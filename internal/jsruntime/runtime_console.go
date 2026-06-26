package jsruntime

import (
	"fmt"
	"io"
	"strings"

	"github.com/dop251/goja"
)

// installConsole exposes a small synchronous console API in JavaScript.
func (r *Runner) installConsole(vm *goja.Runtime) error {
	console := vm.NewObject()
	for _, method := range []string{"debug", "error", "info", "log", "warn"} {
		if err := console.Set(method, r.consoleMethod(method)); err != nil {
			return fmt.Errorf("set console.%s: %w", method, err)
		}
	}

	if err := vm.Set("console", console); err != nil {
		return fmt.Errorf("set console global: %w", err)
	}

	return nil
}

// consoleMethod returns a Goja callback for one console method.
func (r *Runner) consoleMethod(level string) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		r.writeConsoleLine(level, call.Arguments)
		return goja.Undefined()
	}
}

// writeConsoleLine writes one formatted console line to the configured output.
func (r *Runner) writeConsoleLine(level string, arguments []goja.Value) {
	output := r.consoleWriter()
	if output == io.Discard {
		return
	}

	r.consoleMu.Lock()
	defer r.consoleMu.Unlock()

	_, _ = fmt.Fprintf(output, "[js %s]", level)
	if len(arguments) > 0 {
		_, _ = fmt.Fprint(output, " ", formatConsoleArguments(arguments))
	}
	_, _ = fmt.Fprintln(output)
}

// consoleWriter returns the writer used by JavaScript console methods.
func (r *Runner) consoleWriter() io.Writer {
	if r == nil || r.consoleOutput == nil {
		return io.Discard
	}

	return r.consoleOutput
}

// formatConsoleArguments formats JavaScript console arguments for terminal logs.
func formatConsoleArguments(arguments []goja.Value) string {
	formatted := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		if goja.IsUndefined(argument) {
			formatted = append(formatted, "undefined")
			continue
		}

		if goja.IsNull(argument) {
			formatted = append(formatted, "null")
			continue
		}

		formatted = append(formatted, fmt.Sprint(argument.Export()))
	}

	return strings.Join(formatted, " ")
}
