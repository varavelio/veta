package render

import "fmt"

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

// Page contains the page fields render needs from a page manifest.
type Page struct {
	Content    string
	Data       map[string]any
	Date       string
	Layout     string
	OutputPath string
	Permalink  string
	Title      string
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
func (renderer *Renderer) Render(page Page, site any) (Document, error) {
	if renderer == nil {
		renderer = &Renderer{}
	}

	if page.Layout == "" {
		return Document{
			Content:    []byte(page.Content),
			OutputPath: page.OutputPath,
			Permalink:  page.Permalink,
		}, nil
	}
	if renderer.templateRenderer == nil {
		return Document{}, ErrTemplateRendererRequired
	}

	pageContext := pageTemplateContext(page)
	baseContext := map[string]any{"page": pageContext, "site": site}
	content := page.Content
	if renderer.contentProcessor != nil {
		processedContent, err := renderer.contentProcessor.Render(content, baseContext)
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

	context := map[string]any{
		"content": SafeHTML(content),
		"page":    pageContext,
		"site":    site,
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
func (renderer *Renderer) RenderPages(pages []Page, site any) ([]Document, error) {
	documents := make([]Document, 0, len(pages))
	for _, page := range pages {
		document, err := renderer.Render(page, site)
		if err != nil {
			return nil, err
		}

		documents = append(documents, document)
	}

	return documents, nil
}

// pageTemplateContext returns the page namespace exposed to templates.
func pageTemplateContext(page Page) map[string]any {
	data := page.Data
	if data == nil {
		data = map[string]any{}
	}

	return map[string]any{
		"data":       data,
		"date":       page.Date,
		"layout":     page.Layout,
		"outputPath": page.OutputPath,
		"permalink":  page.Permalink,
		"title":      page.Title,
	}
}
