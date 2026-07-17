//go:build linux && arm64

package tailwindcss

import _ "embed"

//go:embed bin/tailwindcss-linux-arm64
var embeddedBinary []byte

const (
	embeddedBinaryName        = "tailwindcss-linux-arm64"
	embeddedBinarySHA256      = "55fd0b241214eff3de1e8ee4f22796662f2d2e7a49bcfca7477cfd0bac398195"
	embeddedBinaryUnsupported = false
)
