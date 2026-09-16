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

func TestWidgetDispatch(t *testing.T) {
	previous := handlers
	handlers = map[string]Handler{}
	t.Cleanup(func() { handlers = previous })

	RegisterWidget("summary", func(context WidgetContext) (Result, error) {
		if context.Surface != "page.details" || context.Page == nil || context.Page.Slug != "guide" || !context.Features["example.enabled"] {
			t.Fatalf("unexpected widget context: %+v", context)
		}
		return Result{Parts: []Part{{Text: "<h2>Summary</h2>"}}, Actions: []WidgetAction{{ID: "all", Kind: "dialog", Label: "All revisions", URL: "/revisions/guide"}}}, nil
	})

	result := Dispatch(Request{APIVersion: Version, Module: "summary", Stage: "widget", Features: map[string]bool{"example.enabled": true}, Widget: &WidgetContext{Surface: "page.details", Page: &Page{Slug: "guide"}}})
	if result.Error != "" || len(result.Parts) != 1 || len(result.Actions) != 1 {
		t.Fatalf("widget: %+v", result)
	}
	if Dispatch(Request{APIVersion: Version, Module: "summary", Stage: "widget"}).Error == "" {
		t.Fatal("widget request without context accepted")
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
		case "pages.links":
			*result.(*PageLinks) = PageLinks{Backlinks: []Page{{Slug: "backlink"}}, Outgoing: []PageLink{{TargetSlug: "target", Exists: true}}}
		case "pages.revisions":
			if params.(RevisionQuery).Limit != 1 {
				t.Fatal("unexpected revision limit")
			}
			*result.(*RevisionHistory) = RevisionHistory{Count: 2, Revisions: []Revision{{Number: 2}}}
		case "pages.recent":
			if params.(PageListQuery).Limit != 3 {
				t.Fatal("unexpected recent limit")
			}
			*result.(*[]Page) = []Page{{Slug: "recent"}}
		case "pages.recent-viewed":
			*result.(*[]Page) = []Page{{Slug: "viewed"}}
		case "pages.favorites":
			*result.(*[]Page) = []Page{{Slug: "favorite"}}
		case "pages.popular":
			*result.(*[]Page) = []Page{{Slug: "popular"}}
		case "pages.recent-edits":
			*result.(*[]RecentEdit) = []RecentEdit{{Page: Page{Slug: "edited"}, RevisionMessage: "Changed"}}
		case "drafts.list":
			*result.(*[]PageDraft) = []PageDraft{{Key: "page:1", PageID: 1, PageSlug: "draft", Title: "Draft"}}
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
	links, err := client.Pages().Links("home")
	if err != nil || len(links.Backlinks) != 1 || len(links.Outgoing) != 1 {
		t.Fatalf("links: %+v %v", links, err)
	}
	history, err := client.Pages().Revisions(RevisionQuery{Slug: "home", Limit: 1})
	if err != nil || history.Count != 2 || len(history.Revisions) != 1 {
		t.Fatalf("revisions: %+v %v", history, err)
	}
	recent, err := client.Pages().Recent(3)
	if err != nil || len(recent) != 1 || recent[0].Slug != "recent" {
		t.Fatalf("recent: %+v %v", recent, err)
	}
	viewed, err := client.Pages().RecentViewed(4)
	if err != nil || len(viewed) != 1 || viewed[0].Slug != "viewed" {
		t.Fatalf("recent viewed: %+v %v", viewed, err)
	}
	favorites, err := client.Pages().Favorites(5)
	if err != nil || len(favorites) != 1 || favorites[0].Slug != "favorite" {
		t.Fatalf("favorites: %+v %v", favorites, err)
	}
	popular, err := client.Pages().Popular(6)
	if err != nil || len(popular) != 1 || popular[0].Slug != "popular" {
		t.Fatalf("popular: %+v %v", popular, err)
	}
	edits, err := client.Pages().RecentEdited(7)
	if err != nil || len(edits) != 1 || edits[0].RevisionMessage != "Changed" {
		t.Fatalf("recent edits: %+v %v", edits, err)
	}
	drafts, err := client.Drafts().List(6)
	if err != nil || len(drafts) != 1 || drafts[0].Key != "page:1" {
		t.Fatalf("drafts: %+v %v", drafts, err)
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

func TestDashboardCapabilityPermissions(t *testing.T) {
	for _, method := range []string{"pages.recent", "pages.popular"} {
		permission, ok := PermissionFor(method)
		if !ok || permission != "pages:read" {
			t.Fatalf("%s: %q %t", method, permission, ok)
		}
	}
	for _, method := range []string{"pages.recent-viewed", "pages.favorites", "pages.recent-edits"} {
		permission, ok := PermissionFor(method)
		if !ok || permission != "activity:read" {
			t.Fatalf("%s: %q %t", method, permission, ok)
		}
	}
	if !ValidPermission("activity:read") {
		t.Fatal("activity:read must be a valid permission")
	}
	permission, ok := PermissionFor("drafts.list")
	if !ok || permission != "drafts:read" || !ValidPermission("drafts:read") {
		t.Fatalf("drafts permission: %q %t", permission, ok)
	}
}

func TestRegisterWidgetRejectsNilRenderer(t *testing.T) {
	previous := handlers
	handlers = map[string]Handler{}
	t.Cleanup(func() { handlers = previous })
	defer func() {
		if recover() == nil {
			t.Fatal("nil widget renderer accepted")
		}
	}()
	RegisterWidget("nil-widget", nil)
}

func TestRegisterMacroRejectsNilCallbacks(t *testing.T) {
	previous := handlers
	handlers = map[string]Handler{}
	t.Cleanup(func() { handlers = previous })
	defer func() {
		if recover() == nil {
			t.Fatal("nil macro callback accepted")
		}
	}()
	RegisterMacro[string]("nil-macro", nil, func(string) (Result, error) { return Result{}, nil })
}

func TestExporterDispatch(t *testing.T) {
	previous := handlers
	handlers = map[string]Handler{}
	t.Cleanup(func() { handlers = previous })

	RegisterExporter("text", func(context ExportContext) (ExportFile, error) {
		if context.Page.Slug != "guide" || context.Source != "# Guide" || !context.Features["example.enabled"] {
			t.Fatalf("unexpected export context: %+v", context)
		}
		return ExportFile{Filename: "guide.txt", MediaType: "text/plain", Data: []byte("Guide")}, nil
	})
	result := Dispatch(Request{
		APIVersion: Version,
		Module:     "text",
		Stage:      "export",
		Features:   map[string]bool{"example.enabled": true},
		Export:     &ExportContext{Page: Page{Slug: "guide"}, Source: "# Guide"},
	})
	if result.Error != "" || result.File == nil || result.File.Filename != "guide.txt" {
		t.Fatalf("export: %+v", result)
	}
	if Dispatch(Request{APIVersion: Version, Module: "text", Stage: "export"}).Error == "" {
		t.Fatal("export request without context accepted")
	}
}

func TestWidgetCommandDispatch(t *testing.T) {
	previous := handlers
	handlers = map[string]Handler{}
	t.Cleanup(func() { handlers = previous })

	RegisterWidgetWithCommands(
		"summary",
		func(WidgetContext) (Result, error) { return Text("summary"), nil },
		func(context WidgetCommandContext) (WidgetCommandResult, error) {
			if context.Surface != "page.details" || context.Page == nil || context.Page.Slug != "guide" || context.Action != "refresh" {
				t.Fatalf("unexpected widget command context: %+v", context)
			}
			return WidgetCommandResult{Redirect: "/pages/guide"}, nil
		},
	)

	result := Dispatch(Request{
		APIVersion: Version,
		Module:     "summary",
		Stage:      "widget-command",
		WidgetCommand: &WidgetCommandContext{
			Surface: "page.details",
			Page:    &Page{Slug: "guide"},
			Action:  "refresh",
		},
	})
	if result.Error != "" || result.WidgetCommand == nil || result.WidgetCommand.Redirect != "/pages/guide" {
		t.Fatalf("widget command: %+v", result)
	}
}
