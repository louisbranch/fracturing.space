---
title: "MCP resources"
parent: "Reference"
nav_order: 3
last_reviewed: "2026-03-14"
---

# MCP resources

Resources exposed by the production internal AI bridge.

- `campaign://{campaign_id}`
- `campaign://{campaign_id}/participants`
- `campaign://{campaign_id}/characters`
- `campaign://{campaign_id}/sessions`
- `campaign://{campaign_id}/sessions/{session_id}/scenes`
- `campaign://{campaign_id}/interaction`
- `campaign://{campaign_id}/artifacts`
- `campaign://{campaign_id}/artifacts/{path}`
- `context://current`

Global campaign listing is intentionally excluded from the production profile.

## Verification

For implementation and registration details, inspect:

- `internal/services/shared/mcpbridge/profile.go`
- `internal/services/mcp/service/server_registration.go`
- `internal/services/mcp/domain/`
- `internal/services/mcp/sessionctx/`
