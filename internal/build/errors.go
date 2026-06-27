package build

import "errors"

var (
	// ErrRootInvalid indicates that the project root option is invalid.
	ErrRootInvalid = errors.New("build root is invalid")

	// ErrOutputDirInvalid indicates that the output directory option is invalid.
	ErrOutputDirInvalid = errors.New("build output directory is invalid")

	// ErrConfigFileInvalid indicates that the explicit config file option is invalid.
	ErrConfigFileInvalid = errors.New("build config file is invalid")

	// ErrTailwindUnsupported indicates that Tailwind CSS execution is not implemented.
	ErrTailwindUnsupported = errors.New("tailwindcss build is not supported yet")
)
