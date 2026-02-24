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

## Where to go next

- [Event-driven system](event-driven-system.md) — Write-path invariants,
  command lifecycle, replay semantics
- [Game systems](game-systems.md) — Full implementation checklist, ownership
  boundaries, adapter patterns
- [Domain language](domain-language.md) — Canonical terms, naming principles
- [Event replay](event-replay.md) — Journal format, replay modes, snapshots
