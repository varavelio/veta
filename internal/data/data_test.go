package data

import (
	"errors"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/veta/internal/js"
)

// TestLoad verifies that every supported data format is loaded into a shared
// namespace.
func TestLoad(t *testing.T) {
	files := fstest.MapFS{
		"data/github.js": {Data: []byte(`
			export default function({ env }) {
				return {
					branch: env.BRANCH,
					items: [1, true, null]
				};
			}
		`)},
		"data/navigation.yaml": {Data: []byte(`
main:
  - label: Home
    href: /
  - label: Blog
    href: /blog/
`)},
		"data/settings.yml": {Data: []byte(`enabled: true`)},
		"data/site.json": {Data: []byte(`{
			"name": "Veta",
			"stars": 42
		}`)},
		"data/theme.toml": {Data: []byte(`
name = "Clean"
published = 2026-06-26T12:34:56Z

[colors]
primary = "blue"
`)},
	}

	values, err := Load(files, WithJSOptions(js.WithEnvironment(js.Environment{"BRANCH": "main"})))
	require.NoError(t, err)
	require.Equal(t, Values{
		"github": map[string]any{
			"branch": "main",
			"items":  []any{int64(1), true, nil},
		},
		"navigation": map[string]any{
			"main": []any{
				map[string]any{"label": "Home", "href": "/"},
				map[string]any{"label": "Blog", "href": "/blog/"},
			},
		},
		"settings": map[string]any{"enabled": true},
		"site":     map[string]any{"name": "Veta", "stars": int64(42)},
		"theme": map[string]any{
			"colors":    map[string]any{"primary": "blue"},
			"name":      "Clean",
			"published": "2026-06-26T12:34:56Z",
		},
	}, values)
}

// TestLoadMissingDataDirectory verifies that projects without data files still
// receive an empty data map.
func TestLoadMissingDataDirectory(t *testing.T) {
	values, err := Load(fstest.MapFS{})
	require.NoError(t, err)
	require.Equal(t, Values{}, values)
}

// TestLoadAcceptsJSONCompatibleTopLevelValues verifies that global data may be
// any JSON-compatible value.
func TestLoadAcceptsJSONCompatibleTopLevelValues(t *testing.T) {
	files := fstest.MapFS{
		"data/array.json":  {Data: []byte(`["go", "ssg"]`)},
		"data/number.json": {Data: []byte(`42`)},
		"data/null.yaml":   {Data: []byte(`null`)},
		"data/text.js":     {Data: []byte(`export default function() { return "hello"; }`)},
	}

	values, err := Load(files)
	require.NoError(t, err)
	require.Equal(t, Values{
		"array":  []any{"go", "ssg"},
		"null":   nil,
		"number": int64(42),
		"text":   "hello",
	}, values)
}

// TestLoadErrors verifies loader validation and parser failures.
func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name    string
		files   fstest.MapFS
		wantErr error
	}{
		{
			name:    "nested directory",
			files:   fstest.MapFS{"data/nested/site.json": {Data: []byte(`{}`)}},
			wantErr: ErrNestedUnsupported,
		},
		{
			name:    "unsupported extension",
			files:   fstest.MapFS{"data/readme.md": {Data: []byte(`# data`)}},
			wantErr: ErrFormatUnsupported,
		},
		{
			name: "duplicate key",
			files: fstest.MapFS{
				"data/site.json": {Data: []byte(`{}`)},
				"data/site.yaml": {Data: []byte(`{}`)},
			},
			wantErr: ErrKeyDuplicate,
		},
		{
			name:    "invalid key",
			files:   fstest.MapFS{"data/site-name.json": {Data: []byte(`{}`)}},
			wantErr: ErrKeyInvalid,
		},
		{
			name:    "malformed json",
			files:   fstest.MapFS{"data/site.json": {Data: []byte(`{`)}},
			wantErr: ErrInvalid,
		},
		{
			name: "multiple json values",
			files: fstest.MapFS{
				"data/site.json": {Data: []byte(`{} {}`)},
			},
			wantErr: ErrInvalid,
		},
		{
			name: "multiple yaml documents",
			files: fstest.MapFS{
				"data/site.yaml": {Data: []byte("title: Veta\n---\ntitle: Other\n")},
			},
			wantErr: ErrInvalid,
		},
		{
			name:    "malformed toml",
			files:   fstest.MapFS{"data/site.toml": {Data: []byte(`name = [`)}},
			wantErr: ErrInvalid,
		},
		{
			name:    "javascript syntax error",
			files:   fstest.MapFS{"data/site.js": {Data: []byte(`export default function() {`)}},
			wantErr: ErrInvalid,
		},
		{
			name:    "javascript undefined result",
			files:   fstest.MapFS{"data/site.js": {Data: []byte(`export default function() {}`)}},
			wantErr: ErrValueUnsupported,
		},
		{
			name: "javascript non finite number",
			files: fstest.MapFS{
				"data/site.js": {Data: []byte(`export default function() { return NaN; }`)},
			},
			wantErr: ErrValueUnsupported,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Load(test.files)
			require.Error(t, err)
			require.True(t, errors.Is(err, test.wantErr), "expected %v, got %v", test.wantErr, err)
		})
	}
}

// TestLoadRequiresFilesystem verifies that callers must provide a filesystem.
func TestLoadRequiresFilesystem(t *testing.T) {
	_, err := Load(nil)
	require.ErrorIs(t, err, ErrFSRequired)
}
