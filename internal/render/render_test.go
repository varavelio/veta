package render

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type testContentProcessor struct{}

func (testContentProcessor) Render(content string, context any) (string, error) {
	return "processed(" + content + ")", nil
}

type failingContentProcessor struct{}

func (failingContentProcessor) Render(string, any) (string, error) {
	return "", errors.New("process failed")
}

type testMarkdownRenderer struct{}

func (testMarkdownRenderer) Render(content string) (string, error) {
	return "markdown(" + content + ")", nil
}

type failingMarkdownRenderer struct{}

func (failingMarkdownRenderer) Render(string) (string, error) {
	return "", errors.New("markdown failed")
}

type testTemplateRenderer struct {
	context  any
	contexts []any
	name     string
}

func (renderer *testTemplateRenderer) Render(name string, context any) (string, error) {
	renderer.name = name
	renderer.context = context
	renderer.contexts = append(renderer.contexts, context)
	contextMap := context.(map[string]any)
	page := contextMap["page"].(map[string]any)
	return fmt.Sprintf("%s:%s:%s", name, page["title"], page["content"]), nil
}

type failingTemplateRenderer struct{}

func (failingTemplateRenderer) Render(string, any) (string, error) {
	return "", errors.New("template failed")
}

// TestRenderWithTemplate verifies the full in-memory render sequence.
func TestRenderWithTemplate(t *testing.T) {
	templateRenderer := &testTemplateRenderer{}
	renderer, err := New(
		WithContentProcessor(testContentProcessor{}),
		WithMarkdownRenderer(testMarkdownRenderer{}),
		WithTemplateRenderer(templateRenderer),
	)
	require.NoError(t, err)

	document, err := renderer.Render(Page{
		Fields: map[string]any{
			"content": "hello",
			"date":    "2026-06-26",
			"kind":    "post",
			"title":   "Blog",
		},
		OutputPath: "blog/index.html",
		Permalink:  "/blog/",
		Template:   "layouts/base",
	}, map[string]any{"site": map[string]any{"name": "Veta"}})
	require.NoError(t, err)
	require.Equal(
		t,
		Document{
			Content:    []byte("layouts/base:Blog:markdown(processed(hello))"),
			OutputPath: "blog/index.html",
			Permalink:  "/blog/",
		},
		document,
	)

	context := templateRenderer.context.(map[string]any)
	require.Equal(t, map[string]any{"site": map[string]any{"name": "Veta"}}, context["data"])
	require.Equal(t, map[string]any{}, context["props"])
	require.Equal(t, map[string]any{
		"content":    SafeHTML("markdown(processed(hello))"),
		"date":       "2026-06-26",
		"kind":       "post",
		"outputPath": "blog/index.html",
		"permalink":  "/blog/",
		"template":   "layouts/base",
		"title":      "Blog",
	}, context["page"])
	require.Equal(t, []map[string]any{{
		"content":    SafeHTML("markdown(processed(hello))"),
		"date":       "2026-06-26",
		"kind":       "post",
		"outputPath": "blog/index.html",
		"permalink":  "/blog/",
		"template":   "layouts/base",
		"title":      "Blog",
	}}, context["pages"])
}

// TestRenderWithoutTemplateReturnsRawContent verifies raw output pages.
func TestRenderWithoutTemplateReturnsRawContent(t *testing.T) {
	renderer, err := New(
		WithContentProcessor(testContentProcessor{}),
		WithMarkdownRenderer(testMarkdownRenderer{}),
	)
	require.NoError(t, err)

	document, err := renderer.Render(
		Page{
			Fields:     map[string]any{"content": "raw"},
			OutputPath: "feed.xml",
			Permalink:  "/feed.xml",
		},
		nil,
	)
	require.NoError(t, err)
	require.Equal(
		t,
		Document{Content: []byte("raw"), OutputPath: "feed.xml", Permalink: "/feed.xml"},
		document,
	)
}

// TestRenderPages verifies rendering multiple pages.
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

// TestRenderErrors verifies dependency and render failures.
func TestRenderErrors(t *testing.T) {
	_, err := (&Renderer{}).Render(Page{Template: "page"}, nil)
	require.ErrorIs(t, err, ErrTemplateRendererRequired)

	renderer, err := New(
		WithTemplateRenderer(&testTemplateRenderer{}),
		WithContentProcessor(failingContentProcessor{}),
	)
	require.NoError(t, err)
	_, err = renderer.Render(Page{Fields: map[string]any{"content": "bad"}, Template: "page"}, nil)
	require.ErrorContains(t, err, "process failed")

	renderer, err = New(
		WithTemplateRenderer(&testTemplateRenderer{}),
		WithMarkdownRenderer(failingMarkdownRenderer{}),
	)
	require.NoError(t, err)
	_, err = renderer.Render(Page{Fields: map[string]any{"content": "bad"}, Template: "page"}, nil)
	require.ErrorContains(t, err, "markdown failed")

	renderer, err = New(WithTemplateRenderer(failingTemplateRenderer{}))
	require.NoError(t, err)
	_, err = renderer.Render(Page{Fields: map[string]any{"content": "bad"}, Template: "page"}, nil)
	require.ErrorContains(t, err, "template failed")
}
