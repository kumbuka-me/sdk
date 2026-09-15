// Package markdown provides reusable Markdown parsing helpers for Kumbuka plugins.
package markdown

import "strings"

// Fence returns the complete opening backtick or tilde run, including its length.
func Fence(line string) string {
	line = strings.TrimSpace(line)
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

// Closes accepts only a matching run at least as long as the opener, followed
// by whitespace. A shorter run or an info string remains code content.
func Closes(line, marker string) bool {
	line = strings.TrimSpace(line)
	return strings.HasPrefix(line, marker) && strings.Trim(line, string(marker[0])) == ""
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
