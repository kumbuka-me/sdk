package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderPolicyNamesAreGenericIdentifiers(t *testing.T) {
	for _, name := range []string{"preserve-programming-operators", "typography.smart_quotes", "v2"} {
		require.True(t, ValidRenderPolicy(name), "expected valid policy %q", name)
	}
	for _, name := range []string{"", "Uppercase", "has space", "-leading", "trailing/segment"} {
		require.False(t, ValidRenderPolicy(name), "expected invalid policy %q", name)
	}
	got := RenderPolicyFeature("preserve-programming-operators")
	require.Equal(t, "render-policy.preserve-programming-operators", got, "unexpected feature key %q", got)
}
