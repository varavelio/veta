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
export default function(context, input) {
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
	writeProjectFile(
		t,
		root,
		"templates/sitemap.pongo",
		`<sitemap>{{ page.content }}{% for item in pages %}<url>{{ item.permalink }}</url>{% endfor %}</sitemap>`,
	)
	writeProjectFile(t, root, "pages/site.js", `
export default function({ data }) {
  return [
    {
      permalink: "/",
      template: "base",
      title: data.site.title,
      content: "<card>**Hello**</card>"
    },
    {
      permalink: "/raw/",
      content: "# Raw"
    },
    {
      permalink: "/sitemap.xml",
      template: "sitemap"
    },
    {
      permalink: "/empty.txt"
    },
    {
      permalink: "/raw.txt",
      content: "hello"
    }
  ];
}
`)
	writeProjectFile(t, root, "public/app.css", `body { color: black; }`)

	result, err := Run(context.Background(), WithRoot(root))
	require.NoError(t, err)
	require.Equal(t, 5, result.Pages)
	require.Equal(t, 5, result.Documents)
	require.ElementsMatch(
		t,
		[]string{"index.html", "raw/index.html", "sitemap.xml", "empty.txt", "raw.txt"},
		result.GeneratedFiles,
	)
	require.Equal(t, filepath.Join(root, DefaultOutputDir), result.OutputDir)

	index := readOutputFile(t, root, "dist/index.html")
	require.Contains(t, index, `<title>Veta</title>`)
	require.Contains(t, index, `<nav>/;/raw/;/sitemap.xml;/empty.txt;/raw.txt;</nav>`)
	require.Contains(t, index, `<section class="card">`)
	require.Contains(t, index, `<strong>Hello</strong>`)
	require.Contains(t, index, `<footer>Veta OK</footer>`)

	raw := readOutputFile(t, root, "dist/raw/index.html")
	require.Equal(t, "# Raw", raw)
	sitemap := readOutputFile(t, root, "dist/sitemap.xml")
	require.Equal(
		t,
		`<sitemap><url>/</url><url>/raw/</url><url>/sitemap.xml</url><url>/empty.txt</url><url>/raw.txt</url></sitemap>`,
		sitemap,
	)
	require.Equal(t, "", readOutputFile(t, root, "dist/empty.txt"))
	require.Equal(t, "hello", readOutputFile(t, root, "dist/raw.txt"))

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
  return [{ permalink: "/", template: "base", content: "Hello" }];
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
  return [{ permalink: "/", template: "base", content: "Hello" }];
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
  return [{ permalink: "/", template: "base.pongo", content: "Hello" }];
}
`)

	result, err := Run(context.Background(), WithConfigFile(filepath.Join(root, "custom.yaml")))
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "custom-dist"), result.OutputDir)
	require.FileExists(t, filepath.Join(root, "custom-dist", "index.html"))
}

func TestRunUsesRuntimeOutputOverrides(t *testing.T) {
	root := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "dev-output")
	writeProjectFile(t, root, "veta.yaml", `
build:
  output: dist
  clean: false
`)
	writeProjectFile(t, root, "pages/site.js", `
export default function() {
  return [{ permalink: "/", content: "Hello" }];
}
`)

	result, err := Run(
		context.Background(),
		WithRoot(root),
		WithOutputDir(outputDir),
		WithClean(true),
	)
	require.NoError(t, err)
	require.Equal(t, root, result.Root)
	require.Equal(t, outputDir, result.OutputDir)
	require.Equal(t, outputDir, result.Config.Build.Output)
	require.True(t, result.Config.Build.Clean)
	require.FileExists(t, filepath.Join(outputDir, "index.html"))
	_, err = os.Stat(filepath.Join(root, "dist"))
	require.True(t, os.IsNotExist(err))
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
  source: "varavelio/veta-theme-remote@main"`,
	)
	writeProjectFile(t, root, "pages/site.js", `
export default function() {
  return [{ permalink: "/", template: "base", content: "Hello" }];
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
theme:
  source: "./theme"
tailwindcss:
  stylesheet: styles.css
  minify: true
`)
	writeProjectFile(t, root, "theme/public/styles.css", `theme css`)
	writeProjectFile(t, root, "public/styles.css", `@import "tailwindcss";`)
	writeProjectFile(
		t,
		root,
		"templates/base.pongo",
		`<html><head><link href="/styles.css" rel="stylesheet"></head><body>{{ page.content }}</body></html>`,
	)
	writeProjectFile(t, root, "pages/site.js", `
export default function() {
  return [{ permalink: "/", template: "base", content: "<div class=\"text-red-500\">Hello</div>" }];
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
	require.Contains(
		t,
		readOutputFile(t, root, "dist/styles.css"),
		"minify=true rendered=true copied=true input=true",
	)
}

// TestRunMinifiesGeneratedHTMLOnly verifies html.minify is wired to output writing.
func TestRunMinifiesGeneratedHTMLOnly(t *testing.T) {
	root := t.TempDir()
	writeProjectFile(t, root, "veta.yaml", `
build:
  clean: true
html:
  minify: true
`)
	writeProjectFile(t, root, "pages/site.js", `
export default function() {
  return [
    {
      permalink: "/",
      content: "<!doctype html>\n<html>\n<body>\n  <main class=\"home\"> Hello </main>\n</body>\n</html>\n"
    },
    {
      permalink: "/feed.xml",
      content: "<feed>\n  <title> Keep XML spacing </title>\n</feed>\n"
    },
    {
      permalink: "/data.json",
      content: "{\n  \"message\": \"keep json spacing\"\n}\n"
    },
    {
      permalink: "/styles.css",
      content: "body {\n  color: red;\n}\n"
    }
  ];
}
`)

	_, err := Run(context.Background(), WithRoot(root))
	require.NoError(t, err)

	index := readOutputFile(t, root, "dist/index.html")
	require.NotContains(t, index, "\n")
	require.Contains(t, index, `<main class=home>Hello</main>`)
	require.Equal(
		t,
		"<feed>\n  <title> Keep XML spacing </title>\n</feed>\n",
		readOutputFile(t, root, "dist/feed.xml"),
	)
	require.Equal(
		t,
		"{\n  \"message\": \"keep json spacing\"\n}\n",
		readOutputFile(t, root, "dist/data.json"),
	)
	require.Equal(t, "body {\n  color: red;\n}\n", readOutputFile(t, root, "dist/styles.css"))
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
			"set in=",
			"set out=",
			"set minify=false",
			":loop",
			"if \"%1\"==\"\" goto done",
			"if \"%1\"==\"-i\" set in=%2& shift& shift& goto loop",
			"if \"%1\"==\"-o\" set out=%2& shift& shift& goto loop",
			"if \"%1\"==\"--minify\" set minify=true& shift& goto loop",
			"shift",
			"goto loop",
			":done",
			"if exist index.html (set rendered=true) else (set rendered=false)",
			"if exist styles.css (set copied=true) else (set copied=false)",
			"set input=false",
			"if not \"%in%\"==\"\" findstr /C:\"tailwindcss\" \"%in%\" >nul && set input=true",
			"> \"%out%\" echo minify=%minify% rendered=%rendered% copied=%copied% input=%input%",
		}, "\r\n"))
	}

	return []byte(strings.Join([]string{
		"#!/bin/sh",
		"in=",
		"out=",
		"minify=false",
		"while [ $# -gt 0 ]; do",
		"  case \"$1\" in",
		"    -i) in=\"$2\"; shift 2 ;;",
		"    -o) out=\"$2\"; shift 2 ;;",
		"    --minify) minify=true; shift ;;",
		"    *) shift ;;",
		"  esac",
		"done",
		"if [ -f index.html ]; then rendered=true; else rendered=false; fi",
		"if [ -f styles.css ]; then copied=true; else copied=false; fi",
		"if [ -n \"$in\" ] && grep -q 'tailwindcss' \"$in\"; then input=true; else input=false; fi",
		"printf 'minify=%s rendered=%s copied=%s input=%s\\n' \"$minify\" \"$rendered\" \"$copied\" \"$input\" > \"$out\"",
	}, "\n"))
}
