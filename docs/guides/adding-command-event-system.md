---
title: "Adding a command/event/system"
parent: "Guides"
nav_order: 1
---

# Adding a command, event, or game system

Canonical how-to for system-extension changes.

## 1. Add or change command/event definitions

1. Update the owning registration surface:
   - system modules: `RegisterCommands` and `RegisterEvents` in `internal/services/game/domain/bridge/<system>/module.go`
   - core modules: registration in the owning `internal/services/game/domain/<area>/registry.go`
2. Update payload structs and validation in the same owning package.
3. Add/update focused registration tests in that package.

## 2. Wire command behavior

1. Add route entry where needed (for core commands) in `internal/services/game/app/domain.go`.
2. Implement/extend decider logic in the owning module.
3. Ensure emitted events are declared and validated by the registry.

## 3. If adding a new game system

1. Implement system module in `internal/services/game/domain/bridge/<system>/`.
2. Register descriptor in:
   - `internal/services/game/domain/bridge/manifest/manifest.go`
   - `internal/services/game/domain/engine/registries.go` integration points
3. Add module conformance/integration tests.
4. Add scenario coverage for the new system:
   - place scenarios under `internal/test/game/scenarios/systems/<system_id>/`
   - add/select smoke entries in `internal/test/game/scenarios/manifests/`
   - use `local <alias> = scene:system(\"<SYSTEM_ID>\")` for system mechanics in Lua scripts

## 4. If exposing MCP tooling

1. Add domain tool/resource handlers in `internal/services/mcp/domain/`.
2. Register in `internal/services/mcp/service/server.go`.
3. Add/update MCP-focused tests.

## 5. Regenerate and verify

- `go test ./...`
- `make integration`
- `make event-catalog-check`
- `make docs-check`

## Canonical references

- Write-path model: [Event-driven system](../architecture/foundations/event-driven-system.md)
- System architecture: [Game systems architecture](../architecture/systems/game-systems.md)
- Generated command/event contracts: [Events index](../events/index.md)
