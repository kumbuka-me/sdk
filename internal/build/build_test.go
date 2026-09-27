package build

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWritePackageSkipsIdenticalArchive(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "dist", "plugin.kumbukaplugin")
	archive := []byte("same archive")
	require.NoError(t, writePackage(destination, archive))

	stamp := time.Unix(123, 0)
	require.NoError(t, os.Chtimes(destination, stamp, stamp))
	require.NoError(t, writePackage(destination, archive))

	info, err := os.Stat(destination)
	require.NoError(t, err)
	assert.Equal(t, stamp, info.ModTime())
}

func TestWritePackageReplacesChangedArchive(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "dist", "plugin.kumbukaplugin")
	require.NoError(t, writePackage(destination, []byte("old")))
	require.NoError(t, writePackage(destination, []byte("new")))

	archive, err := os.ReadFile(destination)
	require.NoError(t, err)
	assert.Equal(t, []byte("new"), archive)

	matches, err := filepath.Glob(filepath.Join(filepath.Dir(destination), ".plugin.kumbukaplugin.tmp-*"))
	require.NoError(t, err)
	assert.Empty(t, matches)
}
