package api

const renderPolicyFeaturePrefix = "render-policy."

// ValidRenderPolicy reports whether name is a bounded public rendering-policy identifier.
func ValidRenderPolicy(name string) bool {
	if !validRenderPolicyLength(name) {
		return false
	}

	for index := 0; index < len(name); index++ {
		if !validRenderPolicyCharacter(name[index], index) {
			return false
		}
	}
	return true
}

// validRenderPolicyLength reports whether name satisfies the policy identifier length bounds.
func validRenderPolicyLength(name string) bool {
	return len(name) > 0 && len(name) <= 128
}

// validRenderPolicyCharacter reports whether one byte is allowed at the given identifier position.
func validRenderPolicyCharacter(character byte, index int) bool {
	if isLowercaseAlphaNumeric(character) {
		return true
	}
	return index > 0 && isRenderPolicySeparator(character)
}

// isLowercaseAlphaNumeric reports whether character is a lowercase ASCII letter or digit.
func isLowercaseAlphaNumeric(character byte) bool {
	return isLowercaseLetter(character) || isDigit(character)
}

// isLowercaseLetter reports whether character is a lowercase ASCII letter.
func isLowercaseLetter(character byte) bool {
	return character >= 'a' && character <= 'z'
}

// isDigit reports whether character is an ASCII decimal digit.
func isDigit(character byte) bool {
	return character >= '0' && character <= '9'
}

// isRenderPolicySeparator reports whether character is an allowed non-leading policy separator.
func isRenderPolicySeparator(character byte) bool {
	return character == '.' || character == '_' || character == '-'
}

// RenderPolicyFeature returns the request feature key for one policy marker.
func RenderPolicyFeature(name string) string { return renderPolicyFeaturePrefix + name }
