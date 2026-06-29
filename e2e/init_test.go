//go:build e2e

package e2e

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestInitBuildsStarterProject verifies the default project that new users see.
func TestInitBuildsStarterProject(t *testing.T) {
	workspace := t.TempDir()

	initResult := runVeta(t, workspace, "init", "site")
	initResult.requireSuccess(t)
	require.Contains(t, initResult.stdout, "Initialized Veta project")
	require.Contains(t, initResult.stdout, "veta build")

	projectRoot := filepath.Join(workspace, "site")
	require.FileExists(t, filepath.Join(projectRoot, "veta.yaml"))
	require.FileExists(t, filepath.Join(projectRoot, "templates", "base.html"))
	require.FileExists(t, filepath.Join(projectRoot, "public", "styles.css"))
	requirePathMissing(t, filepath.Join(projectRoot, "styles"))
	config := readProjectFile(t, projectRoot, "veta.yaml")
	require.Contains(t, config, "# Veta configuration file.")
	require.Contains(t, config, "https://veta.varavel.com/config")
	require.Contains(t, config, "stylesheet: styles.css")
	require.Contains(t, config, "# theme:")
	require.NotContains(t, config, "input: public/styles.css")
	require.NotContains(t, config, "output: styles.css")
	require.Equal(
		t,
		"/* Tailwind CSS entrypoint. Docs: https://veta.varavel.com/tailwindcss */\n@import \"tailwindcss\";\n",
		readProjectFile(t, projectRoot, "public/styles.css"),
	)
	require.Contains(t, readProjectFile(t, projectRoot, "pages/site.js"), "Veta.httpClient")
	require.FileExists(t, filepath.Join(projectRoot, "components", "note.html"))
	require.FileExists(t, filepath.Join(projectRoot, "filters", "label.js"))

	buildResult := runVeta(
		t,
		workspace,
		"build",
		"--config",
		filepath.Join(projectRoot, "veta.yaml"),
	)
	buildResult.requireSuccess(t)
	require.Contains(t, buildResult.stdout, "Veta built 2 pages to dist in ")

	index := readProjectFile(t, projectRoot, "dist/index.html")
	require.Contains(t, index, `<link rel="stylesheet" href="/styles.css">`)
	require.Contains(t, index, `<strong>Veta</strong>`)
	require.Contains(t, index, `Site: Veta Starter`)
	require.Contains(t, index, `<aside class="rounded border border-zinc-200 bg-zinc-50 p-4">`)
	require.Contains(t, index, `href="/about/"`)

	about := readProjectFile(t, projectRoot, "dist/about/index.html")
	require.Contains(t, about, `>About</h1>`)
	require.Contains(t, about, `<code>pages/site.js</code>`)
	require.Equal(
		t,
		"User-agent: *\nAllow: /\n",
		readProjectFile(t, projectRoot, "dist/robots.txt"),
	)

	styles := readProjectFile(t, projectRoot, "dist/styles.css")
	require.NotContains(t, styles, `@import "tailwindcss"`)
	require.Greater(t, len(styles), 100)
}
