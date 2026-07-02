package template

import (
	"fmt"
	"regexp"
)

// WithRegexReplace registers the regex_replace template function.
func WithRegexReplace() Option {
	return WithGlobal("regex_replace", RegexReplace)
}

// RegexReplace replaces all matches of pattern in value with replacement.
func RegexReplace(value, pattern, replacement any) (string, error) {
	expression, err := regexp.Compile(fmt.Sprint(pattern))
	if err != nil {
		return "", fmt.Errorf("compile regex_replace pattern: %w", err)
	}

	return expression.ReplaceAllString(fmt.Sprint(value), fmt.Sprint(replacement)), nil
}
