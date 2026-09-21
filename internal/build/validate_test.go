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
		require.NoError(t, os.WriteFile(filepath.Join(directory, name), []byte(value), 0o600))
	}
	write("plugin.yaml", manifest)
	write("README.md", "# Test")
	write("go.mod", "module example.com/test\n\ngo 1.27.0\n")
	_, err := Validate(directory)
	require.NoError(t, err)
	write("plugin.yaml", manifest+"unknown: value\n")
	_, err = Validate(directory)
	require.Error(t, err, "unknown manifest field accepted")
	write("plugin.yaml", strings.Replace(manifest, "api_version: 1", "api_version: 999", 1))
	_, err = Validate(directory)
	require.Error(t, err, "incompatible version accepted")
	write("plugin.yaml", manifest)
	require.NoError(t, os.Mkdir(filepath.Join(directory, "assets"), 0o700))
	require.NoError(t, os.Symlink(filepath.Join(directory, "README.md"), filepath.Join(directory, "assets", "escape")))
	_, err = Validate(directory)
	require.Error(t, err, "asset symlink accepted")
}

func TestDeclarativeBuildOmitsWASMAndGoModule(t *testing.T) {
	directory := t.TempDir()
	write := func(name, value string) {
		t.Helper()
		require.NoError(t, os.WriteFile(filepath.Join(directory, name), []byte(value), 0o600))
	}
	write("plugin.yaml", "api_version: 1\nid: com.example.syntax\nname: Syntax\nversion: 1.0.0\nmodules:\n  - type: markdown-syntax\n    id: syntax\n    syntax: strikethrough\npermissions: []\n")
	write("README.md", "# Syntax\n")

	destination := filepath.Join(t.TempDir(), "syntax.kumbukaplugin")
	require.NoError(t, Build(context.Background(), directory, destination))
	data, err := os.ReadFile(destination)
	require.NoError(t, err)
	pkg, err := pluginpackage.Read(data)
	require.NoError(t, err)
	require.False(t, pkg.Manifest().RequiresWASM(), "declarative manifest unexpectedly requires WASM")
	require.Len(t, pkg.WASM(), 0, "declarative package contains plugin.wasm")
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
