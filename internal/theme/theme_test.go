package theme

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestResolveWithoutTheme(t *testing.T) {
	projectFiles := fstest.MapFS{
		"templates/base.pongo": {Data: []byte("project")},
	}

	site, err := Resolve(projectFiles, "")
	require.NoError(t, err)
	require.Equal(t, projectFiles, site.Files)
	require.Equal(t, projectFiles, site.Project)
	require.Nil(t, site.Theme)

	content, err := fs.ReadFile(site.Files, "templates/base.pongo")
	require.NoError(t, err)
	require.Equal(t, "project", string(content))
}

func TestResolveLocalTheme(t *testing.T) {
	root := t.TempDir()
	writeThemeFile(t, root, "themes/basic/templates/base.pongo", "theme base")
	writeThemeFile(t, root, "themes/basic/templates/theme-only.pongo", "theme only")
	writeThemeFile(
		t,
		root,
		"themes/basic/pages/ignored.js",
		"export default function() { return []; }",
	)
	writeThemeFile(t, root, "themes/basic/private/secret.txt", "secret")

	projectFiles := fstest.MapFS{
		"templates/base.pongo":         {Data: []byte("project base")},
		"templates/project-only.pongo": {Data: []byte("project only")},
	}

	site, err := Resolve(projectFiles, "themes/basic", WithRoot(root))
	require.NoError(t, err)
	require.NotNil(t, site.Theme)
	require.Equal(t, filepath.Join(root, "themes/basic"), site.Source)

	content, err := fs.ReadFile(site.Files, "templates/base.pongo")
	require.NoError(t, err)
	require.Equal(t, "project base", string(content))

	content, err = fs.ReadFile(site.Files, "templates/theme-only.pongo")
	require.NoError(t, err)
	require.Equal(t, "theme only", string(content))

	content, err = fs.ReadFile(site.Files, "templates/project-only.pongo")
	require.NoError(t, err)
	require.Equal(t, "project only", string(content))

	_, err = fs.ReadFile(site.Files, "pages/ignored.js")
	require.True(t, errors.Is(err, fs.ErrNotExist))

	_, err = fs.ReadFile(site.Files, "private/secret.txt")
	require.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestResolveErrors(t *testing.T) {
	_, err := Resolve(nil, "")
	require.ErrorIs(t, err, ErrProjectFSRequired)

	_, err = Resolve(fstest.MapFS{}, "theme", WithRoot(""))
	require.ErrorIs(t, err, ErrRootInvalid)

	_, err = Resolve(fstest.MapFS{}, "varavelio/veta-theme-clean@v1.0.0")
	require.ErrorIs(t, err, ErrRemoteUnsupported)

	_, err = Resolve(fstest.MapFS{}, "https://example.com/theme.git")
	require.ErrorIs(t, err, ErrRemoteUnsupported)

	_, err = Resolve(fstest.MapFS{}, "missing")
	require.ErrorIs(t, err, ErrSourceInvalid)

	root := t.TempDir()
	writeThemeFile(t, root, "theme.txt", "not a directory")
	_, err = Resolve(fstest.MapFS{}, "theme.txt", WithRoot(root))
	require.ErrorIs(t, err, ErrSourceInvalid)
}

func writeThemeFile(t *testing.T, root, name, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(name))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
