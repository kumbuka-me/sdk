package pluginpackage

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIconResource(t *testing.T) {
	resource, err := ParseIconResource([]byte(`{
  "format": 1,
  "icons": [
    {"name":"example-brand","label":"Example","view_box":"0   0  24 24","paths":["M0 0H24V24H0z"]}
  ]
}`))
	require.NoError(t, err)
	require.Len(t, resource.Icons, 1)
	assert.Equal(t, "example-brand", resource.Icons[0].Name)
	assert.Equal(t, "0 0 24 24", resource.Icons[0].ViewBox)
}

func TestParseIconResourceRejectsInvalidData(t *testing.T) {
	valid := `{"format":1,"icons":[{"name":"example-brand","label":"Example","view_box":"0 0 24 24","paths":["M0 0H24V24H0z"]}]}`
	for _, data := range []string{
		`{"format":2,"icons":[{"name":"example-brand","label":"Example","view_box":"0 0 24 24","paths":["M0 0H24V24H0z"]}]}`,
		`{"format":1,"icons":[]}`,
		strings.Replace(valid, `"example-brand"`, `"Example Brand"`, 1),
		strings.Replace(valid, `"Example"`, `""`, 1),
		strings.Replace(valid, `"0 0 24 24"`, `"0 0 0 24"`, 1),
		strings.Replace(valid, `["M0 0H24V24H0z"]`, `[]`, 1),
		strings.Replace(valid, `"icons":`, `"unknown":true,"icons":`, 1),
		valid + `{}`,
	} {
		_, err := ParseIconResource([]byte(data))
		require.Error(t, err, data)
	}
}
