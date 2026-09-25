// Package api contains the versioned wire types shared by Kumbuka and WASM
// plugins. It intentionally has no dependencies on Kumbuka's internal packages.
package api

import "encoding/json"

const Version = 1

// RenderRequest invokes one declared renderer module. Features contains only
// presentation preferences, never credentials or implicit host capabilities.
type RenderRequest struct {
	// APIVersion identifies the Kumbuka plugin wire protocol version.
	APIVersion int `json:"api_version"`
	// Module identifies the manifest module being invoked.
	Module string `json:"module"`
	// Stage selects the renderer or macro operation.
	Stage string `json:"stage"`
	// Source carries the current Markdown, code, or intermediate HTML input.
	Source string `json:"source"`
	// Language carries the fenced-code language for code-highlighter modules.
	Language string `json:"language,omitempty"`
	// Invocation carries serialized macro arguments between parse and render stages.
	Invocation json.RawMessage `json:"invocation,omitempty"`
	// Features contains request-scoped plugin settings and semantic render-policy markers.
	Features map[string]bool `json:"features,omitempty"`
	// Widget contains the current host surface and page for widget invocations.
	Widget *WidgetContext `json:"widget,omitempty"`
	// WidgetCommand contains one host-validated command invocation for a widget.
	WidgetCommand *WidgetCommandContext `json:"widget_command,omitempty"`
	// Export contains the current page and source for exporter invocations.
	Export *ExportContext `json:"export,omitempty"`
	// ContentChange contains one committed page source change for mutation hooks.
	ContentChange *ContentChangeContext `json:"content_change,omitempty"`
}

// RenderResult returns intermediate output or an error. Markdown fragments are
// rendered by the host after the WASM call finishes; this avoids reentrant guest
// calls for nested blocks. Postprocessors may return text fragments only.
type RenderResult struct {
	// Matched reports whether a macro parser recognized the candidate source.
	Matched bool `json:"matched,omitempty"`
	// Invocation carries serialized macro arguments between parse and render stages.
	Invocation json.RawMessage `json:"invocation,omitempty"`
	// Parts contains ordered intermediate output fragments.
	Parts []RenderPart `json:"parts,omitempty"`
	// Actions contains safe host-rendered controls contributed by a widget.
	Actions []WidgetAction `json:"actions,omitempty"`
	// File contains the bounded download produced by an exporter invocation.
	File *ExportFile `json:"file,omitempty"`
	// WidgetCommand contains the host navigation result of a widget command.
	WidgetCommand *WidgetCommandResult `json:"widget_command,omitempty"`
	// Error contains a guest-visible rendering error when the operation failed.
	Error string `json:"error,omitempty"`
}

// ExportContext describes one authorized page export invocation.
type ExportContext struct {
	// Page contains public metadata for the page being exported.
	Page Page `json:"page"`
	// Source contains the stored Markdown source for the page.
	Source string `json:"source"`
	// Features contains request-scoped plugin settings and semantic policy markers.
	// It is populated by RegisterExporter from the request envelope rather than encoded twice.
	Features map[string]bool `json:"-"`
}

// ContentChangeContext describes canonical Markdown before and after one committed page save.
type ContentChangeContext struct {
	// Page contains public metadata for the committed page version.
	Page Page `json:"page"`
	// PreviousSource is the canonical Markdown stored before the mutation.
	PreviousSource string `json:"previous_source"`
	// Source is the canonical Markdown stored by the mutation.
	Source string `json:"source"`
	// Features contains request-scoped plugin settings and semantic policy markers.
	// It is populated by RegisterContentChange from the request envelope rather than encoded twice.
	Features map[string]bool `json:"-"`
}

// ExportFile is one bounded file returned by an exporter plugin.
type ExportFile struct {
	// Filename is the suggested base filename for the browser download.
	Filename string `json:"filename"`
	// MediaType is the file's IANA media type.
	MediaType string `json:"media_type"`
	// Data contains the complete exported file bytes.
	Data []byte `json:"data"`
}

// RenderPart is either literal intermediate output or a recursive Markdown
// fragment. Text and Markdown must not both be set. Empty text is valid.
type RenderPart struct {
	// Text contains literal intermediate output.
	Text string `json:"text,omitempty"`
	// Markdown contains a recursive Markdown fragment for host-side rendering.
	Markdown *string `json:"markdown,omitempty"`
}

// WidgetContext describes the host surface and current page supplied to a widget.
type WidgetContext struct {
	// Surface identifies where the widget is rendered.
	Surface string `json:"surface"`
	// Page contains the current page on page-scoped surfaces.
	Page *Page `json:"page,omitempty"`
	// Features contains request-scoped plugin settings and semantic policy markers.
	// It is populated by RegisterWidget from the request envelope rather than encoded twice.
	Features map[string]bool `json:"-"`
}

// WidgetAction asks the host to render one safe widget control.
type WidgetAction struct {
	// ID identifies the action within its widget.
	ID string `json:"id"`
	// Kind selects link, dialog, or host-mediated command behavior.
	Kind string `json:"kind"`
	// Label is the visible action text.
	Label string `json:"label"`
	// URL is a local application path handled by the host.
	URL string `json:"url"`
	// Icon is an optional host icon name.
	Icon string `json:"icon,omitempty"`
	// Confirm is optional confirmation text shown before a command is submitted.
	Confirm string `json:"confirm,omitempty"`
}

// WidgetCommandContext describes one host-mediated command invocation.
type WidgetCommandContext struct {
	// Surface identifies the widget placement that emitted the command.
	Surface string `json:"surface"`
	// Page contains the authorized current page on page-scoped surfaces.
	Page *Page `json:"page,omitempty"`
	// Action identifies the command selected by the user.
	Action string `json:"action"`
	// Features contains request-scoped plugin settings and semantic policy markers.
	// It is populated by RegisterWidgetWithCommands from the request envelope rather than encoded twice.
	Features map[string]bool `json:"-"`
}

// WidgetCommandResult controls safe host navigation after a widget command.
type WidgetCommandResult struct {
	// Redirect is an optional local path selected by the plugin after the command succeeds.
	Redirect string `json:"redirect,omitempty"`
}

// ValidWidgetSurface reports whether surface is a public widget placement.
func ValidWidgetSurface(surface string) bool {
	switch surface {
	case "home", "page.details", "page.after-content", "page.aside", "sidebar":
		return true
	default:
		return false
	}
}

// ValidWidgetWidth reports whether width is a supported host layout hint.
func ValidWidgetWidth(width string) bool {
	switch width {
	case "", "normal", "wide":
		return true
	default:
		return false
	}
}
