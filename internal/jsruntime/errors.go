package jsruntime

import "errors"

var (
	// ErrMissingDefaultExport indicates that the source did not contain an
	// export default declaration.
	ErrMissingDefaultExport = errors.New("javascript source must export a default function")

	// ErrMultipleDefaultExports indicates that the source tried to define more
	// than one default export.
	ErrMultipleDefaultExports = errors.New("javascript source must define only one default export")

	// ErrDefaultExportNotFunction indicates that export default evaluated to a
	// value that cannot be called.
	ErrDefaultExportNotFunction = errors.New("javascript default export must be a function")

	// ErrPromiseUnsupported indicates that a script returned a Promise or a
	// Promise-like value.
	ErrPromiseUnsupported = errors.New("javascript promises are not supported")

	// ErrNoResult indicates that a zero Result was used.
	ErrNoResult = errors.New("javascript result is empty")
)
