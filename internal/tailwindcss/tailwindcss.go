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

const renderedDirName = "veta-rendered"

// Config contains Tailwind CSS input and output settings.
type Config struct {
	Input  string
	Minify bool
	Output string
}

// Document is a rendered page available to Tailwind for class scanning.
type Document struct {
	Content []byte
	Path    string
}

// File is a generated Tailwind CSS output file.
type File struct {
	Content []byte
	Path    string
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

// Build runs Tailwind CSS and returns the generated CSS output file.
func Build(
	ctx context.Context,
	files fs.FS,
	documents []Document,
	config Config,
	options ...Option,
) (File, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if files == nil {
		return File{}, fmt.Errorf("%w: filesystem is required", ErrConfigInvalid)
	}

	buildConfig, err := newBuildConfig(options)
	if err != nil {
		return File{}, err
	}
	inputPath, outputPath, err := cleanConfigPaths(config)
	if err != nil {
		return File{}, err
	}
	executable, err := executablePath(buildConfig)
	if err != nil {
		return File{}, err
	}

	tempDir, err := os.MkdirTemp("", "veta-tailwind-*")
	if err != nil {
		return File{}, fmt.Errorf("create tailwindcss workspace: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	sourceRoot := filepath.Join(tempDir, "source")
	if err := materializeFS(files, sourceRoot); err != nil {
		return File{}, err
	}
	if err := materializeDocuments(
		documents,
		filepath.Join(sourceRoot, renderedDirName),
	); err != nil {
		return File{}, err
	}

	inputFile := filepath.Join(sourceRoot, filepath.FromSlash(inputPath))
	if info, err := os.Stat(inputFile); err != nil {
		return File{}, fmt.Errorf("%w: tailwindcss.input %s: %w", ErrConfigInvalid, inputPath, err)
	} else if info.IsDir() {
		return File{}, fmt.Errorf(
			"%w: tailwindcss.input %s is a directory",
			ErrConfigInvalid,
			inputPath,
		)
	}

	generatedFile := filepath.Join(tempDir, "generated.css")
	if err := runTailwind(
		ctx,
		executable,
		sourceRoot,
		inputFile,
		generatedFile,
		config.Minify,
		buildConfig,
	); err != nil {
		return File{}, err
	}

	content, err := os.ReadFile(generatedFile)
	if err != nil {
		return File{}, fmt.Errorf("read tailwindcss output: %w", err)
	}

	return File{Content: content, Path: outputPath}, nil
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

// cleanConfigPaths validates Tailwind input and output paths.
func cleanConfigPaths(config Config) (string, string, error) {
	inputPath, err := cleanRelativePath(config.Input)
	if err != nil {
		return "", "", fmt.Errorf("%w: tailwindcss.input: %w", ErrConfigInvalid, err)
	}
	outputPath, err := cleanRelativePath(config.Output)
	if err != nil {
		return "", "", fmt.Errorf("%w: tailwindcss.output: %w", ErrConfigInvalid, err)
	}

	return inputPath, outputPath, nil
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

// materializeFS copies files into root.
func materializeFS(files fs.FS, root string) error {
	return fs.WalkDir(files, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == "." {
			return os.MkdirAll(root, 0o755)
		}

		target := filepath.Join(root, filepath.FromSlash(name))
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		content, err := fs.ReadFile(files, name)
		if err != nil {
			return fmt.Errorf("read tailwindcss source %s: %w", name, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create tailwindcss source parent %s: %w", name, err)
		}
		if err := os.WriteFile(target, content, 0o644); err != nil {
			return fmt.Errorf("write tailwindcss source %s: %w", name, err)
		}

		return nil
	})
}

// materializeDocuments writes rendered documents for Tailwind class scanning.
func materializeDocuments(documents []Document, root string) error {
	for _, document := range documents {
		cleanPath, err := cleanRelativePath(document.Path)
		if err != nil {
			return fmt.Errorf("%w: rendered document %q: %w", ErrConfigInvalid, document.Path, err)
		}

		target := filepath.Join(root, filepath.FromSlash(cleanPath))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create rendered document parent %s: %w", cleanPath, err)
		}
		if err := os.WriteFile(target, document.Content, 0o644); err != nil {
			return fmt.Errorf("write rendered document %s: %w", cleanPath, err)
		}
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
