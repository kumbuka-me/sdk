package api

// ValidSyntax identifies versioned standard Markdown grammars provided by core.
// Packages declare grammars; they never pass native code across the runtime boundary.
func ValidSyntax(name string) bool {
	switch name {
	case "tables", "strikethrough", "task-list", "definition-list", "footnote", "linkify":
		return true
	default:
		return false
	}
}
