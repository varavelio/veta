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
