package components

import (
	"fmt"
	"strings"
)

type tagToken struct {
	attributes  map[string]string
	closing     bool
	end         int
	name        string
	selfClosing bool
	start       int
}

// renderSegment expands registered component tags in one unbounded content
// segment.
func (processor *Processor) renderSegment(content string, context any) (string, error) {
	ranges := protectedRanges(content)
	var output strings.Builder
	position := 0
	for {
		token, ok, err := processor.nextToken(content, position, ranges)
		if err != nil {
			return "", err
		}
		if !ok {
			output.WriteString(content[position:])
			break
		}
		if token.closing {
			return "", fmt.Errorf("%w: unexpected closing tag </%s>", ErrSyntax, token.name)
		}

		output.WriteString(content[position:token.start])
		if token.selfClosing {
			rendered, err := processor.renderComponent(token, "", context)
			if err != nil {
				return "", err
			}

			output.WriteString(rendered)
			position = token.end
			continue
		}

		closingToken, err := processor.findClosingToken(content, token, ranges)
		if err != nil {
			return "", err
		}

		innerContent, err := processor.renderSegment(content[token.end:closingToken.start], context)
		if err != nil {
			return "", err
		}

		rendered, err := processor.renderComponent(token, innerContent, context)
		if err != nil {
			return "", err
		}

		output.WriteString(rendered)
		position = closingToken.end
	}

	return output.String(), nil
}

// nextToken returns the next registered component tag at or after start.
func (processor *Processor) nextToken(
	content string,
	start int,
	ranges []textRange,
) (tagToken, bool, error) {
	position := start
	for position < len(content) {
		index := nextUnprotectedByte(content, '<', position, ranges)
		if index < 0 {
			return tagToken{}, false, nil
		}

		token, ok, err := processor.parseTag(content, index)
		if err != nil {
			return tagToken{}, false, err
		}
		if ok {
			return token, true, nil
		}

		position = index + 1
	}

	return tagToken{}, false, nil
}

// findClosingToken returns the matching closing tag for an opening component tag.
func (processor *Processor) findClosingToken(
	content string,
	openingToken tagToken,
	ranges []textRange,
) (tagToken, error) {
	depth := 1
	position := openingToken.end
	for {
		token, ok, err := processor.nextToken(content, position, ranges)
		if err != nil {
			return tagToken{}, err
		}
		if !ok {
			return tagToken{}, fmt.Errorf(
				"%w: missing closing tag </%s>",
				ErrSyntax,
				openingToken.name,
			)
		}

		position = token.end
		if token.name != openingToken.name {
			continue
		}
		if token.closing {
			depth--
			if depth == 0 {
				return token, nil
			}

			continue
		}
		if !token.selfClosing {
			depth++
		}
	}
}

// parseTag parses a component tag at start when the tag name is registered.
func (processor *Processor) parseTag(content string, start int) (tagToken, bool, error) {
	name, closing, nameEnd, ok := readTagName(content, start)
	if !ok {
		return tagToken{}, false, nil
	}
	if _, registered := processor.components[name]; !registered {
		return tagToken{}, false, nil
	}

	closeIndex := tagCloseIndex(content, nameEnd)
	if closeIndex < 0 {
		return tagToken{}, false, fmt.Errorf("%w: malformed <%s> tag", ErrSyntax, name)
	}

	body := strings.TrimSpace(content[nameEnd:closeIndex])
	if closing {
		if body != "" {
			return tagToken{}, false, fmt.Errorf(
				"%w: closing tag </%s> cannot have attributes",
				ErrSyntax,
				name,
			)
		}

		return tagToken{closing: true, end: closeIndex + 1, name: name, start: start}, true, nil
	}

	selfClosing := strings.HasSuffix(body, "/")
	if selfClosing {
		body = strings.TrimSpace(strings.TrimSuffix(body, "/"))
	}

	attributes, err := parseAttributes(body)
	if err != nil {
		return tagToken{}, false, fmt.Errorf("%w: <%s>: %w", ErrAttributeInvalid, name, err)
	}

	return tagToken{
		attributes:  attributes,
		end:         closeIndex + 1,
		name:        name,
		selfClosing: selfClosing,
		start:       start,
	}, true, nil
}

// readTagName reads an HTML-like tag name after a less-than sign.
func readTagName(content string, start int) (string, bool, int, bool) {
	if start >= len(content) || content[start] != '<' {
		return "", false, start, false
	}

	position := start + 1
	closing := false
	if position < len(content) && content[position] == '/' {
		closing = true
		position++
	}
	if position >= len(content) || !isTagNameStart(content[position]) {
		return "", false, start, false
	}

	nameStart := position
	position++
	for position < len(content) && isTagNamePart(content[position]) {
		position++
	}

	return content[nameStart:position], closing, position, true
}

// tagCloseIndex returns the closing angle bracket for a tag body.
func tagCloseIndex(content string, start int) int {
	quote := byte(0)
	for index := start; index < len(content); index++ {
		char := content[index]
		if quote != 0 {
			if char == quote {
				quote = 0
			}
			continue
		}

		switch char {
		case '\'', '"':
			quote = char
		case '>':
			return index
		}
	}

	return -1
}

// isTagNameStart reports whether a byte can start a component tag name.
func isTagNameStart(char byte) bool {
	return 'a' <= char && char <= 'z'
}

// isTagNamePart reports whether a byte can continue a component tag name.
func isTagNamePart(char byte) bool {
	return isTagNameStart(char) || '0' <= char && char <= '9' || char == '-'
}
