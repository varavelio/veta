package filters

import (
	"encoding/json"
	"fmt"
)

// markdownFilter returns the native markdown filter.
func markdownFilter(renderer MarkdownRenderer) Func {
	return func(input, _ any) (any, error) {
		if renderer == nil {
			return nil, ErrMarkdownRendererRequired
		}

		output, err := renderer.Render(fmt.Sprint(input))
		if err != nil {
			return nil, fmt.Errorf("render markdown filter: %w", err)
		}

		return SafeHTML(output), nil
	}
}

// jsonFilter returns a JSON string safe for HTML script contexts.
func jsonFilter(input, _ any) (any, error) {
	content, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("render json filter: %w", err)
	}

	return SafeHTML(content), nil
}
