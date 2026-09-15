package api

// ValidRenderPolicy identifies rendering policies that plugins may request from
// the host without receiving access to Kumbuka internals.
func ValidRenderPolicy(name string) bool {
	switch name {
	case "coding-ligatures", "typographer":
		return true
	default:
		return false
	}
}
