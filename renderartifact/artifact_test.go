package renderartifact

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	t.Run("accepts passive SVG", func(t *testing.T) {
		t.Parallel()
		data := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><rect width="20" height="30"/></svg>`)
		require.NoError(t, Validate(SVGMediaType, data))
		require.Len(t, Digest(data), 64)
	})

	t.Run("rejects scripts", func(t *testing.T) {
		t.Parallel()
		require.Error(t, Validate(SVGMediaType, []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)))
	})

	t.Run("allows text labels containing links", func(t *testing.T) {
		t.Parallel()
		data := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><text>https://example.com</text></svg>`)
		require.NoError(t, Validate(SVGMediaType, data))
	})

	t.Run("allows safe inline styles", func(t *testing.T) {
		t.Parallel()
		data := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><style>.node { fill: blue; }</style><rect/></svg>`)
		require.NoError(t, Validate(SVGMediaType, data))
	})

	t.Run("rejects stylesheet imports", func(t *testing.T) {
		t.Parallel()
		data := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><style>@import url(https://evil.test/font.css)</style></svg>`)
		require.Error(t, Validate(SVGMediaType, data))
	})

	t.Run("rejects remote references", func(t *testing.T) {
		t.Parallel()
		require.Error(t, Validate(SVGMediaType, []byte(`<svg xmlns="http://www.w3.org/2000/svg"><rect fill="url(https://example.com/x)"/></svg>`)))
	})

	t.Run("rejects unknown media type", func(t *testing.T) {
		t.Parallel()
		require.Error(t, Validate("text/html", []byte("<html></html>")))
	})
}
