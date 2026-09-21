package markdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseQuotedTitle(t *testing.T) {
	t.Parallel()

	t.Run("parses quoted title", func(t *testing.T) {
		t.Parallel()

		title, ok := ParseQuotedTitle(`"Hello"`)

		require.True(t, ok)
		assert.Equal(t, "Hello", title)
	})

	t.Run("rejects empty title", func(t *testing.T) {
		t.Parallel()

		_, ok := ParseQuotedTitle(`""`)

		assert.False(t, ok)
	})

	t.Run("rejects unquoted title", func(t *testing.T) {
		t.Parallel()

		_, ok := ParseQuotedTitle("Hello")

		assert.False(t, ok)
	})

	t.Run("rejects invalid quoted title", func(t *testing.T) {
		t.Parallel()

		_, ok := ParseQuotedTitle(`"unterminated`)

		assert.False(t, ok)
	})
}

func TestIndentedBody(t *testing.T) {
	t.Parallel()

	t.Run("collects indented and blank lines", func(t *testing.T) {
		t.Parallel()

		body, next := IndentedBody(
			[]string{"header", "    one", "", "\ttwo", "stop"},
			1,
		)

		assert.Equal(t, []string{"one", "", "two"}, body)
		assert.Equal(t, 4, next)
	})

	t.Run("stops at first unindented line", func(t *testing.T) {
		t.Parallel()

		body, next := IndentedBody(
			[]string{"    one", "stop", "    two"},
			0,
		)

		assert.Equal(t, []string{"one"}, body)
		assert.Equal(t, 1, next)
	})

	t.Run("starts at end of input", func(t *testing.T) {
		t.Parallel()

		body, next := IndentedBody([]string{"one"}, 1)

		assert.Empty(t, body)
		assert.Equal(t, 1, next)
	})
}

func TestHasDoubleQuoteDelimiters(t *testing.T) {
	t.Parallel()

	t.Run("accepts quoted value", func(t *testing.T) {
		t.Parallel()

		assert.True(t, hasDoubleQuoteDelimiters(`"Hello"`))
	})

	t.Run("accepts empty quoted value", func(t *testing.T) {
		t.Parallel()

		assert.True(t, hasDoubleQuoteDelimiters(`""`))
	})

	t.Run("rejects empty value", func(t *testing.T) {
		t.Parallel()

		assert.False(t, hasDoubleQuoteDelimiters(""))
	})

	t.Run("rejects single quote", func(t *testing.T) {
		t.Parallel()

		assert.False(t, hasDoubleQuoteDelimiters(`"`))
	})

	t.Run("rejects missing opening quote", func(t *testing.T) {
		t.Parallel()

		assert.False(t, hasDoubleQuoteDelimiters(`Hello"`))
	})

	t.Run("rejects missing closing quote", func(t *testing.T) {
		t.Parallel()

		assert.False(t, hasDoubleQuoteDelimiters(`"Hello`))
	})

	t.Run("rejects single quotes", func(t *testing.T) {
		t.Parallel()

		assert.False(t, hasDoubleQuoteDelimiters(`'Hello'`))
	})
}
