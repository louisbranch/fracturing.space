---
title: "Adding a command/event/system"
parent: "Audience"
nav_order: 3
---

# Adding a command, event, or game system

Use this sequence for first-time contributions to avoid routing or registry regressions.

## Add or change a command/event definition

1. Update the owning domain module registration surface:
   - system modules: `RegisterCommands` and `RegisterEvents` in `internal/services/game/domain/bridge/<system>/module.go`
   - core modules: command/event registration in the owning `internal/services/game/domain/<area>/registry.go`
2. Update any command/event payload structs in the same module package.
3. Update generated catalog expectations by running the repo checks.
4. Add a focused registry test in the owning module package.
5. Update `docs/events/command-catalog.md` and `docs/events/event-catalog.md` through the existing check.

## Add core routing for command behavior

1. Add `coreCommandType*` constant in `internal/services/game/app/domain.go`.
2. Add route entry in `staticCoreCommandRoutes()`.
3. Re-run command/routing tests:
   - `go test ./internal/services/game/app -run TestBuildCoreRouteTable`

## Add a new game system module

1. Implement the system module wiring in `internal/services/game/domain/bridge/<system>/`.
2. Register the module in:
   - `internal/services/game/domain/bridge/manifest/manifest.go`
   - `internal/services/game/domain/engine/registries.go` integration points
3. Add module integration tests in `internal/services/game/domain` for command/event registrations.
4. Re-run startup wiring tests:
   - `go test ./internal/services/game/app -run Test.*System`

## MCP registration follow-up (if system exposes MCP tooling)

1. Add tool/resource handlers in `internal/services/mcp/domain/*`.
2. Register in `internal/services/mcp/service/server.go`.
3. Add/update MCP coverage tests:
   - `go test ./internal/services/mcp/service`
   - `go test ./internal/services/mcp/domain -run Test`

## Pre-merge checks for this change type

- `go test ./...`
- `make integration`
- `make event-catalog-check`
- `make docs-path-check`

## Next docs

- Conceptual write-path rules: [../architecture/event-driven-system.md](../architecture/event-driven-system.md)
- System extension architecture: [../architecture/game-systems.md](../architecture/game-systems.md)
- Generated command/event contracts: [../events/index.md](../events/index.md)
