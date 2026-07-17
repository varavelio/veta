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
		{
			name: "multiline paired component tag is preserved",
			content: strings.Join([]string{
				"<header-base",
				`container="lg"`,
				`menu="Blog|/blog,GitHub|https://github.com/varavelio/veta"`,
				`cta_text="Read > the blog"`,
				`cta_url="/blog/"`,
				">",
				"",
				"Content with **Markdown**.",
				"",
				"</header-base>",
			}, "\n"),
			want: strings.Join([]string{
				"<header-base",
				`container="lg"`,
				`menu="Blog|/blog,GitHub|https://github.com/varavelio/veta"`,
				`cta_text="Read > the blog"`,
				`cta_url="/blog/"`,
				">",
				"<p>Content with <strong>Markdown</strong>.</p>",
				"</header-base>",
			}, "\n"),
		},
		{
			name: "multiline self-closing component tag is preserved",
			content: strings.Join([]string{
				"Before.",
				"",
				"<header-base",
				`container="lg"`,
				`title="A > B"`,
				"/>",
				"",
				"After.",
			}, "\n"),
			want: strings.Join([]string{
				"<p>Before.</p>",
				"<header-base",
				`container="lg"`,
				`title="A > B"`,
				"/>",
				"<p>After.</p>",
				"",
			}, "\n"),
		},
		{
			name: "multiline empty paired component tag is preserved",
			content: strings.Join([]string{
				"<header-base",
				`container="lg"`,
				"></header-base>",
			}, "\n"),
			want: strings.Join([]string{
				"<header-base",
				`container="lg"`,
				"></header-base>",
			}, "\n"),
		},
		{
			name: "standard tag name can contain markdown slot content",
			content: strings.Join([]string{
				"<header",
				`class="hero">`,
				"Content with **Markdown**.",
				"</header>",
			}, "\n"),
			want: strings.Join([]string{
				"<header",
				`class="hero">`,
				"<p>Content with <strong>Markdown</strong>.</p>",
				"</header>",
			}, "\n"),
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

// TestRenderMultilineHTMLContexts verifies multiline tags remain compatible
// with Markdown containers while code examples stay escaped.
func TestRenderMultilineHTMLContexts(t *testing.T) {
	t.Run("raw text html", func(t *testing.T) {
		input := "<script\n type=\"text/javascript\">\nconst value = \"*raw*\";\n</script>"
		got, err := Render(input)
		require.NoError(t, err)
		require.Equal(t, input, got)
	})

	t.Run("does not interrupt paragraph", func(t *testing.T) {
		got, err := Render("Before <ui-card\ntitle=\"Inline\"\n/> after")
		require.NoError(t, err)
		require.Equal(t, "<p>Before <ui-card\ntitle=\"Inline\"\n/> after</p>\n", got)
	})

	t.Run("stops at non-attribute content", func(t *testing.T) {
		got, err := Render("<ui-card\n**regular Markdown**")
		require.NoError(t, err)
		require.Equal(t, "<p>&lt;ui-card\n<strong>regular Markdown</strong></p>\n", got)
	})

	t.Run("restores unclosed quoted attribute", func(t *testing.T) {
		got, err := Render(strings.Join([]string{
			"<ui-card",
			`title="unfinished`,
			"",
			"# Regular Markdown",
			"",
			"- one",
			"- two",
		}, "\n"))
		require.NoError(t, err)
		require.Equal(
			t,
			"<p>&lt;ui-card\ntitle=&quot;unfinished</p>\n<h1>Regular Markdown</h1>\n<ul>\n<li>one</li>\n<li>two</li>\n</ul>\n",
			got,
		)
	})

	t.Run("attribute value can start on a later line", func(t *testing.T) {
		input := "<ui-card\ntitle =\n\"Split Value\"\n/>"
		got, err := Render(input)
		require.NoError(t, err)
		require.Equal(t, input, got)
	})

	t.Run("unrelated bracket does not complete malformed tag", func(t *testing.T) {
		got, err := Render("<ui-card\nregular Markdown > comparison\n\n# Heading")
		require.NoError(t, err)
		require.Equal(
			t,
			"<p><ui-card\nregular Markdown > comparison</p>\n<h1>Heading</h1>\n",
			got,
		)
	})

	t.Run("blockquote", func(t *testing.T) {
		got, err := Render("> <ui-card\n> title=\"Quoted\"\n> />")
		require.NoError(t, err)
		require.Contains(t, got, "<blockquote>\n<ui-card\ntitle=\"Quoted\"\n/>")
	})

	t.Run("list item", func(t *testing.T) {
		got, err := Render("- <ui-card\n  title=\"Listed\"\n  />")
		require.NoError(t, err)
		require.Contains(t, got, "<ui-card\ntitle=\"Listed\"\n/>")
	})

	t.Run("crlf", func(t *testing.T) {
		got, err := Render("<ui-card\r\ntitle=\"CRLF\"\r\n/>\r\n")
		require.NoError(t, err)
		require.Contains(t, got, "<ui-card\r\ntitle=\"CRLF\"\r\n/>")
	})

	t.Run("fenced code", func(t *testing.T) {
		got, err := Render("```html\n<ui-card\ntitle=\"Example\"\n/>\n```")
		require.NoError(t, err)
		require.Contains(t, got, "&lt;ui-card")
		require.NotContains(t, got, "\n<ui-card\n")
	})
}

// TestNilRendererRender verifies that a nil Renderer falls back to defaults.
func TestNilRendererRender(t *testing.T) {
	var renderer *Renderer

	got, err := renderer.Render("**fallback**")
	require.NoError(t, err)
	require.Equal(t, "<p><strong>fallback</strong></p>\n", got)
}
