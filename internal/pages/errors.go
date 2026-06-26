package pages

import "errors"

var (
	// ErrFSRequired indicates that pages were loaded without a filesystem.
	ErrFSRequired = errors.New("pages filesystem is required")

	// ErrNestedUnsupported indicates that the pages directory contains a nested
	// directory.
	ErrNestedUnsupported = errors.New("pages directory must be flat")

	// ErrFormatUnsupported indicates that a page generator file has an unsupported
	// extension.
	ErrFormatUnsupported = errors.New("page generator format is unsupported")

	// ErrGeneratorInvalid indicates that a page generator could not be executed or
	// did not return the expected top-level shape.
	ErrGeneratorInvalid = errors.New("page generator is invalid")

	// ErrPageInvalid indicates that one generated page does not match Veta's page
	// contract.
	ErrPageInvalid = errors.New("page is invalid")

	// ErrPermalinkInvalid indicates that a generated page has an invalid permalink.
	ErrPermalinkInvalid = errors.New("page permalink is invalid")

	// ErrOutputPathDuplicate indicates that multiple pages resolve to the same
	// output path.
	ErrOutputPathDuplicate = errors.New("page output path is duplicated")

	// ErrValueUnsupported indicates that page data cannot be represented as
	// JSON-compatible data.
	ErrValueUnsupported = errors.New("page value is not JSON-compatible")
)
