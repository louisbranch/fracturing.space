# Pass 6: Projection System Consistency

## Summary

The projection package is architecturally well-structured. The `CoreRouter` with
typed generic dispatch, bitmask-based precondition checks, and table-driven store
validation form a coherent system. Event parity tests cover both core and system
events. The exactly-once apply path correctly scopes watermark writes inside the
transaction boundary. Gap detection and repair are well-tested with 7 targeted
tests.

The findings below are primarily about consistency gaps in the precondition
system, a tautological bridge test that should be deleted, missing audit
coverage in the exactly-once path, and minor contributor friction from the
3-config construction hierarchy.

**Severity distribution:** 2 correctness risk, 3 missing best practice,
2 anti-pattern, 2 contributor friction.

---

## Findings

### 1. Soft nil checks bypass storeRequirement precondition system

**Category:** correctness risk

`applySceneCreated` and `applySceneEnded` use inline `if a.SceneInteraction != nil`
guards instead of declaring `storeSceneInteraction` in their handler registration.
The same pattern appears in `applySessionStarted` and `applySessionEnded` for
`SessionInteraction`.

This means:
- `ValidateStorePreconditions()` does not catch a missing `SceneInteraction`
  or `SessionInteraction` store at startup for these handlers.
- The behavior silently degrades: scene creation skips interaction
  initialization, and scene end skips interaction cleanup, with no error or
  audit trail.
- Contributors adding similar optional-store behavior have two competing
  patterns to choose from.

**Files:**
- `apply_scene.go:56` — `if a.SceneInteraction != nil` in `applySceneCreated`
- `apply_scene.go:127` — `if a.SceneInteraction != nil` in `applySceneEnded`
- `apply_session.go:35` — `if a.SessionInteraction == nil` in `applySessionStarted`
- `apply_session.go:62` — `if a.SessionInteraction == nil` in `applySessionEnded`
- `handler_registry_scene.go:6` — registration declares `storeScene, storeSceneCharacter` but not `storeSceneInteraction`
- `handler_registry_scene.go:8` — registration declares `storeScene, storeSceneSpotlight` but not `storeSceneInteraction`
- `handler_registry_session.go:8-9` — registration declares `storeSession` but not `storeSessionInteraction`

**Refactoring proposal:**

Decide whether `SceneInteraction`/`SessionInteraction` is truly optional for
these handlers. If it is required in all production configurations, add
`storeSceneInteraction`/`storeSessionInteraction` to the registration
requirements and remove the soft nil checks. If it is genuinely optional (e.g.
for lightweight replay contexts), document the contract explicitly with an
`// Optional:` comment at the registration site and consider a dedicated
`optionalStores` bitmask that `ValidateStorePreconditions` warns about but does
not reject.

---

### 2. ClaimIndex opt-out from precondition system is undocumented at handler sites

**Category:** missing best practice

`handler_registry.go:46-47` documents that `ClaimIndex` is intentionally absent
from `storeRequirement`, and handlers perform soft nil checks. However, the
individual handler sites in `apply_participant.go` that perform these nil checks
do not reference this design decision, making it easy for a contributor to
"fix" the soft check by adding a store requirement without understanding the
rationale.

**Files:**
- `handler_registry.go:46-47` — the canonical comment
- Handlers in `apply_participant.go` that nil-check `ClaimIndex`

**Refactoring proposal:**

Add a brief `// ClaimIndex: soft nil check — see handler_registry.go:46` comment
at each handler that nil-checks `a.ClaimIndex`, or extract a named helper like
`a.tryClaimIndex(ctx, ...)` that centralizes the nil check and documents the
opt-out once.

---

### 3. Tautological bridge test should be deleted

**Category:** anti-pattern

`TestRegisteredHandlerTypes_MatchesProjectionHandledTypes` in
`handler_registry_test.go:15-49` compares `registeredHandlerTypes()` against
`ProjectionHandledTypes()`. But `ProjectionHandledTypes()` at
`apply_campaign.go:17-19` directly delegates to `registeredHandlerTypes()`:

```go
func ProjectionHandledTypes() []event.Type {
    return registeredHandlerTypes()
}
```

The test is literally asserting that `f()` equals `f()`. The test comment says
"This bridge test ensures the two remain in sync before
`ProjectionHandledTypes()` is refactored to delegate to the map" — but that
refactoring has already happened. The test is now a tombstone.

**Files:**
- `handler_registry_test.go:15-49` — the tautological test
- `apply_campaign.go:17-19` — `ProjectionHandledTypes()` delegates to `registeredHandlerTypes()`

**Refactoring proposal:**

Delete `TestRegisteredHandlerTypes_MatchesProjectionHandledTypes`. The remaining
`TestHandlerRegistry_AllEntriesHaveApply` test at `handler_registry_test.go:53`
already validates that every registered handler has a non-nil apply function,
which is the meaningful invariant.

---

### 4. Exactly-once path omits Auditor, silencing gap audit events

**Category:** missing best practice

`BuildExactlyOnceApply` at `exactly_once_apply.go:36-40` constructs a
`BoundApplierConfig` without setting `Auditor`. Inside the exactly-once
transaction, if a projection gap is detected by `saveProjectionWatermark`, the
slog warning fires (line 71-75 of `applier_watermark.go`) but the structured
audit event at line 80-90 is silently skipped because `a.Auditor == nil`.

This means operational visibility into projection gaps during outbox
consumption relies solely on log scraping rather than structured audit events.

**Files:**
- `exactly_once_apply.go:36-40` — `BoundApplierConfig` without `Auditor`
- `applier_watermark.go:77-90` — nil-guarded audit emit

**Refactoring proposal:**

Accept an `audit.Policy` or `*audit.Emitter` in `BuildExactlyOnceApply` and
thread it through to `BoundApplierConfig.Auditor`. This ensures gap audit events
are emitted uniformly regardless of the apply path.

---

### 5. Three config types create contributor friction in applier construction

**Category:** contributor friction

The construction hierarchy has three config types:
- `ApplierConfig` (line 115) — stores + system stores + events + audit + now
- `BundleApplierConfig` (line 124) — store bundle + system stores + events + audit + now
- `BoundApplierConfig` (line 134) — stores + events + adapters + auditor + now

The chain is `NewApplierFromBundle` -> `NewApplier` -> `NewBoundApplier`.
Contributors need to understand which entry point to use and which config to
fill. The field names also shift between layers (`AuditPolicy` vs `Auditor`,
`SystemStores` vs `Adapters`).

**Files:**
- `applier_construction.go:115-141` — the three config types
- `applier_construction.go:145-195` — the three constructors

**Refactoring proposal:**

This is acceptable complexity given the different scopes (bundle vs grouped vs
pre-bound), but adding a `// Construction hierarchy` doc comment at the top of
`applier_construction.go` explaining when to use each entry point would reduce
onboarding friction. Example:

```
// Construction hierarchy:
//   NewApplierFromBundle — full bundle (production startup, integration tests)
//   NewApplier           — grouped stores (tests that need store-level control)
//   NewBoundApplier      — pre-resolved adapters (exactly-once tx callback, unit tests)
```

---

### 6. coreRouter package-level var is safe (non-finding, documented for completeness)

**Category:** N/A — confirmed safe

`var coreRouter = buildCoreRouter()` at `handler_registry.go:93` is initialized
once at package init and never mutated afterward. All access is read-only via
`coreRouter.handlers[evt.Type]` (applier.go:88) and `coreRouter.Route()`
(applier.go:89). Concurrent reads of a Go map are safe when there are no
concurrent writes. The `HandledTypes()` method returns a defensive copy
(core_router.go:51).

No action needed.

---

### 7. Replay contiguity enforcement is strict but lacks recovery guidance

**Category:** contributor friction

`ReplayCampaignWith` at `replay.go:73-76` aborts on any sequence gap with:
```go
return lastSeq, fmt.Errorf("projection replay sequence gap: expected %d got %d", expectedSeq, evt.Seq)
```

The error message does not indicate whether this is a journal corruption issue
or a `ListEvents` pagination bug, nor does it suggest a recovery path. Operators
seeing this error in production logs need to know whether to investigate the
event store or the replay cursor.

**Files:**
- `replay.go:73-76` — contiguity check error

**Refactoring proposal:**

Enrich the error message with the campaign ID and a hint:
```go
return lastSeq, fmt.Errorf(
    "projection replay sequence gap for campaign %s: expected seq %d got %d (check event journal integrity)",
    campaignID, expectedSeq, evt.Seq,
)
```

---

### 8. Event parity tests mask handler errors as "handled"

**Category:** correctness risk

`TestApplyProjectionRequiredCoreEventsAreHandled` at `event_parity_test.go:51-54`
calls `applier.Apply()` and only checks if the error message matches
`isUnhandledProjectionEventError`. Any other error (store failure, payload
decode failure, precondition failure) is treated as "handled." This means a
handler that is registered but always fails on empty payloads will pass the
parity test.

The test uses `baselineProjectionEvent` which sends `PayloadJSON: []byte("{}")`
— this works for handlers with all-optional fields but will silently fail for
handlers that require specific payload fields. The test's purpose is parity
(ensuring every event has a handler), not correctness, so this is acceptable
as long as the distinction is clear.

**Files:**
- `event_parity_test.go:51-54` — error masking logic
- `event_parity_test.go:108-118` — `baselineProjectionEvent` with empty payload

**Refactoring proposal:**

Add a comment at the test clarifying the intent boundary:
```go
// Invariant: this test verifies handler registration coverage, not handler
// correctness. A handler that fails on empty payloads is still "handled."
// Handler behavior is tested in per-domain applier test files.
```

No code change needed — just clarify intent for future contributors.

---

### 9. validatePreconditions reports only the first missing store

**Category:** anti-pattern

`validatePreconditions` at `handler_registry.go:194` calls `checkMissingStores`
which returns all missing store labels, but then only reports `missing[0]`:

```go
return fmt.Errorf("%s store is not configured", missing[0])
```

In contrast, `ValidateStorePreconditions` at `handler_registry.go:186` reports
all missing stores joined by comma. This inconsistency means that per-event
dispatch errors during runtime only surface one missing store at a time,
forcing repeated apply-fail-fix cycles during development.

**Files:**
- `handler_registry.go:194-196` — single store in error
- `handler_registry.go:185-187` — all stores in error

**Refactoring proposal:**

Change `validatePreconditions` to report all missing stores:
```go
if missing := checkMissingStores(h.stores, a); len(missing) > 0 {
    return fmt.Errorf("projection stores not configured for %s: %s", evt.Type, strings.Join(missing, ", "))
}
```

This is a low-risk change since the error path means the event cannot be
projected anyway.

---

## Package Health Metrics

| Metric | Value |
|--------|-------|
| Production files | ~30 |
| Test files | ~15 |
| Total lines (production) | ~3,200 |
| Total lines (test) | ~5,400 |
| Core handlers registered | ~54 |
| Store requirement bits | 15 |
| ID requirement bits | 3 |
| Config types for construction | 3 |
| Interaction handler files | 5 (611 lines total, well-split by concern) |

## Architecture Assessment

The projection package demonstrates strong design:

1. **Type-safe dispatch** via generics eliminates per-handler payload
   boilerplate and catches type mismatches at compile time.

2. **Bitmask preconditions** provide O(1) startup validation and per-event
   safety checks with a single source of truth in `storeChecks`.

3. **Gap detection and repair** is a complete operational toolkit: watermark
   tracking during apply, periodic gap detection comparing journal vs
   watermark, and automated replay repair.

4. **Exactly-once transactional apply** correctly scopes all projection
   writes (including watermark) inside the transaction callback via
   `StoreGroupsFromBundle(txStore)`.

5. **Event preprocessing** cleanly separates alias resolution and intent
   filtering from the routing/apply path.

The main risks are the soft nil check inconsistency (Finding 1) which could
cause silent data loss during scene/session lifecycle transitions, and the
masked handler errors in parity tests (Finding 8) which could allow broken
handlers to ship undetected.
