---
title: "System developer checklist"
parent: "Architecture"
nav_order: 9
status: canonical
owner: engineering
last_reviewed: "2026-02-26"
---

# System Developer Checklist

Linear walkthrough for adding a new game system. Follow each step in order;
verification commands confirm progress before moving on.

For architecture background, see [game systems](game-systems.md). For a
worked example of a single mechanic, see
[quick start](quick-start-system-developer.md).

---

## Step 1: Add proto identity

Add enum values for your system in the protobuf definition.

**File**: `api/proto/common/v1/game_system.proto`

```proto
GAME_SYSTEM_MY_SYSTEM = <next_number>;
```

```bash
make proto
go build ./...
```

---

## Step 2: Create your system package

**Directory**: `internal/services/game/domain/bridge/<system>/`

Create the package with identity constants and initial files:

### `constants.go`

```go
package mysystem

const (
    SystemID      = "GAME_SYSTEM_MY_SYSTEM"
    SystemVersion = "v1"
)
```

### `event_types.go`

```go
package mysystem

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"

const (
    // Add event types as you implement mechanics.
    // EventTypeExampleHappened event.Type = "sys.my_system.example_happened"
)
```

### `payload.go`

```go
package mysystem

// Add command and event payload structs as you implement mechanics.
// Use absolute values, not deltas (see game-systems.md invariant).
```

---

## Step 3: Implement the domain module

**File**: `internal/services/game/domain/bridge/<system>/module.go`

The module registers commands and events, and provides decider, folder, and
state factory implementations.

```go
package mysystem

import (
    "time"

    "github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
    "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
    "github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

type Module struct{}

func NewModule() *Module { return &Module{} }

func (m *Module) ID() string      { return SystemID }
func (m *Module) Version() string { return SystemVersion }

func (m *Module) RegisterCommands(registry *command.Registry) error {
    for _, def := range commandDefs {
        if err := registry.Register(def); err != nil {
            return err
        }
    }
    return nil
}

func (m *Module) RegisterEvents(registry *event.Registry) error {
    for _, def := range eventDefs {
        if err := registry.Register(def); err != nil {
            return err
        }
    }
    return nil
}

func (m *Module) EmittableEventTypes() []event.Type {
    types := make([]event.Type, len(eventDefs))
    for i, def := range eventDefs {
        types[i] = def.Type
    }
    return types
}

func (m *Module) Decider() module.Decider   { return &decider{} }
func (m *Module) Folder() module.Folder     { return newFolder() }
func (m *Module) StateFactory() module.StateFactory { return &stateFactory{} }

// commandDefs and eventDefs — populated as mechanics are added.
var commandDefs []command.Definition
var eventDefs []event.Definition
```

---

## Step 4: Implement the folder (aggregate state)

**File**: `internal/services/game/domain/bridge/<system>/folder.go`

Use `FoldRouter` for typed dispatch:

```go
package mysystem

import (
    "fmt"

    "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
    "github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

type SnapshotState struct {
    // Add aggregate fields as mechanics are implemented.
}

// assertSnapshotState converts untyped state to *SnapshotState for the fold
// router. FoldRouter expects func(any) (S, error) — two return values.
func assertSnapshotState(state any) (*SnapshotState, error) {
    switch s := state.(type) {
    case nil:
        return &SnapshotState{}, nil
    case *SnapshotState:
        if s != nil {
            return s, nil
        }
        return &SnapshotState{}, nil
    default:
        return nil, fmt.Errorf("unsupported state type %T", state)
    }
}

func newFolder() *folder {
    r := module.NewFoldRouter[*SnapshotState](assertSnapshotState)
    // Register fold handlers:
    // module.HandleFold(r, EventTypeExampleHappened, foldExampleHappened)
    return &folder{router: r}
}

type folder struct {
    router *module.FoldRouter[*SnapshotState]
}

func (f *folder) Fold(state any, evt event.Event) (any, error) {
    return f.router.Fold(state, evt)
}
```

---

## Step 5: Implement the state factory

**File**: `internal/services/game/domain/bridge/<system>/state_factory.go`

```go
package mysystem

type stateFactory struct{}

func (f *stateFactory) NewSnapshotState(campaignID string) (any, error) {
    return &SnapshotState{}, nil
}
```

`NewSnapshotState` must be deterministic: same `campaignID` produces the
same initial state, because replay depends on this guarantee.

---

## Step 6: Implement the decider

**File**: `internal/services/game/domain/bridge/<system>/decider.go`

Use `DecideFunc` helpers for typed dispatch. See
[game-systems.md DecideFunc decision tree](game-systems.md#decidefunc-decision-tree)
to pick the right variant.

```go
package mysystem

import (
    "time"

    "github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
)

type decider struct{}

func (d *decider) Decide(cmd command.Command, state any, now func() time.Time) command.Decision {
    // Route by command type:
    // switch cmd.Type {
    // case commandTypeExample:
    //     return decideExample(cmd, now)
    // }
    return command.Decision{}
}

// DeciderHandledCommands returns the command types this decider handles.
// Required by engine.ValidateDeciderCommandCoverage at startup.
func (d *decider) DeciderHandledCommands() []command.Type {
    types := make([]command.Type, len(commandDefs))
    for i, def := range commandDefs {
        types[i] = def.Type
    }
    return types
}
```

---

## Step 7: Implement the projection adapter

**File**: `internal/services/game/domain/bridge/<system>/adapter.go`

Use `AdapterRouter` for typed dispatch:

```go
package mysystem

import (
    "context"

    "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
    "github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
    "github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type Adapter struct {
    store storage.MySystemStore // Replace with your store interface
    router *module.AdapterRouter
}

func NewAdapter(store storage.MySystemStore) *Adapter {
    a := &Adapter{store: store}
    r := module.NewAdapterRouter()
    // Register handlers:
    // module.HandleAdapter(r, EventTypeExampleHappened, a.handleExampleHappened)
    a.router = r
    return a
}

func (a *Adapter) ID() string      { return SystemID }
func (a *Adapter) Version() string { return SystemVersion }

func (a *Adapter) Apply(ctx context.Context, evt event.Event) error {
    return a.router.Apply(ctx, evt)
}

func (a *Adapter) Snapshot(ctx context.Context, campaignID string) (any, error) {
    // Return system-specific projection snapshot.
    return nil, nil
}

func (a *Adapter) HandledTypes() []event.Type {
    return a.router.HandledTypes()
}
```

If your system stores per-character profile data, also implement
`bridge.ProfileAdapter`. See
[game-systems.md ProfileAdapter](game-systems.md#when-to-implement-profileadapter).

---

## Step 8: Implement the registry metadata system

**File**: `internal/services/game/domain/bridge/<system>/registry_system.go`

```go
package mysystem

import (
    commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
    bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
)

type RegistrySystem struct{}

func NewRegistrySystem() *RegistrySystem { return &RegistrySystem{} }

func (r *RegistrySystem) ID() commonv1.GameSystem {
    return commonv1.GameSystem_GAME_SYSTEM_MY_SYSTEM
}

func (r *RegistrySystem) Version() string { return SystemVersion }
func (r *RegistrySystem) Name() string    { return "My System" }

func (r *RegistrySystem) RegistryMetadata() bridge.RegistryMetadata {
    return bridge.RegistryMetadata{
        ImplementationStage: commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_ALPHA,
        OperationalStatus:   commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL,
        AccessLevel:         commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_BETA,
    }
}

func (r *RegistrySystem) StateHandlerFactory() bridge.StateHandlerFactory { return nil }
func (r *RegistrySystem) OutcomeApplier() bridge.OutcomeApplier           { return nil }
```

---

## Step 9: Register in the manifest

**File**: `internal/services/game/domain/bridge/manifest/manifest.go`

Add your system to `builtInSystems` and `ProjectionStores`:

```go
// In ProjectionStores:
type ProjectionStores struct {
    Daggerheart storage.DaggerheartStore
    MySystem    storage.MySystemStore  // Add your store
}

// In builtInSystems:
{
    ID:      mysystem.SystemID,
    Version: strings.TrimSpace(mysystem.SystemVersion),
    BuildModule:         func() domainsystem.Module { return mysystem.NewModule() },
    BuildMetadataSystem: func() domainbridge.GameSystem { return mysystem.NewRegistrySystem() },
    BuildAdapter: func(stores ProjectionStores) domainbridge.Adapter {
        if stores.MySystem == nil {
            return nil
        }
        return mysystem.NewAdapter(stores.MySystem)
    },
    HasProfileSupport: false, // Set true if implementing ProfileAdapter
},
```

```bash
go build ./...
```

---

## Step 10: Add storage schema

**Migrations**: `internal/services/game/storage/sqlite/migrations/`
**Queries**: `internal/services/game/storage/sqlite/queries/`

1. Create a migration file for your system's projection tables.
2. Add query definitions for CRUD operations.
3. Define the store interface in `internal/services/game/storage/`.
4. Implement the store in `internal/services/game/storage/sqlite/`.

---

## Step 11: Add conformance test

**File**: `internal/services/game/domain/bridge/<system>/module_test.go`

```go
func TestMySystemConformance(t *testing.T) {
    testkit.ValidateSystemConformance(t, NewModule(), NewAdapter(fakeStore{}))
}
```

This validates:
- Every emittable event has a fold handler
- Every projection-and-replay event has an adapter handler
- No fold handlers exist for audit-only events
- State factory is deterministic
- Adapter idempotency

```bash
go test ./internal/services/game/domain/bridge/<system>/...
```

---

## Step 12: Add write-path architecture guard

**File**: `internal/services/game/api/grpc/systems/<system>/write_path_arch_test.go`

```go
func TestMySystemWritePathArchitecture(t *testing.T) {
    testkit.ValidateWritePathArchitecture(t, testkit.WritePathPolicy{
        HandlerDir: handlerDir(t),
        StoreMutationSubstrings: []string{
            ".PutMySystem",
            ".UpdateMySystem",
            ".DeleteMySystem",
        },
    })
}
```

---

## Step 13: Add transport/API handlers

**Directory**: `internal/services/game/api/grpc/systems/<system>/`

Create gRPC service handlers for your system's mechanics. Each handler
should use shared domain write helpers — no direct `Domain.Execute` or
inline event appends.

---

## Step 14: Full verification

```bash
# Regenerate event catalogs (required after adding/removing command/event types)
go run ./internal/tools/eventdocgen

# Unit tests
make test

# Integration tests (gRPC + MCP + storage + event catalog)
make integration

# Coverage impact
make cover
```

**Note**: `make integration` runs `event-catalog-check` which verifies
that generated files in `docs/events/` are up to date. If you added or
changed command/event types without regenerating, CI will fail with a
`git diff --exit-code` error on `docs/events/`.

---

## Review checklist (from game-systems.md step 10)

Before submitting for review, confirm:

- [ ] All emitted events registered with explicit intent
- [ ] Decider outputs round-trip through `BuildRegistries` and replay
- [ ] Adapter coverage tests prove every projection-and-replay event is handled
- [ ] Payload validators reject malformed envelopes
- [ ] Event types documented in generated `docs/events/` catalogs
- [ ] At least one happy-path and one rejection test per command/event pair
- [ ] Event payloads use absolute values, not deltas

---

## Reference files (Daggerheart)

| Concern | File |
|---------|------|
| Module wiring | `domain/bridge/daggerheart/module.go` |
| Decider | `domain/bridge/daggerheart/decider.go` |
| Folder | `domain/bridge/daggerheart/folder.go` |
| Adapter | `domain/bridge/daggerheart/adapter.go` |
| Event types | `domain/bridge/daggerheart/event_types.go` |
| Payloads | `domain/bridge/daggerheart/payload.go` |
| Registry system | `domain/bridge/daggerheart/registry_system.go` |
| State factory | `domain/bridge/daggerheart/state_factory.go` |
| Manifest | `domain/bridge/manifest/manifest.go` |
| gRPC handlers | `internal/services/game/api/grpc/systems/daggerheart/` |
| Conformance test | `domain/bridge/daggerheart/module_test.go` |
| Arch guard | `internal/services/game/api/grpc/systems/daggerheart/write_path_arch_test.go` |

## Where to go next

- [Quick start](quick-start-system-developer.md) — Worked example of a
  single mechanic end-to-end
- [Game systems](game-systems.md) — Architecture, ownership boundaries,
  helper decision trees
- [Event-driven system](event-driven-system.md) — Write-path invariants,
  replay semantics
- [Domain language](domain-language.md) — Canonical naming principles
