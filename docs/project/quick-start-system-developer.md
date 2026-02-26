---
title: "Quick start: system developer"
parent: "Project"
nav_order: 1
---

# Quick Start: System Developer

Your first 30 minutes adding a game system to Fracturing.Space.

```
quick-start (you are here)
  -> event-driven-system.md   (write-path invariants)
  -> game-systems.md          (implementation checklist)
  -> domain-language.md       (canonical naming)
```

## Architecture in 30 seconds

Every mechanic follows one path:

```
Command  ->  Decider  ->  Event(s)  ->  Fold (aggregate)
                                    ->  Adapter (projection)
```

A **command** carries intent ("deal 5 damage"). The **decider** validates it
against aggregate state and emits **events** ("damage applied, 5 HP"). Events
are journaled, then applied in two directions: **fold** rebuilds aggregate
state for future decisions; **adapter** writes denormalized read models for
queries.

## Your system lives in 3 places

| Layer | Path | Defines |
|-------|------|---------|
| Write model | `domain/bridge/{system}/module.go` | Commands, events, decider, folder, state factory |
| Projection adapter | `domain/bridge/{system}/adapter.go` | Read-model handlers for system events |
| API | `api/grpc/systems/{system}/` | gRPC service, mechanics, outcome tools |

All three are wired together by a single `SystemDescriptor` in
`domain/bridge/manifest/manifest.go`.

## The 5 helpers you will use

### DecideFunc — single-event command decisions

```go
decision := module.DecideFunc[MyPayload](cmd, eventType, entityType, entityID, validate, now)
```

Unmarshals the command payload, runs your `validate` function, and emits one
event. Use `DecideFuncWithState` when validation needs current aggregate state,
or `DecideFuncTransform` when the event payload differs from the command
payload.

### FoldRouter — typed event folding

```go
r := module.NewFoldRouter[*MyState](assertMyState)
module.HandleFold(r, EventTypeDamageApplied, func(s *MyState, p DamagePayload) error {
    s.HP -= p.Amount
    return nil
})
```

Dispatches events by type to typed handler functions. Eliminates manual
unmarshal and type-switch boilerplate.

### AdapterRouter — typed projection handling

```go
r := module.NewAdapterRouter()
module.HandleAdapter(r, EventTypeDamageApplied, func(ctx context.Context, evt event.Event, p DamagePayload) error {
    return store.UpdateHP(ctx, evt.CampaignID, evt.EntityID, p.NewHP)
})
```

Same dispatch pattern as FoldRouter, but for projection writes.

### SystemDescriptor — manifest registration

```go
manifest.SystemDescriptor{
    ID:      "my_system",
    Version: "v1",
    BuildModule:         func() module.Module { return NewModule() },
    BuildMetadataSystem: func() bridge.GameSystem { return NewGameSystem() },
    BuildAdapter:        func(s manifest.ProjectionStores) bridge.Adapter { return NewAdapter(s.MySystem) },
}
```

### ValidateSystemConformance — startup safety net

```go
func TestMySystemConformance(t *testing.T) {
    testkit.ValidateSystemConformance(t, NewModule(), NewAdapter(fakeStore{}))
}
```

Asserts every emittable event has a fold handler and an adapter handler, every
command has a decider case, and the state factory is deterministic.

## Your first mechanic end-to-end

1. **Define the event type** in your system's `event_types.go`.
2. **Register the command** in `Module.RegisterCommands()`.
3. **Register the event** in `Module.RegisterEvents()` and add it to
   `EmittableEventTypes()`.
4. **Add the decider case** using `DecideFunc` (or a variant).
5. **Add the fold handler** via `HandleFold` on your `FoldRouter`.
6. **Add the adapter handler** via `HandleAdapter` on your `AdapterRouter`.
7. **Add the manifest entry** (if new system) or verify the existing
   `SystemDescriptor` covers your event.

## Verification

```bash
# Conformance test catches registration gaps
go test ./internal/services/game/domain/module/testkit/...

# Full unit suite
make test

# Integration tests (gRPC + MCP + storage + event catalog)
make integration

# Coverage impact
make cover
```

## Generated event catalogs

After adding or removing command/event types, regenerate the catalogs:

```bash
go run ./internal/tools/eventdocgen
```

Commit the updated files in `docs/events/`. CI enforces this via
`make integration` which runs `event-catalog-check` before tests.

## Worked example: Countdown Create

This traces `sys.daggerheart.countdown.create` through every layer to show
how the 5 helpers compose in real code.

### Step 1 — Command type constant

`domain/commandids/ids.go` declares the shared constant:

```go
DaggerheartCountdownCreate command.Type = "sys.daggerheart.countdown.create"
```

The module's `decider.go` imports it as a package-local alias:

```go
commandTypeCountdownCreate command.Type = commandids.DaggerheartCountdownCreate
```

### Step 2 — Command registration

`module.go` declares command definitions in a slice and registers them in
`RegisterCommands`:

```go
{Type: commandTypeCountdownCreate, Owner: command.OwnerSystem, ValidatePayload: validateCountdownCreatePayload},
```

### Step 3 — Event type constant and registration

`event_types.go`:

```go
EventTypeCountdownCreated event.Type = "sys.daggerheart.countdown_created"
```

`module.go` registers it with `IntentProjectionAndReplay` — it needs both
aggregate fold and projection writes:

```go
{Type: EventTypeCountdownCreated, Owner: event.OwnerSystem, ValidatePayload: validateCountdownCreatedPayload, Intent: event.IntentProjectionAndReplay},
```

### Step 4 — Decider (using `DecideFunc`)

`decider_countdowns.go` — no aggregate state needed, so this uses the simplest
helper:

```go
func decideCountdownCreate(cmd command.Command, now func() time.Time) command.Decision {
    return module.DecideFunc(cmd, EventTypeCountdownCreated, "countdown",
        func(p *CountdownCreatePayload) string { return strings.TrimSpace(p.CountdownID) },
        func(p *CountdownCreatePayload, _ func() time.Time) *command.Rejection {
            p.CountdownID = strings.TrimSpace(p.CountdownID)
            p.Name = strings.TrimSpace(p.Name)
            // ... normalize fields ...
            return nil
        }, now)
}
```

`DecideFunc` unmarshals the command payload, calls `validate`, and emits one
event of type `EventTypeCountdownCreated` with entity type `"countdown"`.

### Step 5 — Fold handler (using `FoldRouter`)

`folder.go` registers a typed fold handler:

```go
module.HandleFold(r, EventTypeCountdownCreated, foldCountdownCreated)
```

The handler upserts into aggregate state using absolute values:

```go
func foldCountdownCreated(state *SnapshotState, payload CountdownCreatedPayload) error {
    applyCountdownUpsert(state, payload.CountdownID, func(cs *CountdownState) {
        cs.Name = payload.Name
        cs.Current = payload.Current
        cs.Max = payload.Max
        // ... set all fields from payload ...
    })
    return nil
}
```

### Step 6 — Adapter handler (using `AdapterRouter`)

`adapter.go` registers a typed projection handler:

```go
module.HandleAdapter(r, EventTypeCountdownCreated, a.handleCountdownCreated)
```

The handler writes a single projection row:

```go
func (a *Adapter) handleCountdownCreated(ctx context.Context, evt event.Event, payload CountdownCreatedPayload) error {
    return a.store.PutDaggerheartCountdown(ctx, storage.DaggerheartCountdown{
        CampaignID:  evt.CampaignID,
        CountdownID: payload.CountdownID,
        Name:        payload.Name,
        // ... map all fields ...
    })
}
```

### Full flow summary

```
sys.daggerheart.countdown.create (command)
  → decideCountdownCreate                    (DecideFunc)
    → sys.daggerheart.countdown_created      (event, appended to journal)
      → foldCountdownCreated                 (FoldRouter → aggregate state)
      → handleCountdownCreated               (AdapterRouter → projection store)
```

## Where to go next

- [Event-driven system](event-driven-system.md) — Write-path invariants,
  command lifecycle, replay semantics
- [Game systems](game-systems.md) — Full implementation checklist, ownership
  boundaries, adapter patterns
- [Domain language](domain-language.md) — Canonical terms, naming principles
- [Event replay](event-replay.md) — Journal format, replay modes, snapshots
