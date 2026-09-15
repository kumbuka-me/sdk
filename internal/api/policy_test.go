package api

import "testing"

func TestRenderPolicyNamesAreGenericIdentifiers(t *testing.T) {
	for _, name := range []string{"preserve-programming-operators", "typography.smart_quotes", "v2"} {
		if !ValidRenderPolicy(name) {
			t.Fatalf("expected valid policy %q", name)
		}
	}
	for _, name := range []string{"", "Uppercase", "has space", "-leading", "trailing/segment"} {
		if ValidRenderPolicy(name) {
			t.Fatalf("expected invalid policy %q", name)
		}
	}
	if got := RenderPolicyFeature("preserve-programming-operators"); got != "render-policy.preserve-programming-operators" {
		t.Fatalf("unexpected feature key %q", got)
	}
}
