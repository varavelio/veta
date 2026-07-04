package functions

import "errors"

var (
	// ErrFormatUnsupported reports a function file with an unsupported format.
	ErrFormatUnsupported = errors.New("function format unsupported")
	// ErrFSRequired reports a missing filesystem.
	ErrFSRequired = errors.New("function filesystem required")
	// ErrLoadDataRequired reports a missing load_data implementation.
	ErrLoadDataRequired = errors.New("load_data function required")
	// ErrNameInvalid reports an invalid function name.
	ErrNameInvalid = errors.New("function name invalid")
	// ErrNestedUnsupported reports nested function directories.
	ErrNestedUnsupported = errors.New("nested function directories unsupported")
	// ErrRunnerRequired reports a missing JavaScript runner.
	ErrRunnerRequired = errors.New("function runner required")
	// ErrScriptInvalid reports a custom JavaScript function execution error.
	ErrScriptInvalid = errors.New("function script invalid")
)
