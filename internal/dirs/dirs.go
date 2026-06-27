package dirs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// HomeEnvName is the environment variable that overrides Veta's home.
	HomeEnvName = "VETA_HOME"

	defaultHomeDirName = ".veta"
	cacheDirName       = "cache"
	themesDirName      = "themes"
	directoryMode      = 0o755
)

var (
	lookupEnv = os.LookupEnv
	userHome  = os.UserHomeDir
	tempDir   = os.TempDir
	absPath   = filepath.Abs
	makeDir   = os.MkdirAll
)

// GetHome returns the root Veta runtime directory.
//
// Resolution order:
//  1. Uses VETA_HOME when defined and non-empty.
//  2. Falls back to ~/.veta.
//  3. Uses the system temp directory when no user home is available.
func GetHome() string {
	if customHome, ok := lookupEnv(HomeEnvName); ok {
		customHome = strings.TrimSpace(customHome)
		if customHome != "" {
			return normalizePath(customHome)
		}
	}

	homeDir, err := userHome()
	if err == nil {
		homeDir = strings.TrimSpace(homeDir)
		if homeDir != "" {
			return filepath.Join(homeDir, defaultHomeDirName)
		}
	}

	return filepath.Join(tempDir(), defaultHomeDirName)
}

// GetCacheDir returns the directory used for Veta caches.
func GetCacheDir() (string, error) {
	return ensureDir(filepath.Join(GetHome(), cacheDirName))
}

// GetThemesCacheDir returns the directory used for cached remote themes.
func GetThemesCacheDir() (string, error) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return "", err
	}

	return ensureDir(filepath.Join(cacheDir, themesDirName))
}

// CachePath returns a named path inside the Veta cache directory.
func CachePath(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsRune(name, 0) || filepath.Base(name) != name {
		return "", fmt.Errorf("%w: %q", ErrNameInvalid, name)
	}

	cacheDir, err := GetCacheDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(cacheDir, name), nil
}

// ensureDir creates path when necessary and returns the normalized directory.
func ensureDir(path string) (string, error) {
	path = normalizePath(path)
	if err := makeDir(path, directoryMode); err != nil {
		return "", fmt.Errorf("create directory %q: %w", path, err)
	}

	return path, nil
}

// normalizePath converts path to an absolute clean path when possible.
func normalizePath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}

	absolutePath, err := absPath(path)
	if err != nil {
		return filepath.Clean(path)
	}

	return filepath.Clean(absolutePath)
}
