//go:build !((linux && (amd64 || arm64)) || (darwin && (amd64 || arm64)) || (windows && amd64))

package tailwindcss

var embeddedBinary []byte

const (
	embeddedBinaryName        = ""
	embeddedBinarySHA256      = ""
	embeddedBinaryUnsupported = true
)
