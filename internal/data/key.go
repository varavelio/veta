package data

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

// dataFileKey derives the nested global data key path and normalized extension
// from a data-relative file path.
func dataFileKey(filePath string) ([]string, string, error) {
	if filePath == "" || strings.ContainsRune(filePath, 0) || path.IsAbs(filePath) ||
		strings.Contains(filePath, "\\") || slices.Contains(strings.Split(filePath, "/"), "..") {
		return nil, "", fmt.Errorf("%w: %q", ErrKeyInvalid, filePath)
	}

	extension := strings.ToLower(path.Ext(filePath))
	if !isSupportedExtension(extension) {
		return nil, "", fmt.Errorf("%w: %s", ErrFormatUnsupported, filePath)
	}

	keyPath := strings.TrimSuffix(filePath, path.Ext(filePath))
	keys := strings.Split(path.Clean(keyPath), "/")
	for _, key := range keys {
		if err := validateKey(key); err != nil {
			return nil, "", err
		}
	}

	return keys, extension, nil
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
