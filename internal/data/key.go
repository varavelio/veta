package data

import (
	"fmt"
	"path"
	"strings"
)

// dataFileKey derives the global data key and normalized extension from a file
// name.
func dataFileKey(fileName string) (string, string, error) {
	if fileName == "" || strings.ContainsAny(fileName, "/\\") || strings.ContainsRune(fileName, 0) {
		return "", "", fmt.Errorf("%w: %q", ErrKeyInvalid, fileName)
	}

	extension := strings.ToLower(path.Ext(fileName))
	if !isSupportedExtension(extension) {
		return "", "", fmt.Errorf("%w: %s", ErrFormatUnsupported, fileName)
	}

	key := strings.TrimSuffix(fileName, path.Ext(fileName))
	if err := validateKey(key); err != nil {
		return "", "", err
	}

	return key, extension, nil
}

// isSupportedExtension reports whether an extension maps to a data parser.
func isSupportedExtension(extension string) bool {
	switch extension {
	case ".js", ".json", ".toml", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

// validateKey checks that a file stem can be used as an ergonomic template key.
func validateKey(key string) error {
	if key == "" {
		return fmt.Errorf("%w: key cannot be empty", ErrKeyInvalid)
	}

	for index, char := range key {
		if index == 0 {
			if !isIdentifierStart(char) {
				return fmt.Errorf("%w: %s", ErrKeyInvalid, key)
			}

			continue
		}

		if !isIdentifierPart(char) {
			return fmt.Errorf("%w: %s", ErrKeyInvalid, key)
		}
	}

	return nil
}

// isIdentifierStart reports whether a rune can start a data key.
func isIdentifierStart(char rune) bool {
	return char == '_' || 'A' <= char && char <= 'Z' || 'a' <= char && char <= 'z'
}

// isIdentifierPart reports whether a rune can continue a data key.
func isIdentifierPart(char rune) bool {
	return isIdentifierStart(char) || '0' <= char && char <= '9'
}
