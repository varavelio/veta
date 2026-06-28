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
	require.Contains(t, result.stdout, "Built 3 page(s)")

	index := readProjectFile(t, projectRoot, "dist/index.html")
	require.Contains(t, index, `<title>Home Page | Veta E2E</title>`)
	require.Contains(t, index, `<a href="/docs/intro/">Docs</a>`)
	require.Contains(t, index, `data-pages="/;/docs/intro/;/feed.xml;"`)
	require.Contains(
		t,
		index,
		`<aside class="rounded-xl border border-sky-400 p-4" data-kind="hero">`,
	)
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
}

// TestBuildDiscoversConfigFromNestedDirectory verifies root discovery and output cleanup.
func TestBuildDiscoversConfigFromNestedDirectory(t *testing.T) {
	projectRoot := copyTestProject(t, "nested-config")
	nestedDir := filepath.Join(projectRoot, "content", "docs")
	require.NoError(t, os.MkdirAll(nestedDir, 0o755))
	writeProjectFile(t, projectRoot, "site-output/stale.txt", "stale")

	result := runVeta(t, nestedDir, "build")
	result.requireSuccess(t)
	require.Contains(t, result.stdout, "Built 2 page(s)")

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
	require.Contains(t, result.stdout, "Built 2 page(s)")

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
