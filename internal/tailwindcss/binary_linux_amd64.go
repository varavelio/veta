//go:build linux && amd64

package tailwindcss

import _ "embed"

//go:embed bin/tailwindcss-linux-x64-musl
var embeddedBinary []byte

const (
	embeddedBinaryName        = "tailwindcss-linux-x64-musl"
	embeddedBinarySHA256      = "daeabe94235912b3773273053d5c8a16325af3fa513aa03b7295d6f445093cf2"
	embeddedBinaryUnsupported = false
)
