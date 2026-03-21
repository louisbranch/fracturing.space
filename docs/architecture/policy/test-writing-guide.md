---
title: "Test writing guide"
parent: "Policy and quality"
nav_order: 4
status: canonical
owner: engineering
last_reviewed: "2026-03-19"
---

# Test writing guide

Practical guide for writing tests in the game service codebase.

## Test levels

Choose the right level for what you are testing:

| Level | When to use | Location |
|-------|------------|----------|
| Unit | Deterministic domain logic, pure functions, state machines | `*_test.go` next to source |
| Integration | Component seams, store interactions, multi-step workflows | `*_test.go` in test packages or `gametest/` |
| Scenario | Critical user/system paths, end-to-end transport flows | `internal/test/game/scenarios/` |

Prefer the lowest level that gives confidence. Unit tests are fast and
deterministic. Integration tests catch wiring issues. Scenario tests validate
real transport paths but are slower.

## Fixture patterns

### Store fakes

Fake stores live in `internal/test/mock/gamefakes/`. They implement store
interfaces with in-memory state and support error injection via `*error` fields.

```go
fake := &gamefakes.FakeCampaignStore{}
fake.GetCampaignErr = errors.New("store unavailable")
```

Use fakes for unit tests that need store interactions without a real database.

### Integration fixtures

Integration tests use `gametest` helpers for common setup patterns:

```go
scenario := gametest.NewCampaignScenario(t).
    WithParticipant("gm", participant.RoleGM).
    WithCharacter("char-1").
    WithActiveSession().
    Build()
```

When `gametest` does not cover your setup needs, write imperative setup but
keep it focused on the specific scenario being tested.

### Daggerheart test helpers

Daggerheart-specific test utilities live in the system's `testkit/` package.
Use these for system-specific state construction and validation.

## Assertion best practices

### Assert on codes, not messages

Error messages are implementation details. Assert on structured codes:

```go
// Prefer: assert on rejection code.
if result.RejectionCode != campaign.RejectionNotFound {
    t.Fatalf("got code %q, want %q", result.RejectionCode, campaign.RejectionNotFound)
}

// Avoid: assert on exact error string.
if err.Error() != "campaign not found" {
    t.Fatalf("unexpected error: %v", err)
}
```

### Negative assertions need rationale

Every negative assertion must include an `Invariant:` comment explaining why
the absence matters:

```go
// Invariant: rejected commands must not emit domain events.
if len(result.Events) != 0 {
    t.Fatal("expected no events on rejection")
}
```

### Table-driven tests

Use table-driven tests for functions with multiple input variations:

```go
tests := []struct {
    name   string
    input  Input
    want   Output
}{
    {"valid input", validInput(), expectedOutput()},
    {"empty name rejected", emptyNameInput(), rejectedOutput()},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got := function(tt.input)
        assertEqual(t, got, tt.want)
    })
}
```

## What not to test

- Thin wiring code (server composition, main functions)
- Generated code (protobuf, sqlc, templ outputs)
- Internal implementation details that may change without affecting behavior
- Exact error message strings (use codes instead)

## Test file organization

| Pattern | Purpose |
|---------|---------|
| `foo_test.go` | Unit tests for `foo.go`, same package |
| `foo_integration_test.go` | Integration tests requiring real dependencies |
| `gametest/` | Shared test fixtures and builders |
| `internal/test/game/scenarios/` | End-to-end scenario tests |
| `internal/test/mock/gamefakes/` | Fake store implementations |
| `testkit/` (per system) | System-specific test utilities |

## Coverage

Coverage is a guardrail, not a target. The CI pipeline enforces non-regression
baselines from `main`. Focus on testing meaningful behavior rather than
maximizing line counts.

See [Testing policy](testing-policy.md) for CI gates and coverage floor policy.

## Related docs

- [Testing policy](testing-policy.md)
- [Validation boundaries](../foundations/validation-boundaries.md)
- [Event-driven system](../foundations/event-driven-system.md)
