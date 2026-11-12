# Kumbuka SDK

Go SDK and development tooling for Kumbuka plugins.

## Install a plugin

1. Download the plugin's `.kumbukaplugin` file.
2. Open **Administration → Plugins** in Kumbuka.
3. Upload the package and select **Install and enable**.

For plugin development, SDK usage, manifests, capabilities, package formats, and the wire protocol, see the [Kumbuka documentation](https://kumbuka.me/development/plugins/).

## Plugin settings and resources

Plugins can declare boolean feature-toggle `settings` modules, typed singleton `settings` groups, and repeatable `admin-resource` records. Typed fields support text, textarea, URL, secret, boolean, select, and color controls. Repeatable admin resources can also use bounded `list` fields with typed columns for structured row editors. Kumbuka renders and validates the administrator UI, namespaces persisted values by plugin ID, and encrypts `secret` fields at rest.

Typed singleton settings use `fields` on a `settings` module. The owning plugin reads them through `Settings().Get("<module>.<field>")`; the host returns manifest defaults when no administrator value has been saved yet. Repeatable records continue to use `Resources()`. A plugin needs `settings:read` to read either form.

## Outbound HTTP

Plugins that declare `network:http` can make bounded HTTP(S) requests with `HTTP().Do`. Kumbuka performs the network I/O and enforces host limits, redirect policy, destination validation, and response bounds. A plugin must additionally declare `network:private` before requesting an administrator-configured private IP and `network:insecure-tls` before disabling origin certificate verification.

The HTTP capability is intentionally generic: provider URLs, credentials, request headers, and protocol behavior belong to the plugin rather than Kumbuka core.

## License

Licensed under the [Apache License 2.0](./LICENSE).

## Optional static browser exports

An existing `browser-module` may advertise build-time image export by bundling
`assets/static-exports.json` (an ordinary asset, so older Kumbuka servers remain
compatible):

```json
{"version":1,"modules":[{"id":"diagrams","media_type":"image/svg+xml"}]}
```

The module's existing JavaScript must expose
`globalThis.kumbukaPlugin.exportStatic(root, {theme})`. The host supplies a
sandboxed DOM element containing the rendered plugin block and invokes it for
`theme="light"` and `theme="dark"`. Return `null` to keep the normal browser
module, or `{content, alt, alignment}` with passive SVG markup and optional
metadata. The CLI validates the SVG, rejects external references and active
content, and writes content-addressed assets. No shell commands or plugin-owned
filesystem access are required. This is **optional** and does not change the
existing Kumbuka web-rendering contract.

The `staticexport` Go package defines and validates this wire contract, so a
future host-backed artifact cache can use the same renderer output.
