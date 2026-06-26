package jsruntime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunnerExecute verifies direct in-memory JavaScript execution.
func TestRunnerExecute(t *testing.T) {
	t.Run("executes default export with runtime argument and Veta global", func(t *testing.T) {
		runner := New(WithRuntime(Runtime{"siteName": "Veta"}))

		result, err := runner.ExecuteString("page.js", `
			const greeting = "Hello";

			export default function(runtime) {
				return {
					title: greeting + ", " + runtime.siteName,
					globalAvailable: Veta.siteName === runtime.siteName,
					keys: Object.keys(runtime).sort()
				};
			}
		`)
		require.NoError(t, err)

		var got map[string]any
		require.NoError(t, result.ExportTo(&got))
		require.Equal(t, "Hello, Veta", got["title"])
		require.Equal(t, true, got["globalAvailable"])
		require.Equal(t, []any{"files", "httpClient", "siteName"}, got["keys"])
	})

	t.Run("supports destructuring the runtime argument", func(t *testing.T) {
		runner := New(WithRuntime(Runtime{"value": 21}))

		result, err := runner.ExecuteString("destructure.js", `
			export default function({ value }) {
				return value * 2;
			}
		`)
		require.NoError(t, err)
		require.Equal(t, int64(42), result.Export())
	})

	t.Run("ignores module words in strings and comments", func(t *testing.T) {
		result, err := New().ExecuteString("strings.js", `
			// import value from "elsewhere";
			const text = "export const hidden = true";

			export default function() {
				return text;
			}
		`)
		require.NoError(t, err)
		require.Equal(t, "export const hidden = true", result.Export())
	})
}

// TestRunnerExecuteFileReadError verifies that file read failures are wrapped
// with useful context.
func TestRunnerExecuteFileReadError(t *testing.T) {
	_, err := New().ExecuteFile(filepath.Join("testdata", "missing.js"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "read javascript file")
}

// TestRunnerRuntimeIsolation verifies that each execution receives a fresh Veta
// object.
func TestRunnerRuntimeIsolation(t *testing.T) {
	runner := New()
	code := `
		export default function() {
			Veta.count = (Veta.count || 0) + 1;
			return Veta.count;
		}
	`

	first, err := runner.ExecuteString("isolation.js", code)
	require.NoError(t, err)
	require.Equal(t, int64(1), first.Export())

	second, err := runner.ExecuteString("isolation.js", code)
	require.NoError(t, err)
	require.Equal(t, int64(1), second.Export())
}

// TestRunnerRuntimeSnapshot verifies that runner configuration is copied before
// execution.
func TestRunnerRuntimeSnapshot(t *testing.T) {
	runtime := Runtime{"value": "before"}
	runner := New(WithRuntime(runtime))
	runtime["value"] = "after"

	result, err := runner.ExecuteString("snapshot.js", `
		export default function() {
			return Veta.value;
		}
	`)
	require.NoError(t, err)
	require.Equal(t, "before", result.Export())
}

// TestRunnerConsole verifies JavaScript console debugging output.
func TestRunnerConsole(t *testing.T) {
	var output bytes.Buffer
	runner := New(WithConsoleOutput(&output))

	result, err := runner.ExecuteString("console.js", `
		export default function() {
			console.log("hello", 123);
			console.info("ready");
			console.warn("careful");
			console.error("broken");
			console.debug("details", undefined, null);
			return "ok";
		}
	`)
	require.NoError(t, err)
	require.Equal(t, "ok", result.Export())
	require.Equal(t, strings.Join([]string{
		"[js log] hello 123",
		"[js info] ready",
		"[js warn] careful",
		"[js error] broken",
		"[js debug] details undefined null",
		"",
	}, "\n"), output.String())
}

// TestRunnerConsoleNilOutput verifies that disabled console output is safe.
func TestRunnerConsoleNilOutput(t *testing.T) {
	runner := New(WithConsoleOutput(nil))

	result, err := runner.ExecuteString("console.js", `
		export default function() {
			console.log("ignored");
			return true;
		}
	`)
	require.NoError(t, err)
	require.Equal(t, true, result.Export())
}

// TestRunnerFileAPIErrors verifies path safety and read/list error handling.
func TestRunnerFileAPIErrors(t *testing.T) {
	root := filepath.Join("testdata", "project")
	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "read outside root",
			code: `
				export default function({ files }) {
					return files.readFile("../page.js");
				}
			`,
			want: ErrPathOutsideRoot.Error(),
		},
		{
			name: "absolute read path",
			code: `
				export default function({ files }) {
					return files.readFile("/content/index.md");
				}
			`,
			want: ErrPathOutsideRoot.Error(),
		},
		{
			name: "missing file",
			code: `
				export default function({ files }) {
					return files.readFile("content/missing.md");
				}
			`,
			want: "read file content/missing.md",
		},
		{
			name: "empty glob",
			code: `
				export default function({ files }) {
					return files.listFiles("");
				}
			`,
			want: ErrEmptyPath.Error(),
		},
		{
			name: "bad glob",
			code: `
				export default function({ files }) {
					return files.listFiles("content/[");
				}
			`,
			want: "list files matching content/[:",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := New(WithRoot(root)).ExecuteString(test.name+".js", test.code)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.want)
		})
	}
}

// TestRunnerInvalidRoot verifies that invalid roots fail before JavaScript runs.
func TestRunnerInvalidRoot(t *testing.T) {
	_, err := New(WithRoot(filepath.Join("testdata", "project", "missing"))).ExecuteString("root.js", `
		export default function() {
			return true;
		}
	`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "stat root directory")
}

// TestRunnerHTTPClient verifies the synchronous HTTP client against a real HTTP
// server.
func TestRunnerHTTPClient(t *testing.T) {
	server := newTestHTTPServer(t)
	defer server.Close()

	assertGoldenExecution(t, New(WithRuntime(Runtime{"baseURL": server.URL})), "http.js", "http.golden.json")
}

// TestRunnerHTTPClientErrors verifies request validation and HTTP option
// errors.
func TestRunnerHTTPClientErrors(t *testing.T) {
	server := newTestHTTPServer(t)
	defer server.Close()

	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "unsupported url scheme",
			code: `
				export default function({ httpClient }) {
					return httpClient.get("ftp://example.com/file.txt");
				}
			`,
			want: ErrHTTPURLUnsupported.Error(),
		},
		{
			name: "invalid method",
			code: `
				export default function({ baseURL, httpClient }) {
					return httpClient.request("", baseURL + "/get");
				}
			`,
			want: ErrHTTPMethodInvalid.Error(),
		},
		{
			name: "invalid options",
			code: `
				export default function({ baseURL, httpClient }) {
					return httpClient.get(baseURL + "/get", "bad");
				}
			`,
			want: ErrHTTPOptionsUnsupported.Error(),
		},
		{
			name: "headers must be object",
			code: `
				export default function({ baseURL, httpClient }) {
					return httpClient.get(baseURL + "/get", { headers: "bad" });
				}
			`,
			want: ErrHTTPHeadersUnsupported.Error(),
		},
		{
			name: "empty header name",
			code: `
				export default function({ baseURL, httpClient }) {
					return httpClient.get(baseURL + "/get", { headers: { "": "bad" } });
				}
			`,
			want: "http header name cannot be empty",
		},
		{
			name: "body json conflict",
			code: `
				export default function({ baseURL, httpClient }) {
					return httpClient.post(baseURL + "/post", { body: "raw", json: { ok: true } });
				}
			`,
			want: ErrHTTPBodyConflict.Error(),
		},
		{
			name: "unsupported body type",
			code: `
				export default function({ baseURL, httpClient }) {
					return httpClient.post(baseURL + "/post", { body: { ok: true } });
				}
			`,
			want: ErrHTTPBodyUnsupported.Error(),
		},
		{
			name: "invalid timeout",
			code: `
				export default function({ baseURL, httpClient }) {
					return httpClient.get(baseURL + "/get", { timeoutMs: 0 });
				}
			`,
			want: ErrHTTPTimeoutInvalid.Error(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := New(WithRuntime(Runtime{"baseURL": server.URL})).ExecuteString(test.name+".js", test.code)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.want)
		})
	}
}

// TestResult verifies Result conversion helpers and zero-value behavior.
func TestResult(t *testing.T) {
	t.Run("zero result", func(t *testing.T) {
		var result Result

		require.Nil(t, result.Export())
		require.ErrorIs(t, result.ExportTo(new(any)), ErrNoResult)
	})

	t.Run("export to struct", func(t *testing.T) {
		result, err := New().ExecuteString("struct.js", `
			export default function() {
				return { Title: "Veta", Count: 2 };
			}
		`)
		require.NoError(t, err)

		var got struct {
			Title string
			Count int
		}
		require.NoError(t, result.ExportTo(&got))
		require.Equal(t, "Veta", got.Title)
		require.Equal(t, 2, got.Count)
		require.NotNil(t, result.Value())
	})
}

// TestRunnerExecuteErrors verifies typed execution contract errors.
func TestRunnerExecuteErrors(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr error
	}{
		{
			name: "missing default export",
			code: `
				const value = 1;
			`,
			wantErr: ErrMissingDefaultExport,
		},
		{
			name: "multiple default exports",
			code: `
				export default function() { return 1; }
				export default function() { return 2; }
			`,
			wantErr: ErrMultipleDefaultExports,
		},
		{
			name: "default export is not function",
			code: `
				export default { value: 1 };
			`,
			wantErr: ErrDefaultExportNotFunction,
		},
		{
			name: "promise like result",
			code: `
				export default function() {
					return { then: function() {} };
				}
			`,
			wantErr: ErrPromiseUnsupported,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := New().ExecuteString(test.name+".js", test.code)
			require.Error(t, err)
			require.True(t, errors.Is(err, test.wantErr), "expected %v, got %v", test.wantErr, err)
		})
	}
}

// TestRunnerExecuteUnsupportedModuleSyntax verifies that unsupported module
// syntax fails instead of being treated as real ESM support.
func TestRunnerExecuteUnsupportedModuleSyntax(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{
			name: "static import",
			code: `
				import value from "./value.js";
				export default function() { return value; }
			`,
		},
		{
			name: "dynamic import",
			code: `
				export default function() { return import("./value.js"); }
			`,
		},
		{
			name: "named export",
			code: `
				export const value = 1;
				export default function() { return value; }
			`,
		},
		{
			name: "require call",
			code: `
				export default function() { return require("value"); }
			`,
		},
		{
			name: "spaced export default",
			code: `
				export   default function() { return "unsupported"; }
			`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := New().ExecuteString(test.name+".js", test.code)
			require.Error(t, err)
		})
	}
}

// TestRunnerExecuteGoldenFiles verifies file-based executions against stable
// output fixtures.
func TestRunnerExecuteGoldenFiles(t *testing.T) {
	tests := []struct {
		name    string
		script  string
		golden  string
		root    string
		runtime Runtime
	}{
		{
			name:   "page",
			script: "page.js",
			golden: "page.golden.json",
			runtime: Runtime{
				"siteName": "Veta",
			},
		},
		{
			name:   "runtime interop",
			script: "runtime.js",
			golden: "runtime.golden.json",
			runtime: Runtime{
				"join": func(left string, right string) string {
					return left + ":" + right
				},
				"nested":   map[string]any{"answer": 42},
				"siteName": "Veta",
				"value":    7,
			},
		},
		{
			name:   "arrow default export",
			script: "arrow.js",
			golden: "arrow.golden.json",
			runtime: Runtime{
				"prefix": "veta",
				"suffix": "ssg",
			},
		},
		{
			name:   "edge values",
			script: "edge.js",
			golden: "edge.golden.json",
		},
		{
			name:   "file api",
			script: "files.js",
			golden: "files.golden.json",
			root:   filepath.Join("testdata", "project"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner := New(WithRuntime(test.runtime))
			if test.root != "" {
				runner = New(WithRuntime(test.runtime), WithRoot(test.root))
			}

			assertGoldenExecution(t, runner, test.script, test.golden)
		})
	}
}

// assertGoldenExecution executes a testdata script and compares its exported
// value with a golden JSON file.
func assertGoldenExecution(t *testing.T, runner *Runner, scriptName string, goldenName string) {
	t.Helper()

	result, err := runner.ExecuteFile(filepath.Join("testdata", scriptName))
	require.NoError(t, err)

	got, err := json.MarshalIndent(result.Export(), "", "  ")
	require.NoError(t, err)

	want, err := os.ReadFile(filepath.Join("testdata", goldenName))
	require.NoError(t, err)

	require.Equal(t, strings.TrimSpace(string(want)), string(got))
}

// newTestHTTPServer returns an HTTP server for runtime client tests.
func newTestHTTPServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/echo":
			body, err := io.ReadAll(request.Body)
			require.NoError(t, err)
			writer.Header().Set("X-Response", "echo")
			_, _ = fmt.Fprintf(writer, `{"method":%q,"body":%q,"contentType":%q}`, request.Method, string(body), request.Header.Get("Content-Type"))
		case "/get":
			writer.Header().Set("Content-Type", "application/json")
			writer.Header().Set("X-Response", "get")
			_, _ = fmt.Fprintf(writer, `{"method":%q,"query":%q,"testHeader":%q}`, request.Method, request.URL.RawQuery, request.Header.Get("X-Test"))
		case "/head":
			writer.Header().Set("X-Head", "true")
			writer.WriteHeader(http.StatusNoContent)
		case "/post":
			body, err := io.ReadAll(request.Body)
			require.NoError(t, err)
			writer.Header().Set("Content-Type", "application/json")
			writer.Header().Set("X-Response", "post")
			writer.WriteHeader(http.StatusCreated)
			response := map[string]any{
				"body":        string(body),
				"contentType": request.Header.Get("Content-Type"),
				"method":      request.Method,
				"traces":      request.Header.Values("X-Trace"),
			}
			require.NoError(t, json.NewEncoder(writer).Encode(response))
		case "/teapot":
			writer.WriteHeader(http.StatusTeapot)
			_, _ = writer.Write([]byte("short and stout"))
		default:
			http.NotFound(writer, request)
		}
	}))
}
