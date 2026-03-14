---
title: "Adding a game system"
parent: "Game systems"
nav_order: 3
status: canonical
owner: engineering
last_reviewed: "2026-03-07"
---

# Adding a Game System

Step-by-step guide for registering a new game system in the game service.
Use Daggerheart as the reference implementation for manifest/module/adapter
shape (`internal/services/game/domain/bridge/daggerheart/`), but do not copy
its intentionally-nil metadata hooks unless your system truly lacks those
surfaces too.

## Overview

A game system requires three required implementations plus optional metadata
hooks registered through a single
`SystemDescriptor` entry in the manifest. Startup parity validation catches
any mismatches between the registered pieces automatically.

| Component | Interface | Purpose |
|-----------|-----------|---------|
| Module | `module.Module` | Commands, events, decider, folder, state factory |
| Metadata System | `bridge.GameSystem` | Registry metadata, name, version, optional state/outcome hooks |
| Adapter | `bridge.Adapter` | Projection event handlers, snapshot, profile adapter |
| Manifest Entry | `manifest.SystemDescriptor` | Unifying builder that wires the three above |

## Step 1: Create the system package

```
internal/services/game/domain/bridge/<system-name>/
  module.go          # Module implementation
  decider.go         # Command decision logic
  folder.go          # Event fold logic
  state.go           # State types and factory
  adapter.go         # Projection adapter
  registry_system.go # Metadata system
  payload.go         # Event payload types
  commands.go        # Command type constants
  events.go          # Event type constants
```

## Step 2: Implement the Module

The module registers commands and events with the domain engine:

```go
type Module struct {
    decider module.Decider
    folder  module.Folder
    factory module.StateFactory
}

func (m *Module) ID() string      { return SystemID }
func (m *Module) Version() string { return SystemVersion }

func (m *Module) RegisterCommands(r *command.Registry) error { ... }
func (m *Module) RegisterEvents(r *event.Registry) error     { ... }
func (m *Module) Decider() module.Decider                    { return m.decider }
func (m *Module) Folder() module.Folder                      { return m.folder }
func (m *Module) StateFactory() module.StateFactory          { return m.factory }
func (m *Module) EmittableEventTypes() []event.Type          { ... }
```

Run the conformance test suite in `domain/module/testkit/` against your
module to validate coverage.

## Step 3: Implement the Metadata System

```go
type RegistrySystem struct{}

func (r *RegistrySystem) ID() bridge.SystemID { return bridge.SystemID<Name> }
func (r *RegistrySystem) Version() string     { return SystemVersion }
func (r *RegistrySystem) Name() string        { return "<Display Name>" }
func (r *RegistrySystem) RegistryMetadata() bridge.RegistryMetadata { ... }
func (r *RegistrySystem) StateHandlerFactory() bridge.StateHandlerFactory { ... }
func (r *RegistrySystem) OutcomeApplier() bridge.OutcomeApplier           { ... }
```
`StateHandlerFactory` and `OutcomeApplier` are optional. Return `nil` only when
the system does not expose those surfaces yet, and document that decision in
the package comment and registry-system comments.

## Step 4: Implement the Adapter

The adapter handles event projection for your system:

```go
type Adapter struct {
    store  storage.<System>Store
    router *module.AdapterRouter
}

func (a *Adapter) ID() string      { return SystemID }
func (a *Adapter) Version() string { return SystemVersion }
func (a *Adapter) Apply(ctx context.Context, evt event.Event) error { ... }
func (a *Adapter) HandledTypes() []event.Type { ... }
```

If your system supports character profiles, implement `bridge.ProfileAdapter`.

## Step 5: Add the Manifest Entry

In `internal/services/game/domain/bridge/manifest/manifest.go`, add a
`SystemDescriptor` to `builtInSystems`:

```go
{
    ID:                  <system>.SystemID,
    Version:             <system>.SystemVersion,
    BuildModule:         func() domainsystem.Module { return <system>.NewModule() },
    BuildMetadataSystem: func() domainbridge.GameSystem { return <system>.NewRegistrySystem() },
    BuildAdapter: func(stores ProjectionStores) domainbridge.Adapter {
        if stores.<System> == nil { return nil }
        return <system>.NewAdapter(stores.<System>)
    },
    HasProfileSupport: true, // if ProfileAdapter is implemented
},
```

Add your system's projection store to `manifest.ProjectionStores`.

## Step 6: Add Storage

If your system requires projection storage:

1. Define store interface in your system package (not in core `storage/`)
2. Add implementation in `storage/sqlite/`
3. Add the store field to `manifest.ProjectionStores`

## Verification

After all steps:

```bash
make test                       # Unit tests pass
make integration                # Component seams work
make game-architecture-check    # Parity validation passes
```

Startup parity validation will fail if:
- Module is registered but metadata system is missing (or vice versa)
- Module version doesn't match metadata or adapter version
- Adapter handles event types not declared as emittable
- Module declares profile support without implementing ProfileAdapter
- Events lack payload validation
