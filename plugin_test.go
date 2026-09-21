package sdk

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMacroDispatch(t *testing.T) {
	previous := handlers
	handlers = map[string]Handler{}
	t.Cleanup(func() { handlers = previous })
	RegisterMacro("sdk-test-macro", func(s string) (string, bool) { return s, s == "{{test}}" }, func(s string) (Result, error) { return Text(s), nil })
	request := Request{APIVersion: Version, Module: "sdk-test-macro", Stage: "parse", Source: "{{test}}"}
	parsed := Dispatch(request)
	require.True(t, parsed.Matched, "parse: %+v", parsed)
	require.Empty(t, parsed.Error, "parse: %+v", parsed)
	request.Stage = "macro"
	request.Invocation = parsed.Invocation
	result := Dispatch(request)
	require.Len(t, result.Parts, 1, "render: %+v", result)
	require.Equal(t, "{{test}}", result.Parts[0].Text, "render: %+v", result)
	request.Invocation = []byte("invalid")
	require.NotEmpty(t, Dispatch(request).Error, "invalid invocation accepted")
	request.APIVersion = 999
	require.NotEmpty(t, Dispatch(request).Error, "invalid version accepted")
	RegisterModule("sdk-test-panic", func(Request) Result { panic("private details") })
	result = Dispatch(Request{APIVersion: Version, Module: "sdk-test-panic"})
	require.Equal(t, "plugin handler panicked", result.Error, "panic escaped: %+v", result)
}

func TestAdminActionDispatch(t *testing.T) {
	previous := handlers
	handlers = map[string]Handler{}
	t.Cleanup(func() { handlers = previous })

	called := false
	RegisterAdminAction("refresh", func() error {
		called = true
		return nil
	})
	result := Dispatch(Request{APIVersion: Version, Module: "refresh", Stage: "admin-action"})
	require.Empty(t, result.Error, "admin action: %+v called=%t", result, called)
	require.True(t, called, "admin action: %+v called=%t", result, called)
	require.NotEmpty(t, Dispatch(Request{APIVersion: Version, Module: "refresh", Stage: "widget"}).Error, "admin action accepted wrong stage")
}

func TestWidgetDispatch(t *testing.T) {
	previous := handlers
	handlers = map[string]Handler{}
	t.Cleanup(func() { handlers = previous })

	RegisterWidget("summary", func(context WidgetContext) (Result, error) {
		require.Equal(t, "page.details", context.Surface, "unexpected widget context: %+v", context)
		require.NotNil(t, context.Page, "unexpected widget context: %+v", context)
		require.Equal(t, "guide", context.Page.Slug, "unexpected widget context: %+v", context)
		require.True(t, context.Features["example.enabled"], "unexpected widget context: %+v", context)
		return Result{Parts: []Part{{Text: "<h2>Summary</h2>"}}, Actions: []WidgetAction{{ID: "all", Kind: "dialog", Label: "All revisions", URL: "/revisions/guide"}}}, nil
	})

	result := Dispatch(Request{APIVersion: Version, Module: "summary", Stage: "widget", Features: map[string]bool{"example.enabled": true}, Widget: &WidgetContext{Surface: "page.details", Page: &Page{Slug: "guide"}}})
	require.Empty(t, result.Error, "widget: %+v", result)
	require.Len(t, result.Parts, 1, "widget: %+v", result)
	require.Len(t, result.Actions, 1, "widget: %+v", result)
	require.NotEmpty(t, Dispatch(Request{APIVersion: Version, Module: "summary", Stage: "widget"}).Error, "widget request without context accepted")
}

func TestTypedCapabilities(t *testing.T) {
	denied := errors.New("permission denied")
	client := NewClient(func(method string, params, result any) error {
		switch method {
		case "pages.get":
			require.Equal(t, "home", params.(PageRef).Slug, "wrong page reference")
			*result.(*Page) = Page{Slug: "home"}
		case "pages.search":
			require.Equal(t, 5, params.(PageQuery).Limit, "unbounded query")
			*result.(*[]Page) = []Page{{Slug: "home"}}
		case "pages.links":
			*result.(*PageLinks) = PageLinks{Backlinks: []Page{{Slug: "backlink"}}, Outgoing: []PageLink{{TargetSlug: "target", Exists: true}}}
		case "pages.revisions":
			require.Equal(t, 1, params.(RevisionQuery).Limit, "unexpected revision limit")
			*result.(*RevisionHistory) = RevisionHistory{Count: 2, Revisions: []Revision{{Number: 2}}}
		case "pages.recent":
			require.Equal(t, 3, params.(PageListQuery).Limit, "unexpected recent limit")
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
			require.FailNowf(t, "unexpected call", "unexpected method %s", method)
		}
		return nil
	})
	page, err := client.Pages().Get("home")
	require.NoError(t, err, "get: %+v %v", page, err)
	require.Equal(t, "home", page.Slug, "get: %+v %v", page, err)
	pages, err := client.Pages().Search(PageQuery{Limit: 5})
	require.NoError(t, err, "search: %+v %v", pages, err)
	require.Len(t, pages, 1, "search: %+v %v", pages, err)
	links, err := client.Pages().Links("home")
	require.NoError(t, err, "links: %+v %v", links, err)
	require.Len(t, links.Backlinks, 1, "links: %+v %v", links, err)
	require.Len(t, links.Outgoing, 1, "links: %+v %v", links, err)
	history, err := client.Pages().Revisions(RevisionQuery{Slug: "home", Limit: 1})
	require.NoError(t, err, "revisions: %+v %v", history, err)
	require.Equal(t, 2, history.Count, "revisions: %+v %v", history, err)
	require.Len(t, history.Revisions, 1, "revisions: %+v %v", history, err)
	recent, err := client.Pages().Recent(3)
	require.NoError(t, err, "recent: %+v %v", recent, err)
	require.Len(t, recent, 1, "recent: %+v %v", recent, err)
	require.Equal(t, "recent", recent[0].Slug, "recent: %+v %v", recent, err)
	viewed, err := client.Pages().RecentViewed(4)
	require.NoError(t, err, "recent viewed: %+v %v", viewed, err)
	require.Len(t, viewed, 1, "recent viewed: %+v %v", viewed, err)
	require.Equal(t, "viewed", viewed[0].Slug, "recent viewed: %+v %v", viewed, err)
	favorites, err := client.Pages().Favorites(5)
	require.NoError(t, err, "favorites: %+v %v", favorites, err)
	require.Len(t, favorites, 1, "favorites: %+v %v", favorites, err)
	require.Equal(t, "favorite", favorites[0].Slug, "favorites: %+v %v", favorites, err)
	popular, err := client.Pages().Popular(6)
	require.NoError(t, err, "popular: %+v %v", popular, err)
	require.Len(t, popular, 1, "popular: %+v %v", popular, err)
	require.Equal(t, "popular", popular[0].Slug, "popular: %+v %v", popular, err)
	edits, err := client.Pages().RecentEdited(7)
	require.NoError(t, err, "recent edits: %+v %v", edits, err)
	require.Len(t, edits, 1, "recent edits: %+v %v", edits, err)
	require.Equal(t, "Changed", edits[0].RevisionMessage, "recent edits: %+v %v", edits, err)
	drafts, err := client.Drafts().List(6)
	require.NoError(t, err, "drafts: %+v %v", drafts, err)
	require.Len(t, drafts, 1, "drafts: %+v %v", drafts, err)
	require.Equal(t, "page:1", drafts[0].Key, "drafts: %+v %v", drafts, err)
	value, err := client.Settings().Get("empty")
	require.NoError(t, err, "empty: %+v %v", value, err)
	require.True(t, value.Found, "empty: %+v %v", value, err)
	value, err = client.Storage().Get("missing")
	require.NoError(t, err, "absent: %+v %v", value, err)
	require.False(t, value.Found, "absent: %+v %v", value, err)
	err = client.Storage().Set("key", []byte("value"))
	require.ErrorIs(t, err, denied, "permission error lost: %v", err)
}

func TestInvalidResultUsesErrorEnvelope(t *testing.T) {
	for _, result := range []Result{
		{Invocation: json.RawMessage("invalid JSON")},
		Text(strings.Repeat("x", 4<<20)),
	} {
		var envelope Result
		err := json.Unmarshal(encodeResult(result), &envelope)
		require.NoError(t, err, "invalid output escaped: %+v %v", envelope, err)
		require.NotEmpty(t, envelope.Error, "invalid output escaped: %+v %v", envelope, err)
	}
}

func TestDashboardCapabilityPermissions(t *testing.T) {
	for _, method := range []string{"pages.recent", "pages.popular"} {
		permission, ok := PermissionFor(method)
		require.True(t, ok, "%s: %q %t", method, permission, ok)
		require.Equal(t, "pages:read", permission, "%s: %q %t", method, permission, ok)
	}
	for _, method := range []string{"pages.recent-viewed", "pages.favorites", "pages.recent-edits"} {
		permission, ok := PermissionFor(method)
		require.True(t, ok, "%s: %q %t", method, permission, ok)
		require.Equal(t, "activity:read", permission, "%s: %q %t", method, permission, ok)
	}
	require.True(t, ValidPermission("activity:read"), "activity:read must be a valid permission")
	permission, ok := PermissionFor("drafts.list")
	require.True(t, ok, "drafts permission: %q %t", permission, ok)
	require.Equal(t, "drafts:read", permission, "drafts permission: %q %t", permission, ok)
	require.True(t, ValidPermission("drafts:read"), "drafts permission: %q %t", permission, ok)
}

func TestRegisterWidgetRejectsNilRenderer(t *testing.T) {
	previous := handlers
	handlers = map[string]Handler{}
	t.Cleanup(func() { handlers = previous })
	require.Panics(t, func() {
		RegisterWidget("nil-widget", nil)
	}, "nil widget renderer accepted")
}

func TestRegisterMacroRejectsNilCallbacks(t *testing.T) {
	previous := handlers
	handlers = map[string]Handler{}
	t.Cleanup(func() { handlers = previous })
	require.Panics(t, func() {
		RegisterMacro[string]("nil-macro", nil, func(string) (Result, error) { return Result{}, nil })
	}, "nil macro callback accepted")
}

func TestExporterDispatch(t *testing.T) {
	previous := handlers
	handlers = map[string]Handler{}
	t.Cleanup(func() { handlers = previous })

	RegisterExporter("text", func(context ExportContext) (ExportFile, error) {
		require.Equal(t, "guide", context.Page.Slug, "unexpected export context: %+v", context)
		require.Equal(t, "# Guide", context.Source, "unexpected export context: %+v", context)
		require.True(t, context.Features["example.enabled"], "unexpected export context: %+v", context)
		return ExportFile{Filename: "guide.txt", MediaType: "text/plain", Data: []byte("Guide")}, nil
	})
	result := Dispatch(Request{
		APIVersion: Version,
		Module:     "text",
		Stage:      "export",
		Features:   map[string]bool{"example.enabled": true},
		Export:     &ExportContext{Page: Page{Slug: "guide"}, Source: "# Guide"},
	})
	require.Empty(t, result.Error, "export: %+v", result)
	require.NotNil(t, result.File, "export: %+v", result)
	require.Equal(t, "guide.txt", result.File.Filename, "export: %+v", result)
	require.NotEmpty(t, Dispatch(Request{APIVersion: Version, Module: "text", Stage: "export"}).Error, "export request without context accepted")
}

func TestWidgetCommandDispatch(t *testing.T) {
	previous := handlers
	handlers = map[string]Handler{}
	t.Cleanup(func() { handlers = previous })

	RegisterWidgetWithCommands(
		"summary",
		func(WidgetContext) (Result, error) { return Text("summary"), nil },
		func(context WidgetCommandContext) (WidgetCommandResult, error) {
			require.Equal(t, "page.details", context.Surface, "unexpected widget command context: %+v", context)
			require.NotNil(t, context.Page, "unexpected widget command context: %+v", context)
			require.Equal(t, "guide", context.Page.Slug, "unexpected widget command context: %+v", context)
			require.Equal(t, "refresh", context.Action, "unexpected widget command context: %+v", context)
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
	require.Empty(t, result.Error, "widget command: %+v", result)
	require.NotNil(t, result.WidgetCommand, "widget command: %+v", result)
	require.Equal(t, "/pages/guide", result.WidgetCommand.Redirect, "widget command: %+v", result)
}
