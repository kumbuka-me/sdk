package api

// ValidSyntax reports whether name identifies a versioned standard Markdown grammar.
func ValidSyntax(name string) bool {
	switch name {
	case "tables", "strikethrough", "task-list", "definition-list", "footnote", "linkify":
		return true
	default:
		return false
	}
}
