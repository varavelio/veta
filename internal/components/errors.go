package components

import "errors"

var (
	// ErrFSRequired indicates that components were loaded without a filesystem.
	ErrFSRequired = errors.New("components filesystem is required")

	// ErrRendererRequired indicates that registered components cannot be rendered
	// because no template renderer was provided.
	ErrRendererRequired = errors.New("component template renderer is required")

	// ErrComponentNameInvalid indicates that a component file cannot be mapped to a
	// valid tag name.
	ErrComponentNameInvalid = errors.New("component name is invalid")

	// ErrFormatUnsupported is retained for compatibility with earlier component
	// extension validation.
	ErrFormatUnsupported = errors.New("component file format is unsupported")

	// ErrAttributeInvalid indicates that a component tag contains malformed
	// attributes.
	ErrAttributeInvalid = errors.New("component attribute is invalid")

	// ErrSyntax indicates that component tags are unbalanced or malformed.
	ErrSyntax = errors.New("component syntax is invalid")
)
