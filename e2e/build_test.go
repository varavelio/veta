//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBuildsRichProjectFixture verifies the core features working together.
func TestBuildsRichProjectFixture(t *testing.T) {
	projectRoot := copyTestProject(t, "rich-site")

	result := runVeta(t, projectRoot, "build")
	result.requireSuccess(t)
	require.Contains(t, result.stdout, "Veta built 3 pages to dist in ")

	index := readProjectFile(t, projectRoot, "dist/index.html")
	require.Contains(t, index, `<title>Home Page | Veta E2E</title>`)
	require.Contains(t, index, `<a href="/docs/intro/">Docs</a>`)
	require.Contains(t, index, `data-pages="/;/docs/intro/;/feed.xml;"`)
	require.Contains(
		t,
		index,
		`"rounded-xl border border-sky-400 p-4"`,
	)
	require.Contains(t, index, `data-kind="hero"`)
	require.Contains(t, index, `<h1>Veta E2E</h1>`)
	require.Contains(t, index, `<strong>entire build pipeline</strong>`)
	require.Contains(t, index, `varavelio/veta · Sky`)

	docs := readProjectFile(t, projectRoot, "dist/docs/intro/index.html")
	require.Contains(t, docs, `<title>Intro Guide | Veta E2E</title>`)
	require.Contains(t, docs, `<h1>Intro</h1>`)
	require.Contains(t, docs, `<p>Repo: varavelio/veta</p>`)
	require.Contains(t, docs, `<p>Theme: Sky</p>`)

	feed := strings.TrimSpace(readProjectFile(t, projectRoot, "dist/feed.xml"))
	require.Equal(t, `<feed>stars:42</feed>`, feed)
	require.Equal(t, "Built by Veta E2E\n", readProjectFile(t, projectRoot, "dist/humans.txt"))

	styles := readProjectFile(t, projectRoot, "dist/styles.css")
	require.NotContains(t, styles, `@import "tailwindcss"`)
	require.Greater(t, len(styles), 100)
	adminStyles := readProjectFile(t, projectRoot, "dist/admin.css")
	require.NotContains(t, adminStyles, `@import "tailwindcss"`)
	require.Greater(t, len(adminStyles), 100)
}

// TestBuildSupportsIncludesFromTemplatesAndComponents verifies shared Pongo
// fragments can be included from both page templates and component templates.
func TestBuildSupportsIncludesFromTemplatesAndComponents(t *testing.T) {
	projectRoot := copyTestProject(t, "includes")

	result := runVeta(t, projectRoot, "build")
	result.requireSuccess(t)

	index := readProjectFile(t, projectRoot, "dist/index.html")
	require.Contains(t, index, `<h1>Includes</h1>`)
	require.Contains(t, index, `<section class="component-panel">`)
	require.Contains(t, index, `<strong>slot</strong>`)
	require.Equal(t, 2, strings.Count(index, `class="shared-include"`))
	require.Equal(t, 2, strings.Count(index, `class="nested-include"`))
	require.Equal(t, 2, strings.Count(index, `Include Fixture`))
}

// TestBuildSupportsLoadDataInPongo verifies templates, includes and components
// can load local structured and text data directly from Pongo.
func TestBuildSupportsLoadDataInPongo(t *testing.T) {
	projectRoot := copyTestProject(t, "template-load-data")

	result := runVeta(t, projectRoot, "build")
	result.requireSuccess(t)

	index := readProjectFile(t, projectRoot, "dist/index.html")
	require.Contains(t, index, `<h1>Template Load Data</h1>`)
	require.Contains(t, index, `<a href="/">Home</a>`)
	require.Contains(t, index, `<a href="/docs/">Docs</a>`)
	require.Contains(t, index, `<p data-snippet="include">Plain text from load_data.`)
	require.Contains(t, index, `<aside data-badge="blue">`)
	require.Contains(t, index, `Loaded Badge: <p>Component <strong>slot</strong>.</p>`)
}

// TestBuildSupportsTemplateHelpers verifies portable URLs, regex replacement,
// and Base64 filters in a real build.
func TestBuildSupportsTemplateHelpers(t *testing.T) {
	projectRoot := copyTestProject(t, "template-helpers")

	result := runVeta(t, projectRoot, "build")
	result.requireSuccess(t)
	require.Contains(t, result.stdout, "Veta built 2 pages to dist in ")

	index := readProjectFile(t, projectRoot, "dist/index.html")
	require.Contains(t, index, `<link rel="stylesheet" href='styles.css?v=1#main'>`)
	require.Contains(t, index, `<img src='images/logo.svg' alt="Logo">`)
	require.Contains(t, index, `<a data-link="root" href='.'>Root</a>`)
	require.Contains(t, index, `<a data-link="docs" href='docs/'>Docs</a>`)
	require.Contains(t, index, `<a data-link="self" href=".">Self</a>`)
	require.Contains(t, index, `data-slug="Hello-Veta-"`)
	require.Contains(t, index, `Hello World`)
	require.Contains(t, index, `data-encoded='aGVsbG8='`)
	require.Contains(t, index, `hello`)

	docs := readProjectFile(t, projectRoot, "dist/docs/intro/index.html")
	require.Contains(t, docs, `<link rel="stylesheet" href='../../styles.css?v=1#main'>`)
	require.Contains(t, docs, `<img src='../../images/logo.svg' alt="Logo">`)
	require.Contains(t, docs, `<a data-link="root" href='../../'>Root</a>`)
	require.Contains(t, docs, `<a data-link="docs" href='../'>Docs</a>`)
	require.Contains(t, docs, `<a data-link="self" href=".">Self</a>`)
	require.Contains(t, docs, `data-slug="Docs-Intro-"`)
	require.Contains(t, docs, `Hello World`)
	require.Contains(t, docs, `data-encoded='aGVsbG8='`)
	require.Contains(t, docs, `hello`)
}

// TestBuildMinifiesGeneratedHTMLOnly verifies html.minify affects generated HTML only.
func TestBuildMinifiesGeneratedHTMLOnly(t *testing.T) {
	projectRoot := t.TempDir()
	writeProjectFile(t, projectRoot, "veta.yaml", `
build:
  output: dist
  clean: true
html:
  minify: true
`)
	writeProjectFile(
		t,
		projectRoot,
		"public/public.html",
		"<html>\n  <body> Public HTML </body>\n</html>\n",
	)
	writeProjectFile(t, projectRoot, "pages/site.js", `
export default function() {
  return [
    {
      permalink: "/",
      content: "<!doctype html>\n<html>\n<body>\n  <section class=\"hero\"> Hello </section>\n</body>\n</html>\n"
    },
    {
      permalink: "/feed.xml",
      content: "<feed>\n  <title> Keep XML spacing </title>\n</feed>\n"
    },
    {
      permalink: "/notes.md",
      content: "# Keep Markdown\n\nBody\n"
    },
    {
      permalink: "/data.json",
      content: "{\n  \"message\": \"keep json spacing\"\n}\n"
    },
    {
      permalink: "/readme.txt",
      content: "hello\n  world\n"
    },
    {
      permalink: "/app.js",
      content: "function test() {\n  return true;\n}\n"
    },
    {
      permalink: "/styles.css",
      content: "body {\n  color: red;\n}\n"
    }
  ];
}
`)

	result := runVeta(t, projectRoot, "build")
	result.requireSuccess(t)

	index := readProjectFile(t, projectRoot, "dist/index.html")
	require.NotContains(t, index, "\n")
	require.Contains(t, index, `<section class=hero>Hello</section>`)
	require.Equal(
		t,
		"<feed>\n  <title> Keep XML spacing </title>\n</feed>\n",
		readProjectFile(t, projectRoot, "dist/feed.xml"),
	)
	require.Equal(t, "# Keep Markdown\n\nBody\n", readProjectFile(t, projectRoot, "dist/notes.md"))
	require.Equal(
		t,
		"{\n  \"message\": \"keep json spacing\"\n}\n",
		readProjectFile(t, projectRoot, "dist/data.json"),
	)
	require.Equal(t, "hello\n  world\n", readProjectFile(t, projectRoot, "dist/readme.txt"))
	require.Equal(
		t,
		"function test() {\n  return true;\n}\n",
		readProjectFile(t, projectRoot, "dist/app.js"),
	)
	require.Equal(
		t,
		"body {\n  color: red;\n}\n",
		readProjectFile(t, projectRoot, "dist/styles.css"),
	)
	require.Equal(
		t,
		"<html>\n  <body> Public HTML </body>\n</html>\n",
		readProjectFile(t, projectRoot, "dist/public.html"),
	)
}

// TestBuildReadsProjectFilesFromJavaScript verifies the file APIs in a real build.
func TestBuildReadsProjectFilesFromJavaScript(t *testing.T) {
	projectRoot := copyTestProject(t, "file-api-content")

	result := runVeta(t, projectRoot, "build")
	result.requireSuccess(t)
	require.Contains(t, result.stdout, "Veta built 2 pages to dist in ")

	index := readProjectFile(t, projectRoot, "dist/index.html")
	require.Contains(t, index, `<h1>File API Fixture</h1>`)
	require.Contains(t, index, `<p data-nav="Docs">/docs/</p>`)
	require.Contains(t, index, `<p data-theme="indigo">Readable</p>`)
	require.Contains(t, index, `<article data-source="yaml">`)
	require.Contains(t, index, `<h2>YAML Article</h2>`)
	require.Contains(t, index, `<p>guide,yaml</p>`)
	require.Contains(t, index, `# YAML Body`)
	require.Contains(t, index, `<article data-source="toml">`)
	require.Contains(t, index, `<h2>TOML Article</h2>`)
	require.Contains(t, index, `<p>Veta</p>`)
	require.Contains(t, index, `# TOML Body`)
	require.Contains(t, index, `data-plain-path="content/snippets/plain.md"`)
	require.Contains(t, index, `# Plain Note`)
	require.Contains(t, index, `data-note="Plain text asset."`)
	require.Contains(
		t,
		index,
		`content/articles/toml.md;content/articles/yaml.md;content/snippets/plain.md`,
	)
	require.Contains(t, index, `data-permalinks="/articles/toml/;/articles/yaml/;/snippets/plain/"`)

	files := readProjectFile(t, projectRoot, "dist/files.json")
	require.Contains(t, files, `"tomlTitle": "TOML Article"`)
	require.Contains(t, files, `"yamlTitle": "YAML Article"`)
	require.Contains(t, files, `"/articles/toml/"`)
	require.Contains(t, files, `"content/articles/yaml.md"`)
}

// TestBuildDiscoversConfigFromNestedDirectory verifies root discovery and output cleanup.
func TestBuildDiscoversConfigFromNestedDirectory(t *testing.T) {
	projectRoot := copyTestProject(t, "nested-config")
	nestedDir := filepath.Join(projectRoot, "content", "docs")
	require.NoError(t, os.MkdirAll(nestedDir, 0o755))
	writeProjectFile(t, projectRoot, "site-output/stale.txt", "stale")

	result := runVeta(t, nestedDir, "build")
	result.requireSuccess(t)
	require.Contains(t, result.stdout, "Veta built 2 pages to site-output in ")

	index := readProjectFile(t, projectRoot, "site-output/index.html")
	require.Contains(t, index, `<body data-page="/">`)
	require.Contains(t, index, `href="/docs/getting-started/"`)
	require.Contains(t, index, `>Docs</a>`)
	require.Contains(t, index, `<h1>Nested Config Site</h1>`)

	docs := readProjectFile(t, projectRoot, "site-output/docs/getting-started/index.html")
	require.Contains(t, docs, `<body data-page="/docs/getting-started/">`)
	require.Contains(t, docs, `<p>Nested config works.</p>`)
	require.Equal(t, "nested asset\n", readProjectFile(t, projectRoot, "site-output/asset.txt"))
	requirePathMissing(t, filepath.Join(projectRoot, "site-output", "stale.txt"))
}

// TestBuildComposesLocalThemeWithProjectOverrides verifies theme composition end-to-end.
func TestBuildComposesLocalThemeWithProjectOverrides(t *testing.T) {
	projectRoot := copyTestProject(t, "theme-overrides")

	result := runVeta(t, projectRoot, "build")
	result.requireSuccess(t)
	require.Contains(t, result.stdout, "Veta built 2 pages to dist in ")

	index := readProjectFile(t, projectRoot, "dist/index.html")
	require.Contains(t, index, "Theme brand: Base Theme")
	require.Contains(t, index, "Project: Theme Override Site")
	require.Contains(t, index, `<div class="project-badge"><p>Project component</p>`)
	require.NotContains(t, index, "theme-badge")

	themeOnly := readProjectFile(t, projectRoot, "dist/theme-only/index.html")
	require.Contains(t, themeOnly, `<section data-template="theme-only">`)
	require.Contains(t, themeOnly, `<p>Theme template page</p>`)
	require.Equal(t, "project asset\n", readProjectFile(t, projectRoot, "dist/theme.txt"))
	require.Equal(t, "theme only asset\n", readProjectFile(t, projectRoot, "dist/theme-only.txt"))
}

// TestBuildSupportsTemplateAndComponentInheritance verifies Pongo2 inheritance end-to-end.
func TestBuildSupportsTemplateAndComponentInheritance(t *testing.T) {
	projectRoot := copyTestProject(t, "template-inheritance")

	result := runVeta(t, projectRoot, "build")
	result.requireSuccess(t)
	require.Contains(t, result.stdout, "Veta built 1 page to dist in ")

	index := readProjectFile(t, projectRoot, "dist/index.html")
	require.Contains(t, index, `<title>Inheritance | Base Inheritance Site</title>`)
	require.Contains(t, index, `<body class="article base-body">`)
	require.Contains(t, index, `Article Header / Inheritance Site`)
	require.Contains(t, index, `<section data-template="article">`)
	require.Contains(t, index, `<p data-template-extra="true">extra from page</p>`)
	require.Contains(t, index, `Base Footer / Article Footer`)
	require.Contains(
		t,
		index,
		`"component-shell panel base"`,
	)
	require.Contains(t, index, `data-tone="success"`)
	require.Contains(t, index, `<header>Panel: Nested component</header>`)
	require.Contains(t, index, `<p>Component <strong>slot</strong> from page.</p>`)
	require.Contains(t, index, `base-footer / child-footer`)
}

// TestBuildRendersComponentsFromMultipleContentSources verifies component expansion
// across Markdown files, raw file content, inline generator strings, nesting, and
// nested component directories.
func TestBuildRendersComponentsFromMultipleContentSources(t *testing.T) {
	projectRoot := copyTestProject(t, "component-pipeline")

	result := runVeta(t, projectRoot, "build")
	result.requireSuccess(t)
	require.Contains(t, result.stdout, "Veta built 3 pages to dist in ")

	markdownPage := readProjectFile(t, projectRoot, "dist/markdown/index.html")
	require.Contains(t, markdownPage, `<title>Markdown Components</title>`)
	require.Contains(
		t,
		markdownPage,
		`<body data-source="parse.markdown" data-permalink="/markdown/">`,
	)
	require.Contains(
		t,
		markdownPage,
		`<section class="component-box" data-title="Markdown Source">`,
	)
	require.Contains(t, markdownPage, `<h1>Markdown Component</h1>`)
	require.Contains(t, markdownPage, `<strong>bold</strong>`)
	require.Contains(t, markdownPage, `<div class="component-stack" data-name="outer">`)
	require.Contains(t, markdownPage, `<strong>stack</strong>`)
	require.Contains(
		t,
		markdownPage,
		`<span class="deep-badge" data-label="Deep Folder">Deep Folder</span>`,
	)
	require.Contains(t, markdownPage, `Inline Code`)
	require.Contains(t, markdownPage, `Code Fence`)
	require.NotContains(t, markdownPage, `data-title="Inline Code"`)
	require.NotContains(t, markdownPage, `data-title="Code Fence"`)
	require.NotContains(t, markdownPage, `<box title=`)
	require.NotContains(t, markdownPage, `<stack name=`)
	require.NotContains(t, markdownPage, `<ui-layout-blocks-deep-badge`)

	filePage := readProjectFile(t, projectRoot, "dist/file/index.html")
	require.Contains(t, filePage, `<body data-source="readFile" data-permalink="/file/">`)
	require.Contains(t, filePage, `<section data-source="read-file">`)
	require.Contains(t, filePage, `<section class="component-box" data-title="File Source">`)
	require.Contains(t, filePage, `<em>emphasis</em>`)
	require.Contains(
		t,
		filePage,
		`<span class="deep-badge" data-label="File Deep">File Deep</span>`,
	)
	require.Contains(t, filePage, `<div data-native-html="preserved">Raw HTML remains.</div>`)
	require.NotContains(t, filePage, `<box title=`)
	require.NotContains(t, filePage, `<ui-layout-blocks-deep-badge`)

	inlinePage := readProjectFile(t, projectRoot, "dist/inline/index.html")
	require.Contains(t, inlinePage, `<body data-source="inline-string" data-permalink="/inline/">`)
	require.Contains(t, inlinePage, `<div class="component-stack" data-name="inline">`)
	require.Contains(t, inlinePage, `<section class="component-box" data-title="Inline Nested">`)
	require.Contains(t, inlinePage, `<strong>slot</strong>`)
	require.Contains(
		t,
		inlinePage,
		`<span class="deep-badge" data-label="Inline Deep">Inline Deep</span>`,
	)
	require.NotContains(t, inlinePage, `<box title=`)
	require.NotContains(t, inlinePage, `<stack name=`)
	require.NotContains(t, inlinePage, `<ui-layout-blocks-deep-badge`)
}
