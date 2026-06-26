package config

import "errors"

var (
	// ErrFSRequired indicates that configuration was loaded without a filesystem.
	ErrFSRequired = errors.New("configuration filesystem is required")

	// ErrPathInvalid indicates that a configuration path is empty, absolute, or
	// tries to escape the filesystem root.
	ErrPathInvalid = errors.New("configuration path is invalid")

	// ErrInvalid indicates that configuration content is malformed or internally
	// inconsistent.
	ErrInvalid = errors.New("configuration is invalid")
)
