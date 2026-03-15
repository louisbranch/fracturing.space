# Session 7: Domain Engine — Handler, Decider, Loader

## Status: `complete`

## Package Summaries

### `domain/engine/handler.go` (491 lines)
Domain write orchestrator implementing the full command pipeline: validate → gate (session then scene) → load state → decide → validate events → append → fold → snapshot/checkpoint. Well-documented with clear gate evaluation ordering rationale. Defines interfaces: `GateStateLoader`, `SceneGateStateLoader`, `StateLoader`, `FreshStateLoader`, `EventJournal`, `Folder`, `Decider`.

### `domain/engine/core_decider.go` (318 lines)
Core command dispatcher routing commands to aggregate-specific deciders (campaign, session, scene, participant, character, invite, action) plus system module delegation.

### Supporting files:
- `gate.go` — `DecisionGate` struct with session and scene gate checking
- `loader.go` — Replay-based state loader implementation
- `errors.go` — `PostPersistError` type for post-append failure classification
- `active_session_policy.go` — Active session enforcement policy
- `core_domain.go` — Core domain type catalog

## Findings

### Finding 1: handler.go at 491 Lines Is Well-Structured
- **Severity**: info
- **Location**: `domain/engine/handler.go`
- **Issue**: Despite 491 lines, the handler is well-decomposed into private methods: `prepareExecution`, `validateCommand`, `evaluateSessionGate`, `evaluateSceneGate`, `loadState`, `decide`, `retryRejectedDecisionWithFreshState`, `validateDecisionEvents`, `appendDecisionEvents`, `applyDecisionEvents`. Each step is 10-30 lines. The public API is two methods: `NewHandler` and `Execute`.
- **Recommendation**: Current structure is clear. No extraction needed.

### Finding 2: retryRejectedDecisionWithFreshState Is Narrowly Scoped
- **Severity**: medium
- **Location**: `domain/engine/handler.go:368-405`
- **Issue**: The retry logic only applies to `session.ai_turn.*` commands with `SESSION_AI_TURN_NOT_ACTIVE` rejections. This is a very specific optimization for AI turn race conditions where a cached snapshot may be stale. The retry loads fresh state (full replay) and re-decides. The narrow scoping is appropriate — broad retry would mask bugs.
- **Recommendation**: Document why this specific retry exists (AI turn queueing race condition with stale snapshots). Consider making the retry criteria configurable via a `RetryPolicy` interface if more command types need similar treatment.

### Finding 3: Interfaces Defined at Consumer — Correct Go Pattern
- **Severity**: info
- **Location**: `domain/engine/handler.go:55-109`
- **Issue**: `GateStateLoader`, `StateLoader`, `EventJournal`, `Folder`, `Decider` are all defined in the engine package where they're consumed. This follows the Go "accept interfaces, return structs" pattern and is explicitly called out in the `Folder` doc comment.
- **Recommendation**: Exemplary Go interface placement.

### Finding 4: PostPersistError Classification Is Well-Designed
- **Severity**: info
- **Location**: `domain/engine/errors.go`
- **Issue**: `PostPersistError` wraps errors that occur after journal append succeeds but before snapshot/checkpoint/fold completes. This is critical in event sourcing — the events are persisted but the cache is potentially stale. The error type includes `Stage` (fold/snapshot/checkpoint), `CampaignID`, and `Seq` for diagnosis.
- **Recommendation**: Clean error design. Callers can distinguish post-persist failures from pre-persist failures and recover appropriately (replay vs retry).

### Finding 5: Gate Policy at the Right Layer
- **Severity**: info
- **Location**: `domain/engine/gate.go`, `domain/engine/handler.go:299-343`
- **Issue**: Gate evaluation happens in the engine handler between command validation and state loading. This is the right layer — gates are a cross-cutting command-routing concern, not an aggregate invariant. The gate uses lightweight state loaders (session/scene only, not full replay) for efficiency.
- **Recommendation**: Correct placement. Gates are properly a command-routing concern.

### Finding 6: Handler.Execute Post-Persist Flow Has Correct Error Semantics
- **Severity**: info
- **Location**: `domain/engine/handler.go:183-222`
- **Issue**: After `prepareExecution` (which includes journal append), the handler saves snapshot and checkpoint. Both failures are wrapped as post-persist errors. The fold failure in `applyDecisionEvents` also correctly wraps as `ErrPostPersistApplyFailed` when journal persist has occurred. This allows callers to know events are durable even if the cache is stale.
- **Recommendation**: Well-designed error semantics for event sourcing.

### Finding 7: Core Decider as Dispatcher Is Appropriate
- **Severity**: info
- **Location**: `domain/engine/core_decider.go`
- **Issue**: At 318 lines, the core decider routes commands to domain-specific deciders based on command type prefix. This is a dispatcher pattern, not a god-object — each case delegates to a focused aggregate decider. System commands are routed to module-specific deciders.
- **Recommendation**: Clean dispatcher pattern. The command type prefix routing is simple and extensible.

### Finding 8: Engine Testability Without Storage
- **Severity**: info
- **Location**: `domain/engine/handler.go:131-144`
- **Issue**: The `Handler` struct has optional fields (StateLoader, Folder, Snapshots, Checkpoints) that are nil-safe at call sites. This means tests can create a Handler with only required dependencies (Commands, Events, Journal, Decider) and omit storage. The `NewHandler` constructor validates required deps while leaving optional ones nil-able.
- **Recommendation**: Good testability design. The test flexibility comment on nil checks is helpful.

## Summary Statistics
- Files reviewed: ~20 (10 production + 10 test files in domain/engine/)
- Findings: 8 (0 critical, 0 high, 1 medium, 0 low, 7 info)
