# Duality Engine ![Coverage](../../raw/badges/coverage.svg)

Duality Engine is a server-authoritative mechanics and campaign-state service for running a Daggerheart Compatible campaign using the Duality Dice system.

Its primary goal is enabling an AI Game Master: an LLM (or agent runtime) drives the campaign by calling MCP tools/resources to resolve mechanics and persist state, while human participants interact through whatever client you build around it.

This project focuses on rules adjudication and campaign state management. It does not generate narrative content and it does not ship any copyrighted Daggerheart material (no rulebook text, artwork, setting, or other Prohibited Content).

## Who this is for

Right now, Duality Engine is primarily for developers experimenting with AI-driven campaign runners:

- You want an AI GM to call tools and get structured outcomes.
- You want deterministic resolution you can replay, test, and audit.
- You do not want clients to embed rules logic.

It is not a VTT. You will need to run the service locally or remotely and integrate a client yourself.

## What it does

- **Rules adjudication**: resolve Duality Dice rolls and return structured outcomes.
- **Campaign state**: persisted campaign entities (see State Model below).
- **Determinism**: supports auditable resolution (useful for testing, replay, and debugging).
- **Two interfaces, same behavior**: gRPC APIs and MCP tools/resources are kept in parity.

## Interfaces

### MCP (primary integration surface)

Use MCP if you are integrating with AI tooling that can call tools/resources.

- **stdio JSON-RPC**: best fit for local agent runtimes and tooling.
- **HTTP**: useful when you want to run the MCP server remotely or behind a gateway.

See [docs](#Documentation) for the tool/resource catalog.

### gRPC (service API)

Use gRPC if you are building a custom client (UI, automation, services) or you want a strongly-typed API. The gRPC API is the source of truth; the MCP server is a transport layer over it.

## Quickstart

### Run from source (recommended for development)

```sh
make run
```

This starts the gRPC server on `localhost:8080`, the MCP server on stdio, and the web client at `http://localhost:8082`.
Ports, endpoints, and configuration are documented in [docs](#Documentation).

### Run with Docker (recommended for local-only execution)

Download the Docker Hub images:

```shell
docker pull louisbranch/duality-grpc:latest
docker pull louisbranch/duality-mcp:latest
```

Notes:
- The gRPC server listens on port 8080, and the MCP HTTP transport listens on port 8081 when enabled.
- Full port/config details live in [docs](#Documentation).

## State Model

Persisted (BoltDB):

- Campaigns
- Participants
- Characters
- Sessions

Ephemeral:

- MCP execution context

## Status and stability

- **Pre-release / prototype**
- gRPC and MCP are kept in parity, but the API is expected to change until a release candidate is published.

## Documentation

- Published docs site: [https://louisbranch.github.io/duality-engine/](https://louisbranch.github.io/duality-engine/)
- Repo docs:
  - [Getting Started](docs/getting-started.md)
  - [Configuration](docs/configuration.md)
  - [MCP](docs/mcp.md)
  - [Integration Tests](docs/integration-tests.md)

## Near-term roadmap

- Expand campaign lifecycle tools
- Improve MCP context handling for multi-client use
- Expand telemetry and request tracing
- Support user-provided content packs (e.g. JSON/Markdown) to extend MCP resources without bundling copyrighted material

## Attribution and licensing

Duality Engine is an independent, fan-made project and is not affiliated with Critical Role Productions LLC, Darrington Press, or their partners.

Daggerheart is a trademark of Critical Role Productions LLC. This project is intended for use under the Darrington Press Community Gaming License (DPCGL). Source code is licensed under the MIT License. See [LICENSE](LICENSE).

All trademarks and copyrighted material remain the property of their respective owners.

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md).
Authors: [AUTHORS](AUTHORS).
