package build

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kumbuka-me/sdk/pluginpackage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationRejectsUnsafeProjects(t *testing.T) {
	directory := t.TempDir()
	manifest := "api_version: 1\nid: com.example.test\nname: Test\nversion: 1.0.0\nmodules:\n  - type: renderer-extension\n    id: test\n    stage: preprocess\npermissions: []\n"
	write := func(name, value string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(directory, name), []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("plugin.yaml", manifest)
	write("README.md", "# Test")
	write("go.mod", "module example.com/test\n\ngo 1.27.0\n")
	if _, err := Validate(directory); err != nil {
		t.Fatal(err)
	}
	write("plugin.yaml", manifest+"unknown: value\n")
	if _, err := Validate(directory); err == nil {
		t.Fatal("unknown manifest field accepted")
	}
	write("plugin.yaml", strings.Replace(manifest, "api_version: 1", "api_version: 999", 1))
	if _, err := Validate(directory); err == nil {
		t.Fatal("incompatible version accepted")
	}
	write("plugin.yaml", manifest)
	if err := os.Mkdir(filepath.Join(directory, "assets"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(directory, "README.md"), filepath.Join(directory, "assets", "escape")); err != nil {
		t.Fatal(err)
	}
	if _, err := Validate(directory); err == nil {
		t.Fatal("asset symlink accepted")
	}
}

func TestDeclarativeBuildOmitsWASMAndGoModule(t *testing.T) {
	directory := t.TempDir()
	write := func(name, value string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(directory, name), []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("plugin.yaml", "api_version: 1\nid: com.example.syntax\nname: Syntax\nversion: 1.0.0\nmodules:\n  - type: markdown-syntax\n    id: syntax\n    syntax: strikethrough\npermissions: []\n")
	write("README.md", "# Syntax\n")

	destination := filepath.Join(t.TempDir(), "syntax.kumbukaplugin")
	if err := Build(context.Background(), directory, destination); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := pluginpackage.Read(data)
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Manifest().RequiresWASM() {
		t.Fatal("declarative manifest unexpectedly requires WASM")
	}
	if len(pkg.WASM()) != 0 {
		t.Fatal("declarative package contains plugin.wasm")
	}
}

func TestExecutablePluginUsesContainingGoModule(t *testing.T) {
	root := t.TempDir()
	write := func(path, value string) {
		t.Helper()
		filename := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0o755))
		require.NoError(t, os.WriteFile(filename, []byte(value), 0o600))
	}

	write("go.mod", "module example.com/plugins\n\ngo 1.27.0\n")
	write("callouts/plugin.yaml", "api_version: 1\nid: com.example.callouts\nname: Callouts\nversion: 1.0.0\nmodules:\n  - type: renderer-extension\n    id: callouts\n    stage: preprocess\npermissions: []\n")
	write("callouts/README.md", "# Callouts\n")

	manifest, err := Validate(filepath.Join(root, "callouts"))
	require.NoError(t, err)
	assert.True(t, manifest.RequiresWASM())
}

func TestValidationRequiresDeclaredIconResourceAsset(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(directory, "assets"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "README.md"), []byte("# Icons\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "plugin.yaml"), []byte(`api_version: 1
id: io.example.icons
name: Icons
version: 1.0.0
modules:
  - type: icon-resource
    id: brands
    name: Brand Icons
    asset: icons.json
permissions: []
`), 0o600))

	_, err := Validate(directory)
	require.Error(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(directory, "assets", "icons.json"), []byte(`{"format":1,"icons":[{"name":"example-brand","label":"Example","view_box":"0 0 24 24","paths":["M0 0H24V24H0z"]}]}`), 0o600))
	_, err = Validate(directory)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(directory, "assets", "icons.json"), []byte(`{"format":1,"icons":[]}`), 0o600))
	_, err = Validate(directory)
	require.Error(t, err)
}
