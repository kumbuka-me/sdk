package pluginpackage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManifestTopLevelIcon(t *testing.T) {
	t.Parallel()

	t.Run("accepts a valid plugin identity icon", func(t *testing.T) {
		t.Parallel()

		manifest, err := ParseManifest([]byte(`api_version: 1
id: io.example.icon
name: Icon
icon: settings-lucide
version: 1.0.0
modules:
  - type: markdown-syntax
    id: grammar
    syntax: strikethrough
permissions: []
`))

		require.NoError(t, err)
		assert.Equal(t, "settings-lucide", manifest.Icon)
	})

	t.Run("rejects an invalid plugin identity icon", func(t *testing.T) {
		t.Parallel()

		_, err := ParseManifest([]byte(`api_version: 1
id: io.example.icon
name: Icon
icon: Bad Icon
version: 1.0.0
modules:
  - type: markdown-syntax
    id: grammar
    syntax: strikethrough
permissions: []
`))

		require.Error(t, err)
	})
}
