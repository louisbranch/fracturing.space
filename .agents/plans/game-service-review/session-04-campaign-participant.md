# Session 4: Core Aggregates — Campaign and Participant

## Status: `complete`

## Package Summaries

### `domain/campaign/` (18 prod files, ~3,523 LOC total)
Campaign aggregate with lifecycle management. Files: decider.go (dispatch), decider_create.go, decider_update.go, decider_lifecycle.go (end/archive/restore), decider_ai.go (bind/unbind/rotate), decider_fork.go, decider_shared.go (helpers), covers.go (cover asset logic), normalize.go (create input normalization), policy.go (status/operation matrix), labels.go, errors.go, fold.go, payload.go, registry.go, state.go, status.go, doc.go.

### `domain/participant/` (12 prod files, ~3,479 LOC total)
Participant aggregate managing campaign membership, roles, and controller bindings. State includes Joined/Left lifecycle flags, role, controller, campaign access level, and avatar.

## Findings

### Finding 1: Campaign's 7 Decider Files Are Warranted
- **Severity**: info
- **Location**: `domain/campaign/decider*.go`
- **Issue**: The 7 decider files split responsibility cleanly: create, update, lifecycle (end/archive/restore), AI binding, fork, shared helpers, and the dispatch router. Each file handles 1-3 command types. The alternative (one large decider) would be ~600+ lines.
- **Recommendation**: Good decomposition. The file-per-concern pattern is clear and consistent.

### Finding 2: campaign/policy.go vs domain/authz/ — Clean Separation
- **Severity**: info
- **Location**: `domain/campaign/policy.go`, `domain/authz/policy.go`
- **Issue**: `campaign/policy.go` handles campaign *status/operation* rules (e.g., "draft campaigns cannot be archived"). `domain/authz/` handles *role/action/resource* authorization (e.g., "members cannot manage sessions"). These are orthogonal concerns — status gates vs access control.
- **Recommendation**: Clean boundary. No changes needed.

### Finding 3: campaign/covers.go Uses Package-Level State
- **Severity**: medium
- **Location**: `domain/campaign/covers.go:11-13`
- **Issue**: `campaignCoverManifest` and `campaignCoverAssetCatalog` are package-level variables initialized at import time. This couples campaign to the `catalog` package at module load and makes testing harder (can't inject different manifests). However, the cover manifest is effectively immutable static data.
- **Recommendation**: If the cover manifest is truly static and immutable, this is acceptable. If it needs to be testable with different data, extract an interface or pass the manifest via the decider constructor.

### Finding 4: campaign/normalize.go Defaults GmMode to AI
- **Severity**: low
- **Location**: `domain/campaign/normalize.go:29`
- **Issue**: `NormalizeCreateInput` defaults `GmMode` to `GmModeAI` when unspecified. This is a business decision buried in normalization code. A contributor might miss this default.
- **Recommendation**: Document this default prominently or move it to the API transport layer where defaults are more visible.

### Finding 5: Session State Has 22 Fields — Complex but Warranted
- **Severity**: medium
- **Location**: `domain/session/state.go:9-55`
- **Issue**: `session.State` has 22 fields spanning: lifecycle (Started/Ended), session identity (SessionID/Name), gate context (GateOpen/GateID/GateType/GateMetadataJSON), spotlight (SpotlightType/SpotlightCharacterID), scene tracking (ActiveSceneID), GM authority, OOC pause state (3 fields), and AI turn state (8 fields). The 8 AI turn fields are a concern — they feel like a sub-aggregate.
- **Recommendation**: Consider grouping AI turn fields into an `AITurnState` sub-struct to improve readability and make the distinct concerns clearer. This is a structural improvement, not a functional one.

### Finding 6: Participant State Is Well-Scoped
- **Severity**: info
- **Location**: `domain/participant/state.go:9-32`
- **Issue**: `participant.State` has 11 fields covering identity, role, controller, access, avatar, and pronouns. Clean and focused.
- **Recommendation**: No changes needed.

### Finding 7: Character State Has Both OwnerParticipantID and ParticipantID
- **Severity**: low
- **Location**: `domain/character/state.go:30-33`
- **Issue**: `character.State` has both `OwnerParticipantID` (governance owner for mutation authority) and `ParticipantID` (controller for operational gameplay). The distinction is documented but the naming is subtle — "owner" vs "controller" would be clearer if the field name were `ControllerParticipantID` instead of just `ParticipantID`.
- **Recommendation**: Consider renaming `ParticipantID` to `ControllerParticipantID` for clarity. The current state works but requires reading docs to understand the distinction.

### Finding 8: Invite State Has Clear Lifecycle
- **Severity**: info
- **Location**: `domain/invite/state.go:9-22`, `domain/invite/status.go`
- **Issue**: Invite has a clean state machine: PENDING → CLAIMED/DECLINED/REVOKED. The `NormalizeStatusLabel` function handles both raw and proto-prefixed values. State is minimal (6 fields).
- **Recommendation**: Clean design. Consider adding a state transition diagram in the doc.go.

### Finding 9: Avatar Fields Duplicated Across Participant and Character
- **Severity**: medium
- **Location**: `domain/participant/state.go:27-29`, `domain/character/state.go:22-25`
- **Issue**: Both `participant.State` and `character.State` have `AvatarSetID` and `AvatarAssetID` fields. Participants have avatars for their profile, characters have avatars for gameplay. These are semantically different (user identity vs character identity) but structurally identical. The naming is consistent.
- **Recommendation**: The duplication is intentional — participants and characters are different entities with independent avatar lifecycles. No refactoring needed, but document the distinction in contributing guide.

### Finding 10: campaign/labels.go Pattern Is Consistent
- **Severity**: info
- **Location**: `domain/campaign/labels.go`, `domain/session/labels.go`
- **Issue**: Both campaign and session have `labels.go` files for stable machine-readable labels. This is consistent across aggregates.
- **Recommendation**: Good pattern.

## Summary Statistics
- Files reviewed: 30+ (18 campaign prod + 12 participant prod + test files)
- Findings: 10 (0 critical, 0 high, 3 medium, 2 low, 5 info)
