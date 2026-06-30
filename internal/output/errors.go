package output

import "errors"

var (
	// ErrDirInvalid indicates that the output directory path is empty or invalid.
	ErrDirInvalid = errors.New("output directory is invalid")

	// ErrPathInvalid indicates that an output file path is empty, absolute, or
	// tries to escape the output directory.
	ErrPathInvalid = errors.New("output path is invalid")

	// ErrPathDuplicate indicates that multiple output files target the same path.
	ErrPathDuplicate = errors.New("output path is duplicated")

	// ErrMinifyFailed indicates that an output file could not be minified.
	ErrMinifyFailed = errors.New("output minification failed")
)
