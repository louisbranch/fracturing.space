# Pass 1: `any`-Typed Seams and Type Safety Boundaries

## Summary

The codebase uses `any` at the domain boundary between the generic engine
pipeline and concrete aggregate/system state. This is a deliberate trade-off
documented in `aggregate/doc.go`: Go generics cannot express a heterogeneous map
where each key maps to a different concrete type, so `any` is the narrowest
practical escape hatch.

The architecture contains **three layers** of `any` recovery:

1. **`aggregate.AssertState[T]`** -- generic, reusable, well-tested.
2. **`module.TypedDecider[S]` / `module.TypedFolder[S]` / `module.FoldRouter[S]`** --
   typed wrappers that hide `any` from system authors.
3. **Ad-hoc type switches** in `engine/loader.go`, `engine/core_decider.go`, and
   `checkpoint/memory.go` that duplicate assertion logic.

The overall design is sound and internally consistent. The findings below
are friction points, minor inconsistencies, and opportunities to narrow `any`
further -- none are correctness bugs in production paths.

---

## Findings

### F01 -- Ad-hoc type switches in `ReplayGateStateLoader` duplicate `AssertState` logic

**Category:** anti-pattern, contributor friction
**Files:**
- `engine/loader.go:147-158` (`LoadSession`)
- `engine/loader.go:169-186` (`LoadScene`)

Both methods manually switch on `aggregate.State` and `*aggregate.State`,
including nil-pointer checks and a `default:` branch returning
`"unsupported state type"`. This is exactly what `aggregate.AssertState[State]`
already does -- with better error messages (includes expected vs got types).

**Proposal:** Replace both methods' switch blocks with
`aggregate.AssertState[aggregate.State](state)`, then extract the sub-field.
This eliminates ~30 lines of duplicated assertion logic and ensures error
messages stay consistent.

---

### F02 -- Ad-hoc type switch in `aggregateState()` (CoreDecider) silently swallows type mismatches

**Category:** correctness risk, anti-pattern
**File:** `engine/core_decider.go:49-59`

```go
func aggregateState(state any) aggregate.State {
    switch typed := state.(type) {
    case aggregate.State:
        return typed
    case *aggregate.State:
        if typed != nil {
            return *typed
        }
    }
    return aggregate.State{}
}
```

When state is neither `aggregate.State` nor `*aggregate.State`, this silently
returns a zero-value `aggregate.State`. In a correctly-wired production system
this path is never hit, but the silent fallback masks wiring bugs during
development. `AssertState` returns an error for the same case.

**Proposal:** Either (a) return `(aggregate.State, error)` and propagate to
`Decide` (which already returns `command.Decision` so can produce a rejection),
or (b) log a warning when the default branch is hit. Since this is the decider
path (pre-persist), a rejection is the correct error surface -- matching
`TypedDecider`'s behavior of emitting `STATE_ASSERT_FAILED`.

---

### F03 -- `engine.Folder` (narrow) vs `fold.Folder` (canonical) -- naming confusion

**Category:** contributor friction
**Files:**
- `engine/handler.go:88-90` -- `engine.Folder` interface (Fold only)
- `fold/fold.go:17-20` -- `fold.Folder` interface (Fold + FoldHandledTypes)
- `module/registry.go:43` -- `module.Folder = fold.Folder` (type alias)
- `replay/replay.go:50-51` -- `replay.Folder` interface (Fold only)

There are **three** narrow `Fold`-only interfaces (`engine.Folder`,
`replay.Folder`, and the implicit compatibility with `fold.Folder`) plus the
canonical `fold.Folder` and its alias `module.Folder`. The doc comments on
`engine.Folder` (line 87) and `aggregate.Folder` (line 17) both explain the
naming rationale, but contributors encountering this for the first time must
read all three packages to understand which `Folder` to use where.

**Proposal:** Minimal intervention -- this is intentional interface segregation.
However, adding a cross-reference comment on `replay.Folder` (which currently
has no doc explaining the relationship) would reduce onboarding friction.
Alternatively, both narrow interfaces could be replaced by a single shared
narrow interface in `fold/` alongside the canonical one.

---

### F04 -- `StateFactory` returns `any` with no compile-time connection to the folder's expected type

**Category:** missing best practice, correctness risk
**File:** `module/registry.go:113-116`

```go
type StateFactory interface {
    NewCharacterState(campaignID ids.CampaignID, characterID ids.CharacterID, kind string) (any, error)
    NewSnapshotState(campaignID ids.CampaignID) (any, error)
}
```

Because both methods return `any`, a module author can return a `FooState`
from the factory and register a `FoldRouter[*BarState]` without any compile-time
error. The mismatch surfaces only at runtime when the first event folds.

**Existing mitigation:**
- `ValidateStateFactoryDeterminism` (`engine/registries_validation_system.go:229-267`)
  checks that repeated calls produce identical results but does NOT verify the
  returned type matches the fold router's expected type.
- `testkit.ValidateSystemConformance` runs fold idempotency but only against
  fresh factory state + empty payload, which may not trigger the type mismatch.

**Proposal:** Add a `ValidateStateFactoryFoldTypeCompatibility` startup
validator that calls `factory.NewSnapshotState(...)`, then calls
`folder.Fold(state, zeroEvent)` and checks that the assertion step succeeds
(i.e. the fold router's assert callback accepts the factory's output type).
This catches wiring bugs at startup instead of at first event.

---

### F05 -- `cloneSnapshotState` in `checkpoint/memory.go` only handles `aggregate.State`

**Category:** correctness risk
**File:** `checkpoint/memory.go:152-164`

```go
func cloneSnapshotState(state any) (any, error) {
    switch typed := state.(type) {
    case aggregate.State:
        return cloneAggregateState(typed), nil
    case *aggregate.State:
        ...
    default:
        return nil, fmt.Errorf("checkpoint: unhandled state type %T -- add a clone case", state)
    }
}
```

The `Systems map[module.Key]any` sub-map is copied by reference
(`cloned.Systems[key] = value` at line 189), not deep-cloned. If system state
contains maps or slices, the in-memory snapshot store shares mutable references
with the live aggregate. In production the SQL-backed store serializes, so this
only affects tests using `checkpoint.Memory`.

**Proposal:** Either document that `checkpoint.Memory` does not deep-clone
system state (test-only concern), or add a system-state clone callback to the
memory store so system modules can register their own deep-clone logic.

---

### F06 -- `aggregate.State.Systems` map values are `any` with no accessor guard

**Category:** contributor friction
**File:** `aggregate/state.go:77`

```go
Systems map[module.Key]any
```

Every consumer of system state must know to type-assert the value. The
`systemCommandDispatcher` (engine/system_command_dispatcher.go:27) reads it
directly: `systemState := current.Systems[key]`. The readiness workflow
(readiness/session_start_workflow.go:97,123) does the same. There is no
accessor method on `State` that performs the assertion -- each call site
independently looks up the map and passes `any` downstream.

**Proposal:** Add a generic accessor:

```go
func SystemState[T any](s State, key module.Key) (T, bool) {
    raw, ok := s.Systems[key]
    if !ok { var zero T; return zero, false }
    v, ok := raw.(T)
    return v, ok
}
```

This would centralize the assertion and eliminate bare map lookups, but is
lower priority since the current callers all pass `any` to typed wrappers
(`TypedDecider`, `FoldRouter`) that perform their own assertion.

---

### F07 -- Nil vs zero-value confusion at fold/decide edges

**Category:** correctness risk (guarded)
**Files:**
- `aggregate/folder.go:126-133` -- lazy initialization with `if systemState == nil`
- `aggregate/state.go:22-38` -- `AssertState` rejects nil
- `engine/core_decider.go:57-58` -- returns zero `aggregate.State{}` for nil
- `engine/handler.go:330-337` -- `loadState` returns `nil` when StateLoader is nil

The codebase has two conventions:
1. **Nil means "no state loaded yet"** (`handler.loadState`, `aggregate.Folder`
   lazy init).
2. **Nil is an error** (`AssertState`, `TypedFolder.Assert` with nil).

These are reconciled by the initialization path:
- Core state always comes from `StateFactory()` (never nil).
- System state in `Systems[key]` starts nil and gets lazily initialized on
  first event via `factory.NewSnapshotState()`.
- The decider path uses `SnapshotOrDefault` which returns a default on nil.

This works but requires every system author to understand three different nil
handling strategies. The `AssertSnapshotState` in Daggerheart (state/snapshot_state.go:144-163)
handles nil by creating a default, while `AssertState` in aggregate rejects nil.

**Observation:** The inconsistency is intentional (documented in
`AssertState`'s comment: "nil state reaching a fold or decider indicates a
missing StateFactory"). But it creates a mental-model split: aggregate folds
reject nil, system folds accept nil and initialize. This is manageable with
one system (Daggerheart) but could cause mistakes when a second system is added.

**Proposal:** Document the nil-handling contract explicitly in `StateFactory`'s
doc comment -- "fold routers must handle nil state from lazy initialization;
`AssertState` is for core aggregate state which is always factory-initialized."

---

### F08 -- `Result.State` is `any` with no typed accessor

**Category:** contributor friction
**File:** `engine/handler.go:167-170`

```go
type Result struct {
    Decision command.Decision
    State    any
}
```

Every consumer of `Result.State` must type-assert. In tests this shows up as
`result.State.(aggregate.State)` (handler_test.go:405) and
`result.State.(map[string]int)` (handler_test.go:647). Transport handlers
need the same assertion to extract typed state for read-after-write responses.

**Proposal:** This is hard to parameterize with generics because `Handler` and
`Result` would need a type parameter that propagates through the entire engine
pipeline. Lower priority -- the assertion is always at the transport boundary
and `AssertState` is available. Consider a helper method:

```go
func (r Result) AggregateState() (aggregate.State, error) {
    return aggregate.AssertState[aggregate.State](r.State)
}
```

---

### F09 -- `StateSnapshotStore` interface uses `any` for state

**Category:** anti-pattern (minor)
**File:** `engine/loader.go:30-33`

```go
type StateSnapshotStore interface {
    GetState(ctx context.Context, campaignID string) (state any, lastSeq uint64, err error)
    SaveState(ctx context.Context, campaignID string, lastSeq uint64, state any) error
}
```

This interface cannot be parameterized because different campaigns could
theoretically use different aggregate shapes (unlikely but the interface
allows it). The `any` here is forced by the same heterogeneous-state
constraint as `Systems map[module.Key]any`.

**Observation:** No action needed -- this is a natural consequence of the
architecture. The snapshot store is transport-level infrastructure and
serialization/deserialization provides the real type boundary.

---

### F10 -- `ReplayStateLoader.StateFactory` is `func() any` instead of a named interface

**Category:** contributor friction
**File:** `engine/loader.go:25`

```go
StateFactory func() any
```

Every caller constructs this as `func() any { return aggregate.NewState() }`.
The function signature provides no documentation about what type the factory
should return. Compare with `module.StateFactory` which is a named interface
with doc comments.

**Proposal:** Define a named type:

```go
type AggregateStateFactory interface {
    NewState() aggregate.State
}
```

Or at minimum, change the field type to `func() aggregate.State` since every
production caller returns `aggregate.State`. This would eliminate one `any`
boundary and make the expected type explicit. The `any` return is only
needed if the loader should be aggregate-type-agnostic, which it currently is
not in practice.

---

### F11 -- `CharacterReadinessChecker.CharacterReady` takes `systemState any`

**Category:** contributor friction (minor)
**File:** `module/registry.go:54`

```go
CharacterReady(systemState any, ch character.State) (ready bool, reason string)
```

The `any` parameter forces implementers (Daggerheart module.go:186) to call
`AssertSnapshotState` internally. This cannot be avoided because the interface
must work for any system module, but it is another manual assertion point that
a typed wrapper could handle.

**Observation:** The same pattern applies to `SessionStartBootstrapper.SessionStartBootstrap`
(registry.go:69). Both are optional extension interfaces that accept `any`
because they are called by the core engine which does not know the system type.
This is the correct design -- the assertion belongs in the implementation.

---

### F12 -- No contract test verifies that `StateFactory` output is fold-compatible

**Category:** missing best practice
**Files:**
- `module/testkit/conformance.go` -- `ValidateSystemConformance` runs
  determinism and idempotency checks but does not verify factory-to-fold
  type compatibility.
- `engine/registries_validation_system.go:229-267` --
  `ValidateStateFactoryDeterminism` checks equal outputs, not type compatibility.

**Proposal:** Add to `ValidateSystemConformance`:

```go
func validateFactoryFoldCompatibility(t *testing.T, mod module.Module) {
    factory := mod.StateFactory()
    folder := mod.Folder()
    if factory == nil || folder == nil { return }
    state, err := factory.NewSnapshotState("compat-check")
    if err != nil { t.Errorf(...); return }
    // Try folding a no-op event; the assertion step in the fold
    // router should succeed even if the handler returns an error
    // for empty payload. The key check is that the type assertion
    // does not fail.
    _, foldErr := folder.Fold(state, event.Event{Type: "nonexistent"})
    // "unhandled fold event type" is acceptable; "unsupported state type" is not.
    if foldErr != nil && strings.Contains(foldErr.Error(), "unsupported state type") {
        t.Errorf("StateFactory output type %T is not compatible with Folder", state)
    }
}
```

---

### F13 -- `aggregate.Folder.Fold` signature takes and returns `any` but always works with `aggregate.State`

**Category:** anti-pattern (structural)
**File:** `aggregate/folder.go:67`

```go
func (a *Folder) Fold(state any, evt event.Event) (any, error) {
```

The `aggregate.Folder` satisfies `engine.Folder` and `fold.Folder`, which
require `any` signatures. Internally, every call immediately asserts to
`aggregate.State`. The `any` boundary here exists solely to satisfy interface
compatibility.

**Observation:** This is the core design constraint. Making `fold.Folder`
generic (`Folder[S any]`) would eliminate this, but Go interfaces cannot be
used polymorphically with different type parameters in the same collection
(e.g. `map[Key]Folder[???]`). The current design is the pragmatic choice.

**Proposal:** No change recommended. The `AssertState` call at the top of
`Fold` is the narrowest possible escape hatch and is consistently applied.

---

### F14 -- Test helpers use bare type assertions instead of `AssertState`

**Category:** contributor friction (minor)
**Files:**
- `aggregate/folder_test.go:24-26,43-45,92-93,100,145,228-230` -- multiple
  `result.(State)` casts without error checking (some use `ok` check, some
  do not).
- `engine/handler_test.go:405,647` -- `result.State.(aggregate.State)` and
  `result.State.(map[string]int)`.
- `engine/loader_test.go:50,198,279` -- `state.(aggregate.State)`.

Some test assertions use the `, ok` pattern while others use direct casts
that will panic on mismatch. This is inconsistent but not a production
concern.

**Proposal:** Low priority. Standardize test assertions to use `AssertState`
or at least consistently use the `, ok` pattern. This primarily affects
test maintainability.

---

## Exhaustive Cast/Assertion Catalog

| # | Location | Source type | Target type | Mechanism | Notes |
|---|----------|-------------|-------------|-----------|-------|
| 1 | `engine/handler.go:66` | `StateLoader.Load` return | `any` | interface contract | Narrowest possible |
| 2 | `engine/handler.go:89` | `Folder.Fold` param+return | `any` | interface contract | Matches `fold.Folder` |
| 3 | `engine/handler.go:97` | `Decider.Decide` param | `any` | interface contract | Matches `module.Decider` |
| 4 | `engine/handler.go:169` | `Result.State` | `any` | struct field | Transport boundary |
| 5 | `engine/core_decider.go:49-59` | `any` | `aggregate.State` | ad-hoc switch | Silent zero-value fallback |
| 6 | `engine/loader.go:25` | `StateFactory` return | `any` | function field | Could be narrowed |
| 7 | `engine/loader.go:31-32` | `StateSnapshotStore` | `any` | interface contract | Forced by heterogeneity |
| 8 | `engine/loader.go:147-158` | `any` | `aggregate.State` | ad-hoc switch | Duplicates `AssertState` |
| 9 | `engine/loader.go:169-186` | `any` | `aggregate.State` | ad-hoc switch | Duplicates `AssertState` |
| 10 | `engine/system_command_dispatcher.go:27` | `Systems[key]` | `any` passed through | map lookup | No assertion needed here |
| 11 | `aggregate/state.go:77` | `Systems` map values | `any` | heterogeneous map | Intentional |
| 12 | `aggregate/state.go:21-38` | `any` | `T` | `AssertState[T]` | Canonical helper |
| 13 | `aggregate/folder.go:67` | `Fold` param+return | `any` | interface contract | Asserts immediately |
| 14 | `aggregate/folder.go:77,86` | `any` | `State` | `AssertState[State]` | Consistent usage |
| 15 | `aggregate/folder.go:113,135` | `Systems[key]` | `any` | map lookup | System routing |
| 16 | `module/registry.go:36-37` | `Decider.Decide` param | `any` | interface | System decider boundary |
| 17 | `module/registry.go:54` | `CharacterReady` param | `any` | interface | Extension point |
| 18 | `module/registry.go:70` | `SessionStartBootstrap` param | `any` | interface | Extension point |
| 19 | `module/registry.go:114-115` | `StateFactory` return | `any` | interface | Factory boundary |
| 20 | `module/registry.go:190,199,206,215` | `RouteCommand`/`RouteEvent` | `any` pass-through | function params | Delegates to typed wrappers |
| 21 | `module/typed.go:31` | `TypedFolder.Fold` | `any` -> `S` | `Assert` callback | Typed wrapper |
| 22 | `module/typed.go:64` | `TypedDecider.Decide` | `any` -> `S` | `Assert` callback | Typed wrapper |
| 23 | `module/fold_router.go:46` | `FoldRouter.Fold` | `any` -> `S` | `assert` callback | Typed wrapper |
| 24 | `fold/fold.go:17-20` | `Folder.Fold` | `any` | canonical interface | Core contract |
| 25 | `fold/fold.go:54` | `CoreFoldRouter.Fold` | typed `S` | generic method | No `any` needed |
| 26 | `checkpoint/memory.go:91,123` | `GetState`/`SaveState` | `any` | interface impl | Snapshot store |
| 27 | `checkpoint/memory.go:153` | `cloneSnapshotState` | `any` -> `aggregate.State` | ad-hoc switch | Only handles one type |
| 28 | `daggerheart/state/snapshot_state.go:124-138` | `any` | `SnapshotState` | `SnapshotOrDefault` | Decider path |
| 29 | `daggerheart/state/snapshot_state.go:144-164` | `any` | `*SnapshotState` | `AssertSnapshotState` | Fold path |
| 30 | `readiness/session_start_workflow.go:97,123` | `Systems[key]` | `any` pass-through | map lookup + pass | Extension call sites |

---

## Priority Rankings

| Priority | Finding | Effort | Impact |
|----------|---------|--------|--------|
| High | F01 -- Replace ad-hoc switches in loader.go with `AssertState` | Small | Eliminates ~30 lines of duplicated logic |
| High | F04/F12 -- Add factory-fold type compatibility validator | Medium | Catches wiring bugs at startup |
| Medium | F02 -- Silent zero-value fallback in `aggregateState()` | Small | Surfaces wiring bugs as rejections |
| Medium | F10 -- Narrow `StateFactory` to `func() aggregate.State` | Small | Eliminates one unnecessary `any` boundary |
| Low | F03 -- Add cross-reference doc on `replay.Folder` | Trivial | Onboarding clarity |
| Low | F05 -- Document shallow clone limitation in checkpoint.Memory | Trivial | Test safety |
| Low | F06 -- Add `SystemState[T]` accessor on `aggregate.State` | Small | Convenience, not required |
| Low | F07 -- Document nil-handling contract in `StateFactory` | Trivial | Second-system author guidance |
| Low | F08 -- Add `Result.AggregateState()` helper | Small | Transport-layer convenience |
| Low | F14 -- Standardize test assertion style | Small | Test maintainability |
| None | F09, F11, F13 -- Observations only | N/A | Correct as-is |
