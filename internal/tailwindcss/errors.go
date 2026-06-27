package tailwindcss

import "errors"

var (
	// ErrBinaryUnavailable indicates that no Tailwind executable is available.
	ErrBinaryUnavailable = errors.New("tailwindcss binary is unavailable")

	// ErrCacheDirInvalid indicates that the binary cache directory is invalid.
	ErrCacheDirInvalid = errors.New("tailwindcss cache directory is invalid")

	// ErrConfigInvalid indicates that the Tailwind build configuration is invalid.
	ErrConfigInvalid = errors.New("tailwindcss config is invalid")

	// ErrExecutableInvalid indicates that the configured executable path is invalid.
	ErrExecutableInvalid = errors.New("tailwindcss executable is invalid")

	// ErrPlatformUnsupported indicates that Veta has no Tailwind binary for the current platform.
	ErrPlatformUnsupported = errors.New("tailwindcss platform is unsupported")

	// ErrRunFailed indicates that the Tailwind CLI failed.
	ErrRunFailed = errors.New("tailwindcss run failed")
)
