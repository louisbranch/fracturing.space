---
title: "MCP"
parent: "Reference"
nav_order: 1
---

# MCP overview

MCP provides a JSON-RPC surface over the same core game capabilities exposed by gRPC.

## Need to know

- Transport modes: `stdio` (default) and `http`
- HTTP endpoints:
  - `POST /mcp`
  - `GET /mcp`
  - `GET /mcp/health`
- Session model: cookie-based (`mcp_session`) for HTTP transport

## Port defaults and precedence

Two defaults exist in different layers and are both intentional:

1. `cmd/mcp` runtime default for `-http-addr` / `FRACTURING_SPACE_MCP_HTTP_ADDR`: `localhost:8085`
2. internal `mcp/service` fallback when embedded with empty HTTP address config: `localhost:8081`

For normal CLI/server startup through `cmd/mcp`, treat `localhost:8085` as canonical.

## Where exact contracts live

- Tool contracts: [mcp-tools.md](mcp-tools.md)
- Resource contracts: [mcp-resources.md](mcp-resources.md)
- Runtime/default configuration: [running/configuration.md](../running/configuration.md)

## Contributor notes

MCP registration architecture and ownership boundaries are implemented in:

- `internal/services/mcp/domain` (domain handlers and definitions)
- `internal/services/mcp/service` (transport and registration wiring)

When changing registrations, keep module ownership narrow and update focused tests in `internal/services/mcp/service`.
