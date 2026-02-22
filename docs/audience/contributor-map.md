---
title: "Contributor Map"
parent: "Audience"
nav_order: 2
---

# Contributor Map

Use this map to find the best first edit point for common contribution types.

## Where to edit for X

| Change you want | Primary files/packages |
| --- | --- |
| Add a command, event, or game system | `docs/audience/adding-command-event-system.md` |
| Add or update Daggerheart action RPC behavior | `internal/services/game/api/grpc/systems/daggerheart/actions_*.go` |
| Add Daggerheart domain rules/mechanics | `internal/services/game/domain/bridge/daggerheart/` |
| Add or update admin page/route or adjust admin rendering | `internal/services/admin/handler_*.go`, `internal/services/admin/templates/` |
| Add or update MCP campaign tools/resources | `internal/services/mcp/domain/campaign_*.go` |
| Add/update MCP campaign tool registration | `internal/services/mcp/domain`, `internal/services/mcp/service/server.go` |
| Add/update MCP session/event/context registration | `internal/services/mcp/domain`, `internal/services/mcp/service/server.go` |
| Add/update game projection/storage behavior | `internal/services/game/storage/sqlite/store_*.go`, `internal/services/game/storage/storage.go` |
| Add game transport-level handlers (non-system) | `internal/services/game/api/grpc/game/` |
| Change game service startup/bootstrap flow | `internal/services/game/app/bootstrap.go`, `internal/services/game/app/server_bootstrap.go` |
| Add shared game test fakes/builders | `internal/testkit/gamefakes/` |
| Update domain write flow/apply behavior | `internal/services/game/api/grpc/internal/domainwrite/` |
| Refactor command startup/config wiring | `internal/platform/cmd`, `internal/cmd/{admin,ai,auth,chat,game,mcp,scenario,seed,web}` |

## Fast orientation flow

1. Read `docs/project/architecture.md` and `docs/project/domain-language.md`.
2. Find your change row in the table above.
3. Run targeted tests for that area first, then full validation:
   - `go test ./internal/services/game/api/grpc/internal/domainwrite -run TestShouldApplyProjectionInline`
   - `go test ./internal/services/game/app -run TestBuildCoreRouteTable`
   - `go test ./internal/services/mcp/service -run Test`
   - `go test ./...`
   - `make integration`
   - `make cover`

## Refactor landmarks (2026-02)

- Daggerheart gRPC action handlers were split from one monolith into:
  - `actions_damage.go`
  - `actions_recovery.go`
  - `actions_conditions.go`
  - `actions_countdowns.go`
  - `actions_session_flows.go`
  - `actions_outcomes.go`
  - `actions_helpers.go`
- Admin HTTP handler was split into:
  - `handler.go` (bootstrap and route wiring)
  - `handler_scenarios.go`
  - `handler_users.go`
  - `handler_campaigns_catalog.go`
  - `handler_dashboard_activity.go`
  - `handler_helpers.go`
- MCP campaign domain was split into:
  - `campaign.go` (shared types)
  - `campaign_tool_defs.go`
  - `campaign_handlers.go`
  - `participant_handlers.go`
  - `character_handlers.go`
  - `campaign_resources.go`
