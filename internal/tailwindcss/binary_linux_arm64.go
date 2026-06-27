//go:build linux && arm64

package tailwindcss

import _ "embed"

//go:embed bin/tailwindcss-linux-arm64-musl
var embeddedBinary []byte

const (
	embeddedBinaryName        = "tailwindcss-linux-arm64-musl"
	embeddedBinarySHA256      = "7ed72712429166d869dc8472e0cd8c61cd46e565a5bc1ba8810612bedfe61e7b"
	embeddedBinaryUnsupported = false
)
