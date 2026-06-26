package jsruntime

import (
	"fmt"
	"math"
	"time"

	"github.com/dop251/goja"
)

// requiredStringArgument returns a JavaScript string argument or a clear script
// author-facing validation error.
func requiredStringArgument(value goja.Value, label string) (string, error) {
	if isJavaScriptNullish(value) {
		return "", fmt.Errorf("%s is required", label)
	}

	text, ok := value.Export().(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", label)
	}

	return text, nil
}

// positiveMilliseconds converts a JavaScript number into a positive duration.
func positiveMilliseconds(value goja.Value, label string) (time.Duration, error) {
	var milliseconds float64
	switch number := value.Export().(type) {
	case int64:
		milliseconds = float64(number)
	case float64:
		milliseconds = number
	default:
		return 0, fmt.Errorf("%s must be a positive number", label)
	}

	maxMilliseconds := float64(int64(1<<63-1) / int64(time.Millisecond))
	if milliseconds <= 0 || math.IsNaN(milliseconds) || math.IsInf(milliseconds, 0) || milliseconds > maxMilliseconds {
		return 0, fmt.Errorf("%s must be a positive number", label)
	}

	return time.Duration(milliseconds * float64(time.Millisecond)), nil
}

// isJavaScriptNullish reports whether a Goja value is absent, undefined, or
// null.
func isJavaScriptNullish(value goja.Value) bool {
	return value == nil || goja.IsUndefined(value) || goja.IsNull(value)
}
