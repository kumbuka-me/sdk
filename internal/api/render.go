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
	// Features contains request-scoped feature flags.
	Features map[string]bool `json:"features,omitempty"`
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
	// Error contains a guest-visible rendering error when the operation failed.
	Error string `json:"error,omitempty"`
}

// RenderPart is either literal intermediate output or a recursive Markdown
// fragment. Text and Markdown must not both be set. Empty text is valid.
type RenderPart struct {
	// Text contains literal intermediate output.
	Text string `json:"text,omitempty"`
	// Markdown contains a recursive Markdown fragment for host-side rendering.
	Markdown *string `json:"markdown,omitempty"`
}
