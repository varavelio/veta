package build

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/veta/internal/theme"
)

func TestRunBuildsSite(t *testing.T) {
	root := t.TempDir()
	writeProjectFile(t, root, "data/site.json", `{"title":"Veta"}`)
	writeProjectFile(t, root, "filters/shout.js", `
export default function(input) {
  return input.toUpperCase();
}
`)
	writeProjectFile(
		t,
		root,
		"components/card.pongo",
		`<section class="card">{{ content }}</section>`,
	)
	writeProjectFile(t, root, "templates/base.pongo", strings.Join([]string{
		`<!doctype html>`,
		`<title>{{ page.title }}</title>`,
		`<main>{{ content }}</main>`,
		`<footer>{{ site.data.site.title }} {{ "ok"|shout }}</footer>`,
	}, ""))
	writeProjectFile(t, root, "pages/site.js", `
export default function({ data }) {
  return [
    {
      permalink: "/",
      layout: "templates/base",
      title: data.site.title,
      content: "<card>**Hello**</card>"
    },
    {
      permalink: "/raw/",
      content: "# Raw"
    }
  ];
}
`)
	writeProjectFile(t, root, "public/app.css", `body { color: black; }`)

	result, err := Run(context.Background(), WithRoot(root), WithClean(true))
	require.NoError(t, err)
	require.Equal(t, 2, result.Pages)
	require.Equal(t, 2, result.Documents)
	require.Equal(t, filepath.Join(root, DefaultOutputDir), result.OutputDir)

	index := readOutputFile(t, root, "dist/index.html")
	require.Contains(t, index, `<title>Veta</title>`)
	require.Contains(t, index, `<section class="card">`)
	require.Contains(t, index, `<strong>Hello</strong>`)
	require.Contains(t, index, `<footer>Veta OK</footer>`)

	raw := readOutputFile(t, root, "dist/raw/index.html")
	require.Equal(t, "# Raw", raw)

	asset := readOutputFile(t, root, "dist/app.css")
	require.Equal(t, `body { color: black; }`, asset)
}

func TestRunUsesLocalTheme(t *testing.T) {
	root := t.TempDir()
	writeProjectFile(t, root, "veta.yaml", `theme: { source: "./theme" }`)
	writeProjectFile(
		t,
		root,
		"theme/templates/base.pongo",
		`<html><body>{{ content }} {{ site.data.theme.name }}</body></html>`,
	)
	writeProjectFile(t, root, "theme/data/theme.json", `{"name":"Theme"}`)
	writeProjectFile(t, root, "theme/public/theme.css", `theme`)
	writeProjectFile(t, root, "theme/pages/ignored.js", `export default function() { return []; }`)
	writeProjectFile(t, root, "pages/site.js", `
export default function() {
  return [{ permalink: "/", layout: "templates/base", content: "Hello" }];
}
`)

	_, err := Run(context.Background(), WithRoot(root), WithClean(true))
	require.NoError(t, err)
	index := readOutputFile(t, root, "dist/index.html")
	require.Contains(t, index, "<p>Hello</p>")
	require.Contains(t, index, "Theme")
	require.Equal(t, "theme", readOutputFile(t, root, "dist/theme.css"))
}

func TestRunUsesRemoteTheme(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			writer.Header().Set("Content-Type", "application/zip")
			_, err := writer.Write(buildThemeArchive(t, map[string]string{
				"veta-theme-remote-main/templates/base.pongo": `<html><body>{{ content }} {{ site.data.theme.name }}</body></html>`,
				"veta-theme-remote-main/data/theme.json":      `{"name":"Remote"}`,
			}))
			require.NoError(t, err)
		}),
	)
	defer server.Close()

	root := t.TempDir()
	cacheDir := t.TempDir()
	writeProjectFile(t, root, "veta.yaml", `theme: { source: "varavelio/veta-theme-remote@main" }`)
	writeProjectFile(t, root, "pages/site.js", `
export default function() {
  return [{ permalink: "/", layout: "templates/base", content: "Hello" }];
}
`)

	_, err := Run(
		context.Background(),
		WithRoot(root),
		WithClean(true),
		WithThemeOptions(
			theme.WithCacheDir(cacheDir),
			theme.WithGitHubBaseURL(server.URL),
			theme.WithHTTPClient(server.Client()),
		),
	)
	require.NoError(t, err)
	require.Contains(t, readOutputFile(t, root, "dist/index.html"), "Remote")
	require.Equal(t, int64(1), requests.Load())

	_, err = Run(
		context.Background(),
		WithRoot(root),
		WithClean(true),
		WithThemeOptions(
			theme.WithCacheDir(cacheDir),
			theme.WithGitHubBaseURL(server.URL),
			theme.WithHTTPClient(server.Client()),
		),
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), requests.Load())
}

func TestRunErrors(t *testing.T) {
	_, err := Run(context.Background(), WithRoot(""))
	require.ErrorIs(t, err, ErrRootInvalid)

	_, err = Run(context.Background(), WithOutputDir(""))
	require.ErrorIs(t, err, ErrOutputDirInvalid)

	_, err = Run(context.Background(), WithConfigFile("bad\x00file"))
	require.ErrorIs(t, err, ErrConfigFileInvalid)

	root := t.TempDir()
	writeProjectFile(t, root, "veta.yaml", `
tailwindcss:
  input: public/app.css
  output: public/build.css
`)
	_, err = Run(context.Background(), WithRoot(root))
	require.ErrorIs(t, err, ErrTailwindUnsupported)

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = Run(canceled)
	require.True(t, errors.Is(err, context.Canceled))
}

func writeProjectFile(t *testing.T, root, name, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(name))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func readOutputFile(t *testing.T, root, name string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	require.NoError(t, err)
	return string(content)
}

func buildThemeArchive(t *testing.T, files map[string]string) []byte {
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
