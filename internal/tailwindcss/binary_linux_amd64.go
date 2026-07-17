//go:build linux && amd64

package tailwindcss

import _ "embed"

//go:embed bin/tailwindcss-linux-x64
var embeddedBinary []byte

const (
	embeddedBinaryName        = "tailwindcss-linux-x64"
	embeddedBinarySHA256      = "dc61b3ac6b8c9ca874c0cc4c57b2409791a64c5540404ca5f5367360babc313a"
	embeddedBinaryUnsupported = false
)
