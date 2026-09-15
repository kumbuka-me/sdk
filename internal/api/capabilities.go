package api

import (
	"encoding/json"
	"time"
)

// CapabilityRequest never carries a caller identity. Kumbuka supplies the identity
// and grants from the executing instance, and the viewer from the render scope.
type CapabilityRequest struct {
	// Method selects the host capability operation.
	Method string `json:"method"`
	// Params contains method-specific JSON parameters.
	Params json.RawMessage `json:"params,omitempty"`
}

// CapabilityResponse is the JSON envelope returned by a Kumbuka host capability call.
type CapabilityResponse struct {
	// Value contains the method-specific JSON response payload.
	Value json.RawMessage `json:"value,omitempty"`
	// Error contains a safe guest-visible capability error.
	Error string `json:"error,omitempty"`
}

// Page is the public, persistence-independent page representation exposed to plugins.
type Page struct {
	// Slug is the canonical page path.
	Slug string
	// Title is the human-readable page title.
	Title string
	// Icon is the optional host icon selected for the page.
	Icon string
	// Status is the page lifecycle state.
	Status string
	// OwnerGroup is the optional human-readable owner group.
	OwnerGroup string
	// UpdatedAt is the page modification time.
	UpdatedAt time.Time
	// Author is the display name of the latest page author.
	Author string
	// Tags contains page tags in stable order.
	Tags []string
	// ViewCount is the recorded page view count.
	ViewCount int64
	// Properties contains public page properties in stable order.
	Properties []Property
}

// Property is one public page metadata property.
type Property struct {
	// Key and Value contain one page property pair.
	Key, Value string
}

// PageLink describes one outgoing wiki-link relationship.
type PageLink struct {
	// TargetSlug is the canonical or requested link destination.
	TargetSlug string
	// TargetTitle is the resolved page title when the destination exists.
	TargetTitle string
	// Exists reports whether the destination currently resolves to a page.
	Exists bool
}

// PageLinks contains authorized incoming and outgoing wiki-link relationships.
type PageLinks struct {
	// Backlinks contains pages that link to the requested page.
	Backlinks []Page
	// Outgoing contains links found in the requested page.
	Outgoing []PageLink
}

// RevisionQuery selects bounded revision metadata for one page.
type RevisionQuery struct {
	// Slug is the canonical page path.
	Slug string
	// Limit bounds the returned newest-first revisions.
	Limit int
}

// Revision contains public revision metadata without stored page bodies.
type Revision struct {
	// Number is the monotonically increasing revision number.
	Number int
	// Author is the display name of the editor that created the revision.
	Author string
	// CreatedAt is the revision creation timestamp.
	CreatedAt time.Time
	// Message describes the recorded change.
	Message string
	// AddedLines and RemovedLines summarize the change against the previous revision.
	AddedLines, RemovedLines int
}

// RevisionHistory contains a bounded newest-first revision list and total count.
type RevisionHistory struct {
	// Count is the total number of revisions for the page.
	Count int
	// Revisions contains at most the requested number of newest revisions.
	Revisions []Revision
}

// NavigationNode is one prepared navigation entry exposed to plugins.
type NavigationNode struct {
	// Title, Icon, and URL contain prepared navigation presentation data.
	Title, Icon, URL string
	// Page indicates whether the navigation node represents a page.
	Page bool
	// Children contains nested navigation nodes.
	Children []NavigationNode
}

// PageQuery describes a bounded plugin page search.
type PageQuery struct {
	// Query is the normal Kumbuka search expression.
	Query string
	// Limit bounds the number of returned pages.
	Limit int
}

// PageListQuery selects a bounded list of pages or activity records.
type PageListQuery struct {
	// Limit bounds the number of returned records.
	Limit int
}

// RecentEdit describes a page recently edited by the current viewer.
type RecentEdit struct {
	Page
	// RevisionMessage is the latest revision message for this edit.
	RevisionMessage string
}

// PageDraft contains bounded private draft metadata without editor form values.
type PageDraft struct {
	// Key is the stable private draft identifier.
	Key string
	// PageID is the persisted page identifier, or zero for a new-page draft.
	PageID int64
	// PageSlug is the current canonical slug for an existing page draft.
	PageSlug string
	// Title is the draft title.
	Title string
	// Stale reports whether the published page changed after the draft started.
	Stale bool
	// UpdatedAt is the last draft update time.
	UpdatedAt time.Time
}

// PageRef identifies one page for a capability request.
type PageRef struct {
	// Slug is the canonical page path.
	Slug string
}

// PageContent contains authorized Markdown content returned to a plugin.
type PageContent struct {
	// Slug is the canonical page path.
	Slug string
	// Markdown is the stored page source.
	Markdown string
}

// StorageValue carries a namespaced plugin storage key and optional value.
type StorageValue struct {
	// Key is the plugin-owned settings or data key.
	Key string
	// Value contains the opaque plugin-owned bytes.
	Value []byte
}

// StoredValue distinguishes an absent storage key from an empty stored value.
type StoredValue struct {
	// Value contains the opaque plugin-owned bytes.
	Value []byte
	// Found indicates whether the requested value exists.
	Found bool
}

// IconRequest identifies an icon that Kumbuka should render for a plugin.
type IconRequest struct {
	// Name identifies the icon to render.
	Name string
	// Size is the requested icon size in pixels.
	Size int
}

// LogMessage carries one bounded plugin log entry.
type LogMessage struct {
	// Message is one bounded plugin log message.
	Message string
}

// PermissionFor is the closed set of host operations supported by API v1.
// An empty permission is an explicitly public, non-sensitive operation.
func PermissionFor(method string) (string, bool) {
	switch method {
	case "pages.get", "pages.search", "pages.navigation", "pages.links", "pages.revisions",
		"pages.recent", "pages.popular":
		return "pages:read", true
	case "pages.recent-viewed", "pages.favorites", "pages.recent-edits":
		return "activity:read", true
	case "drafts.list":
		return "drafts:read", true
	case "pages.content":
		return "pages:content", true
	case "attachments.read":
		return "attachments:read", true
	case "plugin.settings.read":
		return "settings:read", true
	case "plugin.settings.write":
		return "settings:write", true
	case "plugin.storage.read":
		return "storage:read", true
	case "plugin.storage.write":
		return "storage:write", true
	case "icons.render", "log":
		return "", true
	default:
		return "", false
	}
}

// ValidPermission reports whether permission is valid.
func ValidPermission(permission string) bool {
	switch permission {
	case "browser:render", "pages:read", "pages:content", "activity:read", "drafts:read", "attachments:read", "settings:read", "settings:write", "storage:read", "storage:write":
		return true
	default:
		return false
	}
}

// AttachmentRead selects a bounded byte range; a request scope must explicitly
// supply an authorized attachment reader before this operation is available.
type AttachmentRead struct {
	// ID identifies the attachment in the authorized request scope.
	ID int64
	// Offset is the first attachment byte to return.
	Offset int64
	// Length is the maximum number of attachment bytes to return.
	Length int
}

// Attachment contains metadata and bounded bytes returned by an attachment capability.
type Attachment struct {
	// Filename and ContentType describe the attachment payload.
	Filename, ContentType string
	// Size is the complete attachment size in bytes.
	Size int64
	// Data contains the requested attachment byte range.
	Data []byte
}
