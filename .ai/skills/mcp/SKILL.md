---
name: mcp
description: MCP tool/resource guidance and parity rules with gRPC
user-invocable: true
---

# MCP Development

Guidance for MCP tool and resource changes.

## Core Rules

- MCP is a thin transport wrapper; state and rules live in gRPC/domain packages.
- Keep gRPC and MCP APIs in parity.
- When adding MCP tools, update `internal/test/integration/fixtures/blackbox_tools_list.json`.
- Tool exposure should align with the campaign's game system (dynamic exposure is a future requirement).
