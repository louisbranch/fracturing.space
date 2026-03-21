---
title: "Adding a game system"
parent: "Game systems"
nav_order: 3
status: canonical
owner: engineering
last_reviewed: "2026-03-11"
---

# Adding a Game System

Step-by-step guide for registering a new game system in the game service. Use
Daggerheart as the reference implementation for manifest/module/adapter shape
(`internal/services/game/domain/systems/daggerheart/`), but do not copy its
surface area blindly. New systems should follow the manifest-driven path
described here rather than wiring startup through ad hoc app or engine edits.

## Required parts

A game system requires three required implementations plus optional metadata
hooks registered through a single
`SystemDescriptor` entry in the manifest. Startup parity validation catches
any mismatches between the registered pieces automatically.

| Component | Interface | Purpose |
|-----------|-----------|---------|
| Module | `module.Module` | Commands, events, decider, folder, state factory |
| Metadata System | `systems.GameSystem` | Registry metadata, name, version, optional state/outcome hooks |
| Adapter | `systems.Adapter` | Projection event handlers, snapshot, profile adapter |
| Manifest Entry | `manifest.SystemDescriptor` | Unifying builder that wires the three above |

## Module vs systems

The component split above reflects a CQRS boundary. A **module** owns the
write path: it implements command handling (Decider), event folding (Folder),
and state initialization (StateFactory), giving the system its command-execution
and event-replay behavior. A **systems adapter** owns the read path: it
implements projection application (Apply) and snapshot materialization, turning
committed events into queryable state. The companion `systems.GameSystem`
provides system metadata for the read-side registry.

Both sides are registered together through a single `SystemDescriptor` in the
manifest, which validates parity at startup so that every event a module can
emit has a corresponding projection handler in the adapter.

## Authoring path

Built-in system registration should read as one sequence:

1. Implement the system package under `internal/services/game/domain/systems/<system>/`.
2. Add one `manifest.SystemDescriptor` entry in `internal/services/game/domain/systems/manifest/manifest.go`.
3. If needed, add the system-owned projection store contract and backend wiring.
4. Run module conformance, startup parity, generated event docs, and scenario checks.

## Step 1: Create the system package

Recommended layout:

- `module.go`: module implementation and registration
- `decider.go`: command decisions
- `folder.go`: replay fold
- `state.go`: state types and factory
- `adapter.go`: projection apply
- `registry_system.go`: metadata system
- typed payload / command / event files as needed

## Step 2: Implement the Module

Implement `module.Module` with explicit command registration, event
registration, decider, folder, state factory, and emittable event types. Run
`internal/services/game/domain/module/testkit/` against the module to validate
coverage and durable write-path behavior.

## Step 3: Implement the Metadata System

Implement `systems.GameSystem` for system name/version metadata and any optional
state-handler or outcome-application hooks. `StateHandlerFactory` and
`OutcomeApplier` may be `nil`, but only when the system truly does not expose
those surfaces yet; document that choice in package comments.

## Step 4: Implement the Adapter

Implement `systems.Adapter` with explicit handled event types and idempotent
projection apply behavior. If the system supports character profiles, model
them as typed system-owned commands/events; do not route profile writes through
core `map[string]any` envelopes.

## Step 5: Add the Manifest Entry

In `internal/services/game/domain/systems/manifest/manifest.go`, add a
`SystemDescriptor` to `builtInSystems`:

```go
{
    ID:                  <system>.SystemID,
    Version:             <system>.SystemVersion,
    BuildModule:         func() domainsystem.Module { return <system>.NewModule() },
    BuildMetadataSystem: func() domainsystems.GameSystem { return <system>.NewRegistrySystem() },
    BuildAdapter: func(storeSource any) domainsystems.Adapter {
        store := <system>.ProjectionStoreFromSource(storeSource)
        if store == nil { return nil }
        return <system>.NewAdapter(store)
    },
},
```

That descriptor is the built-in source of truth used by `manifest.Modules()`,
`manifest.MetadataSystems()`, and `manifest.AdapterRegistry(...)`. Do not add
separate system registration lists in app or engine startup code.

## Step 6: Add Storage

If your system requires projection storage:

1. Define the store interface in your system package, not in core `internal/services/game/storage/`.
2. Add the backend implementation in the owning backend package.
3. Expose any needed provider method on the concrete projection backend.
4. Keep store extraction inside the owning system descriptor's `BuildAdapter`.

Do not add a manifest-wide store bundle for new systems. Adapter registration
should accept the concrete store source directly and let each system own the
small amount of extraction logic it needs.

## Verification

After all steps:
```bash
make test                    # Unit tests pass
make smoke                   # Quick runtime confidence
make check                   # Final local guard
make game-architecture-check # Parity validation passes
make docs-check              # Authoring docs stay aligned
```

Common parity failures:

- module registered without metadata or adapter
- version mismatch across module, metadata, and adapter
- adapter handles types not declared emittable
- profile events missing folder or adapter coverage
- payload validation missing for registered events

Key files:

- `internal/services/game/domain/systems/manifest/manifest.go`
- `internal/services/game/domain/module/testkit/`
- `internal/services/game/app/system_registration.go`
- `internal/services/game/app/bootstrap_systems.go`
