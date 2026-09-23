package pluginpackage

import (
	"net/url"
	"strings"
	"unicode/utf8"
)

// validConfigurationField validates one bounded structured configuration field.
func validConfigurationField(field ConfigurationField, seen map[string]bool) bool {
	if !validConfigurationFieldIdentity(field, seen) {
		return false
	}
	if !validConfigurationFieldBounds(field) {
		return false
	}
	if field.Type != "list" && (field.MaxItems != 0 || len(field.Columns) != 0) {
		return false
	}

	switch field.Type {
	case "text", "textarea", "url":
		return len(field.Options) == 0 && validConfigurationDefault(field)
	case "color":
		return len(field.Options) == 0 && !field.Key && (field.Default == "" || validConfigurationColor(field.Default))
	case "secret":
		return len(field.Options) == 0 && field.Default == "" && !field.Key
	case "boolean":
		return len(field.Options) == 0 && (field.Default == "" || field.Default == "true" || field.Default == "false") && !field.Key
	case "select":
		return validConfigurationOptions(field) && !field.Key
	case "list":
		return validConfigurationList(field)
	default:
		return false
	}
}

// validConfigurationFieldIdentity reports whether a configuration field has a unique bounded identity.
func validConfigurationFieldIdentity(field ConfigurationField, seen map[string]bool) bool {
	return identifier.MatchString(field.ID) &&
		!seen[field.ID] &&
		strings.TrimSpace(field.Name) != "" &&
		len(field.Name) <= 128
}

// validConfigurationFieldBounds reports whether field size constraints and key usage are supported.
func validConfigurationFieldBounds(field ConfigurationField) bool {
	return field.MaxBytes >= 0 &&
		field.MaxBytes <= 64<<10 &&
		field.MaxItems >= 0 &&
		field.MaxItems <= 64 &&
		validConfigurationKey(field)
}

// validConfigurationKey reports whether a key field is required text or the field is not a key.
func validConfigurationKey(field ConfigurationField) bool {
	return !field.Key || (field.Type == "text" && field.Required)
}

// validConfigurationColor reports whether value is a canonical six-digit CSS hex color.
func validConfigurationColor(value string) bool {
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	for _, char := range value[1:] {
		if !isHexDigit(char) {
			return false
		}
	}
	return true
}

// isHexDigit reports whether character is an ASCII hexadecimal digit.
func isHexDigit(character rune) bool {
	return character >= '0' && character <= '9' ||
		character >= 'a' && character <= 'f' ||
		character >= 'A' && character <= 'F'
}

// validConfigurationList validates one repeatable structured row field.
func validConfigurationList(field ConfigurationField) bool {
	if !validConfigurationListShape(field) {
		return false
	}

	seen := make(map[string]bool, len(field.Columns))
	for _, column := range field.Columns {
		if !validConfigurationListColumn(column) {
			return false
		}
		if !validConfigurationField(column, seen) {
			return false
		}
		seen[column.ID] = true
	}
	return true
}

// validConfigurationListShape reports whether a list field has supported list-only metadata.
func validConfigurationListShape(field ConfigurationField) bool {
	return !field.Key &&
		field.Default == "" &&
		len(field.Options) == 0 &&
		len(field.Columns) > 0 &&
		len(field.Columns) <= 8
}

// validConfigurationListColumn reports whether a field type is supported inside structured list rows.
func validConfigurationListColumn(column ConfigurationField) bool {
	switch column.Type {
	case "list", "secret", "textarea", "boolean":
		return false
	default:
		return !column.Key
	}
}

// validConfigurationDefault validates a bounded default value for scalar configuration fields.
func validConfigurationDefault(field ConfigurationField) bool {
	if !utf8.ValidString(field.Default) || strings.ContainsRune(field.Default, '\x00') {
		return false
	}
	if len(field.Default) > configurationByteLimit(field) {
		return false
	}
	if field.Type != "url" || field.Default == "" {
		return true
	}
	parsed, err := url.ParseRequestURI(field.Default)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

// configurationByteLimit returns the explicit field limit or the host default for its type.
func configurationByteLimit(field ConfigurationField) int {
	if field.MaxBytes != 0 {
		return field.MaxBytes
	}
	if field.Type == "textarea" {
		return 48 << 10
	}
	return 4096
}

// validConfigurationOptions validates the bounded option set and optional default of a select field.
func validConfigurationOptions(field ConfigurationField) bool {
	if len(field.Options) == 0 || len(field.Options) > 32 {
		return false
	}
	seen := make(map[string]bool, len(field.Options))
	limit := min(128, configurationByteLimit(field))
	for _, option := range field.Options {
		if !validConfigurationOption(option, seen, limit) {
			return false
		}
		seen[option] = true
	}
	return field.Default == "" || seen[field.Default]
}

// validConfigurationOption reports whether one select option is unique, bounded, and canonical.
func validConfigurationOption(option string, seen map[string]bool, limit int) bool {
	return option != "" &&
		strings.TrimSpace(option) == option &&
		len(option) <= limit &&
		utf8.ValidString(option) &&
		!strings.ContainsRune(option, '\x00') &&
		!seen[option]
}
