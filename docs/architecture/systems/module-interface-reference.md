---
title: "Module interface reference"
parent: "Game systems"
nav_order: 4
status: canonical
owner: engineering
last_reviewed: "2026-03-27"
---

# Module interface reference

This page documents the three registerable interface contracts that a game
system must (or may) satisfy. All live in the game service domain layer.

## `module.Module`

**Package:** `domain/module/registry.go`

Every game system registers exactly one `Module` with the module registry.

| Method | Return | Purpose |
|--------|--------|---------|
| `ID()` | `string` | Stable system identifier (e.g. `"daggerheart"`). |
| `Version()` | `string` | Ruleset version string (e.g. `"1.0.0"`). |
| `RegisterCommands(*command.Registry)` | `error` | Registers system-owned command definitions at startup. |
| `RegisterEvents(*event.Registry)` | `error` | Registers system-owned event definitions at startup. |
| `EmittableEventTypes()` | `[]event.Type` | Declares all event types the decider can emit; validated against the event registry at startup. |
| `Decider()` | `Decider` | Returns the system command handler (see below). |
| `Folder()` | `Folder` | Returns the system state folder (see below). |
| `StateFactory()` | `StateFactory` | Returns the factory that seeds initial system state for snapshots and characters. |

### `module.Decider`

```go
type Decider interface {
    Decide(state any, cmd command.Command, now func() time.Time) command.Decision
}
```

Pure function: inspects current state and command, returns accepted events or
rejections. Never mutates state.

### `module.Folder` (alias for `fold.Folder`)

```go
type Folder interface {
    Fold(state any, evt event.Event) (any, error)
    FoldHandledTypes() []event.Type
}
```

Applies a single event to state, returning the updated state.
`FoldHandledTypes` enables startup coverage validation.

### `module.StateFactory`

| Method | Purpose |
|--------|---------|
| `NewSnapshotState(campaignID)` | Seeds campaign-level system state on first system event. |
| `NewCharacterState(campaignID, characterID, kind)` | Seeds character-level system state on profile creation. |

Both return `any`. Implementations must be deterministic for replay safety.

## `systems.GameSystem`

**Package:** `domain/systems/registry_bridge.go`

Registered in the `MetadataRegistry` for API-layer dispatch and system metadata.

| Method | Return | Purpose |
|--------|--------|---------|
| `ID()` | `SystemID` | Domain-layer system identifier. |
| `Version()` | `string` | Ruleset version. |
| `Name()` | `string` | Human-readable display name. |
| `RegistryMetadata()` | `RegistryMetadata` | Implementation stage, operational status, access level. |
| `StateHandlerFactory()` | `StateHandlerFactory` | Typed state handlers for the API bridge (resource/damage abstractions). May be nil. |
| `OutcomeApplier()` | `OutcomeApplier` | Applies roll outcomes to game state. May be nil. |

## `systems.Adapter`

**Package:** `domain/systems/adapter_registry.go`

Registered in the `AdapterRegistry` for projection-side event handling.

| Method | Return | Purpose |
|--------|--------|---------|
| `ID()` | `string` | System identifier. |
| `Version()` | `string` | System version. |
| `Apply(ctx, event.Event)` | `error` | Projects a system event into the system-specific projection store. |
| `Snapshot(ctx, campaignID)` | `(any, error)` | Returns the current projected state for a campaign. |
| `HandledTypes()` | `[]event.Type` | Declares handled event types for startup validation against emittable types. |

## Optional module interfaces

A `Module` may additionally implement these interfaces. The engine probes via
type assertion at resolution time.

| Interface | Method | When called |
|-----------|--------|-------------|
| `CharacterReadinessProvider` | `BindCharacterReadiness(campaignID, stateMap)` | Session-start readiness evaluation. Returns a bound `CharacterReadinessEvaluator`. |
| `SessionStartBootstrapProvider` | `BindSessionStartBootstrap(campaignID, stateMap)` | First-session bootstrap. Returns a bound `SessionStartBootstrapEmitter`. |
| `CommandTyper` | `DeciderHandledCommands()` | Startup validation verifies decider covers all registered system commands. |

## Typed escape hatches for `any`-typed state

Because `Module` passes state as `any`, system authors use generic helpers to
recover type safety without hand-written assertions:

- **`TypedDecider[S]`** -- wraps a typed decide function; asserts `any` to `S`
  before calling the inner function.
- **`TypedFolder[S]`** -- wraps a typed fold function with the same pattern.
- **`FoldRouter[S]`** -- dispatches fold events to per-type handler functions
  with automatic JSON payload unmarshaling via `HandleFold[S, P]`.

All three live in `domain/module/`. See `domain/module/typed.go` and
`domain/module/fold_router.go` for signatures.
