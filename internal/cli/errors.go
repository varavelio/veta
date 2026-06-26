package cli

import "errors"

// ErrUnknownCommand indicates that the requested command does not exist.
var ErrUnknownCommand = errors.New("unknown command")
