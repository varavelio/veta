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
	context any
	name    string
}

func (renderer *testTemplateRenderer) Render(name string, context any) (string, error) {
	renderer.name = name
	renderer.context = context
	contextMap := context.(map[string]any)
	page := contextMap["page"].(map[string]any)
	return fmt.Sprintf("%s:%s:%s", name, page["title"], contextMap["content"]), nil
}

type failingTemplateRenderer struct{}

func (failingTemplateRenderer) Render(string, any) (string, error) {
	return "", errors.New("template failed")
}

// TestRenderWithLayout verifies the full in-memory render sequence.
func TestRenderWithLayout(t *testing.T) {
	templateRenderer := &testTemplateRenderer{}
	renderer, err := New(
		WithContentProcessor(testContentProcessor{}),
		WithMarkdownRenderer(testMarkdownRenderer{}),
		WithTemplateRenderer(templateRenderer),
	)
	require.NoError(t, err)

	document, err := renderer.Render(Page{
		Content:    "hello",
		Data:       map[string]any{"kind": "post"},
		Date:       "2026-06-26",
		Layout:     "layouts/base",
		OutputPath: "blog/index.html",
		Permalink:  "/blog/",
		Title:      "Blog",
	}, map[string]any{"name": "Veta"})
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
	require.Equal(t, map[string]any{"name": "Veta"}, context["site"])
	require.Equal(t, SafeHTML("markdown(processed(hello))"), context["content"])
	require.Equal(t, map[string]any{
		"data":       map[string]any{"kind": "post"},
		"date":       "2026-06-26",
		"layout":     "layouts/base",
		"outputPath": "blog/index.html",
		"permalink":  "/blog/",
		"title":      "Blog",
	}, context["page"])
}

// TestRenderWithoutLayoutReturnsRawContent verifies raw output pages.
func TestRenderWithoutLayoutReturnsRawContent(t *testing.T) {
	renderer, err := New(
		WithContentProcessor(testContentProcessor{}),
		WithMarkdownRenderer(testMarkdownRenderer{}),
	)
	require.NoError(t, err)

	document, err := renderer.Render(
		Page{Content: "raw", OutputPath: "feed.xml", Permalink: "/feed.xml"},
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
	renderer, err := New(WithTemplateRenderer(&testTemplateRenderer{}))
	require.NoError(t, err)

	documents, err := renderer.RenderPages([]Page{
		{
			Content:    "one",
			Layout:     "page",
			OutputPath: "one/index.html",
			Permalink:  "/one/",
			Title:      "One",
		},
		{Content: "two", OutputPath: "two.txt", Permalink: "/two.txt"},
	}, nil)
	require.NoError(t, err)
	require.Len(t, documents, 2)
}

// TestRenderErrors verifies dependency and render failures.
func TestRenderErrors(t *testing.T) {
	_, err := (&Renderer{}).Render(Page{Layout: "page"}, nil)
	require.ErrorIs(t, err, ErrTemplateRendererRequired)

	renderer, err := New(
		WithTemplateRenderer(&testTemplateRenderer{}),
		WithContentProcessor(failingContentProcessor{}),
	)
	require.NoError(t, err)
	_, err = renderer.Render(Page{Content: "bad", Layout: "page"}, nil)
	require.ErrorContains(t, err, "process failed")

	renderer, err = New(
		WithTemplateRenderer(&testTemplateRenderer{}),
		WithMarkdownRenderer(failingMarkdownRenderer{}),
	)
	require.NoError(t, err)
	_, err = renderer.Render(Page{Content: "bad", Layout: "page"}, nil)
	require.ErrorContains(t, err, "markdown failed")

	renderer, err = New(WithTemplateRenderer(failingTemplateRenderer{}))
	require.NoError(t, err)
	_, err = renderer.Render(Page{Content: "bad", Layout: "page"}, nil)
	require.ErrorContains(t, err, "template failed")
}
