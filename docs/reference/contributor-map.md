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

- Write-path semantics: [Event-driven system](../architecture/foundations/event-driven-system.md)
- System extension: [Game systems architecture](../architecture/systems/game-systems.md)
