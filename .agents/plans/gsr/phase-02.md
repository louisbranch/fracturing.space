# GSR Phase 2: Domain Model — Aggregates, Entities, Value Objects

## Summary

The domain model has **significant type-safety and invariant-enforcement gaps**. Raw `string` types for identifiers, inconsistent enum types in state structs, unprotected mutable maps, and `map[module.Key]any` for system state create architecture fragility. The aggregate state conflates decision-time invariants with projection-friendly data.

## Findings

### F2.1: Raw String IDs Instead of Newtypes — Critical

**Severity:** critical

All identity fields (`ParticipantID`, `CharacterID`, `CampaignID`, `UserID`, `SessionID`, `InviteID`, `SceneID`, `GateID`) are raw `string`. No compile-time distinction prevents assigning a `CharacterID` to a `SessionID` field.

**Impact:** Aliasing errors compile silently. No IDE support for ID operations. String-based routing ambiguity when multiple IDs are present.

**Recommendation:** Introduce newtype IDs: `type CampaignID string`, `type ParticipantID string`, etc. Apply to all state fields and map keys.

### F2.2: Inconsistent Enum Types in State Structs — Important

**Severity:** important

Some state fields use typed enums (e.g., `campaign.Status Status`), others use raw strings for the same domain concepts (e.g., `participant.Role string` despite `type Role string` existing in labels.go). Same for `character.Kind`, `session.SpotlightType`, `invite.Status`.

**Recommendation:** Replace all raw `string` enum fields with their typed constants. Update fold functions to assign typed values.

### F2.3: `aggregate.State.Systems` as `map[module.Key]any` — Critical

**Severity:** critical

**Location:** `domain/aggregate/state.go:68`

Type safety erasion at the highest architectural level. System state stored as `any`, recovered via `AssertState[T]` which returns zero-value on nil (silent failure). No compile-time checking. Checkpoint cloning does shallow copy only.

**Recommendation:** Consider typed system state container or at minimum make `AssertState[T]` fail-loud on nil input.

### F2.4: Mutable Maps Without Nil-Guard Constructors — Important

**Severity:** important

All entity maps in `aggregate.State` (`Participants`, `Characters`, `Invites`, `Scenes`, `Systems`) are nil by default. No constructor enforces initialization. Nil checks scattered across fold functions, deciders, and helpers — inconsistent coverage.

**Recommendation:** Add constructor that initializes all maps. Centralize nil-guard responsibility.

### F2.5: Missing Invariant Enforcement in State Construction — Important

**Severity:** important

No `New*State()` constructors exist in any domain package. State constructed via zero-values or direct field assignment in fold functions. Empty ParticipantID, empty CharacterID accepted without guard.

**Recommendation:** Add invariant-enforcing constructors for all state types.

### F2.6: Denormalized Data in Aggregate State — Minor

**Severity:** minor

Character.SystemProfile (`map[string]any`), Participant display fields (Name, AvatarSetID, Pronouns), Session/Scene names are projection-friendly but not core decision invariants.

**Recommendation:** Clarify whether aggregate.State is a decision-time snapshot or full read model. Document the design decision.

### F2.7: Anemic Domain Model — Minor

**Severity:** minor

State structs contain only data; behavior scattered across deciders, helpers, and readiness checks. Only one behavior method exists (`Scene.HasPC`).

**Recommendation:** Add semantic query methods to state types where they clarify intent (e.g., `participant.State.IsActive()`, `character.State.IsReadyForPlay()`).

### F2.8: Leaky Aggregate — Maps Exposed Without Access Control — Minor

**Severity:** minor

All maps are public fields with direct indexing. External code can corrupt maps without domain validation. No getter/setter semantics.

**Recommendation:** Consider encapsulating maps with getter methods that handle nil checks internally.

### F2.9: `AssertState[T]` Silent Nil Handling — Minor

**Severity:** minor

**Location:** `domain/aggregate/state.go:16-37`

Returns zero-value on nil input instead of error. Masks architectural fragility.

**Recommendation:** Return error for nil input to fail loud.

## Priority Actions

1. **Immediate:** Introduce ID newtypes for all identifier fields
2. **Immediate:** Enforce typed enum constants in all State structs
3. **Short-term:** Add invariant-enforcing constructors
4. **Short-term:** Encapsulate aggregate maps with getters
5. **Medium-term:** Address `map[module.Key]any` type safety

## Cross-References

- **Phase 3** (Event System): Fold functions construct state without validation
- **Phase 5** (Engine): `AssertState[T]` used in applier
- **Phase 9** (Module Extension): `StateFactory` returns `any`
