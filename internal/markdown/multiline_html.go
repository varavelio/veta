package markdown

import (
	"bytes"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type multilineHTMLBlockParser struct{}

// Trigger returns the byte that can begin a multiline HTML tag.
func (blockParser *multilineHTMLBlockParser) Trigger() []byte {
	return []byte{'<'}
}

// Open starts a raw HTML block when an opening tag continues beyond its first
// line.
func (blockParser *multilineHTMLBlockParser) Open(
	_ ast.Node,
	reader text.Reader,
	_ parser.Context,
) (ast.Node, parser.State) {
	line, segment := reader.PeekLine()
	if !startsMultilineHTMLTag(line) || !hasMultilineHTMLTagClose(reader, line) {
		return nil, parser.NoChildren
	}

	node := ast.NewHTMLBlock(ast.HTMLBlockType7)
	node.Lines().Append(segment)
	reader.AdvanceToEOL()
	return node, parser.NoChildren
}

// Continue preserves tag continuation lines until an unquoted closing angle
// bracket completes the opening tag.
func (blockParser *multilineHTMLBlockParser) Continue(
	node ast.Node,
	reader text.Reader,
	_ parser.Context,
) parser.State {
	htmlBlock := node.(*ast.HTMLBlock)
	line, segment := reader.PeekLine()
	quote := multilineHTMLTagQuote(htmlBlock.Lines(), reader.Source())
	closed, _ := scanHTMLTagClose(line, quote)
	htmlBlock.Lines().Append(segment)
	reader.AdvanceToEOL()
	if closed {
		return parser.Close
	}

	return parser.Continue | parser.NoChildren
}

// Close completes a multiline raw HTML block.
func (blockParser *multilineHTMLBlockParser) Close(
	_ ast.Node,
	_ text.Reader,
	_ parser.Context,
) {
}

// CanInterruptParagraph reports that custom HTML-like tags follow CommonMark
// type-7 behavior and do not interrupt paragraphs.
func (blockParser *multilineHTMLBlockParser) CanInterruptParagraph() bool {
	return false
}

// CanAcceptIndentedLine reports that code-indented lines cannot begin a
// multiline HTML tag.
func (blockParser *multilineHTMLBlockParser) CanAcceptIndentedLine() bool {
	return false
}

// startsMultilineHTMLTag reports whether line begins an HTML-like opening tag
// whose closing angle bracket is on a later line.
func startsMultilineHTMLTag(line []byte) bool {
	line = bytes.TrimSuffix(bytes.TrimSuffix(line, []byte("\n")), []byte("\r"))
	position := 0
	for position < len(line) && position < 4 && line[position] == ' ' {
		position++
	}
	if position > 3 || position >= len(line) || line[position] != '<' {
		return false
	}

	position++
	nameStart := position
	if position >= len(line) || line[position] < 'a' || line[position] > 'z' {
		return false
	}
	position++
	for position < len(line) && isHTMLTagNamePart(line[position]) {
		position++
	}
	if isRawTextHTMLTag(line[nameStart:position]) {
		return false
	}
	if position < len(line) && !isHTMLWhitespace(line[position]) {
		return false
	}

	closed, _ := scanHTMLTagClose(line[position:], 0)
	return !closed
}

// hasMultilineHTMLTagClose looks ahead without changing the reader position and
// reports whether the candidate opening tag has an unquoted closing bracket.
func hasMultilineHTMLTagClose(reader text.Reader, firstLine []byte) bool {
	lineNumber, segment := reader.Position()
	defer reader.SetPosition(lineNumber, segment)

	var candidate bytes.Buffer
	candidate.Write(firstLine)
	closed, quote := scanHTMLTagClose(firstLine, 0)
	for !closed {
		reader.AdvanceLine()
		line, _ := reader.PeekLine()
		if line == nil {
			return false
		}
		candidate.Write(line)
		closed, quote = scanHTMLTagClose(line, quote)
	}

	return validMultilineHTMLTag(candidate.Bytes())
}

// validMultilineHTMLTag reports whether content begins a complete Veta
// component-style tag with quoted attribute values.
func validMultilineHTMLTag(content []byte) bool {
	position := 0
	for position < len(content) && position < 4 && content[position] == ' ' {
		position++
	}
	if position >= len(content) || content[position] != '<' {
		return false
	}
	position++
	if position >= len(content) || content[position] < 'a' || content[position] > 'z' {
		return false
	}
	position++
	for position < len(content) && isHTMLTagNamePart(content[position]) {
		position++
	}

	for {
		position = skipHTMLWhitespace(content, position)
		if position >= len(content) {
			return false
		}
		if content[position] == '>' {
			return true
		}
		if content[position] == '/' {
			return position+1 < len(content) && content[position+1] == '>'
		}
		if !isHTMLAttributeNameStart(content[position]) {
			return false
		}
		position++
		for position < len(content) && isHTMLAttributeNamePart(content[position]) {
			position++
		}
		position = skipHTMLWhitespace(content, position)
		if position >= len(content) || content[position] != '=' {
			return false
		}
		position = skipHTMLWhitespace(content, position+1)
		if position >= len(content) || content[position] != '\'' && content[position] != '"' {
			return false
		}
		quote := content[position]
		position++
		for position < len(content) && content[position] != quote {
			position++
		}
		if position >= len(content) {
			return false
		}
		position++
	}
}

// skipHTMLWhitespace advances past whitespace accepted in component tags.
func skipHTMLWhitespace(content []byte, position int) int {
	for position < len(content) && isHTMLWhitespace(content[position]) {
		position++
	}

	return position
}

// isHTMLAttributeNameStart reports whether char can begin a component
// attribute name.
func isHTMLAttributeNameStart(char byte) bool {
	return 'A' <= char && char <= 'Z' || 'a' <= char && char <= 'z' || char == '_' || char == ':'
}

// isHTMLAttributeNamePart reports whether char can continue a component
// attribute name.
func isHTMLAttributeNamePart(char byte) bool {
	return isHTMLAttributeNameStart(char) || '0' <= char && char <= '9' || char == '-' ||
		char == '.'
}

// multilineHTMLTagQuote returns the active attribute quote after scanning the
// lines already stored in a multiline opening tag.
func multilineHTMLTagQuote(lines *text.Segments, source []byte) byte {
	quote := byte(0)
	for index := range lines.Len() {
		segment := lines.At(index)
		_, quote = scanHTMLTagClose(segment.Value(source), quote)
	}

	return quote
}

// scanHTMLTagClose searches content for a closing angle bracket outside quoted
// attribute values and returns the active quote state.
func scanHTMLTagClose(content []byte, quote byte) (bool, byte) {
	for _, char := range content {
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
			return true, 0
		}
	}

	return false, quote
}

// isHTMLTagNamePart reports whether char can continue a Veta HTML-like tag
// name.
func isHTMLTagNamePart(char byte) bool {
	return 'a' <= char && char <= 'z' || '0' <= char && char <= '9' || char == '-'
}

// isHTMLWhitespace reports whether char is accepted between an HTML-like tag
// name and its attributes.
func isHTMLWhitespace(char byte) bool {
	return char == ' ' || char == '\t' || char == '\r' || char == '\n'
}

// isRawTextHTMLTag reports whether name belongs to an HTML element whose body
// must remain opaque to Markdown parsing.
func isRawTextHTMLTag(name []byte) bool {
	switch string(name) {
	case "code", "pre", "script", "style", "textarea", "title":
		return true
	default:
		return false
	}
}
