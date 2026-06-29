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
						template: "blog/index",
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
				return [{ permalink: "docs/intro", template: "docs/page.pongo", content: "Intro", title: "Intro" }];
			}
		`)},
	}

	manifest, err := Load(files, WithJSOptions(js.WithRuntime(js.Runtime{
		"data": map[string]any{"site": map[string]any{"title": "Veta Blog"}},
	})))
	require.NoError(t, err)
	require.Equal(t, Manifest{Pages: []Page{
		{
			Fields: map[string]any{
				"content": "",
				"data": map[string]any{
					"count":    int64(2),
					"featured": true,
					"tags":     []any{"go", "ssg"},
				},
				"date":       "2026-06-26",
				"generator":  "blog.js",
				"index":      int64(0),
				"outputPath": "blog/index.html",
				"permalink":  "/blog/",
				"template":   "blog/index",
				"title":      "Veta Blog",
			},
			Generator:  "blog.js",
			Index:      0,
			OutputPath: "blog/index.html",
			Permalink:  "/blog/",
			Template:   "blog/index",
		},
		{
			Fields: map[string]any{
				"content":    "<urlset></urlset>",
				"generator":  "blog.js",
				"index":      int64(1),
				"outputPath": "sitemap.xml",
				"permalink":  "/sitemap.xml",
				"template":   "",
			},
			Generator:  "blog.js",
			Index:      1,
			OutputPath: "sitemap.xml",
			Permalink:  "/sitemap.xml",
			Template:   "",
		},
		{
			Fields: map[string]any{
				"content":    "Intro",
				"generator":  "docs.js",
				"index":      int64(0),
				"outputPath": "docs/intro/index.html",
				"permalink":  "/docs/intro/",
				"template":   "docs/page.pongo",
				"title":      "Intro",
			},
			Generator:  "docs.js",
			Index:      0,
			OutputPath: "docs/intro/index.html",
			Permalink:  "/docs/intro/",
			Template:   "docs/page.pongo",
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

// TestLoadDefaultsOmittedContent verifies that content is optional and defaults
// to an empty string in normalized page fields.
func TestLoadDefaultsOmittedContent(t *testing.T) {
	files := fstest.MapFS{
		"pages/site.js": {Data: []byte(`
			export default function() {
				return [
					{ permalink: "/sitemap.xml", template: "sitemap" },
					{ permalink: "/empty.txt" },
					{ permalink: "/raw.txt", content: "hello" },
				];
			}
		`)},
	}

	manifest, err := Load(files)
	require.NoError(t, err)
	require.Equal(t, Manifest{Pages: []Page{
		{
			Fields: map[string]any{
				"content":    "",
				"generator":  "site.js",
				"index":      int64(0),
				"outputPath": "sitemap.xml",
				"permalink":  "/sitemap.xml",
				"template":   "sitemap",
			},
			Generator:  "site.js",
			Index:      0,
			OutputPath: "sitemap.xml",
			Permalink:  "/sitemap.xml",
			Template:   "sitemap",
		},
		{
			Fields: map[string]any{
				"content":    "",
				"generator":  "site.js",
				"index":      int64(1),
				"outputPath": "empty.txt",
				"permalink":  "/empty.txt",
				"template":   "",
			},
			Generator:  "site.js",
			Index:      1,
			OutputPath: "empty.txt",
			Permalink:  "/empty.txt",
			Template:   "",
		},
		{
			Fields: map[string]any{
				"content":    "hello",
				"generator":  "site.js",
				"index":      int64(2),
				"outputPath": "raw.txt",
				"permalink":  "/raw.txt",
				"template":   "",
			},
			Generator:  "site.js",
			Index:      2,
			OutputPath: "raw.txt",
			Permalink:  "/raw.txt",
			Template:   "",
		},
	}}, manifest)
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
			name: "old layout field is rejected",
			files: fstest.MapFS{
				"pages/blog.js": {
					Data: []byte(
						`export default function() { return [{ permalink: "/", layout: "page", content: "" }]; }`,
					),
				},
			},
			wantErr: ErrPageInvalid,
		},
		{
			name: "template is not string",
			files: fstest.MapFS{
				"pages/blog.js": {
					Data: []byte(
						`export default function() { return [{ permalink: "/", template: 1, content: "" }]; }`,
					),
				},
			},
			wantErr: ErrPageInvalid,
		},
		{
			name: "empty template",
			files: fstest.MapFS{
				"pages/blog.js": {
					Data: []byte(
						`export default function() { return [{ permalink: "/", template: "", content: "" }]; }`,
					),
				},
			},
			wantErr: ErrPageInvalid,
		},
		{
			name: "template includes templates prefix",
			files: fstest.MapFS{
				"pages/blog.js": {
					Data: []byte(
						`export default function() { return [{ permalink: "/", template: "templates/page", content: "" }]; }`,
					),
				},
			},
			wantErr: ErrPageInvalid,
		},
		{
			name: "template escapes templates directory",
			files: fstest.MapFS{
				"pages/blog.js": {
					Data: []byte(
						`export default function() { return [{ permalink: "/", template: "../page", content: "" }]; }`,
					),
				},
			},
			wantErr: ErrPageInvalid,
		},
		{
			name: "content is not string",
			files: fstest.MapFS{
				"pages/blog.js": {
					Data: []byte(
						`export default function() { return [{ permalink: "/", template: "page", content: 1 }]; }`,
					),
				},
			},
			wantErr: ErrPageInvalid,
		},
		{
			name: "page contains function",
			files: fstest.MapFS{
				"pages/blog.js": {
					Data: []byte(
						`export default function() { return [{ permalink: "/", custom: function() {} }]; }`,
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
						`export default function() { return [{ permalink: "/contacto", template: "page", content: "A" }]; }`,
					),
				},
				"pages/b.js": {
					Data: []byte(
						`export default function() { return [{ permalink: "/contacto/index.html", template: "page", content: "B" }]; }`,
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
