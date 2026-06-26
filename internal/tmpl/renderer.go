package tmpl

import (
	"fmt"
	"io/fs"
	"reflect"
	"strings"

	"github.com/flosch/pongo2/v7"
)

var defaultExtensions = []string{".pongo", ".html"}

// Context contains template variables. Values may be any Go value supported by
// Pongo2.
type Context map[string]any

// Option configures a Renderer.
type Option func(*rendererConfig) error

// Renderer renders named templates from a filesystem.
type Renderer struct {
	loader *templateLoader
	set    *pongo2.TemplateSet
}

type rendererConfig struct {
	debug      bool
	extensions []string
	filters    map[string]FilterFunc
}

// New creates a Renderer backed by files.
func New(files fs.FS, options ...Option) (*Renderer, error) {
	if files == nil {
		return nil, ErrTemplateFSRequired
	}

	config := rendererConfig{
		extensions: append([]string(nil), defaultExtensions...),
		filters:    map[string]FilterFunc{},
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return nil, err
		}
	}

	loader := &templateLoader{
		extensions: append([]string(nil), config.extensions...),
		files:      files,
	}
	set := pongo2.NewSet("veta", loader)
	set.Debug = config.debug

	for name, filter := range config.filters {
		if err := set.RegisterFilter(name, wrapFilter(filter)); err != nil {
			return nil, fmt.Errorf("register template filter %s: %w", name, err)
		}
	}

	return &Renderer{loader: loader, set: set}, nil
}

// WithDebug configures whether templates are cached. Debug mode disables
// caching so changes in the backing filesystem are picked up immediately.
func WithDebug(debug bool) Option {
	return func(config *rendererConfig) error {
		config.debug = debug
		return nil
	}
}

// WithExtensions configures extension fallbacks for extensionless template
// names. Extensions may be provided with or without a leading dot.
func WithExtensions(extensions ...string) Option {
	return func(config *rendererConfig) error {
		normalized, err := normalizeExtensions(extensions)
		if err != nil {
			return err
		}

		config.extensions = normalized
		return nil
	}
}

// WithFilter registers a template filter for this renderer.
func WithFilter(name string, filter FilterFunc) Option {
	return func(config *rendererConfig) error {
		cleanName, err := cleanFilterName(name)
		if err != nil {
			return err
		}
		if filter == nil {
			return fmt.Errorf("%w: %s", ErrFilterNameInvalid, cleanName)
		}

		config.filters[cleanName] = filter
		return nil
	}
}

// Render renders a named template with context. The context must be nil or a map
// with string keys; nested values may be any Go value supported by Pongo2.
func (renderer *Renderer) Render(name string, context any) (string, error) {
	contextValue, err := normalizeContext(context)
	if err != nil {
		return "", err
	}

	resolvedName, err := renderer.loader.resolve(name)
	if err != nil {
		return "", fmt.Errorf("resolve template %s: %w", name, err)
	}

	template, err := renderer.set.FromCache(resolvedName)
	if err != nil {
		return "", fmt.Errorf("load template %s: %w", name, err)
	}

	output, err := template.Execute(contextValue)
	if err != nil {
		return "", fmt.Errorf("render template %s: %w", name, err)
	}

	return output, nil
}

func normalizeExtensions(extensions []string) ([]string, error) {
	if len(extensions) == 0 {
		return nil, fmt.Errorf(
			"%w: at least one template extension is required",
			ErrTemplateNameInvalid,
		)
	}

	normalized := make([]string, 0, len(extensions))
	for _, extension := range extensions {
		extension = strings.TrimSpace(extension)
		if extension == "" {
			return nil, fmt.Errorf("%w: template extension cannot be empty", ErrTemplateNameInvalid)
		}
		if !strings.HasPrefix(extension, ".") {
			extension = "." + extension
		}
		if strings.ContainsAny(extension, "/\\") || strings.ContainsRune(extension, 0) ||
			extension == "." {
			return nil, fmt.Errorf(
				"%w: invalid template extension %q",
				ErrTemplateNameInvalid,
				extension,
			)
		}

		normalized = append(normalized, extension)
	}

	return normalized, nil
}

func normalizeContext(context any) (pongo2.Context, error) {
	if context == nil {
		return pongo2.Context{}, nil
	}

	switch typedContext := context.(type) {
	case Context:
		return pongo2.Context(typedContext), nil
	case pongo2.Context:
		return typedContext, nil
	case map[string]any:
		return pongo2.Context(typedContext), nil
	}

	value := reflect.ValueOf(context)
	if value.Kind() != reflect.Map || value.Type().Key().Kind() != reflect.String {
		return nil, fmt.Errorf(
			"%w: root context must be a map with string keys",
			ErrContextUnsupported,
		)
	}

	normalized := make(pongo2.Context, value.Len())
	iterator := value.MapRange()
	for iterator.Next() {
		normalized[iterator.Key().String()] = iterator.Value().Interface()
	}

	return normalized, nil
}
