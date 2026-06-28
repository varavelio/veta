//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const commandTimeout = 45 * time.Second

var (
	repositoryRoot string
	vetaBinary     string
)

type commandResult struct {
	args     []string
	exitCode int
	stderr   string
	stdout   string
}

// TestMain builds the Veta CLI once for all e2e tests.
func TestMain(m *testing.M) {
	root, err := findRepositoryRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "find repository root: %s\n", err)
		os.Exit(1)
	}
	repositoryRoot = root

	tempDir, err := os.MkdirTemp("", "veta-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create e2e temp dir: %s\n", err)
		os.Exit(1)
	}
	vetaBinary = filepath.Join(tempDir, binaryName())

	if err := buildVetaBinary(repositoryRoot, vetaBinary); err != nil {
		fmt.Fprintf(os.Stderr, "build e2e binary: %s\n", err)
		_ = os.RemoveAll(tempDir)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(tempDir)
	os.Exit(code)
}

// requireSuccess fails the test when the command did not exit successfully.
func (result commandResult) requireSuccess(t *testing.T) {
	t.Helper()
	require.Equalf(
		t,
		0,
		result.exitCode,
		"veta %s failed\nstdout:\n%s\nstderr:\n%s",
		strings.Join(result.args, " "),
		result.stdout,
		result.stderr,
	)
}

// requireFailure fails the test when the command succeeded.
func (result commandResult) requireFailure(t *testing.T) {
	t.Helper()
	require.NotEqualf(
		t,
		0,
		result.exitCode,
		"veta %s unexpectedly succeeded",
		strings.Join(result.args, " "),
	)
}

// buildVetaBinary compiles the CLI used by e2e tests.
func buildVetaBinary(root, destination string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	command := exec.CommandContext(
		ctx,
		"go",
		"build",
		"-trimpath",
		"-o",
		destination,
		"./cmd/veta/.",
	)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build failed: %w\n%s", err, string(output))
	}

	return nil
}

// runVeta executes the e2e CLI binary in workDir.
func runVeta(t *testing.T, workDir string, args ...string) commandResult {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	command := exec.CommandContext(ctx, vetaBinary, args...)
	command.Dir = workDir
	command.Env = isolatedEnvironment(t, workDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	if ctx.Err() != nil {
		t.Fatalf("veta %s timed out after %s", strings.Join(args, " "), commandTimeout)
	}

	return commandResult{
		args:     append([]string(nil), args...),
		exitCode: exitCode(err),
		stderr:   stderr.String(),
		stdout:   stdout.String(),
	}
}

// copyTestProject copies one e2e test project into a temporary directory.
func copyTestProject(t *testing.T, name string) string {
	t.Helper()

	source := filepath.Join(repositoryRoot, "e2e", "tests", name)
	destination := filepath.Join(t.TempDir(), name)
	copyDir(t, source, destination)

	return destination
}

// copyDir recursively copies source into destination.
func copyDir(t *testing.T, source, destination string) {
	t.Helper()

	require.NoError(
		t,
		filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			relativePath, err := filepath.Rel(source, path)
			if err != nil {
				return err
			}
			if relativePath == "." {
				return nil
			}

			target := filepath.Join(destination, relativePath)
			if entry.IsDir() {
				return os.MkdirAll(target, 0o755)
			}

			info, err := entry.Info()
			if err != nil {
				return err
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			return os.WriteFile(target, content, info.Mode().Perm())
		}),
	)
}

// writeProjectFile writes a file inside root, creating parents as needed.
func writeProjectFile(t *testing.T, root, name, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(name))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// readProjectFile reads a slash-separated file path inside root.
func readProjectFile(t *testing.T, root, name string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	require.NoError(t, err)

	return string(content)
}

// requirePathMissing verifies that path does not exist.
func requirePathMissing(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	require.Truef(t, os.IsNotExist(err), "expected %s to be missing", path)
}

// isolatedEnvironment returns an environment that keeps e2e cache files local.
func isolatedEnvironment(t *testing.T, workDir string) []string {
	t.Helper()

	home := filepath.Join(workDir, ".e2e-home")
	cache := filepath.Join(workDir, ".e2e-cache")
	require.NoError(t, os.MkdirAll(home, 0o755))
	require.NoError(t, os.MkdirAll(cache, 0o755))

	environment := append([]string(nil), os.Environ()...)
	environment = append(environment,
		"HOME="+home,
		"XDG_CACHE_HOME="+cache,
		"NO_COLOR=1",
		"TERM=dumb",
	)

	return environment
}

// exitCode returns a process exit code for err.
func exitCode(err error) int {
	if err == nil {
		return 0
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}

	return -1
}

// findRepositoryRoot returns the repository root by walking up from the cwd.
func findRepositoryRoot() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get cwd: %w", err)
	}

	for {
		if fileExists(filepath.Join(directory, "go.mod")) &&
			fileExists(filepath.Join(directory, "Taskfile.yml")) {
			return directory, nil
		}

		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("repository root not found")
		}
		directory = parent
	}
}

// fileExists reports whether path exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// binaryName returns the current platform's CLI binary filename.
func binaryName() string {
	if runtime.GOOS == "windows" {
		return "veta.exe"
	}

	return "veta"
}
