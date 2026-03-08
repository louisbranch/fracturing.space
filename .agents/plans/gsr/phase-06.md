# GSR Phase 6: Projection Layer

## Summary

The projection layer is **production-ready and well-tested** with type-safe event routing, validated event parity, deterministic replay, and gap-repair mechanisms. Main opportunities: Applier decomposition (18 store fields), `BuildErr` anti-pattern, and documentation of concurrency model and handler ordering assumptions.

## Findings

### F6.1: Applier Struct — 18 Store Fields ("God Object" Smell)

**Severity:** important

**Location:** `projection/applier.go`

Applier holds 18 fields (13 domain stores + Events, Adapters, Watermarks, Now, BuildErr). Testing requires all stores even when testing a subset of handlers.

**Natural groupings:**
- Campaign state: Campaign, Character, Participant, Invite
- Session/Gate/Spotlight: Session, SessionGate, SessionSpotlight, Scene, SceneCharacter, SceneGate, SceneSpotlight
- Infrastructure: Adapters, Watermarks, Now, Events

**Recommendation:** Consider decomposing into domain-grouped facades in a future refactor. Not urgent — `ValidateStorePreconditions()` already does comprehensive startup validation.

### F6.2: `BuildErr` Anti-Pattern

**Severity:** minor

**Location:** `projection/applier.go`

Error stored in struct field instead of returned from constructor. Checked in `Apply()` and `ValidateStorePreconditions()`. Go idiom is `func NewApplier(...) (Applier, error)`.

**Recommendation:** Replace with proper constructor error return.

### F6.3: `coreRouter` Package-Level Var — Safe

**Severity:** style (no action needed)

Single immutable instance built at package import time. Handlers map only read during dispatch. Tests don't share router state. Safe for parallel tests.

### F6.4: Event Parity — Complete

**Severity:** style (no action needed)

`event_parity_test.go` validates all IntentProjectionAndReplay events (core + system) have handlers. Tests pass. No silent handler gaps possible.

### F6.5: Watermark Tracking — Sound

**Severity:** minor

**Location:** `projection/applier_watermark.go`, `projection/gaps.go`

Detects both mid-stream and trailing gaps. Repair is idempotent. Under concurrent projection, watermark may temporarily regress (last-write-wins), but gap repair corrects this.

**Recommendation:** Document concurrency model (eventual consistency guarantee via gap repair).

### F6.6: Handler Ordering Dependencies — Implicit but Protected

**Severity:** minor

Handlers assume strict event sequence order (e.g., Campaign must exist before Participants). `replay.go:73-76` validates sequence contiguity, preventing out-of-order replay.

**Recommendation:** Add package-level documentation explaining ordering assumptions.

### F6.7: System Event Routing — Complete

**Severity:** style (no action needed)

Core events routed via immutable `coreRouter`. System events delegated to `Adapters.GetRequired()` (strict) or `GetOptional()` (graceful for profile updates). All paths tested.

### F6.8: Store Precondition Validation — Excellent

**Severity:** style (no action needed)

**Location:** `projection/handler_registry.go:139-153`

Bitmask-based `storeChecks` table maps all 13 required stores. Dual-layer validation: startup via `ValidateStorePreconditions()` and per-event via `validatePreconditions()`.

## Cross-References

- **Phase 3** (Event System): Intent classification drives projection routing
- **Phase 5** (Engine): Fold integration after persist
- **Phase 7** (Storage): Projection store contracts
- **Phase 9** (Module Extension): System event adapter routing
