package filters

import (
	"errors"
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

type testMarkdownRenderer struct{}

func (testMarkdownRenderer) Render(content string) (string, error) {
	return "<p>" + content + "</p>", nil
}

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

// TestNative verifies native filters.
func TestNative(t *testing.T) {
	set := Native(testMarkdownRenderer{})
	require.Equal(t, []string{"json", "markdown", "slugify"}, set.Names())

	markdown, ok := set.Get("markdown")
	require.True(t, ok)
	markdownOutput, err := markdown("**Veta**", nil)
	require.NoError(t, err)
	require.Equal(t, SafeHTML("<p>**Veta**</p>"), markdownOutput)

	jsonFilter, ok := set.Get("json")
	require.True(t, ok)
	jsonOutput, err := jsonFilter(map[string]any{"tag": "<x>"}, nil)
	require.NoError(t, err)
	require.Equal(t, SafeHTML(`{"tag":"\u003cx\u003e"}`), jsonOutput)

	slugify, ok := set.Get("slugify")
	require.True(t, ok)
	slug, err := slugify(" Hello, Veta SSG! ", nil)
	require.NoError(t, err)
	require.Equal(t, "hello-veta-ssg", slug)
}

// TestNativeMarkdownRequiresRenderer verifies markdown renderer validation.
func TestNativeMarkdownRequiresRenderer(t *testing.T) {
	markdown, ok := Native(nil).Get("markdown")
	require.True(t, ok)

	_, err := markdown("content", nil)
	require.ErrorIs(t, err, ErrMarkdownRendererRequired)
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
	require.Equal(t, []string{"json", "markdown", "slugify", "upper"}, set.Names())

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
	require.Equal(t, []string{"json", "markdown", "slugify"}, set.Names())
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
