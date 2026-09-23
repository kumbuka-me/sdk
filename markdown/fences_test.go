package markdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFenceIndentation(t *testing.T) {
	t.Parallel()

	t.Run("accepts up to three leading spaces", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "```", Fence("   ```go"))
	})

	t.Run("rejects four leading spaces", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, Fence("    ```go"))
	})

	t.Run("rejects leading tab", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, Fence("\t```go"))
	})
}

func TestFenceClosingIndentation(t *testing.T) {
	t.Parallel()

	t.Run("accepts up to three leading spaces", func(t *testing.T) {
		t.Parallel()

		assert.True(t, Closes("   ````   ", "```"))
	})

	t.Run("rejects four leading spaces", func(t *testing.T) {
		t.Parallel()

		assert.False(t, Closes("    ```", "```"))
	})

	t.Run("rejects leading tab", func(t *testing.T) {
		t.Parallel()

		assert.False(t, Closes("\t```", "```"))
	})
}
