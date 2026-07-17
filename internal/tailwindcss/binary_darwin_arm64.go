//go:build darwin && arm64

package tailwindcss

import _ "embed"

//go:embed bin/tailwindcss-macos-arm64
var embeddedBinary []byte

const (
	embeddedBinaryName        = "tailwindcss-macos-arm64"
	embeddedBinarySHA256      = "cdf646702987a743464dff4d9c60fd4480d1c1e73dd819a9a67f1078815dce9d"
	embeddedBinaryUnsupported = false
)
