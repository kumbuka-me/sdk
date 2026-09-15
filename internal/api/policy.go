package api

const renderPolicyFeaturePrefix = "render-policy."

// ValidRenderPolicy reports whether name is a bounded public rendering-policy
// identifier. Policies are semantic markers shared by plugins; core does not
// attach feature-specific behavior to individual policy names.
func ValidRenderPolicy(name string) bool {
	if len(name) == 0 || len(name) > 128 {
		return false
	}
	for index := 0; index < len(name); index++ {
		char := name[index]
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' {
			continue
		}
		if index > 0 && (char == '.' || char == '_' || char == '-') {
			continue
		}
		return false
	}
	return true
}

// RenderPolicyFeature returns the request feature key for one policy marker.
func RenderPolicyFeature(name string) string { return renderPolicyFeaturePrefix + name }
