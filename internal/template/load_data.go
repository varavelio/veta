package template

import (
	"fmt"
	"strings"
)

// LoadDataRequest describes one template load_data call.
type LoadDataRequest struct {
	Path string
	URL  string
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

func positionalLoadData(loader LoadDataFunc) func(string) (any, error) {
	return func(source string) (any, error) {
		request := LoadDataRequest{Path: source}
		if isTemplateRemoteURL(source) {
			request.Path = ""
			request.URL = source
		}

		return loader(request)
	}
}

func isTemplateRemoteURL(source string) bool {
	return strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://")
}
