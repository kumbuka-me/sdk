# Kumbuka SDK

Go SDK and development tooling for Kumbuka plugins.

## Install a plugin

1. Download the plugin's `.kumbukaplugin` file.
2. Open **Administration → Plugins** in Kumbuka.
3. Upload the package and select **Install and enable**.

For plugin development, SDK usage, manifests, capabilities, package formats, and the wire protocol, see the [Kumbuka documentation](https://kumbuka.me/plugins/).

## Plugin settings and resources

Plugins can declare simple boolean `settings` modules and structured `admin-resource` records. Structured fields support text, textarea, URL, secret, boolean, and select controls. Kumbuka renders and validates the administrator UI, namespaces the persisted values by plugin ID, and encrypts `secret` fields at rest. A plugin with `settings:read` can read its own structured records through `Resources()`; secret fields are returned to that plugin after host-side decryption.

## Outbound HTTP

Plugins that declare `network:http` can make bounded HTTP(S) requests with `HTTP().Do`. Kumbuka performs the network I/O and enforces host limits, redirect policy, destination validation, and response bounds. A plugin must additionally declare `network:private` before requesting an administrator-configured private IP and `network:insecure-tls` before disabling origin certificate verification.

The HTTP capability is intentionally generic: provider URLs, credentials, request headers, and protocol behavior belong to the plugin rather than Kumbuka core.

## License

Licensed under the [Apache License 2.0](./LICENSE).
