---
title: "Contributor map"
parent: "Reference"
nav_order: 4
---

# Contributor map

Use this map to find the best first edit point for common contribution types.

## Where to edit for X

| Change you want | Primary files/packages |
| --- | --- |
| Add a command, event, or game system | `docs/guides/adding-command-event-system.md` |
| Add/update Daggerheart mechanics | `internal/services/game/domain/bridge/daggerheart/` |
| Add/update MCP tool/resource handlers | `internal/services/mcp/domain/`, `internal/services/mcp/service/server.go` |
| Add/update auth identity/OAuth/passkey behavior | `internal/services/auth/api/grpc/auth/`, `internal/services/auth/oauth/`, `internal/services/auth/storage/sqlite/` |
| Add/update AI orchestration/agent invocation | `internal/services/ai/api/grpc/ai/`, `internal/services/ai/storage/sqlite/`, `internal/services/ai/app/` |
| Add/update chat realtime transport and room flow | `internal/services/chat/app/` |
| Add/update social profiles/contacts | `internal/services/social/api/grpc/social/`, `internal/services/social/storage/sqlite/` |
| Add/update notifications inbox/delivery behavior | `internal/services/notifications/domain/`, `internal/services/notifications/api/grpc/notifications/`, `internal/services/notifications/storage/sqlite/` |
| Add/update user dashboard aggregation | `internal/services/userhub/domain/`, `internal/services/userhub/api/grpc/userhub/`, `internal/services/userhub/app/` |
| Add/update worker outbox processing | `internal/services/worker/app/`, `internal/services/worker/domain/`, `internal/services/worker/storage/sqlite/` |
| Add/update discovery catalog entries and APIs | `internal/services/discovery/api/grpc/discovery/`, `internal/services/discovery/catalog/`, `internal/services/discovery/storage/sqlite/` |
| Add/update status reporting and overrides | `internal/services/status/api/grpc/status/`, `internal/services/status/domain/`, `internal/services/status/storage/sqlite/` |
| Add/update shared service infrastructure | `internal/services/shared/` |
| Add/update web module routes/handlers | `internal/services/web/modules/<area>/` |
| Add/update web module composition | `internal/services/web/modules/registry_*.go`, `internal/services/web/composition/compose.go` |
| Add/update game transport handlers | `internal/services/game/api/grpc/game/`, `internal/services/game/api/grpc/systems/` |
| Add/update projection/storage behavior | `internal/services/game/storage/sqlite/store_*.go`, `internal/services/game/projection/` |
| Add/update command startup/config wiring | `internal/platform/cmd`, `internal/cmd/*` |

## Validation path

Run targeted tests first, then full checks:

- `go test ./...`
- `make integration`
- `make docs-check`

## Canonical references

- Service runtime ownership map: [Small services topology](small-services-topology.md)
- Write-path semantics: [Event-driven system](../architecture/foundations/event-driven-system.md)
- System extension: [Game systems architecture](../architecture/systems/game-systems.md)
