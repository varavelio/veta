package filters

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/varavelio/veta/internal/parsecontent"
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

// base64EncodeFilter returns the Base64 encoding of the input string.
func base64EncodeFilter(input, _ any) (any, error) {
	return base64.StdEncoding.EncodeToString(fmt.Append(nil, input)), nil
}

// base64DecodeFilter decodes a Base64 input string.
func base64DecodeFilter(input, _ any) (any, error) {
	content, err := base64.StdEncoding.DecodeString(fmt.Sprint(input))
	if err != nil {
		return nil, fmt.Errorf("decode base64 filter: %w", err)
	}

	return string(content), nil
}

// parseJSONFilter parses a JSON string into template data.
func parseJSONFilter(input, _ any) (any, error) {
	value, err := parsecontent.JSON(fmt.Sprint(input))
	if err != nil {
		return nil, fmt.Errorf("parse json filter: %w", err)
	}

	return value, nil
}

// parseYAMLFilter parses a YAML string into template data.
func parseYAMLFilter(input, _ any) (any, error) {
	value, err := parsecontent.YAML(fmt.Sprint(input))
	if err != nil {
		return nil, fmt.Errorf("parse yaml filter: %w", err)
	}

	return value, nil
}

// parseTOMLFilter parses a TOML string into template data.
func parseTOMLFilter(input, _ any) (any, error) {
	value, err := parsecontent.TOML(fmt.Sprint(input))
	if err != nil {
		return nil, fmt.Errorf("parse toml filter: %w", err)
	}

	return value, nil
}

// parseMarkdownFilter parses Markdown frontmatter without rendering Markdown.
func parseMarkdownFilter(input, _ any) (any, error) {
	value, err := parsecontent.MarkdownMap(fmt.Sprint(input))
	if err != nil {
		return nil, fmt.Errorf("parse markdown filter: %w", err)
	}

	return value, nil
}
