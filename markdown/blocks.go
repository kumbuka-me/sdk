package markdown

import (
	"strconv"
	"strings"
)

// ParseQuotedTitle parses one non-empty Go-style double-quoted title.
func ParseQuotedTitle(value string) (string, bool) {
	if !hasDoubleQuoteDelimiters(value) {
		return "", false
	}

	title, err := strconv.Unquote(value)
	if err != nil || strings.TrimSpace(title) == "" {
		return "", false
	}

	return title, true
}

// hasDoubleQuoteDelimiters reports whether value begins and ends with a double quote.
func hasDoubleQuoteDelimiters(value string) bool {
	return len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"'
}

// IndentedBody collects blank and four-space- or tab-indented Markdown body lines.
func IndentedBody(lines []string, start int) ([]string, int) {
	body := make([]string, 0)
	index := start
	for index < len(lines) {
		if strings.TrimSpace(lines[index]) == "" {
			body = append(body, "")
			index++
			continue
		}

		line, ok := stripBlockIndent(lines[index])
		if !ok {
			break
		}
		body = append(body, line)
		index++
	}
	return body, index
}

// stripBlockIndent removes one supported custom-block indentation level.
func stripBlockIndent(line string) (string, bool) {
	if content, ok := strings.CutPrefix(line, "\t"); ok {
		return content, true
	}
	if content, ok := strings.CutPrefix(line, "    "); ok {
		return content, true
	}
	return "", false
}
