package build

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestModuleToolchainAcceptsGoDirectiveFormatting(t *testing.T) {
	t.Parallel()
	for _, source := range []string{"go 1.27.0", "go\t1.27.0", "  go   1.27.0 // minimum toolchain\r\n", "go 1.27"} {
		t.Run(source, func(t *testing.T) {
			toolchain, err := moduleToolchain([]byte("module example.com/test\n" + source))
			require.NoError(t, err)
			assert.Equal(t, "go1.27.0", toolchain)
		})
	}
	for _, source := range []string{"", "// go 1.27.0", "go", "go latest", "go 1.27.0 unexpected"} {
		_, err := moduleToolchain([]byte(source))
		require.Error(t, err)
	}
}
