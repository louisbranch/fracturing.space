---
title: "Game systems"
parent: "Project"
nav_order: 5
---

# Game Systems Architecture

This document explains how a game system extends the shared event-driven core.
For the command-to-event-to-projection lifecycle, read
[Event-driven system](event-driven-system.md) first.
For the high-level design checklist, see [systems checklist](systems.md).
For exact runtime-registered command/event contracts, use generated docs in
[`docs/events/`](../events/index.md).

## Design goals

Fracturing.Space keeps core campaign/session behavior system-agnostic while
allowing each ruleset to own its mechanics.

Benefits:

1. Shared infrastructure for campaigns, participants, sessions, and event
   journaling.
2. System-specific mechanics without polluting core domain packages.
3. Predictable replay and projection behavior across all systems.
4. Independent evolution of systems by version (`system_id + system_version`).

## Ownership boundaries

System support is built on ownership rules:

- Core-owned commands/events: campaign/session/participant/invite/character
  lifecycle.
- System-owned commands/events: mechanics and state transitions specific to a
  game system.

Invariants:

- Core must not emit system-owned events.
- Systems must not emit core-owned events.
- System-owned envelopes must include both `system_id` and `system_version`.
- System-owned type names must match the system namespace:
  `sys.<system_id>.*`.
- Core treats system payloads as opaque and routes them to system handlers.

## Runtime extension surfaces

There are two extension surfaces with different responsibilities.

### Domain module registry (write-path routing)

Location: `internal/services/game/domain/system/registry.go`

Used by the domain engine to:

- register system-owned command and event definitions
- route system commands to module deciders
- route system events to module folders during replay/fold

Module interface:

```go
type Module interface {
    ID() string
    Version() string
    RegisterCommands(registry *command.Registry) error
    RegisterEvents(registry *event.Registry) error
    EmittableEventTypes() []event.Type
    Decider() Decider
    Folder() Folder
    StateFactory() StateFactory
}
```

### System bridge and projection adapters

Locations:

- `internal/services/game/domain/bridge/registry_bridge.go`
- `internal/services/game/domain/bridge/adapter_registry.go`

Used for:

- exposing game-system metadata through API surfaces
- applying system-owned events into system-specific projection tables

Projection adapter interface:

```go
type Adapter interface {
    ID() string
    Version() string
    Apply(context.Context, event.Event) error
    Snapshot(context.Context, string) (any, error)
    HandledTypes() []event.Type
}
```

## Registry map

Three registries serve different purposes. All are populated from a single
`SystemDescriptor` entry in `manifest/manifest.go`.

```
SystemDescriptor (manifest/manifest.go)
├── BuildModule()         → module.Registry      (write-path routing)
│   ├── RegisterCommands  → command.Registry      (shared with core)
│   ├── RegisterEvents    → event.Registry        (shared with core)
│   ├── Decider()         → command decisions
│   └── Folder()          → aggregate fold / replay
│
├── BuildMetadataSystem() → bridge.Registry      (API metadata bridge)
│   └── Maps system_id + version to protobuf enum for transport layers
│
└── BuildAdapter()        → bridge.AdapterRegistry (read-path projections)
    └── Applies system events to denormalized projection tables
```

- **`module.Registry`** (`domain/module/registry.go`): Routes system commands to
  deciders and system events to folders. Used by the domain engine write-path
  and the replay pipeline.
- **`bridge.Registry`** (`domain/bridge/registry_bridge.go`): Maps
  `system_id` + `system_version` to protobuf `GameSystem` enums. Used by gRPC
  and MCP transport layers to expose system metadata.
- **`bridge.AdapterRegistry`** (`domain/bridge/adapter_registry.go`): Routes
  system-owned events to projection adapters that write to system-specific read
  tables. Used by `projection.Applier.applySystemEvent`.

When adding a new system, register all three surfaces from one
`SystemDescriptor` so they stay aligned.

### StateFactory: two interfaces, different layers

The codebase has two `StateFactory` interfaces that serve different layers:

| Interface | Package | Returns | Used by |
|---|---|---|---|
| `module.StateFactory` | `domain/module` | `(any, error)` | Write-path aggregate fold / replay |
| `bridge.StateFactory` | `domain/bridge` | Typed handlers (`CharacterStateHandler`, `SnapshotStateHandler`) | API bridge (gRPC/MCP transport) |

New systems typically implement only the `module.StateFactory` variant. The
`bridge.StateFactory` is satisfied by the metadata/registry system
implementation (e.g. `DaggerheartRegistrySystem`) which wraps domain state
behind the resource/damage abstractions the API layer needs.

## Where systems plug in

Core registration entrypoint:

- `internal/services/game/domain/engine/registries.go`

Server wiring entrypoints:

- domain module registration: `internal/services/game/app/domain.go`
- projection adapter registration: `internal/services/game/api/grpc/game/system_adapters.go`

## Adding a new system (current flow)

### 1. Add identity and versioning

- Add enum values in `api/proto/common/v1/game_system.proto`.
- Run `make proto`.
- Define stable `SystemID` and `SystemVersion` constants in your module package.

### 2. Implement a domain module

Create `internal/services/game/domain/bridge/{system}/module.go` that
implements `system.Module`.

Responsibilities:

- register system-owned command definitions
- register system-owned event definitions
- provide decider/folder/state-factory implementations

#### Infrastructure helpers for system authors

The `module` package provides typed helpers that eliminate boilerplate in
deciders, folders, and adapters. New system authors should use these
instead of writing raw switch/unmarshal code:

- **`module.FoldRouter[S]`** with **`module.HandleFold[S, P]`**: typed fold
  dispatch by event type. Auto-unmarshals payloads into `P`, calls a typed
  handler `func(S, P) error`. Eliminates the per-case unmarshal switch in
  `Fold`. See `daggerheart/projector.go` for usage.

- **`module.AdapterRouter`** with **`module.HandleAdapter[P]`**: typed adapter
  dispatch by event type. Auto-unmarshals payloads, calls a typed handler
  `func(context.Context, event.Event, P) error`. Eliminates unmarshal
  boilerplate in projection adapters. See `daggerheart/adapter.go` for usage.

- **`module.DecideFunc[P]`** / **`module.DecideFuncWithState[S, P]`**: typed
  decider helpers for the common case where one command type maps to one event
  type with the same payload. Handles unmarshal, validation, and event
  construction.

- **`module.DecideFuncTransform[S, PIn, POut]`**: like `DecideFuncWithState`
  but for cases where the emitted event payload type (`POut`) differs from the
  command payload type (`PIn`). Adds a `transform` function to convert between
  them. See `daggerheart/decider.go` for GM fear, hope spend, and stress spend
  cases.

#### DecideFunc decision tree

Use this tree to pick the right helper for a new command handler:

```
Does your command need aggregate snapshot state?
  NO  ─→ Does command payload == event payload?
           YES → DecideFunc[P]                    (e.g. decideLoadoutSwap)
           NO  → Raw Decide switch                (e.g. decideRestTake, multi-event)
  YES ─→ Does event payload differ from command payload?
           YES → DecideFuncTransform[S, PIn, POut] (e.g. decideHopeSpend)
           NO  → DecideFuncWithState[S, P]          (e.g. decideCharacterStatePatch)
```

- **`DecideFunc[P]`**: simplest path — one command type, one event type, same
  payload, no state needed. Just unmarshal, validate, emit.
- **`DecideFuncWithState[S, P]`**: same as above but receives snapshot state
  for idempotency/validation checks (e.g. rejecting no-op mutations).
- **`DecideFuncTransform[S, PIn, POut]`**: command payload and event payload
  are different types. Adds a `transform` function to convert between them
  (e.g. `HopeSpendPayload` → `CharacterStatePatchedPayload`).
- **Raw `Decide` switch**: when one command emits multiple events or needs
  custom routing logic that helpers can't express (e.g. `rest.take` emits both
  `rest_taken` and optionally `countdown_updated`).

#### System state lifecycle

When the aggregate folder encounters the first event for a given
`(system_id, system_version)` pair, it calls `StateFactory.NewSnapshotState`
to seed the initial system state. All subsequent events for the same key
fold into that state. System authors don't need to manually initialize
state — the aggregate folder handles lazy creation.

`NewSnapshotState` must be deterministic: given the same `campaign_id`, it
must return the same initial state, because replay depends on this guarantee.

### 3. Define payload contracts and validation

- Add payload structs for command/event types.
- Validate payload shape and invariants in registry validators.
- Keep validation deterministic and replay-safe.

### 4. Wire into engine startup

Pass your module to `engine.BuildRegistries(...)` in
`internal/services/game/app/domain.go`.

### 5. Register system entry metadata centrally (start here)

> **Start here.** `manifest.go` is the single source of truth for system
> registration. Open it first when onboarding — it shows all three registry
> surfaces (`BuildModule`, `BuildMetadataSystem`, `BuildAdapter`) wired from
> one `SystemDescriptor`, and serves as a map of what exists.

Add or verify your system's `SystemDescriptor` entry in
`internal/services/game/domain/bridge/manifest/manifest.go` so `Modules()`,
`MetadataSystems()`, and `AdapterRegistry(...)` stay aligned.

### 6. Implement system projection adapter

Create `internal/services/game/domain/bridge/{system}/adapter.go` implementing
`systems.Adapter`, then register it in
`internal/services/game/api/grpc/game/system_adapters.go`.

#### When to implement ProfileAdapter

If your system stores per-character data inside `system_profile` (the
system-specific section of a character profile), implement `bridge.ProfileAdapter`
on your projection adapter. The projection applier calls `ApplyProfile` when a
`character.profile_updated` event arrives, passing the system-specific profile
data for your system ID.

```go
type ProfileAdapter interface {
    ApplyProfile(ctx context.Context, campaignID, characterID string, profileData json.RawMessage) error
}
```

Implement this when your system needs to:
- denormalize character profile fields into system-specific projection tables
- keep system state in sync with profile updates (e.g. class, level, traits)

The projection applier iterates all system entries in the profile's
`system_profile` map and delegates each to the corresponding adapter via
`bridge.ProfileAdapter`. See `projection/apply_character.go` for the dispatch
logic.

### 7. Add storage schema and queries

- Add migrations in `internal/services/game/storage/sqlite/migrations/`.
- Add query definitions in `internal/services/game/storage/sqlite/queries/`.
- Extend storage interfaces and conversion helpers.

### 8. Add transport/API handlers

- gRPC endpoints: `internal/services/game/api/grpc/systems/{system}/`.
- MCP mappings (if needed): `internal/services/mcp/`.

### 9. Verify through tests

- Unit tests for decider/folder/validators.
- Projection tests for adapter/store behavior.
- Integration tests across gRPC/MCP + storage.

### 10. Confirm system extension contract (required in reviews)

- All emitted system events are registered with explicit intent (`projection_and_replay`
  vs `audit_only`) so projection obligations are discoverable from registries.
- Core decider outputs have a test that new events can be round-tripped through
  `BuildRegistries` and replayed once in folder tests.
- Core applier and adapter coverage tests prove every `projection_and_replay` event
  is handled (or intentionally ignored by intent).
- Command builders and payload validators reject malformed envelopes with clear
  errors.
- New system event types are documented in both:
  - command/event runtime registrations
  - generated event docs in `docs/events/` (`command-catalog.md`,
    `event-catalog.md`, `usage-map.md`).
- New code path is exercised with at least one happy-path and one rejection/edge-case
  test for each new command and event pairing.

This checklist should be part of review for each new game-system module.

### Runtime execution diagram

```mermaid
sequenceDiagram
    autonumber
    participant API as systems/{system} gRPC handler
    participant ENG as Domain Engine
    participant MOD as system.Module decider
    participant ES as Event Store
    participant AP as projection.Applier
    participant AD as systems.Adapter

    API->>ENG: Execute(system command)
    ENG->>MOD: Decide(...)
    MOD-->>ENG: system-owned event(s)
    ENG->>ES: AppendEvent(seq/hash/signature)
    ES-->>ENG: stored events
    API->>AP: Apply(event)
    AP->>AD: applySystemEvent(system_id/version)
    AD-->>AP: system projection updated
```

## Daggerheart reference implementation

Use Daggerheart as the baseline for structure and naming.

| Concern | Location |
|---|---|
| Module wiring | `internal/services/game/domain/bridge/daggerheart/module.go` |
| Command decisions | `internal/services/game/domain/bridge/daggerheart/decider.go` |
| Replay folder (Fold method) | `internal/services/game/domain/bridge/daggerheart/projector.go` |
| Projection adapter | `internal/services/game/domain/bridge/daggerheart/adapter.go` |
| Event type constants | `internal/services/game/domain/bridge/daggerheart/event_types.go` |
| Payload contracts | `internal/services/game/domain/bridge/daggerheart/payload.go` |
| gRPC system handlers | `internal/services/game/api/grpc/systems/daggerheart/` |

## Consistency expectations for system authors

1. Module deciders and folders must be deterministic.
2. System events must be replay-safe and self-describing through payload + metadata.
3. Projection adapter behavior must be idempotent under replay.
4. Domain writes must happen through commands/events only, never direct projection mutation.

## Event timeline contract requirement (for every new mechanic)

Before implementing a new mechanic, define its timeline contract:

`request -> command -> emitted event(s) -> projection targets -> apply mode -> invariants`

Required process:

1. Add or update a timeline row in the system contract doc before code changes.
2. Ensure handler code uses shared execute-and-apply orchestration.
3. Ensure system-owned side effects are emitted via explicit `sys.*` commands,
   not embedded as system-owned effects in core command payloads.
4. Add/update guard tests that prevent bypass patterns.

For Daggerheart, use:
[Daggerheart Event Timeline Contract](daggerheart-event-timeline-contract.md).

Bypass patterns that are not allowed in request handlers:

1. Direct event append APIs from handler code.
2. Direct projection/store mutation for mutating domain outcomes.
3. Local duplicated execute/reject/apply loops instead of shared orchestration helpers.

## Projection replay safety

System adapters apply events into denormalized projection tables. Several
adapter handlers read existing projection state before writing (e.g. reading
a current value to compute a delta). This is safe because:

1. **Ordered replay**: Events replay in strict sequence order, so the
   read-then-write chain reproduces the same final state deterministically.
2. **Full replay recovers from corruption**: If projection state becomes
   inconsistent, a full replay from the journal rebuilds it from scratch.

Rules for adapter authors:

- **Do not cache state across `Apply` calls.** Each call must read fresh
  projection state. Stale in-memory caches can diverge during replay.
- **Handle `storage.ErrNotFound` gracefully.** During replay the projection
  table may be empty. Seed defaults when a record does not exist yet.
- **Projection adapter behavior must be idempotent under replay.** The same
  event sequence must produce the same projection state regardless of how many
  times it is replayed.
- **Never write projection records outside of adapter `Apply`.** All
  projection mutations for system events must flow through the adapter
  dispatch path so replay remains the single source of truth.

## Idempotency testing for system adapters

System adapter `Apply()` methods must be idempotent: replaying the same event
sequence must produce identical projection state regardless of how many times
it runs. The most common mistake is an adapter that increments a counter or
appends to a list instead of setting an absolute value.

### Test pattern

Use this pattern to verify idempotency for each adapter handler:

```go
func TestAdapter_Apply_Idempotent(t *testing.T) {
    ctx := context.Background()
    store := newFakeStore()
    adapter := NewAdapter(store)

    evt := event.Event{
        CampaignID:    "camp-1",
        Seq:           1,
        Type:          EventTypeMyAction,
        SystemID:      SystemID,
        SystemVersion: SystemVersion,
        EntityType:    "character",
        EntityID:      "char-1",
        PayloadJSON:   []byte(`{"hp_after": 5}`),
    }

    // First apply.
    if err := adapter.Apply(ctx, evt); err != nil {
        t.Fatalf("first apply: %v", err)
    }
    stateAfterFirst := store.Get(ctx, "camp-1", "char-1")

    // Second apply of the same event.
    if err := adapter.Apply(ctx, evt); err != nil {
        t.Fatalf("second apply: %v", err)
    }
    stateAfterSecond := store.Get(ctx, "camp-1", "char-1")

    if !reflect.DeepEqual(stateAfterFirst, stateAfterSecond) {
        t.Fatalf("adapter is not idempotent:\n  first:  %+v\n  second: %+v",
            stateAfterFirst, stateAfterSecond)
    }
}
```

Key points:

- Apply the same event twice (not two different events).
- Compare projection state after each apply using `reflect.DeepEqual`.
- Use absolute values in payloads (e.g. `hp_after`) rather than deltas to
  make idempotency natural.
- See the Daggerheart adapter tests for working examples.

## Common failure modes

1. Missing `system_id/system_version` on system-owned envelopes:
   - command/event registry validation rejects writes.
2. Registering command types without event types (or vice versa):
   - runtime route failures or replay failures.
3. Forgetting projection adapter registration:
   - events append, but system projection state does not update.
4. Non-deterministic decider/folder code:
   - replay divergence and integrity incidents.

## Related docs

- Event lifecycle and consistency model: [Event-driven system](event-driven-system.md)
- Replay/checkpoint operations: [Event replay](event-replay.md)
- High-level system design checklist: [Systems checklist](systems.md)
