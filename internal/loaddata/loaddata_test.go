package loaddata

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoaderLoadPath(t *testing.T) {
	loader, err := New(fstest.MapFS{
		"content/readme.txt": {Data: []byte("Hello Veta")},
		"data/site.json":     {Data: []byte(`{"name":"Veta","count":2}`)},
		"data/nav.yaml":      {Data: []byte("items:\n  - label: Docs\n")},
		"data/theme.toml":    {Data: []byte("name = \"Clean\"\n")},
	})
	require.NoError(t, err)

	text, err := loader.Load(Request{Path: "content/readme.txt"})
	require.NoError(t, err)
	require.Equal(t, "Hello Veta", text)

	jsonValue, err := loader.Load(Request{Path: "data/site.json"})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"name": "Veta", "count": int64(2)}, jsonValue)

	yamlValue, err := loader.Load(Request{Path: "data/nav.yaml"})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"items": []any{map[string]any{"label": "Docs"}}}, yamlValue)

	tomlValue, err := loader.Load(Request{Path: "data/theme.toml"})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"name": "Clean"}, tomlValue)
}

func TestLoaderLoadURL(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			response.Header().Set("Content-Type", "application/json")
			_, _ = response.Write([]byte(`{"name":"Veta"}`))
		}),
	)
	defer server.Close()

	loader, err := New(fstest.MapFS{}, WithHTTPTimeout(time.Second))
	require.NoError(t, err)

	value, err := loader.Load(Request{URL: server.URL})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"name": "Veta"}, value)
}

func TestLoaderFunction(t *testing.T) {
	loader, err := New(fstest.MapFS{"data/site.json": {Data: []byte(`{"name":"Veta"}`)}})
	require.NoError(t, err)

	value, err := loader.Function()("data/site.json")
	require.NoError(t, err)
	require.Equal(t, map[string]any{"name": "Veta"}, value)
}

func TestLoaderErrors(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			response.WriteHeader(http.StatusNotFound)
			_, _ = response.Write([]byte(`missing`))
		}),
	)
	defer server.Close()

	loader, err := New(fstest.MapFS{"data/bad.json": {Data: []byte(`{`)}})
	require.NoError(t, err)

	tests := []struct {
		name    string
		request Request
		want    error
	}{
		{name: "missing source", request: Request{}, want: ErrRequestInvalid},
		{
			name:    "path and url",
			request: Request{Path: "data/site.json", URL: server.URL},
			want:    ErrRequestInvalid,
		},
		{name: "bad path", request: Request{Path: "../data/site.json"}, want: ErrPathInvalid},
		{name: "bad url", request: Request{URL: "file:///tmp/data.json"}, want: ErrURLInvalid},
		{
			name:    "negative timeout",
			request: Request{URL: server.URL, Timeout: -time.Second},
			want:    ErrRequestInvalid,
		},
		{
			name:    "bad format",
			request: Request{Path: "data/bad.json", Format: "csv"},
			want:    ErrFormatInvalid,
		},
		{name: "bad json", request: Request{Path: "data/bad.json"}, want: nil},
		{name: "http status", request: Request{URL: server.URL}, want: ErrHTTPStatus},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := loader.Load(test.request)
			require.Error(t, err)
			if test.want != nil {
				require.True(t, errors.Is(err, test.want), "expected %v, got %v", test.want, err)
			}
		})
	}
}
