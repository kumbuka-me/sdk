# Kumbuka render ABI v1

This is the versioned Kumbuka plugin wire contract. Go authors should use the root `sdk` package. The wire implementation lives under `internal/api`; its public types and helpers are exposed through the root SDK package. Plugins may be written in
any language that can produce a WASI Preview 1 reactor with these exports:

| Export                | Parameters                      | Result                         |
| --------------------- | ------------------------------- | ------------------------------ |
| `_initialize`         | none                            | none                           |
| `kumbuka_api_version` | none                            | `i32`, currently `1`           |
| `kumbuka_alloc`       | `i32` byte length               | `i32` offset into guest memory |
| `kumbuka_transform`   | `i32` offset, `i32` byte length | `i64` packed response          |
| `memory`              | —                               | linear memory                  |

Kumbuka calls `_initialize` before checking the ABI version. For each request it asks
the guest to allocate a buffer, writes a JSON `RenderRequest`, and invokes
`kumbuka_transform`. The returned `i64` packs the response byte length into its high
32 bits and its guest-memory offset into the low 32 bits. Kumbuka validates both
before decoding JSON. The response buffer must remain valid until the next
invocation. The host serializes calls to each reactor.

A renderer module receives its manifest module ID, stage (`preprocess` or
`postprocess`), source, and presentation feature flags. A `RenderResult` contains
an error or ordered fragments. A fragment is literal intermediate text, or Markdown
for the host to render recursively. Only preprocessors may return Markdown
fragments. The WASM call completes before the host renders those fragments; nested
rendering therefore does not re-enter an executing Go guest.

All resulting HTML goes through Kumbuka's central sanitizer. There is no trusted HTML
result type. The guest cannot change sanitizer policy or receive request macro
callbacks, filesystem handles, database clients, or Kumbuka Go pointers. Unknown JSON
fields, trailing JSON, invalid memory ranges, and excessive output are errors.

The bundled example in `plugins/callouts` is an independent Go module. Its
`main.go` owns callout parsing and registers a handler with the root `sdk` package. The SDK
owns all guest exports, buffers and transport. Standard Go builds it with:

```sh
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o plugin.wasm .
```

Go's reactor and export behavior is documented in
[Extensible Wasm Applications with Go](https://go.dev/blog/wasmexport). No TinyGo,
C compiler, external WASM process, or native plugin loading is required.

## Macros

A manifest can declare a macro using `type: macro`, an `id`, and a `name`.
The optional `capability` names an operation required in the current render
scope; without that operation the invocation stays literal Markdown.

Kumbuka protects CommonMark code blocks, then sends candidate lines to the guest
with stage `parse`. Return `matched` and a JSON `invocation` value. Kumbuka later
sends that value with stage `macro`; return ordinary text fragments containing
HTML. Parsing and rendering both execute in WASM. Neither stage may return
recursive Markdown fragments. Macro HTML shares the central sanitizer.

## Capabilities

Guests may import `kumbuka_v1.call(i32, i32, i32, i32) -> i32`. The arguments are
request offset, request length, response offset, and response capacity. Both
buffers belong to guest memory. The response capacity must be between 256 bytes
and the configured wire limit (4 MiB by default). The result is the number of
response bytes, or zero for an invalid buffer. The host never reenters a guest
allocator during a host call. `sdk` capability helpers hides this
transport for Go guests.

Requests are JSON `CapabilityRequest` objects with `method` and `params`.
Responses contain `value` or `error`. Unknown methods and top-level fields are
rejected. Plugin identity cannot be supplied in a request: the host obtains it
from the currently executing instance. Host calls during initialization are
denied. There are at most 512 host calls per invocation, sharing its deadline.

| Operation               | Manifest permission | Parameters / result                           |
| ----------------------- | ------------------- | --------------------------------------------- |
| `pages.get`             | `pages:read`        | `PageRef` / public `Page` metadata            |
| `pages.content`         | `pages:content`     | `PageRef` / authorized `PageContent` Markdown |
| `pages.search`          | `pages:read`        | `PageQuery` (limit 1–100) / `[]Page`          |
| `pages.navigation`      | `pages:read`        | none / render-scoped `[]NavigationNode`       |
| `attachments.read`      | `attachments:read`  | `AttachmentRead` / bounded `Attachment` range |
| `plugin.settings.read`  | `settings:read`     | `StorageValue` key / `StoredValue`            |
| `plugin.settings.write` | `settings:write`    | `StorageValue` / null                         |
| `plugin.storage.read`   | `storage:read`      | `StorageValue` key / `StoredValue`            |
| `plugin.storage.write`  | `storage:write`     | `StorageValue` / null                         |
| `icons.render`          | none                | `IconRequest` / intermediate SVG string       |
| `log`                   | none                | `LogMessage` (up to 4 KiB) / null             |

A permission must be both declared in the package and explicitly granted by
Kumbuka's runtime policy. Missing grants reject loading; an undeclared host call
is denied even when Kumbuka allows that capability to other plugins. Bundled and
installed packages use exactly the same policy. The low-level runtime grants
nothing by default. HTTP rendering explicitly grants page reads and namespaced
settings/storage access. The default/static renderer grants only page reads.

Page operations use the current viewer's authorized catalog, never an
unrestricted database. Navigation URLs are prepared for the rendering target.
Public share capabilities are restricted to the shared page. Static builds have
navigation but no catalog/search or persistent storage. Attachment reads require
an explicitly supplied, authorized range reader; no ambient media access is
bound to current render scopes. Missing context capabilities return an error.
General Kumbuka settings, credentials, users, SQL, networking, and process APIs are
not exposed.

Settings and data are separate namespaces owned by the executing plugin.
Keys are 1–256 bytes, values at most 64 KiB. PostgreSQL storage enforces a total
of 1,024 keys and 16 MiB per plugin across both namespaces, with transactional
quota checks. Reads distinguish an absent value from an empty value. Data
survives renderer/runtime restart. Disabling does not delete plugin data.

Runtime installation, enable/disable, version replacement, and removal are
implemented in Kumbuka's manager. These are trusted application operations, not guest
host calls.
Browser assets use the sandboxed browser-module contract. The Go SDK and
`kumbuka-plugin` CLI provide typed capabilities, registration, testing and packaging.

## Browser rendering

A `browser-module` manifest entry declares `javascript` and optional `css` paths
relative to package `assets/`, and requires `browser:render`. A renderer module
can emit a sanitized `div` with `data-kumbuka-plugin="<plugin-id>"` and
`data-kumbuka-module="<module-id>"`, containing a direct `pre` child as fallback.

Browser JavaScript is a classic script that sets
`globalThis.kumbukaPlugin = { async render(root, { source, theme }) { ... } }`.
It executes in its own opaque sandbox frame, receives the block text and
`light`/`dark` theme, and may modify only its frame DOM. It cannot return HTML
for insertion into Kumbuka's document. Resolve auxiliary package scripts relative
to `document.currentScript.src` captured when the entry script runs.

Browser modules do not expose a general host capability bridge.

## Standard syntax and settings declarations

`markdown-syntax` modules select a standard grammar with a `syntax` field:
`tables`, `strikethrough`, `task-list`, `definition-list`, `footnote`, or `linkify`.
Core constructs the parser components in the current render,
so other inline syntax, references and page variables keep their semantics.
These are public grammar identifiers, not privileged plugin IDs. Custom plugin
behavior continues to use WASM preprocessors/postprocessors; arbitrary native
extensions cannot be installed.

`settings` modules have an `id`, display `name`, optional `description`, and optional `requires` list of module IDs in the same package. Kumbuka renders these declarations as boolean controls on the plugin detail page and persists them in a core-owned namespace. Missing dependencies and cycles are rejected. Request feature keys are `<plugin-id>.<module-id>`; omitted flags default on. Package install/disable/upgrade still operates atomically per plugin.

`admin-resource` modules declare bounded host-rendered record schemas. Exactly one text field is the case-insensitive record key; Kumbuka owns the administrator form, validation, CSRF/authentication boundary, and namespaced persistence. `editor-completion` modules can expose those records to the Markdown editor and `editor-insert` modules add static insertion actions without loading arbitrary plugin code into Kumbuka's editor DOM.

`content-substitution` modules bind an `admin-resource` to an inline `{{prefix:name}}` substitution. Kumbuka replaces matches outside fenced code with opaque request-local tokens, runs all content preprocessors once, and restores values immediately before Markdown parsing. Inserted values are therefore never recursively reinterpreted as new substitutions. A substitution may expose generic page-inspector metadata and temporary export fields. `renderer-extension` modules may additionally use stage `content-preprocess` for executable WASM transformations that must run before the normal Markdown pipeline.

`code-highlighter` modules provide an exclusive fenced-code highlighter. Kumbuka sends the declared module the fenced block source and language with stage `highlight`; the plugin returns sanitized-later HTML and sets `matched` only for languages it supports. At most one code highlighter can be active. An optional `css` asset is filtered to safe code-presentation properties and scoped to that plugin's rendered wrapper, so another plugin can provide different token classes and styling without modifying Kumbuka.

`content-style` modules name a CSS asset that may affect rendered page typography only after Kumbuka filters selectors, properties, and values. `render-policy` modules request a named host rendering behavior from the public policy allowlist. These declarations are metadata contributions; they do not grant filesystem, DOM, or Kumbuka-internal access.

Every `.kumbukaplugin` also contains a bounded `README.md`. Kumbuka sanitizes and renders that documentation on the plugin administration detail page.

For HTML browser inputs, emit a `div` with `data-kumbuka-input="html"` and a direct
child `div data-kumbuka-fallback`. The core sanitizer processes the complete HTML
before the browser sees it. The module receives `context.html` for its isolated
frame. Core normalizes links and transfers bounded same-origin raster images as
data URLs. Unsupported or unavailable image resources leave the native fallback
visible. Plugin JavaScript never receives access to the host DOM.

Core forwards trusted link clicks only to HTTP(S) URLs already present in the
original fallback and only during browser user activation. Themes provide a
bounded set of validated color variables. A filtered stylesheet exposes only
scoped foreground/background/border colors to the original fallback; URLs,
imports, positioning and arbitrary selectors remain excluded. All other package
CSS runs only inside the frame.
