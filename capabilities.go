package sdk

import "github.com/kumbuka-me/sdk/internal/api"

// Transport enables native tests to supply fake host capabilities. Production
// clients use Kumbuka's WASM transport; permissions remain enforced by the host.
type Transport func(method string, params, result any) error

// Client provides typed capabilities bound to one transport.
type Client struct{ call Transport }

// NewClient creates a client. A nil transport uses the Kumbuka host.
func NewClient(transport Transport) Client {
	if transport == nil {
		transport = api.Call
	}
	return Client{call: transport}
}

// Pages returns the host page capabilities.
func Pages() PageClient { return NewClient(nil).Pages() }

// Settings returns this plugin's settings namespace.
func Settings() Values { return NewClient(nil).Settings() }

// Storage returns this plugin's data namespace.
func Storage() Values { return NewClient(nil).Storage() }

// Attachments returns the host attachment capability.
func Attachments() AttachmentClient { return NewClient(nil).Attachments() }

// Log sends a bounded log entry to Kumbuka.
func Log(message string) error { return NewClient(nil).Log(message) }

// Icon requests a host-rendered icon.
func Icon(name string, size int) (string, error) { return NewClient(nil).Icon(name, size) }

// Pages returns page operations on this client.
func (c Client) Pages() PageClient { return PageClient{c} }

// Settings returns the calling plugin's settings namespace.
func (c Client) Settings() Values { return Values{c, "plugin.settings"} }

// Storage returns the calling plugin's data namespace.
func (c Client) Storage() Values { return Values{c, "plugin.storage"} }

// Attachments returns attachment operations on this client.
func (c Client) Attachments() AttachmentClient { return AttachmentClient{c} }

// Log records a host log message.
func (c Client) Log(message string) error {
	return c.call("log", LogMessage{Message: message}, nil)
}

// Icon renders an icon through the public host capability.
func (c Client) Icon(name string, size int) (string, error) {
	var result string
	err := c.call("icons.render", IconRequest{Name: name, Size: size}, &result)
	return result, err
}

// PageClient provides permission-checked page operations.
type PageClient struct{ client Client }

// Get returns authorized metadata for a canonical page slug.
func (p PageClient) Get(slug string) (Page, error) {
	var result Page
	err := p.client.call("pages.get", PageRef{Slug: slug}, &result)
	return result, err
}

// Search returns authorized pages matching a bounded query.
func (p PageClient) Search(query PageQuery) ([]Page, error) {
	var result []Page
	err := p.client.call("pages.search", query, &result)
	return result, err
}

// Content reads authorized Markdown source.
func (p PageClient) Content(slug string) (PageContent, error) {
	var result PageContent
	err := p.client.call("pages.content", PageRef{Slug: slug}, &result)
	return result, err
}

// Navigation returns the prepared navigation tree.
func (p PageClient) Navigation() ([]NavigationNode, error) {
	var result []NavigationNode
	err := p.client.call("pages.navigation", nil, &result)
	return result, err
}

// Values is the calling plugin's host-managed key/value namespace.
type Values struct {
	client Client
	method string
}

// Get reads a key, preserving the distinction between absent and empty values.
func (v Values) Get(key string) (StoredValue, error) {
	var result StoredValue
	err := v.client.call(v.method+".read", StorageValue{Key: key}, &result)
	return result, err
}

// Set writes opaque bytes. Kumbuka enforces permissions and namespace quotas.
func (v Values) Set(key string, value []byte) error {
	return v.client.call(v.method+".write", StorageValue{Key: key, Value: value}, nil)
}

// AttachmentClient reads attachments authorized by the current request scope.
type AttachmentClient struct{ client Client }

// Read requests a bounded attachment byte range.
func (a AttachmentClient) Read(request AttachmentRead) (Attachment, error) {
	var result Attachment
	err := a.client.call("attachments.read", request, &result)
	return result, err
}
