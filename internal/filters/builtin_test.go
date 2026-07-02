package filters

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type testMarkdownRenderer struct{}

func (testMarkdownRenderer) Render(content string) (string, error) {
	return "<p>" + content + "</p>", nil
}

var errTestMarkdownRender = errors.New("boom")

type failingMarkdownRenderer struct{}

func (failingMarkdownRenderer) Render(string) (string, error) {
	return "", errTestMarkdownRender
}

// TestJSONFilter verifies the built-in JSON filter.
func TestJSONFilter(t *testing.T) {
	tests := []struct {
		name      string
		input     any
		want      any
		wantError string
	}{
		{
			name:  "encodes object as safe HTML",
			input: map[string]any{"tag": "<x>", "count": 2},
			want:  SafeHTML(`{"count":2,"tag":"\u003cx\u003e"}`),
		},
		{
			name:      "returns marshal errors",
			input:     make(chan int),
			wantError: "render json filter",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := jsonFilter(test.input, nil)
			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.want, output)
		})
	}
}

// TestMarkdownFilter verifies the built-in Markdown filter.
func TestMarkdownFilter(t *testing.T) {
	tests := []struct {
		name      string
		renderer  MarkdownRenderer
		input     any
		want      any
		wantError error
	}{
		{
			name:     "renders markdown as safe HTML",
			renderer: testMarkdownRenderer{},
			input:    "**Veta**",
			want:     SafeHTML("<p>**Veta**</p>"),
		},
		{
			name:      "requires renderer",
			renderer:  nil,
			input:     "content",
			wantError: ErrMarkdownRendererRequired,
		},
		{
			name:      "returns renderer errors",
			renderer:  failingMarkdownRenderer{},
			input:     "content",
			wantError: errTestMarkdownRender,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := markdownFilter(test.renderer)(test.input, nil)
			if test.wantError != nil {
				require.ErrorIs(t, err, test.wantError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.want, output)
		})
	}
}

// TestBase64Filters verifies the built-in Base64 filters.
func TestBase64Filters(t *testing.T) {
	encoded, err := base64EncodeFilter("hello", nil)
	require.NoError(t, err)
	require.Equal(t, "aGVsbG8=", encoded)

	decoded, err := base64DecodeFilter(encoded, nil)
	require.NoError(t, err)
	require.Equal(t, "hello", decoded)
}

// TestBase64DecodeFilterErrors verifies invalid Base64 input fails clearly.
func TestBase64DecodeFilterErrors(t *testing.T) {
	_, err := base64DecodeFilter("not base64", nil)
	require.ErrorContains(t, err, "decode base64 filter")
}

func TestParseFilters(t *testing.T) {
	tests := []struct {
		name   string
		filter Func
		input  string
		want   any
	}{
		{
			name:   "parse json",
			filter: parseJSONFilter,
			input:  `{"name":"Veta","count":2}`,
			want:   map[string]any{"name": "Veta", "count": int64(2)},
		},
		{
			name:   "parse yaml",
			filter: parseYAMLFilter,
			input:  "items:\n  - label: Docs\n",
			want:   map[string]any{"items": []any{map[string]any{"label": "Docs"}}},
		},
		{
			name:   "parse toml",
			filter: parseTOMLFilter,
			input:  "name = \"Clean\"\n",
			want:   map[string]any{"name": "Clean"},
		},
		{
			name:   "parse markdown",
			filter: parseMarkdownFilter,
			input:  "---\ntitle: Hello\n---\n\n# Body\n",
			want: map[string]any{
				"content":     "# Body\n",
				"frontmatter": map[string]any{"title": "Hello"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.filter(test.input, nil)
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestParseFiltersErrors(t *testing.T) {
	_, err := parseJSONFilter("{", nil)
	require.ErrorContains(t, err, "parse json filter")

	_, err = parseMarkdownFilter("---\ntitle: [broken\n---\nBody\n", nil)
	require.ErrorContains(t, err, "parse markdown filter")
}
