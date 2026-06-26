package jsruntime

import (
	"fmt"
	"io"

	"github.com/dop251/goja"
)

// GlobalName is the name of the runtime object exposed to JavaScript files.
const GlobalName = "Veta"

// Runtime contains the values exposed to JavaScript through the Veta object.
type Runtime map[string]any

// WithRuntime configures additional values exposed through the Veta global and
// the default export argument.
func WithRuntime(runtime Runtime) Option {
	return func(runner *Runner) {
		merged := runner.runtimeSnapshot()
		for name, value := range runtime {
			merged[name] = value
		}
		runner.runtime = merged
	}
}

// WithRoot configures the filesystem root used by Veta file APIs.
func WithRoot(root string) Option {
	return func(runner *Runner) {
		runner.root = root
	}
}

// WithConsoleOutput configures where JavaScript console messages are written.
func WithConsoleOutput(output io.Writer) Option {
	return func(runner *Runner) {
		runner.consoleOutput = output
	}
}

// defaultRuntime returns the built-in JavaScript runtime API exposed by Veta.
func defaultRuntime() Runtime {
	return Runtime{}
}

// newVM creates an isolated JavaScript runtime for one source execution.
func (r *Runner) newVM() (*goja.Runtime, *goja.Object, error) {
	vm := goja.New()
	if err := r.installConsole(vm); err != nil {
		return nil, nil, err
	}

	runtimeValue, err := r.newRuntimeObject(vm)
	if err != nil {
		return nil, nil, err
	}

	if err := vm.Set(GlobalName, runtimeValue); err != nil {
		return nil, nil, fmt.Errorf("set %s global: %w", GlobalName, err)
	}

	if err := vm.Set("Promise", goja.Undefined()); err != nil {
		return nil, nil, fmt.Errorf("disable Promise global: %w", err)
	}

	return vm, runtimeValue, nil
}

// newRuntimeObject converts the configured Go runtime API into a Goja object.
func (r *Runner) newRuntimeObject(vm *goja.Runtime) (*goja.Object, error) {
	runtimeValue := vm.NewObject()
	for name, value := range r.runtimeSnapshot() {
		if err := runtimeValue.Set(name, value); err != nil {
			return nil, fmt.Errorf("set %s.%s: %w", GlobalName, name, err)
		}
	}

	fileAPI, err := r.newFileAPI(vm)
	if err != nil {
		return nil, err
	}

	for name, value := range fileAPI {
		if err := runtimeValue.Set(name, value); err != nil {
			return nil, fmt.Errorf("set %s.%s: %w", GlobalName, name, err)
		}
	}

	return runtimeValue, nil
}

// runtimeSnapshot returns a copy of the runtime API configured on the runner.
func (r *Runner) runtimeSnapshot() Runtime {
	if r == nil || r.runtime == nil {
		return defaultRuntime()
	}

	return cloneRuntime(r.runtime)
}

// cloneRuntime copies a runtime API map so executions cannot mutate runner
// configuration accidentally.
func cloneRuntime(runtime Runtime) Runtime {
	clone := make(Runtime, len(runtime))
	for name, value := range runtime {
		clone[name] = value
	}

	return clone
}
