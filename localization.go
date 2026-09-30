package sdk

import "fmt"

const DefaultLocale = "en"

// Translations contains messages by canonical locale and message key.
type Translations map[string]map[string]string

// Localizer resolves plugin-owned messages for one host-selected locale.
type Localizer struct {
	// locale is the supported request language, falling back to English.
	locale string
	// translations references plugin-owned catalogs that callers must keep immutable.
	translations Translations
}

// NewLocalizer binds immutable plugin translations to one request locale.
func NewLocalizer(locale string, translations Translations) Localizer {
	if _, ok := translations[locale]; !ok {
		locale = DefaultLocale
	}
	return Localizer{locale: locale, translations: translations}
}

// Locale returns the effective locale used by this localizer.
func (l Localizer) Locale() string { return l.locale }

// Text returns one translated message, falling back to English and then the key.
func (l Localizer) Text(key string) string {
	if messages := l.translations[l.locale]; messages != nil {
		if value, ok := messages[key]; ok {
			return value
		}
	}
	if messages := l.translations[DefaultLocale]; messages != nil {
		if value, ok := messages[key]; ok {
			return value
		}
	}
	return key
}

// Textf formats one translated message using fmt-style placeholders.
func (l Localizer) Textf(key string, values ...any) string {
	return fmt.Sprintf(l.Text(key), values...)
}
