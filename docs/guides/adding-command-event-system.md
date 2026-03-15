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

1. For core commands, add the route in the owning core package and keep `internal/services/game/app/domain.go` limited to domain runtime composition.
2. Implement or extend decider logic in the owning module or aggregate package.
3. Ensure emitted events are declared and validated by the registry.
4. Add fold and adapter coverage for any replay- or projection-relevant events.

## 3. If adding a new game system

1. Implement the system package in `internal/services/game/domain/bridge/<system>/`.
2. Add one `SystemDescriptor` in `internal/services/game/domain/bridge/manifest/manifest.go`.
   That descriptor is the built-in source of truth for:
   - `BuildModule` → write-path registration
   - `BuildMetadataSystem` → API-facing system metadata
   - `BuildAdapter` → projection apply and replay repair
3. Add module conformance tests using `internal/services/game/domain/module/testkit/`.
4. If the system needs projection storage, add a system-owned store contract in the system package and keep store extraction owned by that system's descriptor:
   - keep the concrete backend implementation in the owning backend package
   - expose any needed provider method on the concrete projection backend
   - make `BuildAdapter` extract the system store from the concrete store source it receives
5. Add scenario coverage for the new system:
   - place scenarios under `internal/test/game/scenarios/systems/<system_id>/`
   - add/select smoke entries in `internal/test/game/scenarios/manifests/`
   - use a system handle for mechanics (for example `local dh = scn:system(\"<SYSTEM_ID>\")`)
6. Update generated event docs after registration changes land.

## 4. If exposing MCP tooling

1. Add domain tool/resource handlers in `internal/services/mcp/domain/`.
   Session-scoped bridge authority and outgoing metadata helpers belong in `internal/services/mcp/sessionctx/`, not beside gameplay handlers.
2. Register in `internal/services/mcp/service/server_registration.go`.
3. If the production AI bridge surface changes, update `internal/services/shared/mcpbridge/` and the MCP reference docs together.
4. Add/update MCP-focused tests.

## 5. Startup validation debugging

If game service startup fails after registration changes, error messages are
already scoped to the failing phase:

- `system module <id>@<version> <step>: <cause>` points to module registration
  (`register commands`, `register events`, namespace, or emittable checks).
- `registry validation <step>: <cause>` points to post-registration coverage and
  consistency checks.
- `system module registry mismatch: ...` points to manifest-derived parity drift
  between module, metadata, and adapter registration.

Use these step labels first before deep code tracing.

## 6. Regenerate and verify

- targeted package tests for the owning core or system package
- `make smoke`
- `make check`
- `make game-architecture-check`
- `make event-catalog-check`
- `make docs-check`

## Canonical references

- Write-path model: [Event-driven system](../architecture/foundations/event-driven-system.md)
- System architecture: [Game systems architecture](../architecture/systems/game-systems.md)
- System authoring details: [Adding a game system](../architecture/systems/adding-a-game-system.md)
- Generated command/event contracts: [Events index](../events/index.md)
