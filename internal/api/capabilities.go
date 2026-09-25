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

// RevisionQuery selects bounded page revision metadata for one page.
type RevisionQuery struct {
	// Slug is the canonical page path.
	Slug string
	// Limit bounds the returned newest-first revisions.
	Limit int
}

// Revision contains public page revision metadata without stored page bodies.
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
	// Page contains the public page metadata for the edit.
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

// PluginResourceRequest selects one record from a manifest-declared admin resource.
type PluginResourceRequest struct {
	// Resource identifies the admin-resource module in the calling plugin.
	Resource string `json:"resource"`
	// Key identifies the record by its declared key field.
	Key string `json:"key"`
}

// PluginResourceListRequest selects all records from one manifest-declared admin resource.
type PluginResourceListRequest struct {
	// Resource identifies the admin-resource module in the calling plugin.
	Resource string `json:"resource"`
}

// PluginResourceRecord contains one plugin-owned structured setting record.
type PluginResourceRecord struct {
	// Key is the canonical record key.
	Key string `json:"key"`
	// Values contains field values keyed by manifest field ID, including decrypted secrets.
	Values map[string]string `json:"values"`
}

// HTTPRequest describes one bounded outbound HTTP request performed by the Kumbuka host.
type HTTPRequest struct {
	// Method is an HTTP method such as GET or POST.
	Method string `json:"method"`
	// URL is the absolute HTTP or HTTPS destination.
	URL string `json:"url"`
	// Headers contains application-level request headers supplied by the plugin.
	Headers map[string]string `json:"headers,omitempty"`
	// Body contains the optional request payload.
	Body []byte `json:"body,omitempty"`
	// AllowedPrivateIPs lists exact administrator-configured private addresses that may be dialed.
	AllowedPrivateIPs []string `json:"allowed_private_ips,omitempty"`
	// InsecureSkipVerify disables origin TLS certificate verification when separately permitted.
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty"`
}

// HTTPResponse contains one bounded response returned by the Kumbuka HTTP host capability.
type HTTPResponse struct {
	// StatusCode is the upstream HTTP status code.
	StatusCode int `json:"status_code"`
	// Headers contains bounded upstream response headers.
	Headers map[string][]string `json:"headers,omitempty"`
	// Body contains the bounded upstream response payload.
	Body []byte `json:"body,omitempty"`
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

// UserQuery describes a bounded search of the Kumbuka user directory.
type UserQuery struct {
	// Query matches canonical mentions and display names.
	Query string `json:"query"`
	// Limit bounds the number of returned users.
	Limit int `json:"limit"`
}

// UserMention resolves one canonical or case-insensitive Kumbuka mention.
type UserMention struct {
	// Mention contains an @-prefixed Kumbuka username.
	Mention string `json:"mention"`
}

// User is the plugin-safe representation of one enabled Kumbuka user.
type User struct {
	// ID is the stable Kumbuka user identifier.
	ID int64 `json:"id"`
	// Mention is the canonical @-prefixed username.
	Mention string `json:"mention"`
	// DisplayName is the user's human-readable name.
	DisplayName string `json:"display_name"`
}

// NotificationInput describes one in-app notification requested by a plugin.
type NotificationInput struct {
	// RecipientUserID identifies the Kumbuka user who receives the notification.
	RecipientUserID int64 `json:"recipient_user_id"`
	// Title is the concise notification heading.
	Title string `json:"title"`
	// Body is optional plain-text notification detail.
	Body string `json:"body,omitempty"`
	// URL is an optional local Kumbuka destination.
	URL string `json:"url,omitempty"`
	// IdempotencyKey deduplicates retries within the calling plugin and recipient.
	IdempotencyKey string `json:"idempotency_key"`
}

// Notification is the committed receipt returned after creating a notification.
type Notification struct {
	// ID is the stable notification identifier.
	ID int64 `json:"id"`
	// RecipientUserID identifies the notification owner.
	RecipientUserID int64 `json:"recipient_user_id"`
	// CreatedAt records when core committed the notification.
	CreatedAt time.Time `json:"created_at"`
}

// PermissionFor returns the required permission for one supported API v1 host operation.
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
	case "plugin.settings.read", "plugin.resources.get", "plugin.resources.list":
		return "settings:read", true
	case "plugin.settings.write":
		return "settings:write", true
	case "plugin.storage.read":
		return "storage:read", true
	case "plugin.storage.write":
		return "storage:write", true
	case "http.do":
		return "network:http", true
	case "users.search", "users.resolve-mention":
		return "users:read", true
	case "notifications.send":
		return "notifications:send", true
	case "icons.render", "log":
		return "", true
	default:
		return "", false
	}
}

// ValidPermission reports whether permission is valid.
func ValidPermission(permission string) bool {
	switch permission {
	case "browser:render", "pages:read", "pages:content", "activity:read", "drafts:read",
		"attachments:read", "settings:read", "settings:write", "storage:read", "storage:write",
		"network:http", "network:private", "network:insecure-tls", "users:read", "notifications:send":
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
