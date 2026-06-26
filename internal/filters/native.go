package filters

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
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

// slugifyFilter returns a URL slug for input.
func slugifyFilter(input, _ any) (any, error) {
	return slugify(fmt.Sprint(input)), nil
}

// slugify converts text into a lowercase dash-separated slug.
func slugify(input string) string {
	var output strings.Builder
	lastDash := false
	for _, char := range strings.ToLower(input) {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			output.WriteRune(char)
			lastDash = false
			continue
		}
		if output.Len() == 0 || lastDash {
			continue
		}

		output.WriteByte('-')
		lastDash = true
	}

	return strings.TrimSuffix(output.String(), "-")
}
