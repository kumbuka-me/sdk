package sdk

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHTTPAndResourceCapabilities verifies typed generic host capability routing.
func TestHTTPAndResourceCapabilities(t *testing.T) {
	client := NewClient(func(method string, params, result any) error {
		switch method {
		case "plugin.resources.get":
			request := params.(PluginResourceRequest)
			require.Equal(t, "sources", request.Resource, "unexpected resource request: %+v", request)
			require.Equal(t, "engineering", request.Key, "unexpected resource request: %+v", request)
			*result.(*PluginResourceRecord) = PluginResourceRecord{Key: request.Key, Values: map[string]string{"token": "secret"}}
		case "http.do":
			request := params.(HTTPRequest)
			require.Equal(t, "GET", request.Method, "unexpected HTTP request: %+v", request)
			require.Equal(t, "https://example.test/file", request.URL, "unexpected HTTP request: %+v", request)
			require.True(t, request.InsecureSkipVerify, "unexpected HTTP request: %+v", request)
			*result.(*HTTPResponse) = HTTPResponse{StatusCode: 200, Body: []byte("ok")}
		default:
			require.FailNowf(t, "unexpected call", "unexpected method %q", method)
		}
		return nil
	})

	record, err := client.Resources().Get("sources", "engineering")
	require.NoError(t, err, "resource result: %+v %v", record, err)
	require.Equal(t, "secret", record.Values["token"], "resource result: %+v %v", record, err)

	response, err := client.HTTP().Do(HTTPRequest{Method: "GET", URL: "https://example.test/file", InsecureSkipVerify: true})
	require.NoError(t, err, "HTTP result: %+v %v", response, err)
	require.Equal(t, 200, response.StatusCode, "HTTP result: %+v %v", response, err)
	require.Equal(t, "ok", string(response.Body), "HTTP result: %+v %v", response, err)

	permission, ok := PermissionFor("http.do")
	require.True(t, ok, "unexpected HTTP permission: %q %v", permission, ok)
	require.Equal(t, "network:http", permission, "unexpected HTTP permission: %q %v", permission, ok)
	permission, ok = PermissionFor("plugin.resources.get")
	require.True(t, ok, "unexpected resource permission: %q %v", permission, ok)
	require.Equal(t, "settings:read", permission, "unexpected resource permission: %q %v", permission, ok)
	for _, permission := range []string{"network:http", "network:private", "network:insecure-tls"} {
		require.True(t, ValidPermission(permission), "missing permission %q", permission)
	}
}
