package build

import "errors"

var (
	// ErrRootInvalid indicates that the config search directory is invalid.
	ErrRootInvalid = errors.New("build root is invalid")

	// ErrConfigFileInvalid indicates that the explicit config file option is invalid.
	ErrConfigFileInvalid = errors.New("build config file is invalid")

	// ErrConfigNotFound indicates that no Veta config file could be discovered.
	ErrConfigNotFound = errors.New("veta config file was not found")
)
