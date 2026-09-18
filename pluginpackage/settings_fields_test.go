package pluginpackage

import "testing"

// TestSettingsFields validates typed singleton settings groups alongside boolean feature toggles.
func TestSettingsFields(t *testing.T) {
	manifest := Manifest{
		APIVersion: 1,
		ID:         "com.example.appearance",
		Name:       "Appearance",
		Version:    "1.0.0",
		Modules: []Module{
			{Type: "settings", ID: "feature", Name: "Feature toggle"},
			{
				Type:        "settings",
				ID:          "appearance",
				Name:        "Appearance",
				Description: "Presentation defaults.",
				Fields: []ConfigurationField{
					{ID: "position", Name: "Position", Type: "select", Required: true, Default: "right", Options: []string{"left", "right"}},
					{ID: "highlight", Name: "Highlight", Type: "boolean", Default: "true"},
					{ID: "label", Name: "Label", Type: "text", Default: "External file"},
				},
			},
		},
	}

	if err := manifest.Validate(); err != nil {
		t.Fatal(err)
	}

	withKey := manifest
	withKey.Modules = append([]Module(nil), manifest.Modules...)
	withKey.Modules[1].Fields = append([]ConfigurationField(nil), manifest.Modules[1].Fields...)
	withKey.Modules[1].Fields[0].Key = true
	if err := withKey.Validate(); err == nil {
		t.Fatal("settings key field accepted")
	}

	withDependency := manifest
	withDependency.Modules = append([]Module(nil), manifest.Modules...)
	withDependency.Modules[1].Requires = []string{"feature"}
	if err := withDependency.Validate(); err == nil {
		t.Fatal("typed settings dependency accepted")
	}
}
