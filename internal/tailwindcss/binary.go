package tailwindcss

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/varavelio/veta/internal/dirs"
)

const (
	// Version is the bundled Tailwind CSS standalone CLI version.
	Version = "v4.3.1"

	execFileMode = 0o755
	fileMode     = 0o644
)

// Binary contains one Tailwind standalone executable.
type Binary struct {
	Content []byte
	Name    string
	SHA256  string
}

// WithBinary configures the Tailwind executable bytes used by Build.
func WithBinary(binary Binary) Option {
	return func(config *buildConfig) error {
		if binary.Name == "" || strings.ContainsRune(binary.Name, 0) ||
			filepath.Base(binary.Name) != binary.Name || len(binary.Content) == 0 ||
			binary.SHA256 == "" {
			return ErrBinaryUnavailable
		}

		config.binary = binary
		return nil
	}
}

// WithCacheDir configures the directory used for materialized executables.
func WithCacheDir(cacheDir string) Option {
	return func(config *buildConfig) error {
		cacheDir = strings.TrimSpace(cacheDir)
		if cacheDir == "" || strings.ContainsRune(cacheDir, 0) {
			return ErrCacheDirInvalid
		}

		config.cacheDir = normalizeFilesystemPath(cacheDir)
		return nil
	}
}

// WithExecutablePath configures an existing Tailwind executable path.
func WithExecutablePath(path string) Option {
	return func(config *buildConfig) error {
		path = strings.TrimSpace(path)
		if path == "" || strings.ContainsRune(path, 0) {
			return ErrExecutableInvalid
		}

		config.executablePath = normalizeFilesystemPath(path)
		return nil
	}
}

// embeddedBinaryAsset returns the platform-specific embedded Tailwind binary.
func embeddedBinaryAsset() Binary {
	return Binary{Content: embeddedBinary, Name: embeddedBinaryName, SHA256: embeddedBinarySHA256}
}

// executablePath returns the Tailwind executable path for config.
func executablePath(config buildConfig) (string, error) {
	if config.executablePath != "" {
		return config.executablePath, nil
	}

	return materializeBinary(config)
}

// materializeBinary writes the embedded executable into the Veta cache.
func materializeBinary(config buildConfig) (string, error) {
	binary := config.binary
	if len(binary.Content) == 0 || binary.Name == "" || binary.SHA256 == "" {
		if embeddedBinaryUnsupported {
			return "", ErrPlatformUnsupported
		}

		return "", ErrBinaryUnavailable
	}

	cacheRoot := config.cacheDir
	if cacheRoot == "" {
		var err error
		cacheRoot, err = dirs.CachePath("tailwindcss")
		if err != nil {
			return "", err
		}
	}

	targetPath := filepath.Join(cacheRoot, Version, binary.Name)
	if ok, err := executableHashMatches(targetPath, binary.SHA256); err != nil {
		return "", err
	} else if ok {
		if err := chmodExecutable(targetPath); err != nil {
			return "", err
		}

		return targetPath, nil
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", fmt.Errorf("create tailwindcss cache: %w", err)
	}

	actualHash := sha256.Sum256(binary.Content)
	if !strings.EqualFold(hex.EncodeToString(actualHash[:]), binary.SHA256) {
		return "", fmt.Errorf("%w: embedded checksum mismatch", ErrBinaryUnavailable)
	}

	tempPath := targetPath + ".tmp"
	if err := os.WriteFile(tempPath, binary.Content, fileMode); err != nil {
		return "", fmt.Errorf("write tailwindcss executable: %w", err)
	}
	defer func() {
		_ = os.Remove(tempPath)
	}()
	if err := chmodExecutable(tempPath); err != nil {
		return "", err
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		return "", fmt.Errorf("install tailwindcss executable: %w", err)
	}

	return targetPath, nil
}

// executableHashMatches reports whether path has the expected checksum.
func executableHashMatches(path, wantHash string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, fmt.Errorf("read cached tailwindcss executable: %w", err)
	}

	hash := sha256.Sum256(content)
	return strings.EqualFold(hex.EncodeToString(hash[:]), wantHash), nil
}

// chmodExecutable marks path executable on Unix systems.
func chmodExecutable(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	if err := os.Chmod(path, execFileMode); err != nil {
		return fmt.Errorf("chmod tailwindcss executable: %w", err)
	}

	return nil
}
