package render

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type testTemplateRenderer struct {
	context  any
	contexts []any
	name     string
}

// Render records the template call and emits its page content unchanged.
func (renderer *testTemplateRenderer) Render(name string, context any) (string, error) {
	renderer.name = name
	renderer.context = context
	renderer.contexts = append(renderer.contexts, context)
	contextMap := context.(map[string]any)
	page := contextMap["page"].(map[string]any)
	return fmt.Sprintf("%s:%v:%s", name, page["title"], page["content"]), nil
}

type failingTemplateRenderer struct{}

// Render returns a deterministic template failure.
func (failingTemplateRenderer) Render(string, any) (string, error) {
	return "", errors.New("template failed")
}

// TestRenderWithTemplate verifies that generator-provided content is passed to
// the template unchanged and trusted.
func TestRenderWithTemplate(t *testing.T) {
	templateRenderer := &testTemplateRenderer{}
	renderer, err := New(WithTemplateRenderer(templateRenderer))
	require.NoError(t, err)

	document, err := renderer.Render(Page{
		Fields: map[string]any{
			"content": "<p>Hello</p>",
			"date":    "2026-06-26",
			"kind":    "post",
			"title":   "Blog",
		},
		OutputPath: "blog/index.html",
		Permalink:  "/blog/",
		Template:   "layouts/base",
	}, map[string]any{"site": map[string]any{"name": "Veta"}})
	require.NoError(t, err)
	require.Equal(t, Document{
		Content:    []byte("layouts/base:Blog:<p>Hello</p>"),
		OutputPath: "blog/index.html",
		Permalink:  "/blog/",
	}, document)

	context := templateRenderer.context.(map[string]any)
	require.Equal(t, map[string]any{"site": map[string]any{"name": "Veta"}}, context["data"])
	require.Equal(t, map[string]any{}, context["props"])
	require.Equal(t, map[string]any{
		"content":    SafeHTML("<p>Hello</p>"),
		"date":       "2026-06-26",
		"kind":       "post",
		"outputPath": "blog/index.html",
		"permalink":  "/blog/",
		"template":   "layouts/base",
		"title":      "Blog",
	}, context["page"])
	require.Equal(t, []map[string]any{{
		"content":    SafeHTML("<p>Hello</p>"),
		"date":       "2026-06-26",
		"kind":       "post",
		"outputPath": "blog/index.html",
		"permalink":  "/blog/",
		"template":   "layouts/base",
		"title":      "Blog",
	}}, context["pages"])
}

// TestRenderWithTemplatePreservesEveryContentFormat verifies that templated
// content is never implicitly transformed.
func TestRenderWithTemplatePreservesEveryContentFormat(t *testing.T) {
	tests := []struct {
		name    string
		fields  map[string]any
		content string
	}{
		{
			name:    "markdown",
			fields:  map[string]any{"content": "# Heading\n\n**bold**"},
			content: "# Heading\n\n**bold**",
		},
		{
			name:    "component tag",
			fields:  map[string]any{"content": `<card title="Raw">Slot</card>`},
			content: `<card title="Raw">Slot</card>`,
		},
		{
			name:    "rendered html",
			fields:  map[string]any{"content": "<h1>Heading</h1>"},
			content: "<h1>Heading</h1>",
		},
		{name: "omitted content", fields: map[string]any{}, content: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			templateRenderer := &testTemplateRenderer{}
			renderer, err := New(WithTemplateRenderer(templateRenderer))
			require.NoError(t, err)

			document, err := renderer.Render(Page{
				Fields:     test.fields,
				OutputPath: "index.html",
				Permalink:  "/",
				Template:   "page",
			}, nil)
			require.NoError(t, err)
			require.Equal(t, "page:<nil>:"+test.content, string(document.Content))

			context := templateRenderer.context.(map[string]any)
			page := context["page"].(map[string]any)
			require.Equal(t, SafeHTML(test.content), page["content"])
		})
	}
}

// TestRenderWithoutTemplateReturnsRawContent verifies raw output pages.
func TestRenderWithoutTemplateReturnsRawContent(t *testing.T) {
	renderer, err := New()
	require.NoError(t, err)

	document, err := renderer.Render(Page{
		Fields:     map[string]any{"content": "# Raw\n<card />"},
		OutputPath: "feed.xml",
		Permalink:  "/feed.xml",
	}, nil)
	require.NoError(t, err)
	require.Equal(t, Document{
		Content:    []byte("# Raw\n<card />"),
		OutputPath: "feed.xml",
		Permalink:  "/feed.xml",
	}, document)
}

// TestRenderWithoutTemplateDefaultsOmittedContent verifies that template-less
// pages without content render an empty document.
func TestRenderWithoutTemplateDefaultsOmittedContent(t *testing.T) {
	renderer, err := New()
	require.NoError(t, err)

	document, err := renderer.Render(Page{
		OutputPath: "empty.txt",
		Permalink:  "/empty.txt",
	}, nil)
	require.NoError(t, err)
	require.Equal(t, Document{
		Content:    []byte(""),
		OutputPath: "empty.txt",
		Permalink:  "/empty.txt",
	}, document)
}

// TestRenderPages verifies that every template receives the shared normalized
// page list without changing page content.
func TestRenderPages(t *testing.T) {
	templateRenderer := &testTemplateRenderer{}
	renderer, err := New(WithTemplateRenderer(templateRenderer))
	require.NoError(t, err)

	documents, err := renderer.RenderPages([]Page{
		{
			Fields:     map[string]any{"content": "one", "title": "One"},
			OutputPath: "one/index.html",
			Permalink:  "/one/",
			Template:   "page",
		},
		{Fields: map[string]any{"content": "two"}, OutputPath: "two.txt", Permalink: "/two.txt"},
	}, nil)
	require.NoError(t, err)
	require.Len(t, documents, 2)
	require.Len(t, templateRenderer.contexts, 1)

	context := templateRenderer.contexts[0].(map[string]any)
	require.Equal(t, []map[string]any{
		{
			"content":    SafeHTML("one"),
			"outputPath": "one/index.html",
			"permalink":  "/one/",
			"template":   "page",
			"title":      "One",
		},
		{
			"content":    "two",
			"outputPath": "two.txt",
			"permalink":  "/two.txt",
			"template":   "",
		},
	}, context["pages"])
}

// TestRenderErrors verifies dependency and template render failures.
func TestRenderErrors(t *testing.T) {
	_, err := (&Renderer{}).Render(Page{Template: "page"}, nil)
	require.ErrorIs(t, err, ErrTemplateRendererRequired)

	renderer, err := New(WithTemplateRenderer(failingTemplateRenderer{}))
	require.NoError(t, err)
	_, err = renderer.Render(Page{Fields: map[string]any{"content": "bad"}, Template: "page"}, nil)
	require.ErrorContains(t, err, "template failed")
}
