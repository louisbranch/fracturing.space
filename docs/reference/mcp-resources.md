---
title: "MCP resources"
parent: "Reference"
nav_order: 3
last_reviewed: "2026-03-13"
---

# MCP resources

Exact MCP resource URIs currently registered by the server.

- `campaigns://list`
- `campaign://{campaign_id}`
- `campaign://{campaign_id}/participants`
- `campaign://{campaign_id}/characters`
- `campaign://{campaign_id}/sessions`
- `campaign://{campaign_id}/sessions/{session_id}/scenes`
- `campaign://{campaign_id}/interaction`
- `context://current`

## Verification

For implementation and registration details, inspect:

- `internal/services/mcp/service/server.go`
- `internal/services/mcp/service/server_registration.go`
- `internal/services/mcp/domain/`
