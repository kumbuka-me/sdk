# Kumbuka plugin packages

This public package owns the single versioned manifest and bounded ZIP validation
contract used by Kumbuka and the Go developer CLI. `ParseManifest` strictly decodes
one manifest; `Read` validates a complete `.kumbukaplugin`. Returned packages expose
defensive copies and archives are never extracted by the runtime.

See [the SDK](../README.md) for plugin development and [the wire contract](../WIRE.md) for the low-level ABI.

Declarative-only packages do not need `plugin.wasm`. The archive requires a WASM
guest only when the manifest contains an executable renderer extension, macro, or
code highlighter.

## Source usage selectors

Executable modules can optionally declare bounded `usage` selectors so Kumbuka can
build a page render plan without invoking WASM merely to ask whether a plugin is
needed:

```yaml
modules:
  - type: renderer-extension
    id: callouts
    stage: preprocess
    usage:
      - contains: "!!! "

  - type: renderer-extension
    id: includes
    stage: content-preprocess
    usage:
      - substitution: include

  - type: renderer-extension
    id: diagrams
    stage: postprocess
    usage:
      - fence: mermaid
```

A rule sets exactly one of `contains`, `macro`, `substitution`, or `fence`;
`fence: "*"` matches any fenced-code block. Selectors are performance hints and
therefore must not produce false negatives. Modules without selectors remain
always active. `macro` modules are an exception: Kumbuka infers their source selector
from the manifest macro name.

Kumbuka derives plugin usage from Markdown on page writes and stores it as
rebuildable page metadata. Saved-page renders use the index when its renderer
fingerprint and source hash are current. Editor preview, static input, pages without a current usage index, and stale indexes
are analyzed transiently in memory. Markdown remains the source of truth.
