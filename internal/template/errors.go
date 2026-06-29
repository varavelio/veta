package template

import "errors"

var (
	// ErrTemplateNameInvalid indicates that a template name is empty, absolute,
	// or tries to escape the template root.
	ErrTemplateNameInvalid = errors.New("template name is invalid")

	// ErrTemplateNotFound indicates that no template file matched the requested
	// name.
	ErrTemplateNotFound = errors.New("template not found")

	// ErrTemplateAmbiguous indicates that an extensionless template name matched
	// more than one file.
	ErrTemplateAmbiguous = errors.New("template name is ambiguous")

	// ErrContextUnsupported indicates that a render context cannot be converted
	// into a Pongo2 context.
	ErrContextUnsupported = errors.New("template context is unsupported")

	// ErrFilterNameInvalid indicates that a filter name is empty or malformed.
	ErrFilterNameInvalid = errors.New("filter name is invalid")

	// ErrTemplateFSRequired indicates that a renderer was created without a
	// template filesystem.
	ErrTemplateFSRequired = errors.New("template filesystem is required")
)
