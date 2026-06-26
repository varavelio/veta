package markdown

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// Renderer converts Markdown content into HTML.
type Renderer struct {
	markdown goldmark.Markdown
}

// New creates a Renderer configured for Veta page content.
func New() *Renderer {
	return &Renderer{
		markdown: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithRendererOptions(html.WithUnsafe()),
		),
	}
}

// Render converts Markdown content into HTML.
func Render(content string) (string, error) {
	return New().Render(content)
}

// Render converts Markdown content into HTML using renderer's configuration.
func (renderer *Renderer) Render(content string) (string, error) {
	if renderer == nil {
		renderer = New()
	}

	var output bytes.Buffer
	if err := renderer.markdown.Convert([]byte(content), &output); err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}

	return output.String(), nil
}
