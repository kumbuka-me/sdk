# Kumbuka SDK

Go SDK and development tooling for Kumbuka plugins.

## Install a plugin

1. Download the plugin's `.kumbukaplugin` file.
2. Open **Administration → Plugins** in Kumbuka.
3. Upload the package and select **Install and enable**.

For plugin development, SDK usage, manifests, capabilities, package formats, and the wire protocol, see the [Kumbuka documentation](https://kumbuka.me/plugins/).

## Plugin settings and resources

Plugins can declare boolean feature-toggle `settings` modules, typed singleton `settings` groups, and repeatable `admin-resource` records. Typed fields support text, textarea, URL, secret, boolean, select, and color controls. Repeatable admin resources can also use bounded `list` fields with typed columns for structured row editors. Kumbuka renders and validates the administrator UI, namespaces persisted values by plugin ID, and encrypts `secret` fields at rest.

Typed singleton settings use `fields` on a `settings` module. The owning plugin reads them through `Settings().Get("<module>.<field>")`; the host returns manifest defaults when no administrator value has been saved yet. Repeatable records continue to use `Resources()`. A plugin needs `settings:read` to read either form.

## Outbound HTTP

Plugins that declare `network:http` can make bounded HTTP(S) requests with `HTTP().Do`. Kumbuka performs the network I/O and enforces host limits, redirect policy, destination validation, and response bounds. A plugin must additionally declare `network:private` before requesting an administrator-configured private IP and `network:insecure-tls` before disabling origin certificate verification.

The HTTP capability is intentionally generic: provider URLs, credentials, request headers, and protocol behavior belong to the plugin rather than Kumbuka core.

## License

Licensed under the [Apache License 2.0](./LICENSE).
