package jsruntime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dop251/goja"
)

const defaultSourceName = "anonymous.js"

// Source is an in-memory JavaScript file.
type Source struct {
	// Name is used in error messages and JavaScript stack traces.
	Name string

	// Code is the JavaScript source code to execute.
	Code string
}

// name returns the source name used in JavaScript stack traces.
func (s Source) name() string {
	name := strings.TrimSpace(s.Name)
	if name == "" {
		return defaultSourceName
	}

	return name
}

// Option configures a Runner.
type Option func(*Runner)

// Runner executes Veta JavaScript files.
//
// Runner is safe to reuse because each execution receives a fresh Goja runtime.
type Runner struct {
	runtime Runtime
}

// New creates a Runner with the provided options.
func New(options ...Option) *Runner {
	runner := &Runner{runtime: defaultRuntime()}
	for _, option := range options {
		option(runner)
	}

	return runner
}

// ExecuteFile reads and executes a JavaScript file from disk.
func (r *Runner) ExecuteFile(path string) (Result, error) {
	code, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("read javascript file %s: %w", path, err)
	}

	return r.Execute(Source{Name: filepath.ToSlash(path), Code: string(code)})
}

// ExecuteString executes JavaScript source code with the provided source name.
func (r *Runner) ExecuteString(name, code string) (Result, error) {
	return r.Execute(Source{Name: name, Code: code})
}

// Execute runs a Veta JavaScript source synchronously.
func (r *Runner) Execute(source Source) (Result, error) {
	name := source.name()
	programSource := buildProgramSource(source)

	program, err := goja.Compile(name, programSource, true)
	if err != nil {
		return Result{}, fmt.Errorf("%s: compile javascript: %w", name, err)
	}

	vm, runtimeValue, err := r.newVM()
	if err != nil {
		return Result{}, fmt.Errorf("%s: initialize javascript runtime: %w", name, err)
	}

	if _, err := vm.RunProgram(program); err != nil {
		return Result{}, fmt.Errorf("%s: evaluate javascript: %w", name, err)
	}

	defaultFunction, err := exportedDefaultFunction(vm)
	if err != nil {
		return Result{}, fmt.Errorf("%s: %w", name, err)
	}

	value, err := defaultFunction(goja.Undefined(), runtimeValue)
	if err != nil {
		return Result{}, fmt.Errorf("%s: execute default export: %w", name, err)
	}

	if isPromiseLike(vm, value) {
		return Result{}, fmt.Errorf("%s: %w", name, ErrPromiseUnsupported)
	}

	return Result{runtime: vm, value: value}, nil
}

// exportedDefaultFunction returns the function captured by the export-default
// instrumentation.
func exportedDefaultFunction(vm *goja.Runtime) (goja.Callable, error) {
	if vm.Get(defaultExportDuplicateIdentifier).ToBoolean() {
		return nil, ErrMultipleDefaultExports
	}

	if !vm.Get(defaultExportDefinedIdentifier).ToBoolean() {
		return nil, ErrMissingDefaultExport
	}

	defaultFunction, ok := goja.AssertFunction(vm.Get(defaultExportIdentifier))
	if !ok {
		return nil, ErrDefaultExportNotFunction
	}

	return defaultFunction, nil
}

// isPromiseLike reports whether a value behaves like a JavaScript Promise.
func isPromiseLike(vm *goja.Runtime, value goja.Value) bool {
	if value == nil || goja.IsNull(value) || goja.IsUndefined(value) {
		return false
	}

	then := value.ToObject(vm).Get("then")
	_, ok := goja.AssertFunction(then)

	return ok
}
