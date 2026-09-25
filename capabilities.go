package sdk

import "github.com/kumbuka-me/sdk/internal/api"

// Transport enables native tests to supply fake host capabilities. Production
// clients use Kumbuka's WASM transport; permissions remain enforced by the host.
type Transport func(method string, params, result any) error

// Client provides typed capabilities bound to one transport.
type Client struct {
	// call invokes the configured host capability transport.
	call Transport
}

// NewClient creates a client. A nil transport uses the Kumbuka host.
func NewClient(transport Transport) Client {
	if transport == nil {
		transport = api.Call
	}
	return Client{call: transport}
}

// Pages returns the host page capabilities.
func Pages() PageClient { return NewClient(nil).Pages() }

// Drafts returns private draft metadata for the current viewer.
func Drafts() DraftClient { return NewClient(nil).Drafts() }

// Settings returns this plugin's settings namespace.
func Settings() Values { return NewClient(nil).Settings() }

// Resources returns manifest-declared structured settings owned by this plugin.
func Resources() ResourceClient { return NewClient(nil).Resources() }

// Storage returns this plugin's data namespace.
func Storage() Values { return NewClient(nil).Storage() }

// HTTP returns the bounded outbound HTTP client provided by Kumbuka.
func HTTP() HTTPClient { return NewClient(nil).HTTP() }

// Attachments returns the host attachment capability.
func Attachments() AttachmentClient { return NewClient(nil).Attachments() }

// Users returns the plugin-safe Kumbuka user directory.
func Users() UserClient { return NewClient(nil).Users() }

// Notifications returns in-app notification operations.
func Notifications() NotificationClient { return NewClient(nil).Notifications() }

// Log sends a bounded log entry to Kumbuka.
func Log(message string) error { return NewClient(nil).Log(message) }

// Icon requests a host-rendered icon.
func Icon(name string, size int) (string, error) { return NewClient(nil).Icon(name, size) }

// Pages returns page operations on this client.
func (c Client) Pages() PageClient { return PageClient{client: c} }

// Drafts returns private draft operations on this client.
func (c Client) Drafts() DraftClient { return DraftClient{client: c} }

// Settings returns the calling plugin's settings namespace.
func (c Client) Settings() Values { return Values{client: c, method: "plugin.settings"} }

// Resources returns structured settings declared by the calling plugin.
func (c Client) Resources() ResourceClient { return ResourceClient{client: c} }

// Storage returns the calling plugin's data namespace.
func (c Client) Storage() Values { return Values{client: c, method: "plugin.storage"} }

// HTTP returns outbound HTTP operations on this client.
func (c Client) HTTP() HTTPClient { return HTTPClient{client: c} }

// Attachments returns attachment operations on this client.
func (c Client) Attachments() AttachmentClient { return AttachmentClient{client: c} }

// Users returns user-directory operations on this client.
func (c Client) Users() UserClient { return UserClient{client: c} }

// Notifications returns in-app notification operations on this client.
func (c Client) Notifications() NotificationClient { return NotificationClient{client: c} }

// Log records a host log message.
func (c Client) Log(message string) error {
	return c.call("log", LogMessage{Message: message}, nil)
}

// Icon renders an icon through the public host capability.
func (c Client) Icon(name string, size int) (string, error) {
	return callResult[string](c, "icons.render", IconRequest{Name: name, Size: size})
}

// callResult invokes one typed host capability and returns its decoded result.
func callResult[T any](client Client, method string, params any) (T, error) {
	var result T
	err := client.call(method, params, &result)
	return result, err
}

// PageClient provides permission-checked page operations.
type PageClient struct {
	// client carries the host capability transport.
	client Client
}

// Get returns authorized metadata for a canonical page slug.
func (p PageClient) Get(slug string) (Page, error) {
	return callResult[Page](p.client, "pages.get", PageRef{Slug: slug})
}

// Search returns authorized pages matching a bounded query.
func (p PageClient) Search(query PageQuery) ([]Page, error) {
	return callResult[[]Page](p.client, "pages.search", query)
}

// Content reads authorized Markdown source.
func (p PageClient) Content(slug string) (PageContent, error) {
	return callResult[PageContent](p.client, "pages.content", PageRef{Slug: slug})
}

// Links returns authorized incoming and outgoing wiki-link relationships.
func (p PageClient) Links(slug string) (PageLinks, error) {
	return callResult[PageLinks](p.client, "pages.links", PageRef{Slug: slug})
}

// Revisions returns bounded public revision metadata for one authorized page.
func (p PageClient) Revisions(query RevisionQuery) (RevisionHistory, error) {
	return callResult[RevisionHistory](p.client, "pages.revisions", query)
}

// Recent returns the newest authorized pages.
func (p PageClient) Recent(limit int) ([]Page, error) {
	return callResult[[]Page](p.client, "pages.recent", PageListQuery{Limit: limit})
}

// RecentViewed returns pages most recently viewed by the current viewer.
func (p PageClient) RecentViewed(limit int) ([]Page, error) {
	return callResult[[]Page](p.client, "pages.recent-viewed", PageListQuery{Limit: limit})
}

// Favorites returns pages favorited by the current viewer.
func (p PageClient) Favorites(limit int) ([]Page, error) {
	return callResult[[]Page](p.client, "pages.favorites", PageListQuery{Limit: limit})
}

// Popular returns the most viewed authorized pages.
func (p PageClient) Popular(limit int) ([]Page, error) {
	return callResult[[]Page](p.client, "pages.popular", PageListQuery{Limit: limit})
}

// RecentEdited returns pages most recently edited by the current viewer.
func (p PageClient) RecentEdited(limit int) ([]RecentEdit, error) {
	return callResult[[]RecentEdit](p.client, "pages.recent-edits", PageListQuery{Limit: limit})
}

// Navigation returns the prepared navigation tree.
func (p PageClient) Navigation() ([]NavigationNode, error) {
	return callResult[[]NavigationNode](p.client, "pages.navigation", nil)
}

// DraftClient provides access to the current viewer's private draft metadata.
type DraftClient struct {
	// client carries the host capability transport.
	client Client
}

// List returns bounded private draft metadata without editor form values.
func (d DraftClient) List(limit int) ([]PageDraft, error) {
	return callResult[[]PageDraft](d.client, "drafts.list", PageListQuery{Limit: limit})
}

// Values is the calling plugin's host-managed key/value namespace.
type Values struct {
	// client carries the host capability transport.
	client Client
	// method is the fixed capability namespace prefix for this value store.
	method string
}

// Get reads a key, preserving the distinction between absent and empty values.
func (v Values) Get(key string) (StoredValue, error) {
	return callResult[StoredValue](v.client, v.method+".read", StorageValue{Key: key})
}

// Set writes opaque bytes. Kumbuka enforces permissions and namespace quotas.
func (v Values) Set(key string, value []byte) error {
	return v.client.call(v.method+".write", StorageValue{Key: key, Value: value}, nil)
}

// ResourceClient reads structured plugin settings declared as admin resources.
type ResourceClient struct {
	// client carries the host capability transport.
	client Client
}

// Get returns one configured resource record, including decrypted secret fields.
func (r ResourceClient) Get(resource, key string) (PluginResourceRecord, error) {
	return callResult[PluginResourceRecord](r.client, "plugin.resources.get", PluginResourceRequest{Resource: resource, Key: key})
}

// List returns configured records for one resource in deterministic key order.
func (r ResourceClient) List(resource string) ([]PluginResourceRecord, error) {
	return callResult[[]PluginResourceRecord](r.client, "plugin.resources.list", PluginResourceListRequest{Resource: resource})
}

// HTTPClient performs bounded outbound HTTP through Kumbuka's host network policy.
type HTTPClient struct {
	// client carries the host capability transport.
	client Client
}

// Do performs one bounded HTTP request through the Kumbuka host.
func (h HTTPClient) Do(request HTTPRequest) (HTTPResponse, error) {
	return callResult[HTTPResponse](h.client, "http.do", request)
}

// AttachmentClient reads attachments authorized by the current request scope.
type AttachmentClient struct {
	// client carries the host capability transport.
	client Client
}

// Read requests a bounded attachment byte range.
func (a AttachmentClient) Read(request AttachmentRead) (Attachment, error) {
	return callResult[Attachment](a.client, "attachments.read", request)
}

// UserClient provides plugin-safe Kumbuka user-directory operations.
type UserClient struct {
	// client carries the host capability transport.
	client Client
}

// Search returns enabled users matching a canonical mention or display name.
func (u UserClient) Search(query UserQuery) ([]User, error) {
	return callResult[[]User](u.client, "users.search", query)
}

// ResolveMention resolves an @-prefixed mention to one enabled user.
func (u UserClient) ResolveMention(mention string) (User, error) {
	return callResult[User](u.client, "users.resolve-mention", UserMention{Mention: mention})
}

// NotificationClient creates core-owned in-app notifications.
type NotificationClient struct {
	// client carries the host capability transport.
	client Client
}

// Send creates one attributed in-app notification for a Kumbuka user.
func (n NotificationClient) Send(input NotificationInput) (Notification, error) {
	return callResult[Notification](n.client, "notifications.send", input)
}
