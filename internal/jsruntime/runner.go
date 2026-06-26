package jsruntime

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
)

const (
	defaultExecutionTimeout = 10 * time.Minute
	defaultHTTPTimeout      = 30 * time.Second
	defaultRootDir          = "."
	defaultSourceName       = "anonymous.js"
)

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
	runtime          Runtime
	root             string
	environment      Environment
	consoleOutput    io.Writer
	consoleMu        sync.Mutex
	executionTimeout time.Duration
	httpTimeout      time.Duration
}

// New creates a Runner with the provided options.
func New(options ...Option) *Runner {
	runner := &Runner{
		runtime:          defaultRuntime(),
		root:             defaultRootDir,
		environment:      defaultEnvironment(),
		consoleOutput:    os.Stdout,
		executionTimeout: defaultExecutionTimeout,
		httpTimeout:      defaultHTTPTimeout,
	}
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
	cleanupTimeout := r.startExecutionTimeout(vm)
	defer cleanupTimeout()

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

	promiseLike, err := isPromiseLike(vm, value)
	if err != nil {
		return Result{}, fmt.Errorf("%s: inspect default export result: %w", name, err)
	}
	if promiseLike {
		return Result{}, fmt.Errorf("%s: %w", name, ErrPromiseUnsupported)
	}

	return Result{runtime: vm, value: value}, nil
}

// defaultExecutionTimeoutValue returns the maximum duration for one JavaScript
// execution. A non-positive timeout disables the guard.
func (r *Runner) defaultExecutionTimeoutValue() time.Duration {
	if r == nil {
		return defaultExecutionTimeout
	}

	return r.executionTimeout
}

// startExecutionTimeout interrupts JavaScript execution after the configured
// timeout. The timeout is deliberately high; it is a safety net, not normal flow
// control.
func (r *Runner) startExecutionTimeout(vm *goja.Runtime) func() {
	timeout := r.defaultExecutionTimeoutValue()
	if timeout <= 0 {
		return func() {}
	}

	done := make(chan struct{})
	timer := time.AfterFunc(timeout, func() {
		vm.Interrupt(ErrExecutionTimeout)
		close(done)
	})

	return func() {
		if timer.Stop() {
			return
		}

		<-done
		vm.ClearInterrupt()
	}
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
func isPromiseLike(vm *goja.Runtime, value goja.Value) (bool, error) {
	if value == nil || goja.IsNull(value) || goja.IsUndefined(value) {
		return false, nil
	}

	var promiseLike bool
	if exception := vm.Try(func() {
		then := value.ToObject(vm).Get("then")
		_, promiseLike = goja.AssertFunction(then)
	}); exception != nil {
		return false, exception
	}

	return promiseLike, nil
}
