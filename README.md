# Kumbuka SDK

Public Go SDK and plugin package contracts for Kumbuka.

## Packages

Only packages intended for other Go modules are public:

- `github.com/kumbuka-me/sdk` — the Go guest SDK and public API contract.
- `github.com/kumbuka-me/sdk/markdown` — small Markdown parsing helpers for plugins.
- `github.com/kumbuka-me/sdk/pluginpackage` — manifest and `.kumbukaplugin` archive validation used by the Kumbuka host.

Implementation details for the wire transport, package builder, and CLI live under `internal/`.

## Plugin CLI

Build the development command:

```sh
make build
```

The CLI uses TinyFlags subcommands:

```sh
bin/kumbuka-plugin init my-plugin
cd my-plugin
../bin/kumbuka-plugin test
../bin/kumbuka-plugin build
```

`init` accepts either `--sdk-version` or `--sdk-path`. A checkout-built CLI uses the current SDK checkout automatically when neither is supplied.

The build result is written to `dist/<name>.kumbukaplugin`.

## Go plugins

A plugin normally imports only the root SDK:

```go
import sdk "github.com/kumbuka-me/sdk"
```

Register executable contributions from `init`. The SDK owns the WASM ABI, host capability transport, response encoding, and typed capability clients.

See [WIRE.md](WIRE.md) for the low-level ABI contract and [pluginpackage](pluginpackage/README.md) for the package format.

## Development

```sh
make fmt
make vet
make test
```

The SDK has no dependency on the Kumbuka application module. Kumbuka and Kumbuka plugins depend on this module instead.
