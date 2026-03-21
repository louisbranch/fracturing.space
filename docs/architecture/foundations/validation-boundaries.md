---
title: "Validation boundaries"
parent: "Foundations"
nav_order: 9
status: canonical
owner: engineering
last_reviewed: "2026-03-19"
---

# Validation boundaries

Where and why input validation happens at each layer of the game service.

## Two-layer validation model

The game service validates inputs at two distinct boundaries with different
responsibilities:

| Layer | Responsibility | Rejects with |
|-------|---------------|-------------|
| Transport (gRPC handlers) | Input shape, bounds, and well-formedness | gRPC `InvalidArgument` status |
| Domain (deciders) | Semantic invariants, state-dependent rules | Domain rejection codes |

These boundaries are intentionally separate. Transport validation protects the
domain from malformed inputs. Domain validation enforces business rules that
depend on aggregate state.

## Transport validation

Transport handlers validate **syntactic** properties before constructing a
domain command:

- Required fields are present and non-empty (after trimming whitespace)
- String lengths and numeric ranges are within protocol-defined bounds
- Enum values are recognized members of their type
- ID formats are structurally valid
- Collection sizes do not exceed transport limits

Transport validation never reads aggregate state. It answers "is this a
well-formed request?" without considering whether the request makes semantic
sense.

```go
// Transport: validate input shape before building a command.
if strings.TrimSpace(req.GetCampaignId()) == "" {
    return status.Error(codes.InvalidArgument, "campaign_id is required")
}
```

## Domain validation

Domain deciders validate **semantic** properties using current aggregate state:

- The referenced entity exists in the aggregate
- The requested state transition is valid (e.g., session must be active)
- Business invariants hold (e.g., character cannot equip incompatible items)
- Authorization-dependent rules (e.g., only the character owner can transfer)

Domain validation produces rejection codes that are stable, machine-readable
identifiers. The transport layer maps these codes to gRPC statuses and
user-facing messages.

```go
// Domain: validate semantic invariant using aggregate state.
if _, ok := state.Characters[cmd.CharacterID]; !ok {
    return command.Reject(RejectionCharacterNotFound)
}
```

## What belongs where

| Check | Layer | Reason |
|-------|-------|--------|
| "campaign_id is required" | Transport | Syntactic: missing field |
| "name must be <= 200 chars" | Transport | Syntactic: bounds check |
| "campaign does not exist" | Domain | Requires state lookup |
| "session is not active" | Domain | State-dependent invariant |
| "character already equipped" | Domain | Business rule |
| "invalid enum value" | Transport | Syntactic: unrecognized input |

## Daggerheart system validation

The Daggerheart game system adds a third validation point within the domain
layer. System validators run after transport validation but before the system
decider, validating payload structure specific to the game system:

```
transport validate -> build command -> system validate payload -> system decide
```

System validators use `ValidatePayload[P]()` to unmarshal and check
system-specific payload fields. This keeps game-system concerns out of the
core domain decider while maintaining the transport/domain boundary.

## Anti-patterns

- **Domain checks in transport**: Do not query aggregate state in handlers.
  Transport should only validate what it can see in the request message.
- **Transport checks in domain**: Do not re-validate field presence or bounds
  in deciders. Trust that transport has already enforced shape constraints.
- **Duplicated validation**: If both layers check the same condition, one is
  unnecessary. Determine which layer owns the check and remove the duplicate.

## Related docs

- [Event-driven system](event-driven-system.md)
- [gRPC write path](grpc-write-path.md)
- [Testing policy](../policy/testing-policy.md)
