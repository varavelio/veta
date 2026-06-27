//go:build darwin && arm64

package tailwindcss

import _ "embed"

//go:embed bin/tailwindcss-macos-arm64
var embeddedBinary []byte

const (
	embeddedBinaryName        = "tailwindcss-macos-arm64"
	embeddedBinarySHA256      = "a27c43626185953ee19bdace1939c7601e55da654e0b2fc4461e3e29957aa739"
	embeddedBinaryUnsupported = false
)
