package template

import (
	"fmt"
	"reflect"
	"strings"
)

// LoadDataRequest describes one template load_data call.
type LoadDataRequest struct {
	Path      string
	URL       string
	Format    string
	TimeoutMs int
}

// LoadDataFunc loads data for the load_data template helper.
type LoadDataFunc func(LoadDataRequest) (any, error)

// WithLoadData registers the load_data function for this renderer.
func WithLoadData(loader LoadDataFunc) Option {
	return func(config *rendererConfig) error {
		if loader == nil {
			return fmt.Errorf("%w: load_data", ErrGlobalNameInvalid)
		}

		config.globals["load_data"] = positionalLoadData(loader)
		return nil
	}
}

func positionalLoadData(loader LoadDataFunc) func(string, ...any) (any, error) {
	return func(source string, arguments ...any) (any, error) {
		request, err := positionalLoadDataRequest(source, arguments)
		if err != nil {
			return nil, err
		}
		if isTemplateRemoteURL(source) {
			request.Path = ""
			request.URL = source
		}

		return loader(request)
	}
}

func positionalLoadDataRequest(source string, arguments []any) (LoadDataRequest, error) {
	if len(arguments) > 2 {
		return LoadDataRequest{}, fmt.Errorf(
			"load_data accepts source, optional format, and optional timeout_ms",
		)
	}

	request := LoadDataRequest{Path: source}
	if len(arguments) >= 1 {
		format, ok := arguments[0].(string)
		if !ok {
			return LoadDataRequest{}, fmt.Errorf("load_data format must be a string")
		}
		request.Format = format
	}
	if len(arguments) == 2 {
		timeoutMs, err := loadDataTimeoutMs(arguments[1])
		if err != nil {
			return LoadDataRequest{}, err
		}
		request.TimeoutMs = timeoutMs
	}

	return request, nil
}

func loadDataTimeoutMs(value any) (int, error) {
	integer := reflect.ValueOf(value)
	if !integer.IsValid() {
		return 0, fmt.Errorf("load_data timeout_ms must be an integer")
	}

	switch integer.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		timeoutMs := integer.Int()
		if timeoutMs < 0 {
			return 0, fmt.Errorf("load_data timeout_ms cannot be negative")
		}
		return int(timeoutMs), nil
	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr:
		timeoutMs := integer.Uint()
		if timeoutMs > uint64(^uint(0)>>1) {
			return 0, fmt.Errorf("load_data timeout_ms is too large")
		}
		return int(timeoutMs), nil
	default:
		return 0, fmt.Errorf("load_data timeout_ms must be an integer")
	}
}

func isTemplateRemoteURL(source string) bool {
	return strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://")
}
