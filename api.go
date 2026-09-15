// Package sdk provides the public Go SDK for Kumbuka plugins.
package sdk

import "github.com/kumbuka-me/sdk/internal/api"

// Version is the Kumbuka plugin API version supported by this SDK.
const Version = api.Version

type (
	// CapabilityRequest is the low-level host capability request envelope.
	CapabilityRequest = api.CapabilityRequest
	// CapabilityResponse is the low-level host capability response envelope.
	CapabilityResponse = api.CapabilityResponse

	// Request is a render invocation supplied by Kumbuka.
	Request = api.RenderRequest
	// Result contains intermediate output returned to Kumbuka.
	Result = api.RenderResult
	// Part is literal intermediate output or Markdown rendered by Kumbuka.
	Part = api.RenderPart

	// RenderRequest is the wire-level render request used by the Kumbuka host.
	RenderRequest = api.RenderRequest
	// RenderResult is the wire-level render result used by the Kumbuka host.
	RenderResult = api.RenderResult
	// RenderPart is one wire-level render output fragment.
	RenderPart = api.RenderPart

	// Property is one public page metadata property.
	Property = api.Property
	// Page is authorized public page metadata.
	Page = api.Page
	// PageQuery is a bounded page search query.
	PageQuery = api.PageQuery
	// PageRef identifies one page for a capability request.
	PageRef = api.PageRef
	// PageContent is authorized Markdown source.
	PageContent = api.PageContent
	// NavigationNode is an authorized navigation entry.
	NavigationNode = api.NavigationNode

	// StorageValue carries a plugin storage key and optional value.
	StorageValue = api.StorageValue
	// StoredValue distinguishes an absent key from an empty value.
	StoredValue = api.StoredValue

	// AttachmentRead requests a bounded attachment range.
	AttachmentRead = api.AttachmentRead
	// Attachment contains authorized attachment bytes and metadata.
	Attachment = api.Attachment

	// IconRequest identifies an icon to render through the host.
	IconRequest = api.IconRequest
	// LogMessage carries one bounded plugin log entry.
	LogMessage = api.LogMessage
)

// PermissionFor returns the manifest permission required for a host capability.
func PermissionFor(method string) (string, bool) { return api.PermissionFor(method) }

// ValidPermission reports whether permission is part of the public API contract.
func ValidPermission(permission string) bool { return api.ValidPermission(permission) }

// ValidSyntax reports whether name identifies a supported declarative Markdown grammar.
func ValidSyntax(name string) bool { return api.ValidSyntax(name) }

// ValidRenderPolicy reports whether name is a valid public render-policy marker.
func ValidRenderPolicy(name string) bool { return api.ValidRenderPolicy(name) }

// RenderPolicyFeature returns the request feature key for one render-policy marker.
func RenderPolicyFeature(name string) string { return api.RenderPolicyFeature(name) }

// RenderPolicyEnabled reports whether one render-policy marker is active for a request.
func RenderPolicyEnabled(features map[string]bool, name string) bool {
	return features[api.RenderPolicyFeature(name)]
}
