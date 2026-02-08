---
name: web-server
description: Web UI and transport layer conventions
user-invocable: true
---

# Web Server Conventions

Transport-layer guidance for the Web UI and related services.

## Architecture Notes

- Admin dashboard lives under `cmd/admin`.
- Transport services include `cmd/game` (gRPC) and `cmd/mcp` (MCP bridge).
- Keep transport thin: rules and state logic belong in gRPC/domain packages.
