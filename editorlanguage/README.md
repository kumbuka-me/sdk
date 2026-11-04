# Editor code-language contributions

Kumbuka plugins can add entries to the fenced-code language picker without any
plugin-specific definitions in Kumbuka core. Include `assets/code-languages.json`
in the plugin package:

```json
{
  "version": 1,
  "languages": [{ "value": "runbook", "label": "Runbook" }]
}
```

`editorlanguage.Parse` is the SDK validator for this document. Identifiers are
lowercase ASCII language tokens and each plugin can contribute up to 512 entries (128 KiB total).
Optional `aliases` (up to 32 per language) provide alternate fence names and picker search terms.
Identifiers may contain lowercase letters, digits, underscores, dots, hyphens, plus signs and hashes.
Unknown keys, duplicate identifiers, unsupported versions and oversized files
are rejected. Only installed and enabled plugins contribute language choices.
When a plugin is disabled its choices disappear; existing Markdown fences are
preserved and remain ordinary code until a renderer handles them again.

The declaration does not register syntax highlighting or a renderer. Those are
independent plugin capabilities.
