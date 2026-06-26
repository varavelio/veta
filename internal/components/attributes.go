package components

import (
	"fmt"
	"strings"
)

// parseAttributes parses HTML-like quoted string attributes.
func parseAttributes(content string) (map[string]string, error) {
	attributes := map[string]string{}
	position := 0
	for {
		position = skipSpace(content, position)
		if position >= len(content) {
			return attributes, nil
		}

		nameStart := position
		if !isAttributeNameStart(content[position]) {
			return nil, fmt.Errorf("attribute name expected near %q", content[position:])
		}
		position++
		for position < len(content) && isAttributeNamePart(content[position]) {
			position++
		}
		name := content[nameStart:position]

		position = skipSpace(content, position)
		if position >= len(content) || content[position] != '=' {
			return nil, fmt.Errorf("attribute %s must have a quoted value", name)
		}
		position++
		position = skipSpace(content, position)
		if position >= len(content) || content[position] != '"' && content[position] != '\'' {
			return nil, fmt.Errorf("attribute %s must have a quoted value", name)
		}

		quote := content[position]
		position++
		valueStart := position
		for position < len(content) && content[position] != quote {
			position++
		}
		if position >= len(content) {
			return nil, fmt.Errorf("attribute %s is missing closing quote", name)
		}

		attributes[name] = content[valueStart:position]
		position++
	}
}

// skipSpace advances past ASCII whitespace.
func skipSpace(content string, position int) int {
	for position < len(content) && strings.ContainsRune(" \t\r\n", rune(content[position])) {
		position++
	}

	return position
}

// isAttributeNameStart reports whether a byte can start an attribute name.
func isAttributeNameStart(char byte) bool {
	return 'A' <= char && char <= 'Z' || 'a' <= char && char <= 'z' || char == '_' || char == ':'
}

// isAttributeNamePart reports whether a byte can continue an attribute name.
func isAttributeNamePart(char byte) bool {
	return isAttributeNameStart(char) || '0' <= char && char <= '9' || char == '-' || char == '.'
}
