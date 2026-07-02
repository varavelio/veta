package parsecontent

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStructuredParsers(t *testing.T) {
	tests := []struct {
		name   string
		parse  func(string) (any, error)
		input  string
		wanted any
	}{
		{
			name:   "json",
			parse:  JSON,
			input:  `{"title":"Veta","count":2}`,
			wanted: map[string]any{"title": "Veta", "count": int64(2)},
		},
		{
			name:   "yaml",
			parse:  YAML,
			input:  "items:\n  - label: Docs\n",
			wanted: map[string]any{"items": []any{map[string]any{"label": "Docs"}}},
		},
		{
			name:   "toml",
			parse:  TOML,
			input:  "title = \"Veta\"\n",
			wanted: map[string]any{"title": "Veta"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.parse(test.input)
			require.NoError(t, err)
			require.Equal(t, test.wanted, got)
		})
	}
}

func TestMarkdown(t *testing.T) {
	document, err := MarkdownMap("---\ntitle: Hello\n---\n\n# Body\n")
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"content":     "# Body\n",
		"frontmatter": map[string]any{"title": "Hello"},
	}, document)

	plain, err := MarkdownMap("# Plain\n")
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"content":     "# Plain\n",
		"frontmatter": map[string]any{},
	}, plain)
}

func TestParseErrors(t *testing.T) {
	_, err := JSON(`{"a":1} {"b":2}`)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrInvalid))

	_, err = Markdown("---\ntitle: Missing close\n")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrInvalid))
}
