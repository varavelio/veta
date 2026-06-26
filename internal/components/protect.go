package components

import "strings"

type textRange struct {
	end   int
	start int
}

// protectedRanges returns ranges where component tags must not be parsed.
func protectedRanges(content string) []textRange {
	ranges := fencedCodeRanges(content)
	ranges = append(ranges, inlineCodeRanges(content, ranges)...)
	return ranges
}

// fencedCodeRanges returns Markdown fenced code block ranges.
func fencedCodeRanges(content string) []textRange {
	ranges := []textRange{}
	inFence := false
	fenceMarker := ""
	fenceStart := 0
	lineStart := 0
	for lineStart < len(content) {
		lineEnd := strings.IndexByte(content[lineStart:], '\n')
		if lineEnd < 0 {
			lineEnd = len(content)
		} else {
			lineEnd += lineStart + 1
		}

		line := content[lineStart:lineEnd]
		trimmed := strings.TrimSpace(line)
		if !inFence {
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				inFence = true
				fenceMarker = trimmed[:3]
				fenceStart = lineStart
			}
		} else if strings.HasPrefix(trimmed, fenceMarker) {
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

// inlineCodeRanges returns Markdown inline code ranges outside existing ranges.
func inlineCodeRanges(content string, existing []textRange) []textRange {
	ranges := []textRange{}
	for index := 0; index < len(content); index++ {
		if protected(index, existing) || content[index] != '`' {
			continue
		}

		for end := index + 1; end < len(content); end++ {
			if protected(end, existing) {
				break
			}
			if content[end] == '`' {
				ranges = append(ranges, textRange{end: end + 1, start: index})
				index = end
				break
			}
		}
	}

	return ranges
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
	for _, textRange := range ranges {
		if textRange.start <= index && index < textRange.end {
			return true
		}
	}

	return false
}
