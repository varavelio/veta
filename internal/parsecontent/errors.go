package parsecontent

import "errors"

var (
	// ErrInvalid indicates that content could not be parsed.
	ErrInvalid = errors.New("content parse failed")

	// ErrValueUnsupported indicates that parsed content contains a value Veta cannot expose.
	ErrValueUnsupported = errors.New("parsed value is unsupported")
)
