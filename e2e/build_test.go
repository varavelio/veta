//go:build e2e

package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBuildsRichProjectFixture verifies the core features working together.
func TestBuildsRichProjectFixture(t *testing.T) {
	projectRoot := copyFixtureProject(t, "rich-site")

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
	require.Equal(t, `<p><feed>stars:42</feed></p>`, feed)
	require.Equal(t, "Built by Veta E2E\n", readProjectFile(t, projectRoot, "dist/humans.txt"))

	styles := readProjectFile(t, projectRoot, "dist/styles.css")
	require.NotContains(t, styles, `@import "tailwindcss"`)
	require.Greater(t, len(styles), 100)
}
