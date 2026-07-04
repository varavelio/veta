package functions

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLoadDataFunction verifies load_data request normalization.
func TestLoadDataFunction(t *testing.T) {
	function := loadDataFunction(func(request LoadDataRequest) (any, error) {
		return request, nil
	})

	local, err := function(Context{}, "data/site.yaml")
	require.NoError(t, err)
	require.Equal(t, LoadDataRequest{Path: "data/site.yaml"}, local)

	remote, err := function(Context{}, "https://example.test/site.yaml")
	require.NoError(t, err)
	require.Equal(t, LoadDataRequest{URL: "https://example.test/site.yaml"}, remote)
}

// TestLoadDataFunctionErrors verifies load_data argument validation.
func TestLoadDataFunctionErrors(t *testing.T) {
	_, err := loadDataFunction(nil)(Context{}, "data/site.yaml")
	require.ErrorIs(t, err, ErrLoadDataRequired)

	_, err = loadDataFunction(func(LoadDataRequest) (any, error) { return nil, nil })(Context{})
	require.ErrorContains(t, err, "load_data expects 1 argument")
}

// TestRegexReplaceFunction verifies the regex_replace built-in function.
func TestRegexReplaceFunction(t *testing.T) {
	output, err := regexReplaceFunction(Context{}, "World Hello", `(\w+) (\w+)`, "$2 $1")
	require.NoError(t, err)
	require.Equal(t, "Hello World", output)
}

// TestRegexReplaceFunctionErrors verifies regex_replace argument validation.
func TestRegexReplaceFunctionErrors(t *testing.T) {
	_, err := regexReplaceFunction(Context{}, "value")
	require.ErrorContains(t, err, "regex_replace expects 3 arguments")

	_, err = regexReplaceFunction(Context{}, "value", "[", "")
	require.ErrorContains(t, err, "compile regex_replace pattern")
}
