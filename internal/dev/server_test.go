package dev

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInjectLiveReload(t *testing.T) {
	t.Run("inserts before body close", func(t *testing.T) {
		content := []byte("<html><body><main>Hello</main></body></html>")

		injected := string(injectLiveReload(content))

		require.Contains(t, injected, "new EventSource('/_veta/live')")
		require.Less(t, indexOf(t, injected, "new EventSource"), indexOf(t, injected, "</body>"))
		require.Contains(t, injected, "<main>Hello</main>")
	})

	t.Run("appends when body close is missing", func(t *testing.T) {
		content := []byte("<main>Hello</main>")

		injected := string(injectLiveReload(content))

		require.Contains(t, injected, "<main>Hello</main>")
		require.Contains(t, injected, "new EventSource('/_veta/live')")
		require.Greater(t, indexOf(t, injected, "new EventSource"), indexOf(t, injected, "</main>"))
	})
}

func TestInjectHTMLHandlerOnlyInjectsHTMLResponses(t *testing.T) {
	root := t.TempDir()
	writeDevTestFile(t, root, "index.html", "<html><body>Hello</body></html>")
	writeDevTestFile(t, root, "styles.css", "body { color: red; }")
	handler := injectHTMLHandler(http.FileServer(http.Dir(root)), func(*http.Request) bool {
		return true
	})

	html := httptest.NewRecorder()
	handler.ServeHTTP(html, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, html.Code)
	require.Contains(t, html.Body.String(), "new EventSource('/_veta/live')")
	require.NotContains(t, readDevTestFile(t, root, "index.html"), "_veta/live")

	css := httptest.NewRecorder()
	handler.ServeHTTP(
		css,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/styles.css", nil),
	)
	require.Equal(t, http.StatusOK, css.Code)
	require.Equal(t, "body { color: red; }", css.Body.String())
	require.NotContains(t, css.Body.String(), "_veta/live")
}

func TestInjectHTMLHandlerSkipsHeadRequests(t *testing.T) {
	root := t.TempDir()
	writeDevTestFile(t, root, "index.html", "<html><body>Hello</body></html>")
	handler := injectHTMLHandler(http.FileServer(http.Dir(root)), func(*http.Request) bool {
		return true
	})

	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequestWithContext(t.Context(), http.MethodHead, "/", nil),
	)

	require.Equal(t, http.StatusOK, response.Code)
	require.Empty(t, response.Body.String())
}

func TestInjectHTMLHandlerRequiresGeneratedHTMLMatch(t *testing.T) {
	root := t.TempDir()
	writeDevTestFile(t, root, "index.html", "<html><body>Generated</body></html>")
	writeDevTestFile(t, root, "public.html", "<html><body>Public</body></html>")
	handler := injectHTMLHandler(
		http.FileServer(http.Dir(root)),
		newGeneratedHTMLFiles([]string{"index.html"}).matchesRequest,
	)

	generated := httptest.NewRecorder()
	handler.ServeHTTP(
		generated,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil),
	)
	require.Contains(t, generated.Body.String(), "_veta/live")

	public := httptest.NewRecorder()
	handler.ServeHTTP(
		public,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/public.html", nil),
	)
	require.NotContains(t, public.Body.String(), "_veta/live")
}

func indexOf(t *testing.T, content, needle string) int {
	t.Helper()

	index := -1
	for i := range content {
		if len(content[i:]) >= len(needle) && content[i:i+len(needle)] == needle {
			index = i
			break
		}
	}
	require.NotEqual(t, -1, index, "expected %q to contain %q", content, needle)
	return index
}

func writeDevTestFile(t *testing.T, root, name, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(name))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func readDevTestFile(t *testing.T, root, name string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	require.NoError(t, err)
	return string(content)
}
