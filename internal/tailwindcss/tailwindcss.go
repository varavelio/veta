package tailwindcss

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// Config contains Tailwind CSS build settings.
type Config struct {
	Input   string
	Minify  bool
	Output  string
	WorkDir string
}

// Option configures Tailwind CSS builds.
type Option func(*buildConfig) error

type buildConfig struct {
	binary         Binary
	cacheDir       string
	executablePath string
	stderr         io.Writer
	stdout         io.Writer
}

// WithStderr configures where Tailwind stderr output is written.
func WithStderr(writer io.Writer) Option {
	return func(config *buildConfig) error {
		config.stderr = writer
		return nil
	}
}

// WithStdout configures where Tailwind stdout output is written.
func WithStdout(writer io.Writer) Option {
	return func(config *buildConfig) error {
		config.stdout = writer
		return nil
	}
}

// Build runs Tailwind CSS and writes the generated CSS output file.
func Build(
	ctx context.Context,
	files fs.FS,
	config Config,
	options ...Option,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if files == nil {
		return fmt.Errorf("%w: filesystem is required", ErrConfigInvalid)
	}

	buildConfig, err := newBuildConfig(options)
	if err != nil {
		return err
	}
	inputPath, outputPath, workDir, err := cleanConfigPaths(config)
	if err != nil {
		return err
	}
	executable, err := executablePath(buildConfig)
	if err != nil {
		return err
	}

	inputContent, err := readInputFile(files, inputPath)
	if err != nil {
		return err
	}
	inputFile, cleanup, err := materializeInputFile(inputContent, outputPath)
	if err != nil {
		return err
	}
	defer cleanup()

	if err := runTailwind(
		ctx,
		executable,
		workDir,
		inputFile,
		outputPath,
		config.Minify,
		buildConfig,
	); err != nil {
		return err
	}

	return nil
}

// newBuildConfig applies options and defaults.
func newBuildConfig(options []Option) (buildConfig, error) {
	config := buildConfig{binary: embeddedBinaryAsset()}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return buildConfig{}, err
		}
	}

	return config, nil
}

// cleanConfigPaths validates Tailwind input, output, and working directory paths.
func cleanConfigPaths(config Config) (string, string, string, error) {
	inputPath, err := cleanRelativePath(config.Input)
	if err != nil {
		return "", "", "", fmt.Errorf("%w: tailwindcss input: %w", ErrConfigInvalid, err)
	}
	workDir, err := cleanWorkDir(config.WorkDir)
	if err != nil {
		return "", "", "", err
	}
	outputPath, err := cleanOutputPath(config.Output, workDir)
	if err != nil {
		return "", "", "", err
	}

	return inputPath, outputPath, workDir, nil
}

// runTailwind runs the standalone Tailwind CSS executable.
func runTailwind(
	ctx context.Context,
	executable string,
	workDir string,
	inputPath string,
	outputPath string,
	minify bool,
	config buildConfig,
) error {
	args := []string{"-i", inputPath, "-o", outputPath}
	if minify {
		args = append(args, "--minify")
	}

	command := exec.CommandContext(ctx, executable, args...)
	command.Dir = workDir
	command.Stdout = config.stdout
	var stderr bytes.Buffer
	if config.stderr == nil {
		command.Stderr = &stderr
	} else {
		command.Stderr = io.MultiWriter(config.stderr, &stderr)
	}
	if err := command.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			return fmt.Errorf("%w: %w", ErrRunFailed, err)
		}

		return fmt.Errorf("%w: %w: %s", ErrRunFailed, err, message)
	}

	return nil
}

// cleanRelativePath validates a slash-separated relative file path.
func cleanRelativePath(name string) (string, error) {
	rawName := strings.TrimSpace(name)
	if rawName == "" || strings.ContainsRune(rawName, 0) || filepath.VolumeName(rawName) != "" ||
		hasWindowsVolumeName(rawName) || filepath.IsAbs(rawName) {
		return "", ErrConfigInvalid
	}

	rawName = strings.ReplaceAll(rawName, "\\", "/")
	if path.IsAbs(rawName) || slices.Contains(strings.Split(rawName, "/"), "..") {
		return "", ErrConfigInvalid
	}

	cleanPath := path.Clean(rawName)
	if cleanPath == "." || strings.HasSuffix(cleanPath, "/") || !fs.ValidPath(cleanPath) {
		return "", ErrConfigInvalid
	}

	return cleanPath, nil
}

// cleanWorkDir validates and normalizes the Tailwind scan directory.
func cleanWorkDir(name string) (string, error) {
	workDir := strings.TrimSpace(name)
	if workDir == "" || strings.ContainsRune(workDir, 0) {
		return "", fmt.Errorf("%w: tailwindcss work directory is invalid", ErrConfigInvalid)
	}
	workDir = normalizeFilesystemPath(workDir)
	info, err := os.Stat(workDir)
	if err != nil {
		return "", fmt.Errorf(
			"%w: tailwindcss work directory %s: %w",
			ErrConfigInvalid,
			workDir,
			err,
		)
	}
	if !info.IsDir() {
		return "", fmt.Errorf(
			"%w: tailwindcss work directory %s is not a directory",
			ErrConfigInvalid,
			workDir,
		)
	}

	return workDir, nil
}

// cleanOutputPath validates and normalizes the generated CSS path.
func cleanOutputPath(name, workDir string) (string, error) {
	outputPath := strings.TrimSpace(name)
	if outputPath == "" || strings.ContainsRune(outputPath, 0) {
		return "", fmt.Errorf("%w: tailwindcss output is invalid", ErrConfigInvalid)
	}
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(workDir, outputPath)
	}
	outputPath = filepath.Clean(outputPath)
	if info, err := os.Stat(outputPath); err == nil && info.IsDir() {
		return "", fmt.Errorf(
			"%w: tailwindcss output %s is a directory",
			ErrConfigInvalid,
			outputPath,
		)
	} else if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("%w: tailwindcss output %s: %w", ErrConfigInvalid, outputPath, err)
	}

	return outputPath, nil
}

// readInputFile reads the configured Tailwind input from files.
func readInputFile(files fs.FS, inputPath string) ([]byte, error) {
	info, err := fs.Stat(files, inputPath)
	if err != nil {
		return nil, fmt.Errorf("%w: tailwindcss input %s: %w", ErrConfigInvalid, inputPath, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf(
			"%w: tailwindcss input %s is a directory",
			ErrConfigInvalid,
			inputPath,
		)
	}

	content, err := fs.ReadFile(files, inputPath)
	if err != nil {
		return nil, fmt.Errorf("read tailwindcss input %s: %w", inputPath, err)
	}

	return content, nil
}

// materializeInputFile writes the Tailwind input next to the configured output.
func materializeInputFile(content []byte, outputPath string) (string, func(), error) {
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create tailwindcss output parent: %w", err)
	}

	file, err := os.CreateTemp(outputDir, ".veta-tailwind-input-*.css")
	if err != nil {
		return "", nil, fmt.Errorf("create tailwindcss input: %w", err)
	}
	path := file.Name()
	cleanup := func() {
		_ = os.Remove(path)
	}
	if _, err := file.Write(content); err != nil {
		_ = file.Close()
		cleanup()
		return "", nil, fmt.Errorf("write tailwindcss input: %w", err)
	}
	if err := file.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("close tailwindcss input: %w", err)
	}

	return path, cleanup, nil
}

// hasWindowsVolumeName reports whether a path starts with a Windows drive name.
func hasWindowsVolumeName(name string) bool {
	return len(name) >= 2 && name[1] == ':' &&
		('A' <= name[0] && name[0] <= 'Z' || 'a' <= name[0] && name[0] <= 'z')
}

// normalizeFilesystemPath converts path to an absolute clean path when possible.
func normalizeFilesystemPath(name string) string {
	if filepath.IsAbs(name) {
		return filepath.Clean(name)
	}
	absolutePath, err := filepath.Abs(name)
	if err != nil {
		return filepath.Clean(name)
	}

	return filepath.Clean(absolutePath)
}
