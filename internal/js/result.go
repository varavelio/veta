package js

import "github.com/dop251/goja"

// Result contains the value produced by a JavaScript execution.
//
// The value remains tied to the Goja runtime that created it, which allows
// callers to keep JavaScript functions alive for later integration work.
type Result struct {
	runtime *goja.Runtime
	value   goja.Value
}

// Value returns the raw Goja value produced by the executed JavaScript file.
func (r Result) Value() goja.Value {
	return r.value
}

// Export converts the JavaScript value into the closest native Go value.
func (r Result) Export() any {
	if r.value == nil {
		return nil
	}

	return r.value.Export()
}

// ExportTo converts the JavaScript value into target using Goja's conversion
// rules.
func (r Result) ExportTo(target any) error {
	if r.runtime == nil || r.value == nil {
		return ErrNoResult
	}

	return r.runtime.ExportTo(r.value, target)
}
