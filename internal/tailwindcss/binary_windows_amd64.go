//go:build windows && amd64

package tailwindcss

import _ "embed"

//go:embed bin/tailwindcss-windows-x64.exe
var embeddedBinary []byte

const (
	embeddedBinaryName        = "tailwindcss-windows-x64.exe"
	embeddedBinarySHA256      = "e0e260ce048014e9268f6237ff18f8ccf02cef521cbd0ae04e82c2cdf7aa3955"
	embeddedBinaryUnsupported = false
)
