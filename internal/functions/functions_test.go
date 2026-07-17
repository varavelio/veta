package functions

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

func (runner *testScriptRunner) Run(source Source, context Context, args ...any) (any, error) {
	runner.calls = append(runner.calls, source)
	return fmt.Sprintf(
		"%s:%s:%v",
		source.Name,
		context["page"].(map[string]any)["title"],
		args,
	), nil
}

type failingScriptRunner struct{}

func (failingScriptRunner) Run(Source, Context, ...any) (any, error) {
	return nil, errors.New("boom")
}

// TestLoad verifies native plus JavaScript template functions and override behavior.
func TestLoad(t *testing.T) {
	runner := &testScriptRunner{}
	set, err := Load(fstest.MapFS{
		"functions/.draft.js":      {Data: []byte(`invalid`)},
		"functions/custom.test.js": {Data: []byte(`invalid`)},
		"functions/not-valid.js":   {Data: []byte(`invalid`)},
		"functions/custom.js": {
			Data: []byte(`export default function() { return "custom"; }`),
		},
		"functions/regex_replace.js": {
			Data: []byte(`export default function() { return "override"; }`),
		},
	}, WithLoadData(func(request LoadDataRequest) (any, error) {
		return request.Path, nil
	}), WithScriptRunner(runner))
	require.NoError(t, err)
	require.Equal(t, []string{"custom", "load_data", "regex_replace", "url"}, set.Names())

	custom, ok := set.Get("custom")
	require.True(t, ok)
	output, err := custom(Context{"page": map[string]any{"title": "Home"}}, "a", "b")
	require.NoError(t, err)
	require.Equal(t, "functions/custom.js:Home:[a b]", output)

	regexReplace, ok := set.Get("regex_replace")
	require.True(t, ok)
	output, err = regexReplace(Context{"page": map[string]any{"title": "Home"}})
	require.NoError(t, err)
	require.Equal(t, "functions/regex_replace.js:Home:[]", output)
}

// TestLoadMissingDirectory verifies that missing functions directories return native functions.
func TestLoadMissingDirectory(t *testing.T) {
	set, err := Load(fstest.MapFS{}, WithLoadData(func(request LoadDataRequest) (any, error) {
		return request.Path, nil
	}))
	require.NoError(t, err)
	require.Equal(t, []string{"load_data", "regex_replace", "url"}, set.Names())
}

// TestLoadErrors verifies function loading validation.
func TestLoadErrors(t *testing.T) {
	_, err := Load(nil)
	require.ErrorIs(t, err, ErrFSRequired)

	_, err = Load(fstest.MapFS{"functions/custom.js": {Data: []byte("")}})
	require.ErrorIs(t, err, ErrRunnerRequired)

	_, err = Load(
		fstest.MapFS{"functions/nested/custom.js": {Data: []byte("")}},
		WithScriptRunner(&testScriptRunner{}),
	)
	require.ErrorIs(t, err, ErrNestedUnsupported)

	_, err = Load(
		fstest.MapFS{"functions/readme.md": {Data: []byte("")}},
		WithScriptRunner(&testScriptRunner{}),
	)
	require.ErrorIs(t, err, ErrFormatUnsupported)
}

// TestScriptFunctionErrors verifies JavaScript function execution errors.
func TestScriptFunctionErrors(t *testing.T) {
	set, err := Load(
		fstest.MapFS{"functions/broken.js": {Data: []byte("bad")}},
		WithScriptRunner(failingScriptRunner{}),
	)
	require.NoError(t, err)

	function, ok := set.Get("broken")
	require.True(t, ok)
	_, err = function(Context{})
	require.ErrorIs(t, err, ErrScriptInvalid)
}

// TestInject verifies functions are callable from a template context map.
func TestInject(t *testing.T) {
	context := map[string]any{"page": map[string]any{"title": "Home"}}
	Inject(context, Set{functions: map[string]Func{
		"title": func(context Context, args ...any) (any, error) {
			return context["page"].(map[string]any)["title"], nil
		},
	}})

	call, ok := context["title"].(func(...any) (any, error))
	require.True(t, ok)
	output, err := call()
	require.NoError(t, err)
	require.Equal(t, "Home", output)
}
