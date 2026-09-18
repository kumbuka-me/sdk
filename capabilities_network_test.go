package sdk

import "testing"

// TestHTTPAndResourceCapabilities verifies typed generic host capability routing.
func TestHTTPAndResourceCapabilities(t *testing.T) {
	client := NewClient(func(method string, params, result any) error {
		switch method {
		case "plugin.resources.get":
			request := params.(PluginResourceRequest)
			if request.Resource != "sources" || request.Key != "engineering" {
				t.Fatalf("unexpected resource request: %+v", request)
			}
			*result.(*PluginResourceRecord) = PluginResourceRecord{Key: request.Key, Values: map[string]string{"token": "secret"}}
		case "http.do":
			request := params.(HTTPRequest)
			if request.Method != "GET" || request.URL != "https://example.test/file" || !request.InsecureSkipVerify {
				t.Fatalf("unexpected HTTP request: %+v", request)
			}
			*result.(*HTTPResponse) = HTTPResponse{StatusCode: 200, Body: []byte("ok")}
		default:
			t.Fatalf("unexpected method %q", method)
		}
		return nil
	})

	record, err := client.Resources().Get("sources", "engineering")
	if err != nil || record.Values["token"] != "secret" {
		t.Fatalf("resource result: %+v %v", record, err)
	}

	response, err := client.HTTP().Do(HTTPRequest{Method: "GET", URL: "https://example.test/file", InsecureSkipVerify: true})
	if err != nil || response.StatusCode != 200 || string(response.Body) != "ok" {
		t.Fatalf("HTTP result: %+v %v", response, err)
	}

	if permission, ok := PermissionFor("http.do"); !ok || permission != "network:http" {
		t.Fatalf("unexpected HTTP permission: %q %v", permission, ok)
	}
	if permission, ok := PermissionFor("plugin.resources.get"); !ok || permission != "settings:read" {
		t.Fatalf("unexpected resource permission: %q %v", permission, ok)
	}
	for _, permission := range []string{"network:http", "network:private", "network:insecure-tls"} {
		if !ValidPermission(permission) {
			t.Fatalf("missing permission %q", permission)
		}
	}
}
