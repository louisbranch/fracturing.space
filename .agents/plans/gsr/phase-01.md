# GSR Phase 1: Package Structure & Dependency Graph

## Summary

The game service exhibits **excellent architectural discipline** with a clean 7-layer stack, no import cycles, and strict layer separation. All imports flow downward. No god packages, no cycle-breaking `any` laundering, no transport imports in domain code. Architecture is production-ready.

## Findings

### F1.1: Layer Architecture — Excellent

**Severity:** style (no action needed)

Clean 7-layer stack: Platform → Core → Domain → Projection → Storage → App → API. All imports flow strictly downward with no upward or circular dependencies detected.

### F1.2: Package Size Distribution — Well-Balanced

**Severity:** style (no action needed)

- **Tier 1 (8k-33k LOC):** 5 packages — major subsystems (gRPC, Daggerheart, engine, storage, projection)
- **Tier 2 (1k-3k LOC):** 4 packages — core domain aggregates
- **Tier 3 (500-1k LOC):** 7 packages — focused domain logic
- **Tier 4 (<500 LOC):** 13 packages — constants & contracts

No over-decomposition; small packages are leaf utilities, not scattered logic.

### F1.3: No Import Violations Detected

**Severity:** style (no action needed)

Domain packages never import from `api/`, `storage/sqlite`, or `app/`. Exception: `domain/bridge/daggerheart` imports `storage.DaggerheartStore` — this is the contractual system adapter pattern, not a violation.

### F1.4: `any` Type Usage — Legitimate

**Severity:** style (no action needed)

Only occurrence in `domain/engine/loader.go` for `StateFactory func() any`. This decouples state factory from loader (each aggregate has different state type) and is immediately cast back via `aggregate.AssertState[T]`. Not cycle-breaking laundering.

### F1.5: `core/` Package — Genuine Utilities

**Severity:** style (no action needed)

All sub-packages (dice, check, random, filter, encoding, naming) are cohesive RPG mechanics primitives reused across domain and system packages. No dumping-ground anti-pattern.

### F1.6: `domain/commandids` and `domain/coreevent` — Justified Leaf Constants

**Severity:** style (no action needed)

- `commandids` (86 LOC): Central authority for 80+ command type constants. Pure constants, zero logic.
- `coreevent` (49 LOC): Event type aliases + payload shims for test tooling.

No hidden coupling.

### F1.7: `domain/internaltest/` — Justified

**Severity:** style (no action needed)

Test contracts package providing shared test infrastructure for domain packages. Appropriately scoped.

### F1.8: Import Depth — Traceable in Minutes

**Severity:** style (no action needed)

Shallow import depth (4-5 hops max). File naming conventions (`*_service.go`, `*_application.go`, `decider.go`, `fold.go`) act as a navigation map. A newcomer can trace dependencies in under 5 minutes.

## Minor Recommendations

1. Add ADRs documenting why `domain/aggregate` aggregates 6+ substates and why bridge adapters import storage
2. Clarify `domain/bridge/manifest` doc.go with explanation of why storage import is allowed
3. Consider moving `domain/module/testkit` if only used in tests

## Cross-References

- **Phase 2** (Domain Model): Aggregate state composition
- **Phase 9** (Module Extension): Bridge adapter import patterns
- **Phase 8** (gRPC Transport): 172 files in flat package (reviewed separately)
