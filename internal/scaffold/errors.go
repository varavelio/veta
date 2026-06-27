package scaffold

import "errors"

var (
	// ErrRootInvalid indicates that the requested project root is invalid.
	ErrRootInvalid = errors.New("scaffold root is invalid")

	// ErrFileExists indicates that initialization would overwrite existing files.
	ErrFileExists = errors.New("scaffold file already exists")
)

// ExistingFilesError reports starter files that already exist.
type ExistingFilesError struct {
	Paths []string
}

// Error returns a concise existing-files message.
func (err ExistingFilesError) Error() string {
	return ErrFileExists.Error()
}

// Unwrap returns the sentinel error for errors.Is checks.
func (err ExistingFilesError) Unwrap() error {
	return ErrFileExists
}
