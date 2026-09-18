# Kumbuka SDK

Go SDK and development tooling for Kumbuka plugins.

## Install a plugin

1. Download the plugin's `.kumbukaplugin` file.
2. Open **Administration → Plugins** in Kumbuka.
3. Upload the package and select **Install and enable**.

For plugin development, SDK usage, manifests, capabilities, package formats, and the wire protocol, see the [Kumbuka documentation](https://kumbuka.me/plugins/).

## License

Licensed under the [Apache License 2.0](./LICENSE).

## Approved external files

`ExternalFiles().Read(ExternalFileRequest{Source: "engineering", Path: "src/main.go", Start: 10, End: 20})`
requests plain UTF-8 content from a host-approved repository. Omit both bounds for a whole file;
use equal bounds for one line. A plugin must declare `external:read` in its manifest.

This capability requires the accompanying Kumbuka host implementation. The host owns repository
approval, authentication, secrets, network policy, and limits. The SDK accepts no arbitrary URL,
headers, credentials, repository override, or caller identity. Treat the returned content as
untrusted text and escape it before rendering. Never return it as a Markdown fragment.
