// Package markdown provides reusable Markdown parsing helpers for Kumbuka plugins.
package markdown

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
	if len(line) < 3 {
		return ""
	}
	delimiter := line[0]
	if delimiter != '`' && delimiter != '~' {
		return ""
	}

	length := 1
	for length < len(line) && line[length] == delimiter {
		length++
	}
	if length < 3 {
		return ""
	}
	return line[:length]
}

// validFenceInfo reports whether the opening fence accepts the trailing info string.
func validFenceInfo(marker, info string) bool {
	if marker[0] != '`' {
		return true
	}
	for index := 0; index < len(info); index++ {
		if info[index] == '`' {
			return false
		}
	}
	return true
}

// Closes reports whether line is a CommonMark-compatible closing fence for marker.
func Closes(line, marker string) bool {
	if marker == "" {
		return false
	}
	line, ok := fenceLineContent(line)
	if !ok {
		return false
	}

	end := len(line)
	for end > 0 && (line[end-1] == ' ' || line[end-1] == '\t') {
		end--
	}
	if end < len(marker) {
		return false
	}

	delimiter := marker[0]
	for index := 0; index < end; index++ {
		if line[index] != delimiter {
			return false
		}
	}
	return end >= len(marker)
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
