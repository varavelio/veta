package js

import (
	"errors"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/veta/internal/components"
	"github.com/varavelio/veta/internal/markdown"
	"github.com/varavelio/veta/internal/template"
)

type failingMarkdownRenderer struct {
	err error
}

// Render returns the configured Markdown rendering failure.
func (renderer failingMarkdownRenderer) Render(string) (string, error) {
	return "", renderer.err
}

type failingComponentRenderer struct {
	err error
}

// Render returns the configured component rendering failure.
func (renderer failingComponentRenderer) Render(string, any) (string, error) {
	return "", renderer.err
}

// TestParseMarkdown verifies YAML, TOML, and absent frontmatter results include
// the raw body and its rendered HTML.
func TestParseMarkdown(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		frontmatter map[string]any
		content     string
		html        string
	}{
		{
			name:        "yaml frontmatter",
			input:       "---\ntitle: YAML\nweight: 2\n---\n\n# Body\n",
			frontmatter: map[string]any{"title": "YAML", "weight": int64(2)},
			content:     "# Body\n",
			html:        "<h1>Body</h1>\n",
		},
		{
			name:        "toml frontmatter",
			input:       "+++\ntitle = \"TOML\"\nfeatured = true\n+++\n\n## Body\n",
			frontmatter: map[string]any{"featured": true, "title": "TOML"},
			content:     "## Body\n",
			html:        "<h2>Body</h2>\n",
		},
		{
			name:        "no frontmatter",
			input:       "Plain **body**\n",
			frontmatter: map[string]any{},
			content:     "Plain **body**\n",
			html:        "<p>Plain <strong>body</strong></p>\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := New(WithMarkdownRenderer(markdown.New())).Call(
				Source{
					Name: test.name + ".js",
					Code: `export default function({ parse }, input) { return parse.markdown(input); }`,
				},
				test.input,
			)
			require.NoError(t, err)

			var got map[string]any
			require.NoError(t, result.ExportTo(&got))
			require.Equal(t, test.frontmatter, got["frontmatter"])
			require.Equal(t, test.content, got["content"])
			require.Equal(t, test.html, got["html"])
		})
	}
}

// TestParseMarkdownErrors verifies argument, dependency, parser, and renderer
// failures remain clear to JavaScript authors.
func TestParseMarkdownErrors(t *testing.T) {
	renderFailure := errors.New("markdown renderer failed")
	tests := []struct {
		name   string
		runner *Runner
		code   string
		want   string
	}{
		{
			name:   "missing argument",
			runner: New(WithMarkdownRenderer(markdown.New())),
			code:   `export default function({ parse }) { return parse.markdown(); }`,
			want:   "parse.markdown content is required",
		},
		{
			name:   "non-string argument",
			runner: New(WithMarkdownRenderer(markdown.New())),
			code:   `export default function({ parse }) { return parse.markdown(42); }`,
			want:   "parse.markdown content must be a string",
		},
		{
			name:   "invalid frontmatter",
			runner: New(WithMarkdownRenderer(markdown.New())),
			code:   `export default function({ parse }) { return parse.markdown("---\ntitle: [broken\n---\nBody"); }`,
			want:   "parse markdown",
		},
		{
			name:   "missing renderer",
			runner: New(),
			code:   `export default function({ parse }) { return parse.markdown("Body"); }`,
			want:   ErrMarkdownRendererRequired.Error(),
		},
		{
			name:   "renderer failure",
			runner: New(WithMarkdownRenderer(failingMarkdownRenderer{err: renderFailure})),
			code:   `export default function({ parse }) { return parse.markdown("Body"); }`,
			want:   renderFailure.Error(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.runner.ExecuteString(test.name+".js", test.code)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.want)
		})
	}
}

// TestParseRenderComponents verifies passthrough behavior and registered
// component rendering with props, slots, nesting, includes, and runtime context.
func TestParseRenderComponents(t *testing.T) {
	files := fstest.MapFS{
		"components/badge.j2": {
			Data: []byte(`<strong data-text="{{ props.text }}">{{ props.text }}</strong>`),
		},
		"components/card.j2": {
			Data: []byte(
				`{% include "includes/context.j2" %}<article data-title="{{ props.title }}">{{ props.content }}</article>`,
			),
		},
		"components/outer.j2": {Data: []byte(`<section>{{ props.content }}</section>`)},
		"includes/context.j2": {
			Data: []byte(
				`<span data-context="{{ data.site.name }}:{{ page.title }}">{% for item in pages %}{{ item.title }};{% endfor %}</span>`,
			),
		},
	}
	templateRenderer, err := template.New(files)
	require.NoError(t, err)
	componentRenderer, err := components.New(files, templateRenderer)
	require.NoError(t, err)
	runner := New(
		WithComponentRenderer(componentRenderer),
		WithRuntime(Runtime{
			"data":  map[string]any{"site": map[string]any{"name": "Veta"}},
			"page":  map[string]any{"title": "Current"},
			"pages": []map[string]any{{"title": "One"}, {"title": "Two"}},
		}),
	)

	t.Run("leaves plain text and unregistered tags unchanged", func(t *testing.T) {
		input := `plain <unknown value="kept" />`
		result, err := runner.Call(Source{
			Name: "plain.js",
			Code: `export default function({ parse }, input) {
				return parse.renderComponents(input);
			}`,
		}, input)
		require.NoError(t, err)
		require.Equal(t, input, result.Export())
	})

	t.Run("renders registered nested components", func(t *testing.T) {
		result, err := runner.Call(Source{
			Name: "components.js",
			Code: `export default function({ parse }) {
				return parse.renderComponents('before <outer><card title="Sale">Hello **raw** <badge text="New" /></card></outer> after');
			}`,
		})
		require.NoError(t, err)
		require.Equal(
			t,
			`before <section><span data-context="Veta:Current">One;Two;</span><article data-title="Sale">Hello **raw** <strong data-text="New">New</strong></article></section> after`,
			result.Export(),
		)
	})
}

// TestParseRenderComponentsErrors verifies argument, dependency, syntax, and
// renderer failures are reported through the JavaScript call.
func TestParseRenderComponentsErrors(t *testing.T) {
	renderFailure := errors.New("component renderer failed")
	files := fstest.MapFS{"components/card.j2": {Data: []byte(`{{ props.content }}`)}}
	templateRenderer, err := template.New(files)
	require.NoError(t, err)
	componentRenderer, err := components.New(files, templateRenderer)
	require.NoError(t, err)

	tests := []struct {
		name   string
		runner *Runner
		code   string
		want   string
	}{
		{
			name:   "missing argument",
			runner: New(WithComponentRenderer(componentRenderer)),
			code:   `export default function({ parse }) { return parse.renderComponents(); }`,
			want:   "parse.renderComponents content is required",
		},
		{
			name:   "non-string argument",
			runner: New(WithComponentRenderer(componentRenderer)),
			code:   `export default function({ parse }) { return parse.renderComponents({}); }`,
			want:   "parse.renderComponents content must be a string",
		},
		{
			name:   "missing renderer",
			runner: New(),
			code:   `export default function({ parse }) { return parse.renderComponents("plain"); }`,
			want:   ErrComponentRendererRequired.Error(),
		},
		{
			name:   "invalid component syntax",
			runner: New(WithComponentRenderer(componentRenderer)),
			code:   `export default function({ parse }) { return parse.renderComponents("<card>"); }`,
			want:   components.ErrSyntax.Error(),
		},
		{
			name:   "renderer failure",
			runner: New(WithComponentRenderer(failingComponentRenderer{err: renderFailure})),
			code:   `export default function({ parse }) { return parse.renderComponents("plain"); }`,
			want:   renderFailure.Error(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.runner.ExecuteString(test.name+".js", test.code)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.want)
		})
	}
}
