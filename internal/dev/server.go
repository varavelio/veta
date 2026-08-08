package dev

import (
	"bytes"
	"fmt"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"
)

const (
	liveEndpoint       = "/_veta/live"
	liveReloadInterval = time.Second
)

// liveReloadScript returns the polling live-reload script. The revision at
// serve time seeds the baseline so a page served just before a completed build
// is still refreshed by the first poll. Polls time out and never overlap, so a
// temporarily unreachable server is retried instead of breaking the page.
func liveReloadScript(revision uint64) []byte {
	return fmt.Appendf(nil, `
		<script>
			(function () {
				var last = %[3]d;
				var inFlight = false;
				function poll() {
					if (inFlight) {
						return;
					}
					inFlight = true;
					var controller = new AbortController();
					var timeout = setTimeout(function () { controller.abort(); }, 5000);
					fetch('%[1]s', { cache: 'no-store', signal: controller.signal })
						.then(function (response) {
							if (!response.ok) {
								throw new Error('live reload unavailable');
							}
							return response.json();
						})
						.then(function (data) {
							if (typeof data.revision !== 'number') {
								return;
							}
							if (data.revision !== last) {
								window.location.reload();
								return;
							}
							last = data.revision;
						})
						.catch(function () {})
						.finally(function () {
							clearTimeout(timeout);
							inFlight = false;
						});
				}
				poll();
				setInterval(poll, %[2]d);
			})();
		</script>
	`, liveEndpoint, liveReloadInterval/time.Millisecond, revision)
}

// newHandler returns the HTTP handler used by the development server.
func newHandler(
	outputRoot *outputRoot,
	revision *revision,
	generatedHTML *generatedHTMLFiles,
) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(liveEndpoint, revision)
	mux.Handle("/", injectHTMLHandler(
		dirFileHandler{outputRoot: outputRoot},
		generatedHTML.matchesRequest,
		revision,
	))

	return mux
}

// dirFileHandler serves static files from the current build output directory.
// The directory is resolved per request so completed builds swap atomically.
type dirFileHandler struct {
	outputRoot *outputRoot
}

// ServeHTTP serves one request from the current build output directory.
func (handler dirFileHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	http.FileServer(http.Dir(handler.outputRoot.get())).ServeHTTP(writer, request)
}

// injectHTMLHandler injects live reload into successful HTML file responses.
func injectHTMLHandler(
	next http.Handler,
	matchesRequest func(*http.Request) bool,
	revision *revision,
) http.Handler {
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
			body = injectLiveReload(body, revision.current())
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
func injectLiveReload(content []byte, revision uint64) []byte {
	script := liveReloadScript(revision)
	bodyClose := []byte("</body>")
	index := bytes.LastIndex(bytes.ToLower(content), bodyClose)
	if index == -1 {
		injected := make([]byte, 0, len(content)+len(script))
		injected = append(injected, content...)
		injected = append(injected, script...)
		return injected
	}

	injected := make([]byte, 0, len(content)+len(script))
	injected = append(injected, content[:index]...)
	injected = append(injected, script...)
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

// outputRoot holds the current build output directory served to browsers.
type outputRoot struct {
	mutex sync.RWMutex
	dir   string
}

// get returns the current build output directory.
func (root *outputRoot) get() string {
	root.mutex.RLock()
	defer root.mutex.RUnlock()

	return root.dir
}

// set replaces the build output directory and returns the previous one.
func (root *outputRoot) set(dir string) string {
	root.mutex.Lock()
	defer root.mutex.Unlock()

	previous := root.dir
	root.dir = dir
	return previous
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
