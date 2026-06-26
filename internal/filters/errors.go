package filters

import "errors"

var (
	// ErrFSRequired indicates that filters were loaded without a filesystem.
	ErrFSRequired = errors.New("filters filesystem is required")

	// ErrRunnerRequired indicates that JavaScript filters cannot be loaded because
	// no script runner was provided.
	ErrRunnerRequired = errors.New("filter script runner is required")

	// ErrMarkdownRendererRequired indicates that the native markdown filter was
	// called without a markdown renderer.
	ErrMarkdownRendererRequired = errors.New("markdown filter renderer is required")

	// ErrNestedUnsupported indicates that the filters directory contains a nested
	// directory.
	ErrNestedUnsupported = errors.New("filters directory must be flat")

	// ErrFormatUnsupported indicates that a filter file has an unsupported
	// extension.
	ErrFormatUnsupported = errors.New("filter file format is unsupported")

	// ErrNameInvalid indicates that a filter file cannot be mapped to a valid
	// filter name.
	ErrNameInvalid = errors.New("filter name is invalid")

	// ErrScriptInvalid indicates that a JavaScript filter failed during execution.
	ErrScriptInvalid = errors.New("filter script is invalid")
)
