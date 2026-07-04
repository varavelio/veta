package functions

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestURLFunction verifies the page-scoped URL helper.
func TestURLFunction(t *testing.T) {
	tests := []struct {
		name    string
		current string
		target  string
		want    string
	}{
		{name: "root page asset", current: "/", target: "/styles.css", want: "styles.css"},
		{
			name:    "nested page asset",
			current: "/docs/intro/",
			target:  "/styles.css",
			want:    "../../styles.css",
		},
		{
			name:    "nested page directory link",
			current: "/docs/intro/",
			target:  "/docs/",
			want:    "../",
		},
		{
			name:    "current directory link",
			current: "/docs/intro/",
			target:  "/docs/intro/",
			want:    ".",
		},
		{name: "root link from nested page", current: "/docs/intro/", target: "/", want: "../../"},
		{
			name:    "query and fragment",
			current: "/docs/intro/",
			target:  "/styles.css?v=1#main",
			want:    "../../styles.css?v=1#main",
		},
		{
			name:    "external url",
			current: "/docs/intro/",
			target:  "https://example.com/styles.css",
			want:    "https://example.com/styles.css",
		},
		{
			name:    "protocol relative url",
			current: "/docs/intro/",
			target:  "//cdn.example.com/styles.css",
			want:    "//cdn.example.com/styles.css",
		},
		{name: "fragment only", current: "/docs/intro/", target: "#section", want: "#section"},
		{
			name:    "relative input",
			current: "/docs/intro/",
			target:  "images/logo.svg",
			want:    "images/logo.svg",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := urlFunction(
				Context{"page": map[string]any{"permalink": test.current}},
				test.target,
			)
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}
