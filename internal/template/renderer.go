package template

import (
	"fmt"
	"io/fs"
	"maps"
	"reflect"

	"github.com/flosch/pongo2/v7"
)

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
	filters map[string]FilterFunc
	globals map[string]any
}

// New creates a Renderer backed by files.
func New(files fs.FS, options ...Option) (*Renderer, error) {
	if files == nil {
		return nil, ErrTemplateFSRequired
	}

	config := rendererConfig{
		filters: map[string]FilterFunc{},
		globals: map[string]any{},
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return nil, err
		}
	}

	loader := &templateLoader{files: files}
	set := pongo2.NewSet("veta", loader)
	maps.Copy(set.Globals, config.globals)

	for name, filter := range config.filters {
		wrappedFilter := wrapFilter(filter)
		if set.FilterExists(name) {
			if err := set.ReplaceFilter(name, wrappedFilter); err != nil {
				return nil, fmt.Errorf("replace template filter %s: %w", name, err)
			}

			continue
		}

		if err := set.RegisterFilter(name, wrappedFilter); err != nil {
			return nil, fmt.Errorf("register template filter %s: %w", name, err)
		}
	}

	return &Renderer{loader: loader, set: set}, nil
}

// WithGlobal registers a global template value or function for this renderer.
func WithGlobal(name string, value any) Option {
	return func(config *rendererConfig) error {
		cleanName, err := cleanGlobalName(name)
		if err != nil {
			return err
		}
		if value == nil {
			return fmt.Errorf("%w: %s", ErrGlobalNameInvalid, cleanName)
		}

		config.globals[cleanName] = value
		return nil
	}
}

// WithExtensions is retained for compatibility.
//
// Deprecated: Veta now resolves extensionless template names by scanning for any
// non-ignored file with a matching stem, regardless of extension.
func WithExtensions(_ ...string) Option {
	return func(config *rendererConfig) error {
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

// normalizeContext converts supported root context maps into Pongo2 context.
func normalizeContext(context any) (pongo2.Context, error) {
	if context == nil {
		return pongo2.Context{}, nil
	}

	switch typedContext := context.(type) {
	case Context:
		return normalizeContextMap(map[string]any(typedContext)), nil
	case pongo2.Context:
		return normalizeContextMap(map[string]any(typedContext)), nil
	case map[string]any:
		return normalizeContextMap(typedContext), nil
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
		normalized[iterator.Key().String()] = normalizeContextValue(iterator.Value().Interface())
	}

	return normalized, nil
}

// normalizeContextMap converts map values into Pongo2-compatible values.
func normalizeContextMap(context map[string]any) pongo2.Context {
	normalized := make(pongo2.Context, len(context))
	for key, value := range context {
		normalized[key] = normalizeContextValue(value)
	}

	return normalized
}

// normalizeContextValue preserves trusted HTML markers inside context values.
func normalizeContextValue(value any) any {
	switch typedValue := value.(type) {
	case safeHTML:
		return pongo2.AsSafeValue(typedValue.SafeHTML())
	case map[string]any:
		return normalizeContextMap(typedValue)
	case Context:
		return normalizeContextMap(map[string]any(typedValue))
	case pongo2.Context:
		return normalizeContextMap(map[string]any(typedValue))
	default:
		return value
	}
}
