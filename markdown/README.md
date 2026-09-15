# Markdown helpers

`markdown` contains small parsing helpers shared by Kumbuka plugin implementations.

This directory is not a Kumbuka plugin. It has no `plugin.yaml`, is not packaged as a `.kumbukaplugin`, and does not appear in the plugin administration UI.

## Fenced code blocks

`fences.go` handles Markdown fenced code-block boundaries. Plugins use these helpers when a feature needs to recognize code blocks without interpreting their contents.

The package exposes:

- `Fence` to recognize an opening backtick or tilde fence and return its complete marker;
- `Closes` to determine whether a line closes a previously opened fence;
- `AppendFence` to copy a complete fenced block without interpreting its body.

Keep this package focused on reusable Markdown helpers. Plugin-specific parsing belongs in the plugin that owns the syntax.
