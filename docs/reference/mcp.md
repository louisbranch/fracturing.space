---
title: "MCP"
parent: "Reference"
nav_order: 1
last_reviewed: "2026-03-14"
---

# MCP overview

MCP is an internal HTTP bridge between the AI service and game/AI capabilities.
It is not a public user or third-party integration surface.

## Need to know

- Transport: HTTP only
- Endpoints:
  - `POST /mcp`
  - `GET /mcp`
  - `GET /mcp/health`
- Session authority is fixed per bridge session with:
  - `X-Fracturing-Space-MCP-Campaign-Id`
  - `X-Fracturing-Space-MCP-Session-Id`
  - `X-Fracturing-Space-MCP-Participant-Id`
- Production exposure is intentionally limited to the GM-safe AI profile.
- Public OAuth, host-allowlist, and browser-session behavior are not part of the
  supported contract.

## Runtime default

For normal local startup through `cmd/mcp`, treat `localhost:8085` as canonical.

## Where exact contracts live

- Production tool profile: [mcp-tools.md](mcp-tools.md)
- Production resource profile: [mcp-resources.md](mcp-resources.md)
- Runtime/default configuration: [configuration](../running/configuration.md)

## Contributor notes

MCP registration architecture and ownership boundaries are implemented in:

- `internal/services/mcp/domain` (tool/resource handlers and MCP-facing payloads)
- `internal/services/mcp/sessionctx` (session-scoped authority, request metadata, and gRPC call context)
- `internal/services/mcp/service` (transport and registration wiring)
- `internal/services/shared/mcpbridge` (internal AI bridge headers/profile)

When changing the production AI surface, keep the GM-safe profile, runtime docs,
and registration modules aligned.
