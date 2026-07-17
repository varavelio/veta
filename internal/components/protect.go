package components

import (
	"sort"
	"strings"
)

type textRange struct {
	end   int
	start int
}

// protectedRanges returns ranges where registered component tags must not be
// parsed.
func protectedRanges(content string, components map[string]Component) []textRange {
	fencedRanges := normalizeRanges(fencedCodeRanges(content))
	markdownRanges := append([]textRange(nil), fencedRanges...)
	markdownRanges = append(markdownRanges, inlineCodeRanges(content, fencedRanges)...)
	markdownRanges = normalizeRanges(markdownRanges)

	ranges := append([]textRange(nil), fencedRanges...)
	ranges = append(ranges, htmlProtectedRanges(content, markdownRanges, components)...)
	ranges = normalizeRanges(ranges)
	inlineExclusions := append([]textRange(nil), ranges...)
	inlineExclusions = append(
		inlineExclusions,
		registeredTagRanges(content, markdownRanges, components)...,
	)
	inlineExclusions = normalizeRanges(inlineExclusions)
	ranges = append(ranges, inlineCodeRanges(content, inlineExclusions)...)
	return normalizeRanges(ranges)
}

// normalizeRanges sorts and merges overlapping protected ranges.
func normalizeRanges(ranges []textRange) []textRange {
	sort.Slice(ranges, func(left, right int) bool {
		return ranges[left].start < ranges[right].start
	})
	merged := make([]textRange, 0, len(ranges))
	for _, current := range ranges {
		if len(merged) == 0 || current.start > merged[len(merged)-1].end {
			merged = append(merged, current)
			continue
		}
		if current.end > merged[len(merged)-1].end {
			merged[len(merged)-1].end = current.end
		}
	}

	return merged
}

// fencedCodeRanges returns Markdown fenced code block ranges.
func fencedCodeRanges(content string) []textRange {
	ranges := []textRange{}
	inFence := false
	fenceLength := 0
	fenceMarker := byte(0)
	fenceStart := 0
	lineStart := 0
	for lineStart < len(content) {
		lineEnd := strings.IndexByte(content[lineStart:], '\n')
		if lineEnd < 0 {
			lineEnd = len(content)
		} else {
			lineEnd += lineStart + 1
		}

		line := strings.TrimSuffix(strings.TrimSuffix(content[lineStart:lineEnd], "\n"), "\r")
		if !inFence {
			marker, length, rest, ok := fenceStartDelimiter(line)
			if ok && (marker != '`' || !strings.ContainsRune(rest, '`')) {
				inFence = true
				fenceMarker = marker
				fenceLength = length
				fenceStart = lineStart
			}
		} else if fenceEndDelimiter(line, fenceMarker, fenceLength) {
			ranges = append(ranges, textRange{end: lineEnd, start: fenceStart})
			inFence = false
		}

		lineStart = lineEnd
	}
	if inFence {
		ranges = append(ranges, textRange{end: len(content), start: fenceStart})
	}

	return ranges
}

// fenceStartDelimiter parses a Markdown fenced-code opening delimiter.
func fenceStartDelimiter(line string) (byte, int, string, bool) {
	position := 0
	for position < len(line) && position < 4 && line[position] == ' ' {
		position++
	}
	if position > 3 || position >= len(line) || line[position] != '`' && line[position] != '~' {
		return 0, 0, "", false
	}

	marker := line[position]
	start := position
	for position < len(line) && line[position] == marker {
		position++
	}
	if position-start < 3 {
		return 0, 0, "", false
	}

	return marker, position - start, line[position:], true
}

// fenceEndDelimiter reports whether line closes a Markdown fenced-code block.
func fenceEndDelimiter(line string, marker byte, minimumLength int) bool {
	position := 0
	for position < len(line) && position < 4 && line[position] == ' ' {
		position++
	}
	if position > 3 {
		return false
	}

	start := position
	for position < len(line) && line[position] == marker {
		position++
	}
	return position-start >= minimumLength && strings.TrimSpace(line[position:]) == ""
}

// inlineCodeRanges returns Markdown inline code ranges outside existing ranges.
func inlineCodeRanges(content string, existing []textRange) []textRange {
	ranges := []textRange{}
	for index := 0; index < len(content); {
		if protected(index, existing) || content[index] != '`' {
			index++
			continue
		}

		delimiterLength := byteRunLength(content, index, '`')
		matched := false
		for end := index + delimiterLength; end < len(content); {
			if protected(end, existing) {
				break
			}
			if content[end] != '`' {
				end++
				continue
			}

			closingLength := byteRunLength(content, end, '`')
			if closingLength == delimiterLength {
				ranges = append(ranges, textRange{end: end + closingLength, start: index})
				index = end + closingLength
				matched = true
				break
			}
			end += closingLength
		}
		if !matched {
			index += delimiterLength
		}
	}

	return ranges
}

// byteRunLength returns the number of consecutive occurrences at start.
func byteRunLength(content string, start int, char byte) int {
	end := start
	for end < len(content) && content[end] == char {
		end++
	}
	return end - start
}

// htmlProtectedRanges returns HTML comments and raw-text element ranges outside
// existing Markdown code ranges where component-like text must remain unchanged.
func htmlProtectedRanges(
	content string,
	existing []textRange,
	components map[string]Component,
) []textRange {
	rawTags := map[string]struct{}{
		"code": {}, "pre": {}, "script": {}, "style": {}, "textarea": {}, "title": {},
	}
	lowerContent := strings.ToLower(content)
	ranges := []textRange{}
	for position := 0; position < len(content); {
		index := strings.IndexByte(content[position:], '<')
		if index < 0 {
			break
		}
		index += position
		if end, ok := containingRangeEnd(index, existing); ok {
			position = end
			continue
		}
		if strings.HasPrefix(content[index:], "<!--") {
			end := strings.Index(content[index+4:], "-->")
			if end < 0 {
				return append(ranges, textRange{start: index, end: len(content)})
			}
			end += index + 7
			ranges = append(ranges, textRange{start: index, end: end})
			position = end
			continue
		}

		name, closing, nameEnd, ok := readHTMLTagName(lowerContent, index)
		if !ok {
			position = index + 1
			continue
		}
		openEnd := tagCloseIndex(content, nameEnd)
		if openEnd < 0 {
			break
		}
		_, registered := components[name]
		if closing {
			if !registered {
				ranges = append(ranges, textRange{start: index, end: openEnd + 1})
			}
			position = openEnd + 1
			continue
		}
		if _, raw := rawTags[name]; !raw {
			if !registered {
				ranges = append(ranges, textRange{start: index, end: openEnd + 1})
			}
			position = openEnd + 1
			continue
		}

		if strings.HasSuffix(strings.TrimSpace(content[nameEnd:openEnd]), "/") {
			ranges = append(ranges, textRange{start: index, end: openEnd + 1})
			position = openEnd + 1
			continue
		}
		end := rawElementEnd(content, lowerContent, name, openEnd+1)
		ranges = append(ranges, textRange{start: index, end: end})
		position = end
	}

	return ranges
}

// containingRangeEnd returns the end of the range containing index.
func containingRangeEnd(index int, ranges []textRange) (int, bool) {
	position := sort.Search(len(ranges), func(position int) bool {
		return ranges[position].end > index
	})
	if position >= len(ranges) || ranges[position].start > index {
		return 0, false
	}

	return ranges[position].end, true
}

// registeredTagRanges returns component tag markup ranges so Markdown backticks
// in attributes cannot create code spans across component boundaries.
func registeredTagRanges(
	content string,
	existing []textRange,
	components map[string]Component,
) []textRange {
	ranges := []textRange{}
	lowerContent := strings.ToLower(content)
	for position := 0; position < len(content); {
		index := strings.IndexByte(content[position:], '<')
		if index < 0 {
			break
		}
		index += position
		if end, ok := containingRangeEnd(index, existing); ok {
			position = end
			continue
		}

		name, _, nameEnd, ok := readHTMLTagName(lowerContent, index)
		if !ok {
			position = index + 1
			continue
		}
		closeIndex := tagCloseIndex(content, nameEnd)
		if closeIndex < 0 {
			break
		}
		if _, registered := components[name]; registered {
			ranges = append(ranges, textRange{start: index, end: closeIndex + 1})
		}
		position = closeIndex + 1
	}

	return ranges
}

// readHTMLTagName reads a case-normalized HTML tag name.
func readHTMLTagName(content string, start int) (string, bool, int, bool) {
	if start >= len(content) || content[start] != '<' {
		return "", false, start, false
	}

	position := start + 1
	closing := false
	if position < len(content) && content[position] == '/' {
		closing = true
		position++
	}
	if position >= len(content) || content[position] < 'a' || content[position] > 'z' {
		return "", false, start, false
	}

	nameStart := position
	for position < len(content) {
		char := content[position]
		if !isTagNamePart(char) {
			break
		}
		position++
	}
	if !isTagNameBoundary(content, position) {
		return "", false, start, false
	}

	return content[nameStart:position], closing, position, true
}

// rawElementEnd returns the end of a raw-text element or the input when its
// closing tag is missing.
func rawElementEnd(content, lowerContent, name string, start int) int {
	prefix := "</" + name
	position := start
	for position < len(content) {
		index := strings.Index(lowerContent[position:], prefix)
		if index < 0 {
			return len(content)
		}
		index += position
		nameEnd := index + len(prefix)
		if !isTagNameBoundary(content, nameEnd) {
			position = nameEnd
			continue
		}
		closeIndex := tagCloseIndex(content, nameEnd)
		if closeIndex < 0 {
			return len(content)
		}

		return closeIndex + 1
	}

	return len(content)
}

// nextUnprotectedByte returns the next byte outside protected ranges.
func nextUnprotectedByte(content string, char byte, start int, ranges []textRange) int {
	for index := start; index < len(content); index++ {
		if content[index] == char && !protected(index, ranges) {
			return index
		}
	}

	return -1
}

// protected reports whether index belongs to any protected range.
func protected(index int, ranges []textRange) bool {
	position := sort.Search(len(ranges), func(position int) bool {
		return ranges[position].start > index
	})
	if position == 0 {
		return false
	}

	previous := ranges[position-1]
	return index < previous.end
}
