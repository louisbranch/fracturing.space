# Session 3: Aggregate State and Fold Infrastructure

## Status: `complete`

## Package Summaries

### `domain/aggregate/` (8 files: 4 prod, 4 test)
- `state.go` — `State` struct (8 fields: Campaign, Session, Action, Participants map, Characters map, Invites map, Scenes map, Systems map) with `NewState()` factory and generic `AssertState[T]` helper
- `fold_registry.go` — Declarative fold dispatch table (`coreFoldEntries()`) mapping event types to fold functions. Generic `foldEntityKeyed` helper. Special `foldScene` handler for scene ID extraction from payloads.
- `folder.go` — `Folder` struct with lazy fold index initialization (`sync.Once`), event alias resolution, intent filtering (skip audit-only), core vs system event routing
- `doc.go` — Package documentation

### `domain/decide/` (3 files: 1 prod + doc, 1 test)
- `flow.go` — Generic decision flow helpers: `DecideFunc[P]`, `DecideFuncWithState[S,P]`, `DecideFuncTransform[S,PIn,POut]`, `DecideFuncMulti[S,P]`. These eliminate boilerplate for the common unmarshal→validate→marshal→emit pattern.

## Findings

### Finding 1: aggregate.State Is a Reasonable Aggregate Root — Not a God Struct
- **Severity**: info
- **Location**: `domain/aggregate/state.go:66-83`
- **Issue**: `State` has 8 fields (3 value types + 4 maps + 1 any map). The review plan asked if this is a god struct. It is not — it's the in-memory campaign-wide projection snapshot that the CQRS decider needs as input. Each field corresponds to a distinct entity type. The `Systems` map (`map[module.Key]any`) is the extension point for game-system-specific state.
- **Recommendation**: Current structure is appropriate for the aggregate root of an event-sourced system. The field count is manageable.

### Finding 2: AssertState Generic Helper Is Well-Designed
- **Severity**: info
- **Location**: `domain/aggregate/state.go:22-39`
- **Issue**: `AssertState[T]` handles both `T` and `*T` with clear error messages including "missing StateFactory?" hints. This is used throughout the fold pipeline to convert `any`-typed state back to concrete types.
- **Recommendation**: Clean design with good error diagnostics. No changes needed.

### Finding 3: Fold Registry Is Declarative and Extensible
- **Severity**: info
- **Location**: `domain/aggregate/fold_registry.go:64-122`
- **Issue**: `coreFoldEntries()` returns a declarative table mapping event types to fold functions. Adding a new core domain requires only adding an entry here. The generic `foldEntityKeyed` helper eliminates boilerplate for entity-keyed domains.
- **Recommendation**: Clean pattern. Well-documented.

### Finding 4: Scene Fold Uses Payload Extraction — Inconsistent with Other Entity-Keyed Folds
- **Severity**: medium
- **Location**: `domain/aggregate/fold_registry.go:130-159`
- **Issue**: `foldScene` extracts `scene_id` from the JSON payload because scene events use different EntityID conventions (some use SceneID, gate events use GateID). This is a special case that breaks the uniform `foldEntityKeyed` pattern used by participant/character/invite. The payload extraction adds JSON unmarshal overhead per scene event and couples the fold infrastructure to payload structure.
- **Recommendation**: Consider normalizing scene event entity addressing so that EntityID always contains the SceneID (with GateID in the payload), which would allow scene folds to use the standard `foldEntityKeyed` pattern. This may require an event schema change.

### Finding 5: Folder Uses sync.Once for Lazy Index — Correct Thread Safety
- **Severity**: info
- **Location**: `domain/aggregate/folder.go:31-47`
- **Issue**: `sync.Once` ensures the fold index is built exactly once even under concurrent access. This is correct — after initialization, the index map is read-only.
- **Recommendation**: Clean design.

### Finding 6: Folder.Fold Handles Core and System Events in One Path
- **Severity**: low
- **Location**: `domain/aggregate/folder.go:67-143`
- **Issue**: `Fold` handles both core events (via fold index) and system events (via module registry) in a single method. System events may also have a core fold handler (when an event affects both core and system state). The flow is: resolve aliases → check should-fold → try core fold → if system metadata present, also fold via system module. This dual-path is correct but the method is 76 lines with nested conditionals.
- **Recommendation**: The method could be clearer with an early-return structure or by extracting the system fold path into a helper. But the current code is well-commented and correct.

### Finding 7: decide.Flow Functions Eliminate Boilerplate Effectively
- **Severity**: info
- **Location**: `domain/decide/flow.go:24-229`
- **Issue**: The four `DecideFunc*` variants cover the common decider patterns progressively: simple (payload only), with state, with transform, and multi-event. Each handles unmarshal→validate→marshal→emit with proper error handling and entity ID extraction.
- **Recommendation**: Well-designed generic helpers. The progression from simple to complex is clear. Good candidate for a contributor guide example.

### Finding 8: DecideFunc Variants Have Duplicated Unmarshal/Entity/Emit Logic
- **Severity**: medium
- **Location**: `domain/decide/flow.go:24-229`
- **Issue**: All four `DecideFunc*` variants contain identical unmarshal, entity-ID-extraction, and event-construction code. This is ~30 lines duplicated four times. If the pattern changes (e.g., adding a new envelope field), all four must be updated.
- **Recommendation**: Extract common phases into shared helpers: `unmarshalPayload[P]`, `resolveEntity`, `emitEvent`. The variants would then compose these phases differently. This reduces the risk of drift between variants.

### Finding 9: decide Package Has No Redundancy with Engine
- **Severity**: info
- **Location**: `domain/decide/flow.go`, `domain/engine/`
- **Issue**: The review plan asked about `decide.Flow` vs engine redundancy. `decide/` provides generic decision flow helpers used by individual deciders. The engine orchestrates the full command pipeline (validation → gate → load → decide → append). They operate at different abstraction levels — no overlap.
- **Recommendation**: Clean separation. `decide/` is the right abstraction for reducing decider boilerplate.

### Finding 10: Fold Error Classification Is Implicit
- **Severity**: low
- **Location**: `domain/aggregate/fold_registry.go:40`, `domain/aggregate/folder.go:98`
- **Issue**: Fold errors are returned as plain `error` values with `fmt.Errorf` wrapping. There's no distinction between infrastructure errors (JSON unmarshal failure) and domain errors (invalid state transition). The engine and callers treat all fold errors as fatal. In an event-sourced system this is correct — if a persisted event can't be folded, it's a data integrity issue, not a retryable error.
- **Recommendation**: Current approach is correct for event sourcing. All fold errors indicate data corruption or programming bugs. No classification needed.

## Summary Statistics
- Files reviewed: 8 (4 production, 4 test)
- Findings: 10 (0 critical, 0 high, 2 medium, 2 low, 6 info)
