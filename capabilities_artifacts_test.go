package sdk

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestArtifactCapabilities verifies that plugins cannot select storage destinations.
func TestArtifactCapabilities(t *testing.T) {
	t.Parallel()

	client := NewClient(func(method string, params, result any) error {
		switch method {
		case "artifacts.put":
			request, ok := params.(RenderedArtifactWrite)
			if !ok || request.MediaType != "image/svg+xml" {
				return fmt.Errorf("unexpected write payload: %T", params)
			}
		case "artifacts.render":
			request, ok := params.(RenderedArtifactRender)
			if !ok || request.Script != "headless.js" {
				return fmt.Errorf("unexpected render payload: %T", params)
			}
		default:
			return fmt.Errorf("unexpected method %s", method)
		}
		*result.(*RenderedArtifactRef) = RenderedArtifactRef{URL: "/assets/rendered/example.svg"}
		return nil
	})

	t.Run("put", func(t *testing.T) {
		t.Parallel()
		ref, err := client.Artifacts().Put(RenderedArtifactWrite{MediaType: "image/svg+xml"})
		require.NoError(t, err)
		require.Contains(t, ref.URL, "example.svg")
	})

	t.Run("render", func(t *testing.T) {
		t.Parallel()
		ref, err := client.Artifacts().Render(RenderedArtifactRender{Script: "headless.js"})
		require.NoError(t, err)
		require.Contains(t, ref.URL, "example.svg")
	})

	permission, known := PermissionFor("artifacts.render")
	require.True(t, known)
	require.Equal(t, "artifacts:write", permission)
}
