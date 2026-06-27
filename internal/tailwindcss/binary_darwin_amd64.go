//go:build darwin && amd64

package tailwindcss

import _ "embed"

//go:embed bin/tailwindcss-macos-x64
var embeddedBinary []byte

const (
	embeddedBinaryName        = "tailwindcss-macos-x64"
	embeddedBinarySHA256      = "e9e830ceb3e70b7e0775a3dd79eee8ec82c6b31270f08f2fa2857d0077045ac3"
	embeddedBinaryUnsupported = false
)
