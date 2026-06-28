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
	require.FileExists(t, filepath.Join(projectRoot, "public", "styles.css"))
	requirePathMissing(t, filepath.Join(projectRoot, "styles"))
	require.Contains(t, readProjectFile(t, projectRoot, "veta.yaml"), "input: public/styles.css")
	require.Equal(
		t,
		"@import \"tailwindcss\";\n",
		readProjectFile(t, projectRoot, "public/styles.css"),
	)

	buildResult := runVeta(
		t,
		workspace,
		"build",
		"--config",
		filepath.Join(projectRoot, "veta.yaml"),
	)
	buildResult.requireSuccess(t)
	require.Contains(t, buildResult.stdout, "Built 2 page(s)")

	index := readProjectFile(t, projectRoot, "dist/index.html")
	require.Contains(t, index, `<link rel="stylesheet" href="/styles.css">`)
	require.Contains(t, index, `<strong>Veta</strong>`)
	require.Contains(t, index, `rounded-2xl`)
	require.Contains(t, index, `href="/about/"`)

	about := readProjectFile(t, projectRoot, "dist/about/index.html")
	require.Contains(t, about, `<h1>About</h1>`)
	require.Equal(
		t,
		"User-agent: *\nAllow: /\n",
		readProjectFile(t, projectRoot, "dist/robots.txt"),
	)

	styles := readProjectFile(t, projectRoot, "dist/styles.css")
	require.NotContains(t, styles, `@import "tailwindcss"`)
	require.Greater(t, len(styles), 100)
}
