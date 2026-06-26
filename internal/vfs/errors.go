package vfs

import "errors"

var (
	// ErrLayerRequired indicates that an overlay was created without layers.
	ErrLayerRequired = errors.New("at least one filesystem layer is required")

	// ErrLayerInvalid indicates that a filesystem layer is incomplete.
	ErrLayerInvalid = errors.New("filesystem layer is invalid")

	// ErrFSRequired indicates that a filesystem wrapper was created without a
	// filesystem.
	ErrFSRequired = errors.New("filesystem is required")

	// ErrPathInvalid indicates that a filesystem path is empty, absolute, or tries
	// to escape the filesystem root.
	ErrPathInvalid = errors.New("filesystem path is invalid")

	// ErrTopDirInvalid indicates that a root allowlist entry is not a single
	// top-level directory or file name.
	ErrTopDirInvalid = errors.New("top-level filesystem name is invalid")
)
