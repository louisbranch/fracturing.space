---
title: "MCP resources"
parent: "Reference"
nav_order: 3
---

# MCP resources

Exact MCP resource URIs currently registered by the server.

- `campaigns://list`
- `campaign://{campaign_id}`
- `campaign://{campaign_id}/participants`
- `campaign://{campaign_id}/characters`
- `campaign://{campaign_id}/sessions`
- `context://current`

## Verification

For implementation and registration details, inspect:

- `internal/services/mcp/service/server.go`
- `internal/services/mcp/service/server_registration.go`
- `internal/services/mcp/domain/`
