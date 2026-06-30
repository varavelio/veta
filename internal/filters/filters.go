package filters

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"sort"
	"strings"
)

// DirName is the project directory containing custom filter scripts.
const DirName = "filters"

// Func transforms a template value. The parameter is nil when the filter is
// called without an argument.
type Func func(input, parameter any) (any, error)

// Source is a filter script loaded from the filters directory.
type Source struct {
	Name string
	Code string
}

// ScriptRunner executes a JavaScript filter with input and parameter values.
type ScriptRunner interface {
	Run(source Source, input, parameter any) (any, error)
}

// MarkdownRenderer renders Markdown for the native markdown filter.
type MarkdownRenderer interface {
	Render(content string) (string, error)
}

// SafeHTML marks filter output as trusted HTML for downstream template adapters
// that understand this structural interface.
type SafeHTML string

// SafeHTML returns the trusted HTML string.
func (html SafeHTML) SafeHTML() string {
	return string(html)
}

// Set contains filters keyed by their template name.
type Set struct {
	filters map[string]Func
}

// Option configures filter loading.
type Option func(*loadConfig) error

type loadConfig struct {
	markdownRenderer MarkdownRenderer
	native           bool
	runner           ScriptRunner
}

// Load returns built-in filters plus user filters from the filters directory.
func Load(files fs.FS, options ...Option) (Set, error) {
	if files == nil {
		return Set{}, ErrFSRequired
	}

	config, err := newLoadConfig(options)
	if err != nil {
		return Set{}, err
	}

	set := Set{filters: map[string]Func{}}
	if config.native {
		set.Merge(Builtin(config.markdownRenderer))
	}

	entries, err := fs.ReadDir(files, DirName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return set, nil
		}

		return Set{}, fmt.Errorf("read filters directory %s: %w", DirName, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			return Set{}, fmt.Errorf("%w: %s/%s", ErrNestedUnsupported, DirName, entry.Name())
		}
		if config.runner == nil {
			return Set{}, ErrRunnerRequired
		}

		name, err := filterName(entry.Name())
		if err != nil {
			return Set{}, err
		}

		filePath := path.Join(DirName, entry.Name())
		content, err := fs.ReadFile(files, filePath)
		if err != nil {
			return Set{}, fmt.Errorf("read filter %s: %w", filePath, err)
		}

		set.filters[name] = scriptFilter(
			config.runner,
			Source{Name: filePath, Code: string(content)},
		)
	}

	return set, nil
}

// Builtin returns Veta's built-in filters.
func Builtin(markdownRenderer MarkdownRenderer) Set {
	return Set{filters: map[string]Func{
		"json":     jsonFilter,
		"markdown": markdownFilter(markdownRenderer),
	}}
}

// WithMarkdownRenderer configures the native markdown filter.
func WithMarkdownRenderer(renderer MarkdownRenderer) Option {
	return func(config *loadConfig) error {
		config.markdownRenderer = renderer
		return nil
	}
}

// WithNative configures whether native filters are included.
func WithNative(enabled bool) Option {
	return func(config *loadConfig) error {
		config.native = enabled
		return nil
	}
}

// WithScriptRunner configures the runner used for JavaScript filters.
func WithScriptRunner(runner ScriptRunner) Option {
	return func(config *loadConfig) error {
		config.runner = runner
		return nil
	}
}

// Get returns a filter by name.
func (set Set) Get(name string) (Func, bool) {
	filter, ok := set.filters[name]
	return filter, ok
}

// Merge copies filters from other into set, overriding existing names.
func (set Set) Merge(other Set) {
	if set.filters == nil {
		set.filters = map[string]Func{}
	}
	maps.Copy(set.filters, other.filters)
}

// Names returns filter names in deterministic order.
func (set Set) Names() []string {
	names := make([]string, 0, len(set.filters))
	for name := range set.filters {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

// Functions returns a copy of the filter map.
func (set Set) Functions() map[string]Func {
	filters := make(map[string]Func, len(set.filters))
	maps.Copy(filters, set.filters)

	return filters
}

// newLoadConfig applies options and defaults.
func newLoadConfig(options []Option) (loadConfig, error) {
	config := loadConfig{native: true}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return loadConfig{}, err
		}
	}

	return config, nil
}

// filterName derives a filter name from a filter script file name.
func filterName(fileName string) (string, error) {
	if strings.ToLower(path.Ext(fileName)) != ".js" {
		return "", fmt.Errorf("%w: %s", ErrFormatUnsupported, path.Join(DirName, fileName))
	}

	name := strings.TrimSuffix(fileName, path.Ext(fileName))
	if name == "" || strings.ContainsAny(name, "/\\ \t\r\n|:()") || strings.ContainsRune(name, 0) {
		return "", fmt.Errorf("%w: %s", ErrNameInvalid, fileName)
	}

	return name, nil
}

// scriptFilter returns a filter backed by a script runner.
func scriptFilter(runner ScriptRunner, source Source) Func {
	return func(input, parameter any) (any, error) {
		output, err := runner.Run(source, input, parameter)
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrScriptInvalid, source.Name, err)
		}

		return output, nil
	}
}
