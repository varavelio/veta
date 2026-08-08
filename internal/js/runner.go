package js

import (
	"fmt"
	"io"
	"io/fs"
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
	componentRenderer ComponentRenderer
	files             fs.FS
	runtime           Runtime
	root              string
	environment       Environment
	consoleOutput     io.Writer
	consoleMu         sync.Mutex
	executionTimeout  time.Duration
	httpTimeout       time.Duration
	markdownRenderer  MarkdownRenderer
	programs          map[programCacheKey]*goja.Program
	programsMu        sync.RWMutex
}

type programCacheKey struct {
	code string
	name string
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
		programs:         map[programCacheKey]*goja.Program{},
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
	return r.execute(source, nil, func(vm *goja.Runtime, runtimeValue *goja.Object) []goja.Value {
		return []goja.Value{runtimeValue}
	})
}

// Call runs a Veta JavaScript source and invokes its default export with the
// runtime context followed by args.
func (r *Runner) Call(source Source, args ...any) (Result, error) {
	return r.call(nil, source, args...)
}

// CallWithRuntime runs a Veta JavaScript source with per-call runtime values and
// invokes its default export with the runtime context followed by args.
func (r *Runner) CallWithRuntime(runtime Runtime, source Source, args ...any) (Result, error) {
	return r.call(runtime, source, args...)
}

func (r *Runner) call(runtime Runtime, source Source, args ...any) (Result, error) {
	return r.execute(
		source,
		runtime,
		func(vm *goja.Runtime, runtimeValue *goja.Object) []goja.Value {
			values := make([]goja.Value, 0, len(args)+1)
			values = append(values, runtimeValue)
			for _, arg := range args {
				values = append(values, vm.ToValue(arg))
			}

			return values
		},
	)
}

// execute runs source and invokes the default export with caller-provided arguments.
func (r *Runner) execute(
	source Source,
	runtime Runtime,
	arguments func(*goja.Runtime, *goja.Object) []goja.Value,
) (Result, error) {
	name := source.name()
	program, err := r.compile(source, name)
	if err != nil {
		return Result{}, fmt.Errorf("%s: compile javascript: %w", name, err)
	}

	vm, runtimeValue, err := r.newVM(runtime)
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

	value, err := defaultFunction(goja.Undefined(), arguments(vm, runtimeValue)...)
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

func (r *Runner) compile(source Source, name string) (*goja.Program, error) {
	key := programCacheKey{name: name, code: source.Code}
	if program, ok := r.cachedProgram(key); ok {
		return program, nil
	}

	program, err := goja.Compile(name, buildProgramSource(source), true)
	if err != nil {
		return nil, err
	}

	r.storeProgram(key, program)
	return program, nil
}

func (r *Runner) cachedProgram(key programCacheKey) (*goja.Program, bool) {
	r.programsMu.RLock()
	defer r.programsMu.RUnlock()

	program, ok := r.programs[key]
	return program, ok
}

func (r *Runner) storeProgram(key programCacheKey, program *goja.Program) {
	r.programsMu.Lock()
	defer r.programsMu.Unlock()

	r.programs[key] = program
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
