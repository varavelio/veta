package theme

import (
	"archive/zip"
	"bytes"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
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

func TestResolveRemoteTheme(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			requests.Add(1)
			require.Equal(t, "/varavelio/veta-theme-basic/zip/main", request.URL.Path)
			writer.Header().Set("Content-Type", "application/zip")
			_, err := writer.Write(themeArchive(t, map[string]string{
				"veta-theme-basic-main/templates/base.pongo":       "theme base",
				"veta-theme-basic-main/templates/theme-only.pongo": "theme only",
				"veta-theme-basic-main/pages/ignored.js":           "ignored",
				"veta-theme-basic-main/private/secret.txt":         "secret",
			}))
			require.NoError(t, err)
		}),
	)
	defer server.Close()

	projectFiles := fstest.MapFS{
		"templates/base.pongo":         {Data: []byte("project base")},
		"templates/project-only.pongo": {Data: []byte("project only")},
	}
	cacheDir := t.TempDir()

	site, err := Resolve(
		projectFiles,
		"varavelio/veta-theme-basic@main",
		WithCacheDir(cacheDir),
		WithGitHubBaseURL(server.URL),
		WithHTTPClient(server.Client()),
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), requests.Load())
	require.FileExists(t, filepath.Join(filepath.Dir(site.Source), archiveFileName))

	content, err := fs.ReadFile(site.Files, "templates/base.pongo")
	require.NoError(t, err)
	require.Equal(t, "project base", string(content))

	content, err = fs.ReadFile(site.Files, "templates/theme-only.pongo")
	require.NoError(t, err)
	require.Equal(t, "theme only", string(content))

	_, err = fs.ReadFile(site.Files, "pages/ignored.js")
	require.True(t, errors.Is(err, fs.ErrNotExist))

	site, err = Resolve(
		projectFiles,
		"varavelio/veta-theme-basic@main",
		WithCacheDir(cacheDir),
		WithGitHubBaseURL(server.URL),
		WithHTTPClient(server.Client()),
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), requests.Load())
	require.DirExists(t, site.Source)
}

func TestResolveErrors(t *testing.T) {
	_, err := Resolve(nil, "")
	require.ErrorIs(t, err, ErrProjectFSRequired)

	_, err = Resolve(fstest.MapFS{}, "theme", WithRoot(""))
	require.ErrorIs(t, err, ErrRootInvalid)

	_, err = Resolve(fstest.MapFS{}, "varavelio/theme-clean@v1.0.0")
	require.ErrorIs(t, err, ErrSourceInvalid)

	_, err = Resolve(fstest.MapFS{}, "https://example.com/theme.git")
	require.ErrorIs(t, err, ErrRemoteUnsupported)

	_, err = Resolve(fstest.MapFS{}, "varavelio/veta-theme-clean", WithCacheDir(""))
	require.ErrorIs(t, err, ErrCacheDirInvalid)

	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			http.Error(writer, "missing", http.StatusNotFound)
		}),
	)
	defer server.Close()
	_, err = Resolve(
		fstest.MapFS{},
		"varavelio/veta-theme-clean@main",
		WithCacheDir(t.TempDir()),
		WithGitHubBaseURL(server.URL),
		WithHTTPClient(server.Client()),
	)
	require.ErrorIs(t, err, ErrDownloadFailed)

	_, err = Resolve(fstest.MapFS{}, "missing")
	require.ErrorIs(t, err, ErrSourceInvalid)

	root := t.TempDir()
	writeThemeFile(t, root, "theme.txt", "not a directory")
	_, err = Resolve(fstest.MapFS{}, "theme.txt", WithRoot(root))
	require.ErrorIs(t, err, ErrSourceInvalid)
}

func TestParseRemoteReference(t *testing.T) {
	reference, err := parseRemoteReference("varavelio/veta-theme-clean@v1.0.0")
	require.NoError(t, err)
	require.Equal(
		t,
		remoteReference{Owner: "varavelio", Ref: "v1.0.0", Repo: "veta-theme-clean"},
		reference,
	)

	for _, source := range []string{
		"varavelio/veta-clean@v1.0.0",
		"varavelio/veta-theme-clean",
		"varavelio/veta-theme-clean@",
		"varavelio/veta-theme-clean@bad ref",
		"../veta-theme-clean@main",
	} {
		t.Run(source, func(t *testing.T) {
			_, err := parseRemoteReference(source)
			require.ErrorIs(t, err, ErrSourceInvalid)
		})
	}
}

func writeThemeFile(t *testing.T, root, name, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(name))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func themeArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range files {
		file, err := writer.Create(name)
		require.NoError(t, err)
		_, err = file.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	return buffer.Bytes()
}
