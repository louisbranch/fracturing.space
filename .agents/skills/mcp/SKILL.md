---
name: mcp
description: MCP transport boundaries and gRPC parity workflow
user-invocable: true
---

# MCP Development

Guidance for MCP tool and resource changes.

## Architecture Rules

- MCP is a thin transport wrapper; state and rules live in gRPC/domain packages.
- If new behavior requires domain logic, implement it in domain or gRPC first, then expose through MCP.
- Keep MCP handlers thin: validate transport payload, map to service calls, and map service responses/errors back to MCP.

## Parity Rules

- Keep gRPC and MCP APIs in parity.
- Keep auth requirements, error semantics, and capability names aligned across transports.
- If parity intentionally diverges, document the reason in `docs/` and the PR.

## Tool Exposure

- When adding MCP tools, update `internal/test/integration/fixtures/blackbox_tools_list.json`.
- Tool exposure should align with the campaign's game system (dynamic exposure is a future requirement).

## Verification

- Add or update integration coverage for MCP-to-gRPC parity.
- Run project verification commands from `AGENTS.md` after MCP changes.
