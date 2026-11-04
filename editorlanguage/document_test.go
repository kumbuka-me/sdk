package editorlanguage

import (
	"encoding/json"
	"fmt"
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
		_, err := Parse([]byte(strings.Repeat("x", 128<<10+1)))
		require.Error(t, err)
	})
	t.Run("trailing JSON", func(t *testing.T) {
		t.Parallel()
		_, err := Parse([]byte(`{"version":1,"languages":[{"value":"go","label":"Go"}]} {}`))
		require.Error(t, err)
	})
}

func TestLargeCatalogAndAliases(t *testing.T) {
	document := Document{Version: 1}
	for i := range 260 {
		document.Languages = append(document.Languages, Language{Value: fmt.Sprintf("lang%d", i), Label: "Language"})
	}
	document.Languages = append(document.Languages, Language{Value: "c#", Label: "C#", Aliases: []string{"csharp"}})
	data, err := json.Marshal(document)
	require.NoError(t, err)
	parsed, err := Parse(data)
	require.NoError(t, err)
	require.Equal(t, document, parsed)
}
func TestRejectInvalidAliases(t *testing.T) {
	for _, aliases := range []string{`["JS"]`, `["js","js"]`, `["javascript"]`, `["<script>"]`} {
		_, err := Parse([]byte(`{"version":1,"languages":[{"value":"javascript","label":"JavaScript","aliases":` + aliases + `}]}`))
		require.Error(t, err)
	}
}
