package markdown

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRender verifies common Markdown rendering behavior.
func TestRender(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "empty content",
			content: "",
			want:    "",
		},
		{
			name:    "basic markdown",
			content: "# Hello\n\nThis is **Veta** with [docs](/docs).",
			want:    "<h1>Hello</h1>\n<p>This is <strong>Veta</strong> with <a href=\"/docs\">docs</a>.</p>\n",
		},
		{
			name: "gfm table",
			content: strings.Join([]string{
				"| Name | Value |",
				"| --- | --- |",
				"| Veta | SSG |",
			}, "\n"),
			want: strings.Join([]string{
				"<table>",
				"<thead>",
				"<tr>",
				"<th>Name</th>",
				"<th>Value</th>",
				"</tr>",
				"</thead>",
				"<tbody>",
				"<tr>",
				"<td>Veta</td>",
				"<td>SSG</td>",
				"</tr>",
				"</tbody>",
				"</table>",
				"",
			}, "\n"),
		},
		{
			name:    "gfm task list",
			content: "- [x] Build core\n- [ ] Ship release",
			want: strings.Join([]string{
				"<ul>",
				"<li><input checked=\"\" disabled=\"\" type=\"checkbox\"> Build core</li>",
				"<li><input disabled=\"\" type=\"checkbox\"> Ship release</li>",
				"</ul>",
				"",
			}, "\n"),
		},
		{
			name:    "gfm strikethrough",
			content: "This is ~~old~~ new.",
			want:    "<p>This is <del>old</del> new.</p>\n",
		},
		{
			name:    "raw html is preserved",
			content: "<div class=\"note\">Trusted HTML</div>",
			want:    "<div class=\"note\">Trusted HTML</div>",
		},
		{
			name:    "custom component tag is preserved",
			content: "<ui-card title=\"Hello\">\n\nContent\n\n</ui-card>",
			want:    "<ui-card title=\"Hello\">\n<p>Content</p>\n</ui-card>",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Render(test.content)
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

// TestRendererRender verifies that a reusable Renderer can render content.
func TestRendererRender(t *testing.T) {
	renderer := New()

	got, err := renderer.Render("## Reusable")
	require.NoError(t, err)
	require.Equal(t, "<h2>Reusable</h2>\n", got)
}

// TestNilRendererRender verifies that a nil Renderer falls back to defaults.
func TestNilRendererRender(t *testing.T) {
	var renderer *Renderer

	got, err := renderer.Render("**fallback**")
	require.NoError(t, err)
	require.Equal(t, "<p><strong>fallback</strong></p>\n", got)
}
