package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalizerUsesLocaleAndEnglishFallbacks(t *testing.T) {
	translations := Translations{
		"en": {"title": "Tasks", "count": "%d tasks"},
		"de": {"title": "Aufgaben"},
	}
	localizer := NewLocalizer("de", translations)
	assert.Equal(t, "de", localizer.Locale())
	assert.Equal(t, "Aufgaben", localizer.Text("title"))
	assert.Equal(t, "2 tasks", localizer.Textf("count", 2))
	assert.Equal(t, "missing", localizer.Text("missing"))
	assert.Equal(t, "en", NewLocalizer("fr", translations).Locale())
}
