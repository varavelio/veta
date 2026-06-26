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

	// ErrHTTPBodyConflict indicates that an HTTP request received both body and
	// json options.
	ErrHTTPBodyConflict = errors.New("http request options cannot define both body and json")

	// ErrHTTPBodyUnsupported indicates that an HTTP request body has an
	// unsupported type.
	ErrHTTPBodyUnsupported = errors.New("http request body must be a string; use json for JSON bodies")

	// ErrHTTPHeadersUnsupported indicates that HTTP headers were not provided as
	// an object.
	ErrHTTPHeadersUnsupported = errors.New("http request headers must be an object")

	// ErrHTTPMethodInvalid indicates that an HTTP method is empty or malformed.
	ErrHTTPMethodInvalid = errors.New("http method is invalid")

	// ErrHTTPOptionsUnsupported indicates that HTTP options were not provided as
	// an object.
	ErrHTTPOptionsUnsupported = errors.New("http request options must be an object")

	// ErrHTTPTimeoutInvalid indicates that a timeout option is not positive.
	ErrHTTPTimeoutInvalid = errors.New("http timeout must be greater than zero")

	// ErrHTTPURLUnsupported indicates that a URL is not an absolute HTTP(S) URL.
	ErrHTTPURLUnsupported = errors.New("http client only supports absolute http and https URLs")
)
