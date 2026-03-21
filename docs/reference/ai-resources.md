---
title: "AI orchestration resources"
parent: "Reference"
nav_order: 3
last_reviewed: "2026-03-18"
---

# AI orchestration resources

Resources available during campaign-turn orchestration.

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

- `internal/services/ai/orchestration/gametools/tools.go`
- `internal/services/ai/orchestration/gametools/resources.go`
