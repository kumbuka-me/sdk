package sdk

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestMacroDispatch(t *testing.T) {
	previous := handlers
	handlers = map[string]Handler{}
	t.Cleanup(func() { handlers = previous })
	RegisterMacro("sdk-test-macro", func(s string) (string, bool) { return s, s == "{{test}}" }, func(s string) (Result, error) { return Text(s), nil })
	request := Request{APIVersion: Version, Module: "sdk-test-macro", Stage: "parse", Source: "{{test}}"}
	parsed := Dispatch(request)
	if !parsed.Matched || parsed.Error != "" {
		t.Fatalf("parse: %+v", parsed)
	}
	request.Stage = "macro"
	request.Invocation = parsed.Invocation
	result := Dispatch(request)
	if len(result.Parts) != 1 || result.Parts[0].Text != "{{test}}" {
		t.Fatalf("render: %+v", result)
	}
	request.Invocation = []byte("invalid")
	if Dispatch(request).Error == "" {
		t.Fatal("invalid invocation accepted")
	}
	request.APIVersion = 999
	if Dispatch(request).Error == "" {
		t.Fatal("invalid version accepted")
	}
	RegisterModule("sdk-test-panic", func(Request) Result { panic("private details") })
	result = Dispatch(Request{APIVersion: Version, Module: "sdk-test-panic"})
	if result.Error != "plugin handler panicked" {
		t.Fatalf("panic escaped: %+v", result)
	}
}

func TestTypedCapabilities(t *testing.T) {
	denied := errors.New("permission denied")
	client := NewClient(func(method string, params, result any) error {
		switch method {
		case "pages.get":
			if params.(PageRef).Slug != "home" {
				t.Fatal("wrong page reference")
			}
			*result.(*Page) = Page{Slug: "home"}
		case "pages.search":
			if params.(PageQuery).Limit != 5 {
				t.Fatal("unbounded query")
			}
			*result.(*[]Page) = []Page{{Slug: "home"}}
		case "plugin.settings.read":
			*result.(*StoredValue) = StoredValue{Found: true, Value: []byte{}}
		case "plugin.storage.read":
			*result.(*StoredValue) = StoredValue{}
		case "plugin.storage.write":
			return denied
		default:
			t.Fatalf("unexpected method %s", method)
		}
		return nil
	})
	page, err := client.Pages().Get("home")
	if err != nil || page.Slug != "home" {
		t.Fatalf("get: %+v %v", page, err)
	}
	pages, err := client.Pages().Search(PageQuery{Limit: 5})
	if err != nil || len(pages) != 1 {
		t.Fatalf("search: %+v %v", pages, err)
	}
	value, err := client.Settings().Get("empty")
	if err != nil || !value.Found {
		t.Fatalf("empty: %+v %v", value, err)
	}
	value, err = client.Storage().Get("missing")
	if err != nil || value.Found {
		t.Fatalf("absent: %+v %v", value, err)
	}
	if err := client.Storage().Set("key", []byte("value")); !errors.Is(err, denied) {
		t.Fatalf("permission error lost: %v", err)
	}
}

func TestInvalidResultUsesErrorEnvelope(t *testing.T) {
	for _, result := range []Result{
		{Invocation: json.RawMessage("invalid JSON")},
		Text(strings.Repeat("x", 4<<20)),
	} {
		var envelope Result
		if err := json.Unmarshal(encodeResult(result), &envelope); err != nil || envelope.Error == "" {
			t.Fatalf("invalid output escaped: %+v %v", envelope, err)
		}
	}
}
