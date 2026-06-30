package dev

import (
	"bytes"
	"net/http"
	"path"
	"strings"
	"sync"
)

const liveEndpoint = "/_veta/live"

var liveReloadScript = []byte(`
<script>
(function () {
  var source = new EventSource('/_veta/live');
  source.addEventListener('reload', function () {
    window.location.reload();
  });
})();
</script>
`)

// newHandler returns the HTTP handler used by the development server.
func newHandler(
	outputDir string,
	broadcaster *broadcaster,
	generatedHTML *generatedHTMLFiles,
) http.Handler {
	files := http.FileServer(http.Dir(outputDir))
	mux := http.NewServeMux()
	mux.Handle(liveEndpoint, broadcaster)
	mux.Handle("/", injectHTMLHandler(files, generatedHTML.matchesRequest))

	return mux
}

// injectHTMLHandler injects live reload into successful HTML file responses.
func injectHTMLHandler(next http.Handler, matchesRequest func(*http.Request) bool) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		buffer := newBufferedResponseWriter()
		next.ServeHTTP(buffer, request)

		for key, values := range buffer.header {
			for _, value := range values {
				writer.Header().Add(key, value)
			}
		}

		body := buffer.body.Bytes()
		if shouldInjectLiveReload(request, buffer, matchesRequest) {
			body = injectLiveReload(body)
			writer.Header().Del("Content-Length")
		}

		writer.WriteHeader(buffer.statusCode())
		if request.Method == http.MethodHead {
			return
		}
		_, _ = writer.Write(body)
	})
}

// shouldInjectLiveReload reports whether a buffered response is an HTML page.
func shouldInjectLiveReload(
	request *http.Request,
	response *bufferedResponseWriter,
	matchesRequest func(*http.Request) bool,
) bool {
	if request.Method != http.MethodGet || response.statusCode() != http.StatusOK {
		return false
	}
	if matchesRequest != nil && !matchesRequest(request) {
		return false
	}

	contentType := response.header.Get("Content-Type")
	return strings.Contains(strings.ToLower(contentType), "text/html")
}

// injectLiveReload inserts the dev live-reload script into an HTML document.
func injectLiveReload(content []byte) []byte {
	bodyClose := []byte("</body>")
	index := bytes.LastIndex(bytes.ToLower(content), bodyClose)
	if index == -1 {
		injected := make([]byte, 0, len(content)+len(liveReloadScript))
		injected = append(injected, content...)
		injected = append(injected, liveReloadScript...)
		return injected
	}

	injected := make([]byte, 0, len(content)+len(liveReloadScript))
	injected = append(injected, content[:index]...)
	injected = append(injected, liveReloadScript...)
	injected = append(injected, content[index:]...)
	return injected
}

type bufferedResponseWriter struct {
	body            bytes.Buffer
	header          http.Header
	statusCodeValue int
}

// newBufferedResponseWriter creates a response writer that captures responses.
func newBufferedResponseWriter() *bufferedResponseWriter {
	return &bufferedResponseWriter{header: http.Header{}}
}

// Header returns the buffered response headers.
func (writer *bufferedResponseWriter) Header() http.Header {
	return writer.header
}

// Write appends content to the buffered response body.
func (writer *bufferedResponseWriter) Write(content []byte) (int, error) {
	if writer.statusCodeValue == 0 {
		writer.statusCodeValue = http.StatusOK
	}

	return writer.body.Write(content)
}

// WriteHeader stores the buffered response status code.
func (writer *bufferedResponseWriter) WriteHeader(statusCode int) {
	if writer.statusCodeValue != 0 {
		return
	}

	writer.statusCodeValue = statusCode
}

// statusCode returns the response status code, defaulting to 200.
func (writer *bufferedResponseWriter) statusCode() int {
	if writer.statusCodeValue == 0 {
		return http.StatusOK
	}

	return writer.statusCodeValue
}

type generatedHTMLFiles struct {
	mutex sync.RWMutex
	paths map[string]struct{}
}

// newGeneratedHTMLFiles creates a generated HTML output path set.
func newGeneratedHTMLFiles(files []string) *generatedHTMLFiles {
	generated := &generatedHTMLFiles{}
	generated.update(files)
	return generated
}

// update replaces the generated HTML output path set.
func (files *generatedHTMLFiles) update(paths []string) {
	updated := map[string]struct{}{}
	for _, filePath := range paths {
		filePath = path.Clean(strings.ReplaceAll(filePath, "\\", "/"))
		if strings.EqualFold(path.Ext(filePath), ".html") {
			updated[filePath] = struct{}{}
		}
	}

	files.mutex.Lock()
	files.paths = updated
	files.mutex.Unlock()
}

// matchesRequest reports whether request maps to a generated HTML output path.
func (files *generatedHTMLFiles) matchesRequest(request *http.Request) bool {
	if files == nil {
		return false
	}

	files.mutex.RLock()
	defer files.mutex.RUnlock()

	for _, candidate := range htmlRequestCandidates(request.URL.Path) {
		if _, ok := files.paths[candidate]; ok {
			return true
		}
	}

	return false
}

// htmlRequestCandidates returns generated output paths that could serve urlPath.
func htmlRequestCandidates(urlPath string) []string {
	if strings.HasSuffix(urlPath, "/") {
		return []string{strings.TrimPrefix(path.Join(urlPath, "index.html"), "/")}
	}

	cleanPath := strings.TrimPrefix(path.Clean("/"+urlPath), "/")
	if cleanPath == "." || cleanPath == "" {
		cleanPath = "index.html"
	}

	return []string{cleanPath}
}
