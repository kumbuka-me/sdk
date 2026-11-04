package editorlanguage

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseLanguages(t *testing.T) {
	t.Parallel()
	parsed, err := Parse([]byte(`{"version":1,"languages":[{"value":"mermaid","label":"Mermaid"},{"value":"runbook","label":"Runbook"}]}`))
	require.NoError(t, err)
	require.Equal(t, "mermaid", parsed.Languages[0].Value)
	require.Equal(t, "Runbook", parsed.Languages[1].Label)
}

func TestParseRejectsInvalidDocuments(t *testing.T) {
	t.Parallel()
	t.Run("duplicate", func(t *testing.T) {
		t.Parallel()
		_, err := Parse([]byte(`{"version":1,"languages":[{"value":"mermaid","label":"Mermaid"},{"value":"mermaid","label":"Other"}]}`))
		require.Error(t, err)
	})
	t.Run("unknown fields", func(t *testing.T) {
		t.Parallel()
		_, err := Parse([]byte(`{"version":1,"languages":[{"value":"mermaid","label":"Mermaid","script":"bad"}]}`))
		require.Error(t, err)
	})
	t.Run("invalid id", func(t *testing.T) {
		t.Parallel()
		_, err := Parse([]byte(`{"version":1,"languages":[{"value":"<svg>","label":"Bad"}]}`))
		require.Error(t, err)
	})
	t.Run("oversized", func(t *testing.T) {
		t.Parallel()
		_, err := Parse([]byte(strings.Repeat("x", 16<<10+1)))
		require.Error(t, err)
	})
	t.Run("trailing JSON", func(t *testing.T) {
		t.Parallel()
		_, err := Parse([]byte(`{"version":1,"languages":[{"value":"go","label":"Go"}]} {}`))
		require.Error(t, err)
	})
}
