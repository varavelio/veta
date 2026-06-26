package pages

import (
	"errors"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/veta/internal/js"
)

// TestLoad verifies that page generators produce a deterministic manifest.
func TestLoad(t *testing.T) {
	files := fstest.MapFS{
		"pages/blog.js": {Data: []byte(`
			export default function({ data }) {
				return [
					{
						permalink: "/blog",
						layout: "blog/index",
						title: data.site.title,
						content: "",
						date: "2026-06-26",
						data: { count: 2, featured: true, tags: ["go", "ssg"] }
					},
					{
						permalink: "/sitemap.xml",
						content: "<urlset></urlset>"
					}
				];
			}
		`)},
		"pages/docs.js": {Data: []byte(`
			export default function() {
				return [{ permalink: "docs/intro", layout: "docs/page", title: "Intro" }];
			}
		`)},
	}

	manifest, err := Load(files, WithJSOptions(js.WithRuntime(js.Runtime{
		"data": map[string]any{"site": map[string]any{"title": "Veta Blog"}},
	})))
	require.NoError(t, err)
	require.Equal(t, Manifest{Pages: []Page{
		{
			Content: "",
			Data: map[string]any{
				"count":    int64(2),
				"featured": true,
				"tags":     []any{"go", "ssg"},
			},
			Date:       "2026-06-26",
			Generator:  "blog.js",
			Index:      0,
			Layout:     "blog/index",
			OutputPath: "blog/index.html",
			Permalink:  "/blog/",
			Title:      "Veta Blog",
		},
		{
			Content:    "<urlset></urlset>",
			Data:       map[string]any{},
			Generator:  "blog.js",
			Index:      1,
			OutputPath: "sitemap.xml",
			Permalink:  "/sitemap.xml",
		},
		{
			Data:       map[string]any{},
			Generator:  "docs.js",
			Index:      0,
			Layout:     "docs/page",
			OutputPath: "docs/intro/index.html",
			Permalink:  "/docs/intro/",
			Title:      "Intro",
		},
	}}, manifest)
}

// TestLoadMissingPagesDirectory verifies that projects without generators still
// receive an empty manifest.
func TestLoadMissingPagesDirectory(t *testing.T) {
	manifest, err := Load(fstest.MapFS{})
	require.NoError(t, err)
	require.Equal(t, Manifest{}, manifest)
}

// TestLoadRequiresFilesystem verifies that callers must provide a filesystem.
func TestLoadRequiresFilesystem(t *testing.T) {
	_, err := Load(nil)
	require.ErrorIs(t, err, ErrFSRequired)
}

// TestLoadErrors verifies filesystem, generator, and page contract validation.
func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name    string
		files   fstest.MapFS
		wantErr error
	}{
		{
			name: "nested directory",
			files: fstest.MapFS{
				"pages/blog/index.js": {Data: []byte(`export default function() { return []; }`)},
			},
			wantErr: ErrNestedUnsupported,
		},
		{
			name:    "unsupported file extension",
			files:   fstest.MapFS{"pages/README.md": {Data: []byte(`# pages`)}},
			wantErr: ErrFormatUnsupported,
		},
		{
			name:    "javascript syntax error",
			files:   fstest.MapFS{"pages/blog.js": {Data: []byte(`export default function() {`)}},
			wantErr: ErrGeneratorInvalid,
		},
		{
			name:    "missing default export",
			files:   fstest.MapFS{"pages/blog.js": {Data: []byte(`const pages = [];`)}},
			wantErr: ErrGeneratorInvalid,
		},
		{
			name:    "generator returns undefined",
			files:   fstest.MapFS{"pages/blog.js": {Data: []byte(`export default function() {}`)}},
			wantErr: ErrGeneratorInvalid,
		},
		{
			name: "generator returns object",
			files: fstest.MapFS{
				"pages/blog.js": {Data: []byte(`export default function() { return {}; }`)},
			},
			wantErr: ErrGeneratorInvalid,
		},
		{
			name: "page item is not object",
			files: fstest.MapFS{
				"pages/blog.js": {Data: []byte(`export default function() { return [42]; }`)},
			},
			wantErr: ErrPageInvalid,
		},
		{
			name: "missing permalink",
			files: fstest.MapFS{
				"pages/blog.js": {Data: []byte(`export default function() { return [{}]; }`)},
			},
			wantErr: ErrPageInvalid,
		},
		{
			name: "permalink is not string",
			files: fstest.MapFS{
				"pages/blog.js": {
					Data: []byte(`export default function() { return [{ permalink: 42 }]; }`),
				},
			},
			wantErr: ErrPageInvalid,
		},
		{
			name: "invalid permalink",
			files: fstest.MapFS{
				"pages/blog.js": {
					Data: []byte(
						`export default function() { return [{ permalink: "../secret" }]; }`,
					),
				},
			},
			wantErr: ErrPermalinkInvalid,
		},
		{
			name: "unknown field",
			files: fstest.MapFS{
				"pages/blog.js": {
					Data: []byte(
						`export default function() { return [{ permalink: "/", draft: true }]; }`,
					),
				},
			},
			wantErr: ErrPageInvalid,
		},
		{
			name: "layout is not string",
			files: fstest.MapFS{
				"pages/blog.js": {
					Data: []byte(
						`export default function() { return [{ permalink: "/", layout: 1 }]; }`,
					),
				},
			},
			wantErr: ErrPageInvalid,
		},
		{
			name: "data is not object",
			files: fstest.MapFS{
				"pages/blog.js": {
					Data: []byte(
						`export default function() { return [{ permalink: "/", data: [] }]; }`,
					),
				},
			},
			wantErr: ErrPageInvalid,
		},
		{
			name: "data contains function",
			files: fstest.MapFS{
				"pages/blog.js": {
					Data: []byte(
						`export default function() { return [{ permalink: "/", data: { bad: function() {} } }]; }`,
					),
				},
			},
			wantErr: ErrValueUnsupported,
		},
		{
			name: "duplicate output path",
			files: fstest.MapFS{
				"pages/a.js": {
					Data: []byte(
						`export default function() { return [{ permalink: "/contacto" }]; }`,
					),
				},
				"pages/b.js": {
					Data: []byte(
						`export default function() { return [{ permalink: "/contacto/index.html" }]; }`,
					),
				},
			},
			wantErr: ErrOutputPathDuplicate,
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

// TestNormalizePermalink verifies Veta's pretty URL and explicit extension
// routing rules.
func TestNormalizePermalink(t *testing.T) {
	tests := []struct {
		name          string
		permalink     string
		wantPermalink string
		wantOutput    string
	}{
		{
			name:          "root",
			permalink:     "/",
			wantPermalink: "/",
			wantOutput:    "index.html",
		},
		{
			name:          "relative pretty URL",
			permalink:     "contacto",
			wantPermalink: "/contacto/",
			wantOutput:    "contacto/index.html",
		},
		{
			name:          "absolute pretty URL",
			permalink:     "/contacto",
			wantPermalink: "/contacto/",
			wantOutput:    "contacto/index.html",
		},
		{
			name:          "trailing slash pretty URL",
			permalink:     "/contacto/",
			wantPermalink: "/contacto/",
			wantOutput:    "contacto/index.html",
		},
		{
			name:          "explicit html",
			permalink:     "/contacto.html",
			wantPermalink: "/contacto.html",
			wantOutput:    "contacto.html",
		},
		{
			name:          "explicit xml",
			permalink:     "/sitemap.xml",
			wantPermalink: "/sitemap.xml",
			wantOutput:    "sitemap.xml",
		},
		{
			name:          "nested explicit json",
			permalink:     "/api/posts.json",
			wantPermalink: "/api/posts.json",
			wantOutput:    "api/posts.json",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotPermalink, gotOutput, err := normalizePermalink(test.permalink)
			require.NoError(t, err)
			require.Equal(t, test.wantPermalink, gotPermalink)
			require.Equal(t, test.wantOutput, gotOutput)
		})
	}
}

// TestNormalizePermalinkErrors verifies invalid permalink inputs.
func TestNormalizePermalinkErrors(t *testing.T) {
	tests := []string{
		"",
		"   ",
		".",
		"../secret",
		"/../secret",
		"https://example.com/page",
		"//example.com/page",
		"/page?draft=true",
		"/page#section",
		`C:\site\page`,
	}

	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			_, _, err := normalizePermalink(test)
			require.ErrorIs(t, err, ErrPermalinkInvalid)
		})
	}
}
