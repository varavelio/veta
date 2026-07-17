package functions

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"sort"
	"strings"

	"github.com/varavelio/veta/internal/sourcefile"
)

// DirName is the project directory containing custom template function scripts.
const DirName = "functions"

// Context contains values available to one template function call.
type Context map[string]any

// Func executes one template function with the current template context.
type Func func(context Context, args ...any) (any, error)

// Source is a function script loaded from the functions directory.
type Source struct {
	Name string
	Code string
}

// ScriptRunner executes a JavaScript function with context and explicit args.
type ScriptRunner interface {
	Run(source Source, context Context, args ...any) (any, error)
}

// Set contains template functions keyed by their template name.
type Set struct {
	functions map[string]Func
}

// Option configures function loading.
type Option func(*loadConfig) error

type loadConfig struct {
	loadData LoadDataFunc
	native   bool
	runner   ScriptRunner
}

// Load returns built-in functions plus user functions from the functions directory.
func Load(files fs.FS, options ...Option) (Set, error) {
	if files == nil {
		return Set{}, ErrFSRequired
	}

	config, err := newLoadConfig(options)
	if err != nil {
		return Set{}, err
	}

	set := Set{functions: map[string]Func{}}
	if config.native {
		set.Merge(Builtin(config.loadData))
	}

	entries, err := fs.ReadDir(files, DirName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return set, nil
		}

		return Set{}, fmt.Errorf("read functions directory %s: %w", DirName, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			return Set{}, fmt.Errorf("%w: %s/%s", ErrNestedUnsupported, DirName, entry.Name())
		}
		if !sourcefile.Allowed(entry.Name()) {
			continue
		}
		if config.runner == nil {
			return Set{}, ErrRunnerRequired
		}

		name, err := functionName(entry.Name())
		if err != nil {
			return Set{}, err
		}

		filePath := path.Join(DirName, entry.Name())
		content, err := fs.ReadFile(files, filePath)
		if err != nil {
			return Set{}, fmt.Errorf("read function %s: %w", filePath, err)
		}

		set.functions[name] = scriptFunction(
			config.runner,
			Source{Name: filePath, Code: string(content)},
		)
	}

	return set, nil
}

// Builtin returns Veta's built-in template functions.
func Builtin(loadData LoadDataFunc) Set {
	return Set{functions: map[string]Func{
		"load_data":     loadDataFunction(loadData),
		"regex_replace": regexReplaceFunction,
		"url":           urlFunction,
	}}
}

// WithLoadData configures the load_data built-in function.
func WithLoadData(loadData LoadDataFunc) Option {
	return func(config *loadConfig) error {
		config.loadData = loadData
		return nil
	}
}

// WithNative configures whether native functions are included.
func WithNative(enabled bool) Option {
	return func(config *loadConfig) error {
		config.native = enabled
		return nil
	}
}

// WithScriptRunner configures the runner used for JavaScript functions.
func WithScriptRunner(runner ScriptRunner) Option {
	return func(config *loadConfig) error {
		config.runner = runner
		return nil
	}
}

// Get returns a function by name.
func (set Set) Get(name string) (Func, bool) {
	function, ok := set.functions[name]
	return function, ok
}

// Merge copies functions from other into set, overriding existing names.
func (set Set) Merge(other Set) {
	if set.functions == nil {
		set.functions = map[string]Func{}
	}
	maps.Copy(set.functions, other.functions)
}

// Names returns function names in deterministic order.
func (set Set) Names() []string {
	names := make([]string, 0, len(set.functions))
	for name := range set.functions {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

// Functions returns a copy of the function map.
func (set Set) Functions() map[string]Func {
	functions := make(map[string]Func, len(set.functions))
	maps.Copy(functions, set.functions)

	return functions
}

// Inject adds callable template functions to context.
func Inject(context map[string]any, set Set) {
	for name, function := range set.Functions() {
		context[name] = bind(context, function)
	}
}

// Runtime returns JavaScript runtime values for a template function call.
func Runtime(context Context) map[string]any {
	return map[string]any{
		"data":  context["data"],
		"pages": context["pages"],
		"page":  context["page"],
		"props": context["props"],
	}
}

func bind(context map[string]any, function Func) func(...any) (any, error) {
	return func(args ...any) (any, error) {
		return function(Context(context), args...)
	}
}

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

func functionName(fileName string) (string, error) {
	if strings.ToLower(path.Ext(fileName)) != ".js" {
		return "", fmt.Errorf("%w: %s", ErrFormatUnsupported, path.Join(DirName, fileName))
	}

	name := strings.TrimSuffix(fileName, path.Ext(fileName))
	if name == "" || strings.ContainsAny(name, "/\\ \t\r\n|:()") || strings.ContainsRune(name, 0) {
		return "", fmt.Errorf("%w: %s", ErrNameInvalid, fileName)
	}

	return name, nil
}

func scriptFunction(runner ScriptRunner, source Source) Func {
	return func(context Context, args ...any) (any, error) {
		output, err := runner.Run(source, context, args...)
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrScriptInvalid, source.Name, err)
		}

		return output, nil
	}
}
