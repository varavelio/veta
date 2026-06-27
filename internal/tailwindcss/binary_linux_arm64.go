//go:build linux && arm64

package tailwindcss

import _ "embed"

//go:embed bin/tailwindcss-linux-arm64
var embeddedBinary []byte

const (
	embeddedBinaryName        = "tailwindcss-linux-arm64"
	embeddedBinarySHA256      = "3d662377a86d71c43b549dc06b90db4586b4acd412bf827a3268e951661e5adf"
	embeddedBinaryUnsupported = false
)
