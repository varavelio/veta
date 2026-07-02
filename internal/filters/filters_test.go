package filters

import (
	"errors"
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

type testScriptRunner struct {
	calls []Source
}

func (runner *testScriptRunner) Run(source Source, input, parameter any) (any, error) {
	runner.calls = append(runner.calls, source)
	return fmt.Sprintf("%s:%v:%v", source.Name, input, parameter), nil
}

type failingScriptRunner struct{}

func (failingScriptRunner) Run(Source, any, any) (any, error) {
	return nil, errors.New("boom")
}

// TestLoad verifies native plus JavaScript filters and override behavior.
func TestLoad(t *testing.T) {
	runner := &testScriptRunner{}
	set, err := Load(fstest.MapFS{
		"filters/markdown.js": {Data: []byte(`export default function() { return "custom"; }`)},
		"filters/upper.js": {
			Data: []byte(`export default function(input) { return input.toUpperCase(); }`),
		},
	}, WithMarkdownRenderer(testMarkdownRenderer{}), WithScriptRunner(runner))
	require.NoError(t, err)
	require.Equal(
		t,
		[]string{
			"json",
			"markdown",
			"parse_json",
			"parse_markdown",
			"parse_toml",
			"parse_yaml",
			"upper",
		},
		set.Names(),
	)

	upper, ok := set.Get("upper")
	require.True(t, ok)
	output, err := upper("veta", "param")
	require.NoError(t, err)
	require.Equal(t, "filters/upper.js:veta:param", output)

	markdown, ok := set.Get("markdown")
	require.True(t, ok)
	output, err = markdown("veta", nil)
	require.NoError(t, err)
	require.Equal(t, "filters/markdown.js:veta:<nil>", output)
}

// TestLoadMissingDirectory verifies that missing filters directories return
// native filters.
func TestLoadMissingDirectory(t *testing.T) {
	set, err := Load(fstest.MapFS{}, WithMarkdownRenderer(testMarkdownRenderer{}))
	require.NoError(t, err)
	require.Equal(
		t,
		[]string{"json", "markdown", "parse_json", "parse_markdown", "parse_toml", "parse_yaml"},
		set.Names(),
	)
}

// TestLoadErrors verifies filter loading validation.
func TestLoadErrors(t *testing.T) {
	_, err := Load(nil)
	require.ErrorIs(t, err, ErrFSRequired)

	_, err = Load(fstest.MapFS{"filters/upper.js": {Data: []byte("")}})
	require.ErrorIs(t, err, ErrRunnerRequired)

	_, err = Load(
		fstest.MapFS{"filters/text/upper.js": {Data: []byte("")}},
		WithScriptRunner(&testScriptRunner{}),
	)
	require.ErrorIs(t, err, ErrNestedUnsupported)

	_, err = Load(
		fstest.MapFS{"filters/readme.md": {Data: []byte("")}},
		WithScriptRunner(&testScriptRunner{}),
	)
	require.ErrorIs(t, err, ErrFormatUnsupported)

	_, err = Load(
		fstest.MapFS{"filters/bad name.js": {Data: []byte("")}},
		WithScriptRunner(&testScriptRunner{}),
	)
	require.ErrorIs(t, err, ErrNameInvalid)
}

// TestScriptFilterErrors verifies JavaScript filter execution errors.
func TestScriptFilterErrors(t *testing.T) {
	set, err := Load(
		fstest.MapFS{"filters/broken.js": {Data: []byte("bad")}},
		WithScriptRunner(failingScriptRunner{}),
	)
	require.NoError(t, err)

	filter, ok := set.Get("broken")
	require.True(t, ok)
	_, err = filter("input", nil)
	require.ErrorIs(t, err, ErrScriptInvalid)
}
