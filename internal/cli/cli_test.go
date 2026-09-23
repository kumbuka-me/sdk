package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootRequiresSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	require.NoError(t, Run(context.Background(), nil, "v1.2.3", &stdout, &stderr))
	assert.Empty(t, stdout.String())
	assert.Contains(t, stderr.String(), "init")
	assert.Contains(t, stderr.String(), "test")
	assert.Contains(t, stderr.String(), "build")
}

func TestVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	require.NoError(t, Run(context.Background(), []string{"--version"}, "v1.2.3", &stdout, &stderr))
	assert.Contains(t, stdout.String(), "v1.2.3")
	assert.Empty(t, stderr.String())
}

func TestPluginGoMod(t *testing.T) {
	got := pluginGoMod("example", "v0.1.0", "replace github.com/kumbuka-me/sdk => /tmp/sdk")
	assert.Contains(t, got, "module example.com/example")
	assert.Contains(t, got, "require github.com/kumbuka-me/sdk v0.1.0")
	assert.Contains(t, got, "replace github.com/kumbuka-me/sdk => /tmp/sdk")
}

func TestLocalSDKPathWithSpacesProducesValidModule(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "SDK checkout with spaces")
	require.NoError(t, os.Mkdir(directory, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module "+sdkModule+"\n\ngo 1.27.0\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "plugin.go"), []byte("package sdk\n"), 0o600))
	version, replacement, err := resolveSDKPath(directory)
	require.NoError(t, err)
	project := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(project, "go.mod"), []byte(pluginGoMod("sample", version, replacement)), 0o600))
	command := exec.Command("go", "mod", "edit", "-json")
	command.Dir = project
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	var module struct {
		Replace []struct{ New struct{ Path string } }
	}
	require.NoError(t, json.Unmarshal(output, &module))
	require.Len(t, module.Replace, 1)
	assert.Equal(t, filepath.ToSlash(directory), module.Replace[0].New.Path)
}

func TestSDKModulePathAcceptsGoModFormatting(t *testing.T) {
	t.Parallel()

	modulePath, err := sdkModulePath([]byte("// SDK module\r\n  module   github.com/kumbuka-me/sdk   // canonical\r\n\r\ngo 1.27.0\r\n"))
	require.NoError(t, err)
	assert.Equal(t, sdkModule, modulePath)
}

func TestSDKModulePathRejectsAmbiguousModuleDirectives(t *testing.T) {
	t.Parallel()

	_, err := sdkModulePath([]byte("module github.com/kumbuka-me/sdk\nmodule example.com/other\n"))
	require.Error(t, err)
}
