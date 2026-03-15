# Session 1: Domain Primitives and Type Foundations

## Status: `complete`

## Package Summaries

### `domain/ids/` (3 files)
Leaf package defining type-safe domain identifiers as named string types. Each ID type has a `String()` method. Contains one sentinel error (`ErrCampaignIDRequired`). No validation at construction — IDs are free-form strings with compile-time type safety only.

### `domain/commandids/` (2 files)
Constant declarations for all command type strings (`command.Type`). Organized into core, scene, and daggerheart system commands (~50 constants). Single import dependency on `domain/command`.

### `domain/coreevent/` (2 files)
Constant declarations for core event type strings (`event.Type`). Also re-exports `SessionGateOpenedPayload` as a type alias. Imports both `domain/event` and `domain/session`.

### `domain/command/` non-registry (5 prod + 4 test files)
- `decision.go` — `Decision` struct with `Accept`/`Reject` constructors, shared rejection codes
- `event.go` — `NewEvent` helper copying command envelope to event
- `marshal.go` — `MustMarshalJSON` utility
- `time.go` — `NowFunc` clock validator

### `domain/event/hash.go` (1 prod + 1 test)
Content hashing and chain integrity hashing for events. Uses `core/encoding.ContentHash` for event hashes and direct SHA-256 for chain hashes.

### `domain/authz/` (2 prod + 2 test files)
Role/action/resource authorization policy matrix. Pure domain logic — no transport dependencies. Defines capabilities, policy decisions with machine-readable reason codes, and layered evaluators (`CanCampaignAccess`, `CanCharacterMutation`, `CanParticipantMutation`, etc.).

## Findings

### Finding 1: IDs Are Not Validated at Construction
- **Severity**: medium
- **Location**: `domain/ids/ids.go:12-60`
- **Issue**: All ID types are simple `type XxxID string` with no factory function or validation. Empty strings, whitespace, and arbitrary content are valid IDs. Validation happens ad-hoc at call sites (e.g., `strings.TrimSpace` in registry validation). This is a deliberate trade-off for simplicity, but means invalid IDs can propagate deep into the system before detection.
- **Recommendation**: Consider whether a `NewCampaignID(s string) (CampaignID, error)` constructor pattern would catch bugs earlier. If the current approach is intentional (IDs come from trusted sources like UUID generators), document this invariant in the package doc. The current doc says "compile-time safety against cross-entity aliasing" which accurately describes the value provided.

### Finding 2: AdversaryID and CountdownID Are System-Specific in Core ids Package
- **Severity**: low
- **Location**: `domain/ids/ids.go:52-60`
- **Issue**: `AdversaryID` and `CountdownID` are Daggerheart-specific concepts living in the core `ids` package alongside universal IDs (CampaignID, SessionID, etc.). This couples the core identity layer to a specific game system.
- **Recommendation**: Move system-specific IDs to `domain/bridge/daggerheart/` or a `daggerheart/ids` sub-package. Core `ids` should only contain identifiers used across all game systems.

### Finding 3: coreevent Imports domain/session — Unexpected Coupling
- **Severity**: medium
- **Location**: `domain/coreevent/refs.go:6`
- **Issue**: `coreevent` imports `domain/session` to re-export `SessionGateOpenedPayload`. A "core event references" package should be a leaf dependency, but importing a specific aggregate breaks this. Any change to `session.GateOpenedPayload` now forces recompilation of all `coreevent` consumers.
- **Recommendation**: Either move the payload type alias to the consuming package, or define the payload type in a shared location (e.g., in `coreevent` itself) and have `session` import it. The ref package should remain a leaf.

### Finding 4: commandids Doc Comment Misleads ("generates and validates")
- **Severity**: low
- **Location**: `domain/commandids/doc.go:1`
- **Issue**: Doc says "generates and validates stable command and invocation identifiers" but the package only declares constants — no generation or validation logic exists.
- **Recommendation**: Fix doc to: "Package commandids declares stable command type constants for the game service."

### Finding 5: ErrCampaignIDRequired Defined in ids, Re-exported in Both command and event
- **Severity**: low
- **Location**: `domain/ids/errors.go:10`, `domain/command/registry.go:20`, `domain/event/registry.go:19`
- **Issue**: The same sentinel error is defined in `ids` and re-exported via `var ErrCampaignIDRequired = ids.ErrCampaignIDRequired` in both `command` and `event` packages. Three packages all expose the same error. While the re-export prevents import changes for callers, `errors.Is` works with all three since they share identity.
- **Recommendation**: This is acceptable as a migration convenience. Document which is canonical (ids) and eventually remove re-exports.

### Finding 6: event/hash.go — Chain Hash Uses Direct SHA-256 While Event Hash Uses ContentHash
- **Severity**: low
- **Location**: `domain/event/hash.go:86-101`
- **Issue**: `EventHash` delegates to `coreencoding.ContentHash` which presumably does canonical JSON + SHA-256, while `ChainHash` does its own `canonicalJSON` + `sha256.Sum256` directly. The two hashing paths use slightly different abstractions for the same operation. `ChainHash` exists because it needs different input fields (seq, prev_hash), but the final hash computation is duplicated.
- **Recommendation**: Consider extracting the common `canonicalJSON → sha256` step into a shared helper. This is minor — the current code is correct and well-tested.

### Finding 7: event/hash.go Placement Is Correct for Domain
- **Severity**: info
- **Location**: `domain/event/hash.go`
- **Issue**: The review plan asked whether hash.go belongs in domain vs storage/integrity. The content hash is part of the event envelope's identity (used for deduplication and chain integrity) and is computed before storage. The chain hash links events in sequence. Both are domain-level invariants, not storage implementation details.
- **Recommendation**: Current placement is correct. The `storage/integrity/` package likely handles verification of stored events, which is a different concern.

### Finding 8: authz Policy Is Clean Domain Logic — No Transport Conflation
- **Severity**: info
- **Location**: `domain/authz/policy.go`
- **Issue**: The review plan asked whether `authz/` conflates domain policy with transport enforcement. It does not — the package is pure domain logic with no gRPC, HTTP, or transport imports. It defines the policy matrix and evaluators. Transport enforcement happens separately (likely in `api/grpc/game/authz/`).
- **Recommendation**: Clean separation. The existence of both `domain/authz/` and `api/grpc/game/authz/` (13 files) suggests proper layering. Session 22 will verify the transport side.

### Finding 9: authz CanCampaignAccess Scans Full Table Per Call
- **Severity**: low
- **Location**: `domain/authz/policy.go:229-238`
- **Issue**: `CanCampaignAccess` does a linear scan of the policy table (~27 rows) for every authorization check. This is functionally correct but could be a map lookup.
- **Recommendation**: The table is small enough that performance is not a concern. A `map[Capability]map[CampaignAccess]bool` would be cleaner but adds initialization complexity. Keep as-is unless profiling shows issues.

### Finding 10: authz Imports domain/participant — Correct but Creates Coupling Direction
- **Severity**: low
- **Location**: `domain/authz/policy.go:8`
- **Issue**: `authz` imports `participant` to use `CampaignAccess` type. This means `participant` cannot import `authz` without a cycle. The coupling direction is correct (policy evaluates access levels defined by participant), but it means participant-level authorization must be invoked from outside the participant package.
- **Recommendation**: Current design is correct. The `CampaignAccess` type could theoretically live in a shared package, but moving it would be over-engineering given the clean acyclic relationship.

## Summary Statistics
- Files reviewed: 19 (10 production, 9 test)
- Findings: 10 (0 critical, 0 high, 3 medium, 5 low, 2 info)
