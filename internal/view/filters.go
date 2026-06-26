package view

import (
	"fmt"
	"strings"

	"github.com/flosch/pongo2/v7"
)

// FilterFunc transforms a template value. The parameter is nil when the filter
// is called without an argument.
type FilterFunc func(input any, parameter any) (any, error)

// SafeString marks trusted HTML as safe for Pongo2 output.
type SafeString string

func cleanFilterName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, " \t\r\n|:()") {
		return "", ErrFilterNameInvalid
	}

	return name, nil
}

func wrapFilter(filter FilterFunc) pongo2.FilterFunction {
	return func(input *pongo2.Value, parameter *pongo2.Value) (*pongo2.Value, error) {
		output, err := filter(pongoValue(input), pongoValue(parameter))
		if err != nil {
			return nil, err
		}

		return asPongoValue(output), nil
	}
}

func pongoValue(value *pongo2.Value) any {
	if value == nil || value.IsNil() {
		return nil
	}

	return value.Interface()
}

func asPongoValue(value any) *pongo2.Value {
	switch typedValue := value.(type) {
	case SafeString:
		return pongo2.AsSafeValue(string(typedValue))
	case fmt.Stringer:
		return pongo2.AsValue(typedValue.String())
	default:
		return pongo2.AsValue(value)
	}
}
