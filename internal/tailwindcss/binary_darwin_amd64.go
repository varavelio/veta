//go:build darwin && amd64

package tailwindcss

import _ "embed"

//go:embed bin/tailwindcss-macos-x64
var embeddedBinary []byte

const (
	embeddedBinaryName        = "tailwindcss-macos-x64"
	embeddedBinarySHA256      = "7922e0953f2110c05976e3bf58f14e643d90427575e766b7d433f5f80cbee7e1"
	embeddedBinaryUnsupported = false
)
