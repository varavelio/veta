package permalink

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNormalize verifies Veta's pretty URL and explicit extension routing rules.
func TestNormalize(t *testing.T) {
	tests := []struct {
		name          string
		permalink     string
		wantPermalink string
		wantOutput    string
	}{
		{name: "root", permalink: "/", wantPermalink: "/", wantOutput: "index.html"},
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
			gotPermalink, gotOutput, err := Normalize(test.permalink)
			require.NoError(t, err)
			require.Equal(t, test.wantPermalink, gotPermalink)
			require.Equal(t, test.wantOutput, gotOutput)
		})
	}
}

// TestNormalizeErrors verifies invalid permalink inputs.
func TestNormalizeErrors(t *testing.T) {
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
			_, _, err := Normalize(test)
			require.ErrorIs(t, err, ErrInvalid)
		})
	}
}

// TestFromPath verifies generic path-to-permalink conversion.
func TestFromPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		options PathOptions
		want    string
	}{
		{name: "plain file", path: "content/about.md", want: "/content/about/"},
		{
			name:    "strip prefix",
			path:    "content/blog/hello.md",
			options: PathOptions{StripPrefix: "content"},
			want:    "/blog/hello/",
		},
		{
			name:    "strip prefix index",
			path:    "content/index.md",
			options: PathOptions{StripPrefix: "content"},
			want:    "/",
		},
		{
			name:    "nested index",
			path:    "content/blog/index.md",
			options: PathOptions{StripPrefix: "content"},
			want:    "/blog/",
		},
		{name: "extensionless", path: "docs/guide", want: "/docs/guide/"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := FromPath(test.path, test.options)
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

// TestFromPathErrors verifies invalid path conversion inputs.
func TestFromPathErrors(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		options PathOptions
	}{
		{name: "empty", path: ""},
		{name: "absolute", path: "/content/index.md"},
		{name: "parent", path: "../content/index.md"},
		{
			name:    "missing strip prefix",
			path:    "posts/index.md",
			options: PathOptions{StripPrefix: "content"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := FromPath(test.path, test.options)
			require.ErrorIs(t, err, ErrInvalid)
		})
	}
}
