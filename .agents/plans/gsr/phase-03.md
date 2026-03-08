# GSR Phase 3: Event System — Registration, Intent, Folding

## Summary

The event system demonstrates **strong architectural clarity** with well-designed registry patterns, correct intent classification, pure fold functions, and comprehensive validation. Two significant concerns: events carry computed before/after data that bloats payloads, and the `extractSceneID` workaround reveals an addressing inconsistency in gate events.

## Findings

### F3.1: Computed Data in Event Payloads — Important

**Severity:** important

Events carry before/after values and derived state (e.g., `OutcomeAppliedChange.Before/After`, `CharacterStatePatchedPayload.HPBefore/HPAfter`, `LevelUpApplyPayload.Tier/PreviousTier/IsTierEntry`). These are projection data, not domain facts.

**Impact:** Event immutability principle weakened. Payload bloat. Data consistency risk if decider computes incorrectly. Fold functions don't use these fields — they're purely for projection display.

**Recommendation:** Separate payload into command input (persist) and computed effects (derive at read time). Create `docs/architecture/policy/event-payload-design.md`.

### F3.2: `extractSceneID` Workaround — Addressing Inconsistency — Important

**Severity:** important

**Location:** `domain/aggregate/fold_registry.go:128-157`

Scene gate events use `EntityID = GateID` but scene folds need `SceneID`. The workaround mines the JSON payload to recover the scene ID — fold functions shouldn't need JSON parsing for addressing.

**Options:**
- **Option A (cleaner):** Change gate events to use `EntityID=SceneID`, store GateID in payload
- **Option B (document):** If GateID addressing is required, document the exception explicitly

### F3.3: Intent Classifications — Excellent

**Severity:** style (no action needed)

All 41 core events correctly classified across IntentProjectionAndReplay, IntentReplayOnly, and IntentAuditOnly. `ShouldFold()` and `ShouldProject()` methods correctly implement intent filtering.

### F3.4: PayloadValidator Coverage — Complete

**Severity:** style (no action needed)

All 39 non-audit events have PayloadValidators. Audit-only events correctly exempted. `MissingPayloadValidators()` method properly excludes them.

### F3.5: Fold Function Purity — Verified

**Severity:** style (no action needed)

All fold functions are pure: no I/O, no `time.Now()`, no `context.Context`. Same events always produce same state. Fully replay-friendly.

### F3.6: Event Type Naming — Minor Inconsistency

**Severity:** minor

Commands use dot-separated namespacing (`action.roll.resolve`) while events use underscore (`action.roll_resolved`). Not breaking (different type systems), but creates cognitive friction.

**Recommendation:** If intentional, document the convention. If accidental, consider standardization.

### F3.7: RegisterAlias — Designed but Unused

**Severity:** minor

**Location:** `domain/event/registry.go:409-445`

Alias infrastructure exists for event type deprecation but has zero production callers. No documented migration path.

**Recommendation:** Either document deprecation criteria or remove if not needed.

### F3.8: Hash Computation — Correct

**Severity:** style (no action needed)

Content hash correctly separates from chain hash. Omitted fields (Signature, SignatureKeyID, ChainHash) are intentionally excluded as append-time integrity metadata.

## Cross-References

- **Phase 2** (Domain Model): State constructed in folds without validation
- **Phase 4** (Command/Decision): Event emission from deciders
- **Phase 6** (Projection): Projection handlers consuming computed payload data
- **Phase 9** (Module Extension): System event registration
