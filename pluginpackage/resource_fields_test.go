package pluginpackage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestConfigurationFieldTypes verifies typed configuration schemas and secret defaults.
func TestConfigurationFieldTypes(t *testing.T) {
	manifest := Manifest{
		APIVersion: 1,
		ID:         "com.example.settings",
		Name:       "Settings",
		Version:    "1.0.0",
		Modules: []Module{{
			Type: "admin-resource",
			ID:   "sources",
			Name: "Sources",
			Fields: []ConfigurationField{
				{ID: "name", Name: "Name", Type: "text", Required: true, Key: true},
				{ID: "endpoint", Name: "Endpoint", Type: "url"},
				{ID: "token", Name: "Token", Type: "secret"},
				{ID: "enabled", Name: "Enabled", Type: "boolean", Default: "true"},
				{ID: "provider", Name: "Provider", Type: "select", Options: []string{"github", "gitlab"}, Default: "github"},
			},
		}},
		Permissions: []string{"settings:read", "network:http"},
	}
	require.NoError(t, manifest.Validate())

	manifest.Modules[0].Fields[2].Default = "secret"
	require.Error(t, manifest.Validate(), "secret default accepted")
}

// TestConfigurationListFields verifies repeatable structured configuration rows and color columns.
func TestConfigurationListFields(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		manifest := Manifest{
			APIVersion: 1,
			ID:         "com.example.statuses",
			Name:       "Statuses",
			Version:    "1.0.0",
			Modules: []Module{{
				Type: "admin-resource",
				ID:   "sets",
				Name: "Sets",
				Fields: []ConfigurationField{
					{ID: "name", Name: "Name", Type: "text", Required: true, Key: true},
					{
						ID:       "statuses",
						Name:     "Statuses",
						Type:     "list",
						Required: true,
						MaxItems: 16,
						Columns: []ConfigurationField{
							{ID: "label", Name: "Status", Type: "text", Required: true, MaxBytes: 64},
							{ID: "color", Name: "Color", Type: "color", Required: true, Default: "#64748b"},
						},
					},
				},
			}},
		}
		require.NoError(t, manifest.Validate())
	})

	t.Run("nested list", func(t *testing.T) {
		field := ConfigurationField{
			ID: "rows", Name: "Rows", Type: "list", Columns: []ConfigurationField{{
				ID: "nested", Name: "Nested", Type: "list", Columns: []ConfigurationField{{ID: "value", Name: "Value", Type: "text"}},
			}},
		}
		require.False(t, validConfigurationField(field, map[string]bool{}))
	})

	t.Run("invalid color", func(t *testing.T) {
		field := ConfigurationField{ID: "color", Name: "Color", Type: "color", Default: "rosa"}
		require.False(t, validConfigurationField(field, map[string]bool{}))
	})
}
