package dirs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetHome(t *testing.T) {
	t.Run("environment override", func(t *testing.T) {
		withDirStubs(t)
		lookupEnv = func(name string) (string, bool) {
			require.Equal(t, HomeEnvName, name)
			return " ./custom-home ", true
		}
		absPath = func(path string) (string, error) {
			require.Equal(t, "./custom-home", path)
			return filepath.Join(string(filepath.Separator), "abs", "custom-home"), nil
		}

		require.Equal(t, filepath.Join(string(filepath.Separator), "abs", "custom-home"), GetHome())
	})

	t.Run("user home fallback", func(t *testing.T) {
		withDirStubs(t)
		lookupEnv = func(string) (string, bool) { return "", false }
		userHome = func() (string, error) { return filepath.Join("home", "user"), nil }

		require.Equal(t, filepath.Join("home", "user", defaultHomeDirName), GetHome())
	})

	t.Run("temp fallback", func(t *testing.T) {
		withDirStubs(t)
		lookupEnv = func(string) (string, bool) { return "", false }
		userHome = func() (string, error) { return "", errors.New("missing") }
		tempDir = func() string { return filepath.Join("tmp") }

		require.Equal(t, filepath.Join("tmp", defaultHomeDirName), GetHome())
	})
}

func TestGetCacheDirs(t *testing.T) {
	root := t.TempDir()
	withDirStubs(t)
	lookupEnv = func(string) (string, bool) { return root, true }

	cacheDir, err := GetCacheDir()
	require.NoError(t, err)
	require.DirExists(t, cacheDir)
	require.Equal(t, filepath.Join(root, cacheDirName), cacheDir)

	themesDir, err := GetThemesCacheDir()
	require.NoError(t, err)
	require.DirExists(t, themesDir)
	require.Equal(t, filepath.Join(root, cacheDirName, themesDirName), themesDir)
}

func TestCachePath(t *testing.T) {
	root := t.TempDir()
	withDirStubs(t)
	lookupEnv = func(string) (string, bool) { return root, true }

	path, err := CachePath("downloads")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, cacheDirName, "downloads"), path)

	_, err = CachePath("nested/name")
	require.ErrorIs(t, err, ErrNameInvalid)

	_, err = CachePath("")
	require.ErrorIs(t, err, ErrNameInvalid)
}

func withDirStubs(t *testing.T) {
	t.Helper()

	originalLookupEnv := lookupEnv
	originalUserHome := userHome
	originalTempDir := tempDir
	originalAbsPath := absPath
	originalMakeDir := makeDir
	t.Cleanup(func() {
		lookupEnv = originalLookupEnv
		userHome = originalUserHome
		tempDir = originalTempDir
		absPath = originalAbsPath
		makeDir = originalMakeDir
	})

	lookupEnv = os.LookupEnv
	userHome = os.UserHomeDir
	tempDir = os.TempDir
	absPath = filepath.Abs
	makeDir = os.MkdirAll
}
