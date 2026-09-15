package cli

import (
	"bytes"
	"context"
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
