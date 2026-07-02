package template

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRegexReplace verifies the regex_replace template function implementation.
func TestRegexReplace(t *testing.T) {
	tests := []struct {
		name        string
		value       any
		pattern     any
		replacement any
		want        string
	}{
		{
			name:        "replaces matches",
			value:       "Hello, Veta!",
			pattern:     `[^a-zA-Z0-9]+`,
			replacement: "-",
			want:        "Hello-Veta-",
		},
		{
			name:        "supports capture groups",
			value:       "World Hello",
			pattern:     `(\w+) (\w+)`,
			replacement: "$2 $1",
			want:        "Hello World",
		},
		{
			name:        "supports named capture groups",
			value:       "World Hello",
			pattern:     `(?P<subject>\w+) (?P<greeting>\w+)`,
			replacement: "$greeting $subject",
			want:        "Hello World",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := RegexReplace(test.value, test.pattern, test.replacement)
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

// TestRegexReplaceErrors verifies invalid regex patterns fail clearly.
func TestRegexReplaceErrors(t *testing.T) {
	_, err := RegexReplace("value", "[", "")
	require.ErrorContains(t, err, "compile regex_replace pattern")
}
