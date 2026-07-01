package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreate(t *testing.T) {
	root := filepath.Join(t.TempDir(), "site")

	result, err := Create(Config{Root: root})
	require.NoError(t, err)
	require.Equal(t, root, result.Root)
	require.Contains(t, result.Directories, "pages")
	require.NotContains(t, result.Directories, "styles")
	require.Contains(t, result.Files, "veta.yaml")
	require.Contains(t, result.Files, "public/styles.css")
	require.FileExists(t, filepath.Join(root, "veta.yaml"))
	require.FileExists(t, filepath.Join(root, "pages", "site.js"))
	require.FileExists(t, filepath.Join(root, "public", "styles.css"))
	require.FileExists(t, filepath.Join(root, "templates", "base.html"))
	config, err := os.ReadFile(filepath.Join(root, "veta.yaml"))
	require.NoError(t, err)
	require.Contains(t, string(config), "# Veta configuration file.")
	require.Contains(t, string(config), "https://veta.varavel.com/config")
	require.Contains(t, string(config), "\nbuild:\n")
	require.Contains(t, string(config), "\ndev:\n")
	require.Contains(t, string(config), "watch: []")
	require.Contains(t, string(config), "\nhtml:\n")
	require.Contains(t, string(config), "\ntailwindcss:\n")
	require.Contains(t, string(config), "# theme:")
	require.Contains(t, string(config), "#   source:")
	require.NotContains(t, string(config), "input:")
	require.NotContains(t, string(config), "output: styles.css")
	pages, err := os.ReadFile(filepath.Join(root, "pages", "site.js"))
	require.NoError(t, err)
	require.Contains(t, string(pages), `template: "base"`)
	require.Contains(t, string(pages), "https://veta.varavel.com/pages")
	require.Contains(t, string(pages), "{ data, files, httpClient }")
	require.NotContains(t, string(pages), "layout:")
	require.FileExists(t, filepath.Join(root, "components", "note.html"))
}

func TestCreateRefusesExistingFiles(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "veta.yaml"), []byte("old"), 0o644))

	_, err := Create(Config{Root: root})
	require.ErrorIs(t, err, ErrFileExists)
	var existingFiles ExistingFilesError
	require.ErrorAs(t, err, &existingFiles)
	require.Equal(t, []string{"veta.yaml"}, existingFiles.Paths)
}

func TestCreateForceOverwritesExistingFiles(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "veta.yaml"), []byte("old"), 0o644))

	_, err := Create(Config{Force: true, Root: root})
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(root, "veta.yaml"))
	require.NoError(t, err)
	require.Contains(t, string(content), "stylesheet: styles.css")
	require.NotContains(t, string(content), "input: public/styles.css")
}

func TestCreateErrors(t *testing.T) {
	_, err := Create(Config{Root: "bad\x00root"})
	require.ErrorIs(t, err, ErrRootInvalid)
}
