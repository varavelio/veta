package cli

import "errors"

// ErrUsage indicates that command-line arguments are invalid.
var ErrUsage = errors.New("usage error")
