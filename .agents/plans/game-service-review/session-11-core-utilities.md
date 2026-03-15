# Session 11: Core Utilities

## Status: `complete`

## Package Summaries

### `core/check/` — Validation helpers (invariant checks)
### `core/dice/` — Dice rolling types and logic
### `core/encoding/` — Canonical JSON serialization
### `core/filter/` — Generic filter/predicate helpers
### `core/naming/` — System namespace validation
### `core/random/` — Deterministic seed generation

## Findings

### Finding 1: All Packages Belong Under core/ — Good Cohesion
- **Severity**: info
- **Location**: `core/`
- **Issue**: Each package provides a focused utility that multiple domain packages depend on. `encoding` provides canonical JSON for hashing. `naming` validates system namespace prefixes (`sys.daggerheart.*`). `dice` and `random` handle randomness. `filter` and `check` are small helper packages.
- **Recommendation**: Good package organization. All are genuine cross-cutting utilities.

### Finding 2: random/ Injectability for Testing
- **Severity**: info
- **Location**: `core/random/seed.go`
- **Issue**: The random package provides seed generation for deterministic random sources. This supports injectable randomness — callers can provide a fixed seed for testing while using crypto/rand in production.
- **Recommendation**: Good testability design.

### Finding 3: encoding/canonical.go Is Critical Path Infrastructure
- **Severity**: info
- **Location**: `core/encoding/canonical.go`
- **Issue**: `CanonicalJSON` and `ContentHash` are used by event hashing and chain integrity. Changes to canonical JSON serialization would break all existing event hashes. This is effectively frozen.
- **Recommendation**: Add a prominent warning comment: "This function must remain backwards-compatible — changes break event hash integrity."

### Finding 4: naming/ Validates System Namespaces — Correct Placement
- **Severity**: info
- **Location**: `core/naming/system.go`
- **Issue**: `ValidateSystemNamespace` checks that event/command type strings match their system ID prefix (e.g., `sys.daggerheart.*` must have `systemID=daggerheart`). Used by both command and event registries. Core utility is the right location — it's used across domain packages.
- **Recommendation**: Clean placement.

### Finding 5: dice/ Types Should Document Testing Approach
- **Severity**: low
- **Location**: `core/dice/`
- **Issue**: Dice rolling is inherently random. The package should document how to test with deterministic outcomes (injected random source).
- **Recommendation**: Add a doc comment explaining the deterministic testing pattern.

## Summary Statistics
- Files reviewed: ~14 (6 packages, prod and test)
- Findings: 5 (0 critical, 0 high, 0 medium, 1 low, 4 info)
