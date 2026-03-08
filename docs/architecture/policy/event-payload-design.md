---
title: "Event payload design"
parent: "Policy and quality"
nav_order: 3
status: canonical
owner: engineering
last_reviewed: "2026-03-07"
---

# Event Payload Design

Conventions for structuring command and event payload structs in game system
modules.

## Command vs event payloads

Every mutation has two payload structs:

| Struct | Role | Example |
|--------|------|---------|
| `FooPayload` | Command input: what the caller requests | `CharacterStatePatchPayload` |
| `FooedPayload` | Event output: what actually happened | `CharacterStatePatchedPayload` |

**Command payloads** carry the full decision context: Before/After pairs, source
metadata, and any fields the decider needs to validate or compute the outcome.

**Event payloads** carry only authoritative outcome data: the resulting state
after the mutation. They never include Before fields — those are
decision-time context, not durable facts.

## Go field naming

Event payload Go field names use short names without `Before` or `After`
suffixes:

```go
// Command payload (decision input):
type CharacterStatePatchPayload struct {
    CharacterID string `json:"character_id"`
    HPBefore    *int   `json:"hp_before,omitempty"`
    HPAfter     *int   `json:"hp_after,omitempty"`
}

// Event payload (authoritative outcome):
type CharacterStatePatchedPayload struct {
    CharacterID string `json:"character_id"`
    HP          *int   `json:"hp_after,omitempty"`
}
```

Rules:

1. Event struct Go field names are short: `HP`, `Hope`, `Stress`, `Value`, etc.
2. JSON tags retain `_after` suffix for serialization compatibility with stored
   events.
3. Command struct Go field names keep `Before`/`After` suffixes since both are
   semantically relevant.
4. Local variables that hold event field values should also use short names
   (`hp`, `hope`) rather than `hpAfter`, `hopeAfter`.

## DecideFuncTransform pattern

When command and event payloads differ, use `DecideFuncTransform` to map
command → event:

```go
deciders.DecideFuncTransform(
    EventTypeFooChanged,
    func(_ SnapshotState, _ bool, cmd FooPayload) FooChangedPayload {
        return FooChangedPayload{
            Value: cmd.ValueAfter,
        }
    },
    validateFooPayload,
)
```

The transform function:

- Receives the validated command payload.
- Returns the event payload with short field names and no Before fields.
- Keeps the mapping explicit and co-located with the event type registration.

When command and event payloads are identical (no Before/After split needed),
use a type alias: `type FooedPayload = FooPayload`.

## Before field removal rationale

Before fields in events are redundant — they duplicate state that already
exists in the projection at the time the event is applied. Including them
creates problems:

1. **Stale coupling**: Before values are captured at decision time. If replay
   order changes or events are reprocessed, the Before value in the payload may
   not match the actual prior state.
2. **Payload bloat**: Every mutation carries twice the data it needs.
3. **False authority**: Consumers may treat Before values as ground truth instead
   of reading projected state.

Projectors and adapters should read current state from the projection, not from
event payloads. Timeline or audit displays that need "X → Y" transitions should
compute them from projected state history, not from embedded Before fields.

## Replay-safety notes

See [Event payload change policy](event-payload-change-policy.md) for rules on
adding, removing, or renaming payload fields — particularly around `omitempty`,
pointer fields, and backward compatibility with stored events.
