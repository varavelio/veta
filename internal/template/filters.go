package template

import (
	"fmt"
	"strings"

	"github.com/flosch/pongo2/v7"
)

// FilterFunc transforms a template value. The parameter is nil when the filter
// is called without an argument.
type FilterFunc func(input, parameter any) (any, error)

// SafeString marks trusted HTML as safe for Pongo2 output.
type SafeString string

// safeHTML marks structurally compatible trusted HTML values.
type safeHTML interface {
	SafeHTML() string
}

// cleanFilterName validates a template filter name.
func cleanFilterName(name string) (string, error) {
	return cleanIdentifierName(name, ErrFilterNameInvalid)
}

// cleanGlobalName validates a template global name.
func cleanGlobalName(name string) (string, error) {
	return cleanIdentifierName(name, ErrGlobalNameInvalid)
}

func cleanIdentifierName(name string, invalidError error) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, " \t\r\n|:()") {
		return "", invalidError
	}

	return name, nil
}

// wrapFilter converts a Veta filter into a Pongo2 filter function.
func wrapFilter(filter FilterFunc) pongo2.FilterFunction {
	return func(input, parameter *pongo2.Value) (*pongo2.Value, error) {
		output, err := filter(pongoValue(input), pongoValue(parameter))
		if err != nil {
			return nil, err
		}

		return asPongoValue(output), nil
	}
}

// pongoValue converts an optional Pongo2 value into a Go value.
func pongoValue(value *pongo2.Value) any {
	if value == nil || value.IsNil() {
		return nil
	}

	return value.Interface()
}

// asPongoValue converts a Go value into a Pongo2 value.
func asPongoValue(value any) *pongo2.Value {
	switch typedValue := value.(type) {
	case SafeString:
		return pongo2.AsSafeValue(string(typedValue))
	case safeHTML:
		return pongo2.AsSafeValue(typedValue.SafeHTML())
	case fmt.Stringer:
		return pongo2.AsValue(typedValue.String())
	default:
		return pongo2.AsValue(value)
	}
}
