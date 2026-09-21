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
