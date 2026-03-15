---
title: "Contributor map"
parent: "Reference"
nav_order: 4
last_reviewed: "2026-03-14"
---

# Contributor map

Use this map to find the best first edit point for common contribution types.

## Where to edit for X

| Change you want | Primary files/packages |
| --- | --- |
| Add a command, event, or game system | `docs/guides/adding-command-event-system.md` |
| Add/update Daggerheart mechanics or gRPC gameplay/content flows | `internal/services/game/domain/bridge/daggerheart/`, `internal/services/game/api/grpc/systems/daggerheart/` |
| Add/update MCP gameplay tool/resource handlers or production bridge exposure | `internal/services/mcp/domain/`, `internal/services/mcp/sessionctx/`, `internal/services/mcp/service/server_registration.go`, `internal/services/shared/mcpbridge/` |
| Add/update MCP AI campaign artifacts or system-reference tools/resources | `internal/services/mcp/campaigncontext/`, `internal/services/mcp/service/server_registration.go`, `internal/services/shared/mcpbridge/` |
| Add/update MCP HTTP bridge transport, session lifecycle, or host validation | `internal/services/mcp/httptransport/`, `internal/services/mcp/service/server_runtime.go`, `internal/services/mcp/service/http_transport_runtime_adapter.go` |
| Add/update auth identity/OAuth/passkey behavior | `internal/services/auth/api/grpc/auth/`, `internal/services/auth/oauth/`, `internal/services/auth/storage/sqlite/` |
| Add/update AI orchestration/agent invocation | [AI service contributor map](ai-service-contributor-map.md) |
| Add/update play realtime transport, transcript flow, or play-session/auth handoff | `internal/services/play/app/`, `internal/services/play/storage/sqlite/`, `internal/services/shared/playlaunchgrant/`, `internal/services/shared/playorigin/` |
| Add/update play SPA UI, system-specific game views, or websocket/bootstrap client behavior | `internal/services/play/ui/src/`, `internal/services/play/ui/src/app_mode.ts`, `internal/services/play/ui/src/systems/` |
| Add/update social profiles/contacts | `internal/services/social/api/grpc/social/`, `internal/services/social/storage/sqlite/` |
| Add/update notifications inbox/delivery behavior | `internal/services/notifications/domain/`, `internal/services/notifications/api/grpc/notifications/`, `internal/services/notifications/storage/sqlite/` |
| Add/update user dashboard aggregation | `internal/services/userhub/domain/`, `internal/services/userhub/api/grpc/userhub/`, `internal/services/userhub/app/` |
| Add/update worker outbox processing | `internal/services/worker/app/`, `internal/services/worker/domain/`, `internal/services/worker/storage/sqlite/` |
| Add/update discovery catalog entries and APIs | `internal/services/discovery/api/grpc/discovery/`, `internal/services/discovery/catalog/`, `internal/services/discovery/storage/sqlite/` |
| Add/update status reporting and overrides | `internal/services/status/api/grpc/status/`, `internal/services/status/domain/`, `internal/services/status/storage/sqlite/` |
| Add/update shared service infrastructure | `internal/services/shared/` |
| Add/update web module routes/handlers | `internal/services/web/modules/<area>/` |
| Add/update web module composition | `internal/services/web/modules/registry_*.go`, `internal/services/web/composition/compose.go` |
| Add/update game transport handlers or game-domain workflows | [Game service contributor map](game-service-contributor-map.md) |
| Add/update campaign reads, campaign list behavior, or campaign protobuf mapping | `internal/services/game/api/grpc/game/campaigntransport/` |
| Add/update participant reads, roster listing/get flows, or participant protobuf mapping | `internal/services/game/api/grpc/game/participanttransport/` |
| Add/update character reads, sheet assembly, profile/state listing, or character protobuf mapping | `internal/services/game/api/grpc/game/charactertransport/` |
| Add/update fork lineage, fork replay/copy behavior, or source-campaign fork orchestration | `internal/services/game/api/grpc/game/forktransport/` |
| Add/update snapshot reads or Daggerheart snapshot/state mutation flows | `internal/services/game/api/grpc/game/snapshottransport/`, `internal/services/game/api/grpc/game/charactertransport/` |
| Add/update event history reads, campaign update subscriptions, or timeline assembly | `internal/services/game/api/grpc/game/eventtransport/` |
| Add/update authorization policy checks or batch authorization transport | `internal/services/game/api/grpc/game/authorizationtransport/`, `internal/services/game/api/grpc/game/authz/` |
| Add/update interaction context/control flows | `internal/services/game/api/grpc/game/interaction_*.go`, `internal/services/game/api/grpc/game/sessiontransport/` |
| Add/update session gRPC mapping, spotlight/gate protobuf conversions | `internal/services/game/api/grpc/game/sessiontransport/` |
| Add/update session-gate authority, workflow validation, progress tracking, or structured gate storage | `internal/services/game/domain/session/decider_gate.go`, the `internal/services/game/domain/session/gate_workflow_*.go`, `gate_progress_*.go`, and `gate_projection_*.go` families, `internal/services/game/projection/apply_session.go`, and `internal/services/game/storage/sqlite/coreprojection/store_projection_session_gate.go` |
| Add/update game bootstrap/service registration | `internal/services/game/app/`, `internal/services/game/app/bootstrap_service_registration.go` |
| Add/update projection/storage behavior | `internal/services/game/storage/sqlite/coreprojection/` for shared projection records, conversion seams, apply-once, snapshots, and watermarks; `internal/services/game/storage/sqlite/eventjournal/` for immutable event persistence; `internal/services/game/storage/sqlite/integrationoutbox/` for worker delivery persistence; and `internal/services/game/projection/` for apply logic |
| Add/update command startup/config wiring | `internal/platform/cmd`, `internal/cmd/*` |

## Validation path

Run targeted tests first, then full checks:

- `go test ./internal/services/game/...`
- `make smoke`
- `make check`
- `make game-architecture-check` (when changing `internal/services/game/**`)
- `make docs-check`

## Canonical references

- Service runtime ownership map: [Small services topology](small-services-topology.md)
- Play routing: [Play contributor map](play-contributor-map.md)
- Write-path semantics: [Event-driven system](../architecture/foundations/event-driven-system.md)
- System extension: [Game systems architecture](../architecture/systems/game-systems.md)
- Game service routing: [Game service contributor map](game-service-contributor-map.md)
