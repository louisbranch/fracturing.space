# GSR Phase 5: Engine Orchestration

## Summary

The engine orchestration implements a clean, event-sourced write-path with strong separation of concerns. Issues are **clarity and consistency concerns** rather than functional bugs: dead `HandlerConfig` alias, test-only `Handle()` API, value receivers on struct with interface fields, and scattered nil checks undermining startup validation.

## Findings

### F5.1: `HandlerConfig = Handler` — Dead Compatibility Shim

**Severity:** minor

**Location:** `domain/engine/handler.go:120`

Type alias with single caller (`app/domain.go:73`). Provides zero value.

**Recommendation:** Remove alias, update single call site to use `engine.Handler{}` directly.

### F5.2: `Handle()` vs `Execute()` Dual API — Confusing

**Severity:** important

**Location:** `domain/engine/handler.go:163-207`

`Handle()` returns only `Decision` (no post-persist state). `Execute()` returns `Result{Decision, State}` (with snapshot/checkpoint). No production code calls `Handle()` — it's test-only. The `Domain` interface exposes only `Execute()`.

**Recommendation:** Remove `Handle()` entirely. Tests should use `Execute()` and ignore the State field. Eliminates dead API and unifies the contract.

### F5.3: Value Receivers on Handler — Convention Issue

**Severity:** minor

All 14 Handler methods use value receivers despite Handler containing multiple interface fields. Technically correct but unconventional. Risks confusion when refactoring.

**Recommendation:** Change to pointer receivers for Go convention alignment.

### F5.4: Nil Checks Undermining Production Invariants — Important

**Severity:** important

**Location:** `domain/engine/handler.go` — 18 nil checks across 420 lines

Fields validated at startup via `NewHandler()` are re-checked at runtime "for test flexibility." This trades zero-cost flexibility for cognitive overhead and signals fields are optional when they're required.

**Recommendation:** Require all tests to use `NewHandler()`. Remove runtime nil checks for startup-validated fields. Document optional vs. required fields explicitly.

### F5.5: Post-Persist Error Handling — Correct

**Severity:** style (no action needed)

All three stages (fold, snapshot, checkpoint) properly wrapped as `PostPersistError` with `NonRetryable()`. Transport boundary (`grpcerror/helper.go`) correctly maps to `codes.FailedPrecondition`. Test coverage exists for all three stages.

### F5.6: `NonRetryable` Adoption — Complete

**Severity:** style (no action needed)

`IsNonRetryable()` checked at gRPC transport boundary. All post-persist stages classified correctly.

### F5.7: Gate Evaluation Ordering — Correct, Undocumented

**Severity:** minor

**Location:** `domain/engine/handler.go:215-229`

Session gate (global precondition) evaluated before scene gate (scoped). Correct design but ordering rationale not documented.

**Recommendation:** Add inline documentation explaining the ordering rationale.

### F5.8: Checkpoint Capping Logic — Sound

**Severity:** style (no action needed)

**Location:** `domain/engine/loader.go:88-109`

`checkpointCapStore` prevents replay corruption when snapshots and checkpoints drift. Excellent inline comments explain the invariant.

### F5.9: `Registries` Struct — Acceptable

**Severity:** style (no action needed)

Lightweight bundle grouping 3 cohesive registries. Defensible as a convenience struct.

## Recommended Refactor Sequence

1. **Immediate:** Remove `HandlerConfig` alias, add gate ordering docs
2. **Short-term:** Remove `Handle()` API, consolidate to `Execute()`, change to pointer receivers
3. **Follow-up:** Remove runtime nil checks for startup-validated fields, document optional vs. required

## Cross-References

- **Phase 3** (Event System): Event validation in engine pipeline
- **Phase 4** (Command/Decision): Gate policy evaluation
- **Phase 6** (Projection): Fold integration after persist
- **Phase 7** (Storage): Snapshot/checkpoint store contracts
