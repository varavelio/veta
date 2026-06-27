//go:build windows && amd64

package tailwindcss

import _ "embed"

//go:embed bin/tailwindcss-windows-x64.exe
var embeddedBinary []byte

const (
	embeddedBinaryName        = "tailwindcss-windows-x64.exe"
	embeddedBinarySHA256      = "dc4fd46acd354d976df2a31b6425fbe865a38229a06bc005a4c59f2b3d24ab4a"
	embeddedBinaryUnsupported = false
)
