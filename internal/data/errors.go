package data

import "errors"

var (
	// ErrFSRequired indicates that data was loaded without a filesystem.
	ErrFSRequired = errors.New("data filesystem is required")

	// ErrInvalid indicates that a data file is malformed or cannot be used as
	// global data.
	ErrInvalid = errors.New("data is invalid")

	// ErrKeyDuplicate indicates that more than one data file maps to the same
	// global key.
	ErrKeyDuplicate = errors.New("data key is duplicated")

	// ErrKeyInvalid indicates that a data file name cannot be used as a global key.
	ErrKeyInvalid = errors.New("data key is invalid")

	// ErrNestedUnsupported indicates that the data directory contains a nested
	// directory.
	ErrNestedUnsupported = errors.New("data directory must be flat")

	// ErrFormatUnsupported indicates that a data file has an unsupported extension.
	ErrFormatUnsupported = errors.New("data file format is unsupported")

	// ErrValueUnsupported indicates that a data file produced a value that cannot
	// be represented as JSON-compatible data.
	ErrValueUnsupported = errors.New("data value is not JSON-compatible")
)
