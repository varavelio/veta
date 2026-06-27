package render

import (
	"fmt"
	"maps"
)

// TemplateRenderer renders a template by name.
type TemplateRenderer interface {
	Render(name string, context any) (string, error)
}

// ContentProcessor transforms page content before Markdown rendering.
type ContentProcessor interface {
	Render(content string, context any) (string, error)
}

// MarkdownRenderer renders Markdown into HTML.
type MarkdownRenderer interface {
	Render(content string) (string, error)
}

// SafeHTML marks rendered content as trusted HTML for downstream template
// adapters that understand this structural interface.
type SafeHTML string

// SafeHTML returns the trusted HTML string.
func (html SafeHTML) SafeHTML() string {
	return string(html)
}

// Page contains one normalized page and its template context fields.
type Page struct {
	// Fields contains the page object exposed to templates as page.
	Fields     map[string]any
	Layout     string
	OutputPath string
	Permalink  string
}

// Document is a rendered file ready for output writing.
type Document struct {
	Content    []byte
	OutputPath string
	Permalink  string
}

// Renderer composes pages with injected renderers.
type Renderer struct {
	contentProcessor ContentProcessor
	markdownRenderer MarkdownRenderer
	templateRenderer TemplateRenderer
}

// Option configures a Renderer.
type Option func(*Renderer) error

// New creates a Renderer.
func New(options ...Option) (*Renderer, error) {
	renderer := &Renderer{}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(renderer); err != nil {
			return nil, err
		}
	}

	return renderer, nil
}

// WithContentProcessor configures the pre-Markdown content processor.
func WithContentProcessor(processor ContentProcessor) Option {
	return func(renderer *Renderer) error {
		renderer.contentProcessor = processor
		return nil
	}
}

// WithMarkdownRenderer configures the Markdown renderer.
func WithMarkdownRenderer(markdownRenderer MarkdownRenderer) Option {
	return func(renderer *Renderer) error {
		renderer.markdownRenderer = markdownRenderer
		return nil
	}
}

// WithTemplateRenderer configures the template renderer.
func WithTemplateRenderer(templateRenderer TemplateRenderer) Option {
	return func(renderer *Renderer) error {
		renderer.templateRenderer = templateRenderer
		return nil
	}
}

// Render renders one page into a document.
func (renderer *Renderer) Render(page Page, data any) (Document, error) {
	if renderer == nil {
		renderer = &Renderer{}
	}

	pageContext := pageTemplateContext(page)
	if page.Layout == "" {
		return Document{
			Content:    []byte(rawPageContent(pageContext)),
			OutputPath: page.OutputPath,
			Permalink:  page.Permalink,
		}, nil
	}
	if renderer.templateRenderer == nil {
		return Document{}, ErrTemplateRendererRequired
	}

	context := baseTemplateContext(data, pageContext, map[string]any{})
	if content, ok := pageStringField(pageContext, "content"); ok {
		if renderer.contentProcessor != nil {
			processedContent, err := renderer.contentProcessor.Render(content, context)
			if err != nil {
				return Document{}, fmt.Errorf("process page content %s: %w", page.OutputPath, err)
			}

			content = processedContent
		}
		if renderer.markdownRenderer != nil {
			renderedContent, err := renderer.markdownRenderer.Render(content)
			if err != nil {
				return Document{}, fmt.Errorf("render page markdown %s: %w", page.OutputPath, err)
			}

			content = renderedContent
		}

		pageContext["content"] = SafeHTML(content)
	}
	output, err := renderer.templateRenderer.Render(page.Layout, context)
	if err != nil {
		return Document{}, fmt.Errorf("render page layout %s: %w", page.Layout, err)
	}

	return Document{
		Content:    []byte(output),
		OutputPath: page.OutputPath,
		Permalink:  page.Permalink,
	}, nil
}

// RenderPages renders multiple pages into documents.
func (renderer *Renderer) RenderPages(pages []Page, data any) ([]Document, error) {
	documents := make([]Document, 0, len(pages))
	for _, page := range pages {
		document, err := renderer.Render(page, data)
		if err != nil {
			return nil, err
		}

		documents = append(documents, document)
	}

	return documents, nil
}

// pageTemplateContext returns the page namespace exposed to templates.
func pageTemplateContext(page Page) map[string]any {
	context := make(map[string]any, len(page.Fields)+3)
	maps.Copy(context, page.Fields)
	context["outputPath"] = page.OutputPath
	context["permalink"] = page.Permalink
	if _, exists := context["layout"]; !exists {
		context["layout"] = page.Layout
	}

	return context
}

// baseTemplateContext returns the only root keys exposed to templates.
func baseTemplateContext(data any, page, props map[string]any) map[string]any {
	if props == nil {
		props = map[string]any{}
	}
	return map[string]any{
		"data":  data,
		"page":  page,
		"props": props,
	}
}

// rawPageContent returns the raw content string for layout-less pages.
func rawPageContent(page map[string]any) string {
	content, _ := pageStringField(page, "content")
	return content
}

// pageStringField returns a string field from a page context.
func pageStringField(page map[string]any, key string) (string, bool) {
	value, ok := page[key]
	if !ok {
		return "", false
	}

	content, ok := value.(string)
	return content, ok
}
