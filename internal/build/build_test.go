package build

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/veta/internal/tailwindcss"
	"github.com/varavelio/veta/internal/theme"
)

func TestRunBuildsSite(t *testing.T) {
	root := t.TempDir()
	writeProjectFile(t, root, "veta.yaml", `
build:
  output: dist
  clean: true
`)
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
		`<section class="card">{{ props.content }}</section>`,
	)
	writeProjectFile(t, root, "templates/base.pongo", strings.Join([]string{
		`<!doctype html>`,
		`<title>{{ page.title }}</title>`,
		`<nav>{% for item in pages %}{{ item.permalink }};{% endfor %}</nav>`,
		`<main>{{ page.content }}</main>`,
		`<footer>{{ data.site.title }} {{ "ok"|shout }}</footer>`,
	}, ""))
	writeProjectFile(t, root, "templates/plain.pongo", `{{ page.content }}`)
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
	      layout: "templates/plain",
	      content: "# Raw"
	    }
  ];
}
`)
	writeProjectFile(t, root, "public/app.css", `body { color: black; }`)

	result, err := Run(context.Background(), WithRoot(root))
	require.NoError(t, err)
	require.Equal(t, 2, result.Pages)
	require.Equal(t, 2, result.Documents)
	require.Equal(t, filepath.Join(root, DefaultOutputDir), result.OutputDir)

	index := readOutputFile(t, root, "dist/index.html")
	require.Contains(t, index, `<title>Veta</title>`)
	require.Contains(t, index, `<nav>/;/raw/;</nav>`)
	require.Contains(t, index, `<section class="card">`)
	require.Contains(t, index, `<strong>Hello</strong>`)
	require.Contains(t, index, `<footer>Veta OK</footer>`)

	raw := readOutputFile(t, root, "dist/raw/index.html")
	require.Equal(t, "<h1>Raw</h1>\n", raw)

	asset := readOutputFile(t, root, "dist/app.css")
	require.Equal(t, `body { color: black; }`, asset)
}

func TestRunUsesLocalTheme(t *testing.T) {
	root := t.TempDir()
	writeProjectFile(t, root, "veta.yaml", `
build:
  clean: true
theme:
  source: "./theme"
`)
	writeProjectFile(
		t,
		root,
		"theme/templates/base.pongo",
		`<html><body>{{ page.content }} {{ data.theme.name }}</body></html>`,
	)
	writeProjectFile(t, root, "theme/data/theme.json", `{"name":"Theme"}`)
	writeProjectFile(t, root, "theme/public/theme.css", `theme`)
	writeProjectFile(t, root, "theme/pages/ignored.js", `export default function() { return []; }`)
	writeProjectFile(t, root, "pages/site.js", `
export default function() {
  return [{ permalink: "/", layout: "templates/base", content: "Hello" }];
}
`)

	_, err := Run(context.Background(), WithRoot(root))
	require.NoError(t, err)
	index := readOutputFile(t, root, "dist/index.html")
	require.Contains(t, index, "<p>Hello</p>")
	require.Contains(t, index, "Theme")
	require.Equal(t, "theme", readOutputFile(t, root, "dist/theme.css"))
}

func TestRunDiscoversConfigFromAncestors(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "content", "docs")
	require.NoError(t, os.MkdirAll(child, 0o755))
	writeProjectFile(t, root, "veta.yaml", `
build:
  output: public-build
  clean: true
`)
	writeProjectFile(t, root, "templates/base.pongo", `{{ page.content }}`)
	writeProjectFile(t, root, "pages/site.js", `
export default function() {
  return [{ permalink: "/", layout: "templates/base", content: "Hello" }];
}
`)

	result, err := Run(context.Background(), WithRoot(child))
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "public-build"), result.OutputDir)
	require.FileExists(t, filepath.Join(root, "public-build", "index.html"))
}

func TestRunUsesExplicitConfigFile(t *testing.T) {
	root := t.TempDir()
	writeProjectFile(t, root, "custom.yaml", `
build:
  output: custom-dist
  clean: true
`)
	writeProjectFile(t, root, "templates/base.pongo", `{{ page.content }}`)
	writeProjectFile(t, root, "pages/site.js", `
export default function() {
  return [{ permalink: "/", layout: "templates/base", content: "Hello" }];
}
`)

	result, err := Run(context.Background(), WithConfigFile(filepath.Join(root, "custom.yaml")))
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "custom-dist"), result.OutputDir)
	require.FileExists(t, filepath.Join(root, "custom-dist", "index.html"))
}

func TestRunUsesRemoteTheme(t *testing.T) {
	var requests atomic.Int64
	archive := buildThemeArchive(t, map[string]string{
		"veta-theme-remote-main/templates/base.pongo": `<html><body>{{ page.content }} {{ data.theme.name }}</body></html>`,
		"veta-theme-remote-main/data/theme.json":      `{"name":"Remote"}`,
	})
	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			writer.Header().Set("Content-Type", "application/zip")
			_, err := writer.Write(archive)
			require.NoError(t, err)
		}),
	)
	defer server.Close()

	root := t.TempDir()
	cacheDir := t.TempDir()
	writeProjectFile(
		t,
		root,
		"veta.yaml",
		`build:
  clean: true
theme:
  source: "varavelio/veta-theme-remote@main"
  sha256: "`+bytesSHA256(archive)+`"`,
	)
	writeProjectFile(t, root, "pages/site.js", `
export default function() {
  return [{ permalink: "/", layout: "templates/base", content: "Hello" }];
}
`)

	_, err := Run(
		context.Background(),
		WithRoot(root),
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
		WithThemeOptions(
			theme.WithCacheDir(cacheDir),
			theme.WithGitHubBaseURL(server.URL),
			theme.WithHTTPClient(server.Client()),
		),
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), requests.Load())
}

func TestRunBuildsTailwindCSS(t *testing.T) {
	root := t.TempDir()
	writeProjectFile(t, root, "veta.yaml", `
build:
  clean: true
tailwindcss:
  input: styles/app.css
  output: app.css
  minify: true
`)
	writeProjectFile(t, root, "styles/app.css", `@import "tailwindcss";`)
	writeProjectFile(t, root, "public/app.css", `public css`)
	writeProjectFile(
		t,
		root,
		"templates/base.pongo",
		`<html><head><link href="/app.css" rel="stylesheet"></head><body>{{ page.content }}</body></html>`,
	)
	writeProjectFile(t, root, "pages/site.js", `
export default function() {
  return [{ permalink: "/", layout: "templates/base", content: "<div class=\"text-red-500\">Hello</div>" }];
}
`)

	_, err := Run(
		context.Background(),
		WithRoot(root),
		WithTailwindOptions(
			tailwindcss.WithBinary(fakeTailwindBinary()),
			tailwindcss.WithCacheDir(t.TempDir()),
		),
	)
	require.NoError(t, err)
	require.Contains(t, readOutputFile(t, root, "dist/app.css"), "minify=true rendered=true")
}

func TestRunErrors(t *testing.T) {
	_, err := Run(context.Background(), WithRoot(""))
	require.ErrorIs(t, err, ErrRootInvalid)

	_, err = Run(context.Background(), WithConfigFile("bad\x00file"))
	require.ErrorIs(t, err, ErrConfigFileInvalid)

	_, err = Run(context.Background(), WithRoot(t.TempDir()))
	require.ErrorIs(t, err, ErrConfigNotFound)

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

// bytesSHA256 returns the SHA-256 hex digest for content.
func bytesSHA256(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

func fakeTailwindBinary() tailwindcss.Binary {
	content := fakeTailwindBinaryContent()
	hash := sha256.Sum256(content)
	name := "tailwindcss-fake"
	if runtime.GOOS == "windows" {
		name += ".cmd"
	}

	return tailwindcss.Binary{Content: content, Name: name, SHA256: hex.EncodeToString(hash[:])}
}

func fakeTailwindBinaryContent() []byte {
	if runtime.GOOS == "windows" {
		return []byte(strings.Join([]string{
			"@echo off",
			"set out=",
			"set minify=false",
			":loop",
			"if \"%1\"==\"\" goto done",
			"if \"%1\"==\"-o\" set out=%2& shift& shift& goto loop",
			"if \"%1\"==\"--minify\" set minify=true& shift& goto loop",
			"shift",
			"goto loop",
			":done",
			"if exist veta-rendered\\index.html (set rendered=true) else (set rendered=false)",
			"> \"%out%\" echo minify=%minify% rendered=%rendered%",
		}, "\r\n"))
	}

	return []byte(strings.Join([]string{
		"#!/bin/sh",
		"out=",
		"minify=false",
		"while [ $# -gt 0 ]; do",
		"  case \"$1\" in",
		"    -o) out=\"$2\"; shift 2 ;;",
		"    --minify) minify=true; shift ;;",
		"    *) shift ;;",
		"  esac",
		"done",
		"if [ -f veta-rendered/index.html ]; then rendered=true; else rendered=false; fi",
		"printf 'minify=%s rendered=%s\\n' \"$minify\" \"$rendered\" > \"$out\"",
	}, "\n"))
}
