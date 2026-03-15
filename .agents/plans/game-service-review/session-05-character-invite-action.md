# Session 5: Core Aggregates — Character, Invite, Action

## Status: `complete`

## Package Summaries

### `domain/character/` (~20 files, ~2,189 LOC total)
Character aggregate managing identity, avatar, aliases, ownership, and kind (PC/NPC). Includes fold, decider, registry, payload, and state files.

### `domain/invite/` (~17 files, ~1,769 LOC total)
Invite aggregate with PENDING→CLAIMED/DECLINED/REVOKED lifecycle. Clean state machine with status normalization for proto compatibility.

### `domain/action/` (~16 files, ~1,596 LOC total)
Action aggregate tracking roll→outcome causal chain. State is minimal (roll reference tracking), decider handles resolve/apply/reject, with story notes.

## Findings

### Finding 1: Character Aliases and Avatar Are Properly Scoped
- **Severity**: info
- **Location**: `domain/character/state.go:22-29`
- **Issue**: Character `aliases` (ordered list) and `avatar` (set+asset IDs) are character-specific identity data, not shared with participant. While participant also has avatar fields, these represent different entities. The alias list is stored on character state for fold/replay.
- **Recommendation**: No overlap concern. Different entities, same pattern.

### Finding 2: Invite Lifecycle Is a Clear State Machine
- **Severity**: info
- **Location**: `domain/invite/state.go`, `domain/invite/status.go`
- **Issue**: Invite transitions are enforced by the decider: only PENDING invites can be claimed/declined/revoked. Status normalization handles both raw strings ("PENDING") and proto-prefixed values ("INVITE_STATUS_PENDING"). The state struct is minimal (6 fields).
- **Recommendation**: Clean implementation. A state diagram in docs would help contributors.

### Finding 3: Action Aggregate State Is Minimal — Good Design
- **Severity**: info
- **Location**: `domain/action/state.go`
- **Issue**: `action.State` tracks just enough for replay-time invariant checks (e.g., ensuring an outcome references a valid roll). This keeps the aggregate lightweight.
- **Recommendation**: Good design — action state exists for command invariants, not for read models.

### Finding 4: Action Decider Has Clean Roll→Outcome→Note Boundaries
- **Severity**: info
- **Location**: `domain/action/` decider files
- **Issue**: The action decider handles three distinct concerns: roll resolution, outcome application/rejection, and story notes. Each is a separate command type with clear semantics.
- **Recommendation**: Well-decomposed.

### Finding 5: Consistent Error Pattern Across All Three Aggregates
- **Severity**: info
- **Location**: `domain/character/errors.go`, `domain/invite/errors.go` (if exists), `domain/action/` error handling
- **Issue**: All three aggregates use the same error patterns: sentinel errors for registration, rejection codes for domain-level command failures, and `fmt.Errorf` wrapping for infrastructure errors.
- **Recommendation**: Consistent patterns. No changes needed.

### Finding 6: Cross-Aggregate References Are Properly Scoped
- **Severity**: info
- **Location**: All three packages
- **Issue**: Character references `ids.ParticipantID` (ownership) and `ids.CharacterID` (identity). Invite references `ids.ParticipantID` and `ids.UserID` (claim target). Action has no cross-aggregate references beyond `ids.CampaignID`. All cross-references are via stable ID types, not direct state access.
- **Recommendation**: Clean dependency direction through the `ids` package.

### Finding 7: Character Kind (PC/NPC) Should Be Documented
- **Severity**: low
- **Location**: `domain/character/state.go:19`
- **Issue**: `Kind` field is typed but the available kinds and their behavioral implications aren't immediately visible from the state file. Contributors need to find the kind type definition to understand the options.
- **Recommendation**: Add a brief doc comment listing the kind values and their significance.

## Summary Statistics
- Files reviewed: ~53 (20 character + 17 invite + 16 action, prod and test)
- Findings: 7 (0 critical, 0 high, 0 medium, 1 low, 6 info)
