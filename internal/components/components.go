package components

import (
	"fmt"
	"io/fs"
	"sort"
)

// DirName is the project directory containing component templates.
const DirName = "components"

var defaultExtensions = []string{".pongo", ".html"}

// TemplateRenderer renders a component template by name.
type TemplateRenderer interface {
	Render(name string, context any) (string, error)
}

// SlotRenderer renders the inner content of a paired component tag.
type SlotRenderer func(content string, context any) (string, error)

// SafeHTML marks component content as trusted HTML for downstream template
// adapters that understand this structural interface.
type SafeHTML string

// SafeHTML returns the trusted HTML string.
func (html SafeHTML) SafeHTML() string {
	return string(html)
}

// Component describes one registered component template.
type Component struct {
	Tag      string
	Template string
	Path     string
	Depth    int
}

// Conflict describes a component tag registration conflict.
type Conflict struct {
	Tag     string
	Winner  string
	Ignored string
}

// Processor expands registered component tags in content strings.
type Processor struct {
	components   map[string]Component
	conflicts    []Conflict
	renderer     TemplateRenderer
	slotRenderer SlotRenderer
}

// Option configures a Processor.
type Option func(*processorConfig) error

type processorConfig struct {
	extensions   []string
	slotRenderer SlotRenderer
}

// New creates a Processor backed by component templates from files.
func New(files fs.FS, renderer TemplateRenderer, options ...Option) (*Processor, error) {
	if files == nil {
		return nil, ErrFSRequired
	}

	config, err := newProcessorConfig(options)
	if err != nil {
		return nil, err
	}

	registry, conflicts, err := scan(files, config.extensions)
	if err != nil {
		return nil, err
	}
	if len(registry) > 0 && renderer == nil {
		return nil, ErrRendererRequired
	}

	return &Processor{
		components:   registry,
		conflicts:    conflicts,
		renderer:     renderer,
		slotRenderer: config.slotRenderer,
	}, nil
}

// WithExtensions configures component template extensions.
func WithExtensions(extensions ...string) Option {
	return func(config *processorConfig) error {
		normalized, err := normalizeExtensions(extensions)
		if err != nil {
			return err
		}

		config.extensions = normalized
		return nil
	}
}

// WithSlotRenderer configures how paired component inner content is rendered.
func WithSlotRenderer(renderer SlotRenderer) Option {
	return func(config *processorConfig) error {
		config.slotRenderer = renderer
		return nil
	}
}

// Components returns the registered components sorted by tag.
func (processor *Processor) Components() []Component {
	if processor == nil || len(processor.components) == 0 {
		return nil
	}

	components := make([]Component, 0, len(processor.components))
	for _, component := range processor.components {
		components = append(components, component)
	}
	sort.Slice(components, func(left, right int) bool {
		return components[left].Tag < components[right].Tag
	})

	return components
}

// Conflicts returns component registration conflicts detected during discovery.
func (processor *Processor) Conflicts() []Conflict {
	if processor == nil || len(processor.conflicts) == 0 {
		return nil
	}

	return append([]Conflict(nil), processor.conflicts...)
}

// Render expands registered component tags in content.
func (processor *Processor) Render(content string, context any) (string, error) {
	if processor == nil || len(processor.components) == 0 {
		return content, nil
	}
	if processor.renderer == nil {
		return "", ErrRendererRequired
	}

	output, err := processor.renderSegment(content, context)
	if err != nil {
		return "", err
	}

	return output, nil
}

// newProcessorConfig applies options and defaults.
func newProcessorConfig(options []Option) (processorConfig, error) {
	config := processorConfig{extensions: append([]string(nil), defaultExtensions...)}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return processorConfig{}, err
		}
	}

	return config, nil
}

// renderComponent renders one component invocation.
func (processor *Processor) renderComponent(
	token tagToken,
	content string,
	context any,
) (string, error) {
	component := processor.components[token.name]
	renderedContent := content
	if processor.slotRenderer != nil {
		var err error
		renderedContent, err = processor.slotRenderer(content, context)
		if err != nil {
			return "", fmt.Errorf("render component %s slot: %w", token.name, err)
		}
	}

	output, err := processor.renderer.Render(
		component.Template,
		componentContext(context, token.attributes, renderedContent),
	)
	if err != nil {
		return "", fmt.Errorf("render component %s: %w", token.name, err)
	}

	return output, nil
}

// componentContext builds the context passed to component templates.
func componentContext(base any, props map[string]string, content string) map[string]any {
	componentProps := make(map[string]any, len(props)+1)
	for key, value := range props {
		componentProps[key] = value
	}
	componentProps["content"] = SafeHTML(content)

	return map[string]any{
		"data":  contextValue(base, "data"),
		"pages": contextValue(base, "pages"),
		"page":  contextValue(base, "page"),
		"props": componentProps,
	}
}

// contextValue extracts a named value from a context map.
func contextValue(context any, key string) any {
	values, ok := context.(map[string]any)
	if !ok {
		return nil
	}

	return values[key]
}
