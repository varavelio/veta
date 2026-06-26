package theme

import "errors"

var (
	// ErrProjectFSRequired indicates that a project filesystem was not provided.
	ErrProjectFSRequired = errors.New("project filesystem is required")

	// ErrRootInvalid indicates that the project root path is invalid.
	ErrRootInvalid = errors.New("theme root is invalid")

	// ErrSourceInvalid indicates that a configured theme source is invalid.
	ErrSourceInvalid = errors.New("theme source is invalid")

	// ErrRemoteUnsupported indicates that remote theme sources are not implemented.
	ErrRemoteUnsupported = errors.New("remote theme sources are not supported")
)
