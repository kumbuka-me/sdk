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
	// WidgetContext describes one host widget invocation.
	WidgetContext = api.WidgetContext
	// WidgetAction is one safe host-rendered widget control.
	WidgetAction = api.WidgetAction
	// WidgetCommandContext describes one host-mediated widget command invocation.
	WidgetCommandContext = api.WidgetCommandContext
	// WidgetCommandResult controls safe host navigation after a widget command.
	WidgetCommandResult = api.WidgetCommandResult
	// ExportContext describes one authorized page export invocation.
	ExportContext = api.ExportContext
	// ExportFile is one bounded file returned by an exporter plugin.
	ExportFile = api.ExportFile

	// Property is one public page metadata property.
	Property = api.Property
	// Page is authorized public page metadata.
	Page = api.Page
	// PageQuery is a bounded page search query.
	PageQuery = api.PageQuery
	// PageListQuery selects a bounded page/activity list.
	PageListQuery = api.PageListQuery
	// RecentEdit is public metadata for one recent edit.
	RecentEdit = api.RecentEdit
	// PageDraft is bounded private draft metadata for the current viewer.
	PageDraft = api.PageDraft
	// PageRef identifies one page for a capability request.
	PageRef = api.PageRef
	// PageContent is authorized Markdown source.
	PageContent = api.PageContent
	// PageLink is one outgoing wiki-link relationship.
	PageLink = api.PageLink
	// PageLinks contains incoming and outgoing page links.
	PageLinks = api.PageLinks
	// RevisionQuery selects bounded page revision metadata.
	RevisionQuery = api.RevisionQuery
	// Revision is public page revision metadata.
	Revision = api.Revision
	// RevisionHistory contains bounded revision metadata and the total count.
	RevisionHistory = api.RevisionHistory
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

// ValidWidgetSurface reports whether surface is a supported widget placement.
func ValidWidgetSurface(surface string) bool { return api.ValidWidgetSurface(surface) }

// ValidWidgetWidth reports whether width is a supported host layout hint.
func ValidWidgetWidth(width string) bool { return api.ValidWidgetWidth(width) }
