package staticexport

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDeclaration(t *testing.T) {
	t.Parallel()

	t.Run("accepts declared browser export", func(t *testing.T) {
		t.Parallel()

		decl, err := ParseDeclaration([]byte(`{"version":1,"modules":[{"id":"diagrams","media_type":"image/svg+xml"}]}`))

		require.NoError(t, err)
		assert.Equal(t, "diagrams", decl.Modules[0].ID)
	})

	t.Run("rejects unsupported media", func(t *testing.T) {
		t.Parallel()

		_, err := ParseDeclaration([]byte(`{"version":1,"modules":[{"id":"diagram","media_type":"text/html"}]}`))

		require.Error(t, err)
	})

	t.Run("rejects unsupported version", func(t *testing.T) {
		t.Parallel()

		_, err := ParseDeclaration([]byte(`{"version":2,"modules":[{"id":"diagram","media_type":"image/svg+xml"}]}`))

		require.Error(t, err)
	})

	t.Run("rejects unknown fields", func(t *testing.T) {
		t.Parallel()

		_, err := ParseDeclaration([]byte(`{"version":1,"unknown":true,"modules":[{"id":"diagram","media_type":"image/svg+xml"}]}`))

		require.Error(t, err)
	})
}

func TestValidateResult(t *testing.T) {
	t.Parallel()

	t.Run("accepts passive SVG", func(t *testing.T) {
		t.Parallel()

		result := Result{Content: `<svg xmlns="http://www.w3.org/2000/svg"><path d="M0 0"/></svg>`, Alignment: "left"}

		require.NoError(t, ValidateResult(result, SVGMediaType))
	})

	t.Run("rejects active SVG", func(t *testing.T) {
		t.Parallel()

		result := Result{Content: `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`}

		require.Error(t, ValidateResult(result, SVGMediaType))
	})

	t.Run("rejects external SVG resources", func(t *testing.T) {
		t.Parallel()

		result := Result{Content: `<svg xmlns="http://www.w3.org/2000/svg"><style>path{fill:url(https://example.test/x)}</style></svg>`}

		require.Error(t, ValidateResult(result, SVGMediaType))
	})

	t.Run("rejects unsafe alignment", func(t *testing.T) {
		t.Parallel()

		result := Result{Content: `<svg xmlns="http://www.w3.org/2000/svg"/>`, Alignment: "left;position:absolute"}

		require.Error(t, ValidateResult(result, SVGMediaType))
	})
}
