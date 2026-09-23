// Package markdown provides reusable Markdown parsing helpers for Kumbuka plugins.
package markdown

import "strings"

// Fence returns the complete opening backtick or tilde run, including its length.
func Fence(line string) string {
	line, ok := fenceLineContent(line)
	if !ok {
		return ""
	}
	marker := openingMarker(line)
	if marker == "" || !validFenceInfo(marker, line[len(marker):]) {
		return ""
	}
	return marker
}

// openingMarker returns a valid opening fence marker or an empty string.
func openingMarker(line string) string {
	for _, delimiter := range []string{"`", "~"} {
		length := len(line) - len(strings.TrimLeft(line, delimiter))
		if length >= 3 {
			return line[:length]
		}
	}
	return ""
}

// validFenceInfo reports whether the opening fence accepts the trailing info string.
func validFenceInfo(marker, info string) bool {
	return marker[0] != '`' || !strings.ContainsRune(info, '`')
}

// Closes reports whether line is a CommonMark-compatible closing fence for marker.
func Closes(line, marker string) bool {
	line, ok := fenceLineContent(line)
	if !ok {
		return false
	}
	line = strings.TrimRight(line, " \t")
	return strings.HasPrefix(line, marker) && strings.Trim(line, string(marker[0])) == ""
}

// fenceLineContent removes at most three leading spaces and rejects deeper or tab indentation.
func fenceLineContent(line string) (string, bool) {
	spaces := 0
	for spaces < len(line) && line[spaces] == ' ' {
		spaces++
	}
	if spaces > 3 || (spaces < len(line) && line[spaces] == '\t') {
		return "", false
	}
	return line[spaces:], true
}

// AppendFence copies a complete fenced block without interpreting its body.
func AppendFence(lines []string, start int, marker string, out *[]string) int {
	*out = append(*out, lines[start])
	for index := start + 1; index < len(lines); index++ {
		*out = append(*out, lines[index])
		if Closes(lines[index], marker) {
			return index + 1
		}
	}
	return len(lines)
}
