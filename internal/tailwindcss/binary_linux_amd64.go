//go:build linux && amd64

package tailwindcss

import _ "embed"

//go:embed bin/tailwindcss-linux-x64
var embeddedBinary []byte

const (
	embeddedBinaryName        = "tailwindcss-linux-x64"
	embeddedBinarySHA256      = "2526d063ba03b71f9a3ea7d5cee14f0aec147f117f222d5adc97b1d736d45999"
	embeddedBinaryUnsupported = false
)
