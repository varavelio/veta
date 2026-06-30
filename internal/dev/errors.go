package dev

import "errors"

var (
	// ErrAddressInvalid indicates that the dev server host or port is invalid.
	ErrAddressInvalid = errors.New("dev server address is invalid")

	// ErrListenFailed indicates that the dev server could not bind its address.
	ErrListenFailed = errors.New("dev server listen failed")
)
