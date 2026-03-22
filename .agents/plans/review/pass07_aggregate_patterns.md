# Pass 7: Aggregate Domain Pattern Consistency

## Summary

Six aggregate packages were reviewed: campaign, session, participant, character, scene, action.
Overall the codebase shows strong structural consistency across aggregates. The core
decider/fold/registry/payload pattern is well-established, and most deviations are
justified by domain complexity (e.g., scene's multi-entity Decide signature). The
findings below focus on genuine inconsistencies, correctness risks, and contributor
friction points.

**Critical findings:** 2 (correctness risks)
**Moderate findings:** 8 (inconsistencies and anti-patterns)
**Minor findings:** 8 (contributor friction / naming drift)

---

## Findings

### 1. [correctness risk] `validateEmptyPayload` rejects valid empty JSON forms

**File:** `campaign/registry.go:306`

```go
func validateEmptyPayload(raw json.RawMessage) error {
    if string(raw) != "{}" {
        return errors.New("payload must be empty")
    }
    return nil
}
```

This check uses string comparison `string(raw) != "{}"`. It rejects:
- `null` (valid JSON, common default for empty payloads)
- empty string `""` (possible edge case from transport)
- `{ }` (whitespace-padded empty object)
- `{}` with trailing newline

Only campaign uses this pattern (for `end`, `archive`, `restore`). No other aggregate
enforces empty-payload semantics this way. The other aggregates simply unmarshal into a
typed payload struct and tolerate empty/null JSON.

**Proposal:** Replace with `json.Unmarshal` into an empty struct or `map[string]any` and
check `len(fields) == 0`, or use `json.Valid(raw)` plus an empty-object check that
trims whitespace.

---

### 2. [correctness risk] action decider uses hardcoded rejection code string instead of shared constant

**File:** `action/decider.go:62`

```go
default:
    return command.Reject(command.Rejection{
        Code:    "COMMAND_TYPE_UNSUPPORTED",
        Message: "command type is not supported by action decider",
    })
```

Every other aggregate uses `command.RejectionCodeCommandTypeUnsupported` for the default
case. Action hardcodes the string literal `"COMMAND_TYPE_UNSUPPORTED"`. If the shared
constant ever changes, action will drift. Additionally, action does not include this
code in its `RejectionCodes()` return, creating a collision-detection blind spot.

**Proposal:** Use `command.RejectionCodeCommandTypeUnsupported` like all other aggregates.

---

### 3. [anti-pattern] Registry contract type naming inconsistency across aggregates

**Files:**
- `campaign/registry.go:12-13` - `commandContract`, `eventProjectionContract`
- `session/registry_support.go:8-9` - `commandContract`, `eventProjectionContract`
- `participant/registry.go:11-19` - `commandRegistration`, `eventProjectionRegistration`
- `character/registry.go:12-13` - `commandContract`, `eventProjectionContract`
- `scene/registry_support.go:8-9` - `commandContract`, `eventProjectionContract`
- `action/registry.go` - uses inline `command.Definition` slices, no wrapper struct

Three naming conventions coexist:
1. `commandContract` / `eventProjectionContract` (campaign, session, character, scene)
2. `commandRegistration` / `eventProjectionRegistration` (participant)
3. No wrapper struct at all (action)

**Proposal:** Standardize on `commandContract` / `eventProjectionContract` across all six
aggregates. Action should adopt the struct wrapper pattern for consistency with the
other five.

---

### 4. [anti-pattern] Registry organization: monolithic vs split registry files

**Files:**
- Campaign: single `registry.go` (all contracts, validation, and API in one file)
- Session: split across `registry.go`, `registry_lifecycle.go`, `registry_gate.go`,
  `registry_spotlight.go`, `registry_interaction.go`, `registry_support.go`
- Scene: split across `registry.go`, `registry_lifecycle.go`, `registry_character.go`,
  `registry_gate.go`, `registry_spotlight.go`, `registry_interaction.go`,
  `registry_support.go`
- Participant: single `registry.go`
- Character: single `registry.go`
- Action: single `registry.go`

Session and scene use multi-file registry organization with concat helpers
(`appendSessionCommandContracts`, `appendSceneCommandContracts`). The other four use
single files. This is justifiable by size (session has 21 commands, scene has 20)
versus campaign's 10 and participant's 6.

**Assessment:** Acceptable divergence driven by scale. No action needed unless the
smaller aggregates grow significantly.

---

### 5. [missing best practice] Event definition `Intent` field inconsistently set

**Files:**
- Campaign events: no `Intent` field set in `eventProjectionContract.definition`
- Participant events: no `Intent` field set
- Character events: no `Intent` field set
- Session events: all set to `event.IntentProjectionAndReplay`
- Scene events: all set to `event.IntentProjectionAndReplay`
- Action events: explicitly set to `IntentReplayOnly` or `IntentAuditOnly`

Campaign, participant, and character event definitions omit `Intent` entirely, relying on
the zero-value. Session, scene, and action explicitly declare intent. This creates
ambiguity about whether the omission is intentional (default-projection) or accidental.

**Proposal:** Explicitly set `Intent` on all event definitions across all aggregates.
This makes the contract self-documenting and prevents surprises when a new consumer
checks the field.

---

### 6. [contributor friction] Decider `now` normalization placement varies

**Files:**
- Campaign `Decide()` (decider.go:75): does NOT call `command.NowFunc(now)` at
  dispatcher level; individual handlers call it implicitly via `command.NowFunc(now)()`
  inline
- Session `Decide()` (decider.go:126): calls `now = command.NowFunc(now)` at top of
  dispatcher
- Participant `Decide()` (decider.go:75): does NOT call at dispatcher level; each sub-
  decider calls `now = command.NowFunc(now)` at its own top
- Character `Decide()` (decider.go:54): calls `now = command.NowFunc(now)` at top of
  dispatcher
- Scene `Decide()` (decider.go:143): calls `now = command.NowFunc(now)` at top of
  dispatcher
- Action `Decide()` (decider.go:49): calls `now = command.NowFunc(now)` at top of
  dispatcher

Campaign's deciders use `command.NowFunc(now)()` inline at call sites (e.g.,
`decider_create.go:31`). Participant's sub-deciders each independently call
`now = command.NowFunc(now)`. This redundant normalization in every sub-decider is
error-prone for new handlers.

**Proposal:** Normalize `now` once at the `Decide()` dispatcher level in all aggregates,
matching session/character/scene/action.

---

### 7. [contributor friction] Shared helper function naming and location inconsistency

**Files:**
- Campaign: `commandDecodeMessage()` in `decider_shared.go`
- Participant: `decodeCommandPayload[T]()`, `ensureParticipantActive()`,
  `acceptParticipantEvent()` in `decider_shared.go`
- Character: `normalizeCharacterKindLabel()`, `acceptCharacterEvent()` in
  `decider_shared.go`
- Scene: `requireActiveScene()`, `rejectPayloadDecode()`, helper functions in
  `decider_shared.go`
- Action: `acceptActionEvent()` in `decider_shared.go`
- Session: no `decider_shared.go`, inline helpers

Participant and character have centralized `acceptXxxEvent()` helpers that standardize
event construction. Campaign lacks this; each campaign sub-decider manually constructs
events. Session also lacks this centralization.

Campaign's `commandDecodeMessage()` is a single-line string formatter. Participant uses
a generic `decodeCommandPayload[T]()`. Neither campaign nor session uses generics for
payload decode, doing manual `json.Unmarshal` inline in every handler.

**Proposal:** Campaign and session should adopt the `acceptXxxEvent()` pattern and
consider adopting the generic `decodeCommandPayload[T]()` from participant.

---

### 8. [anti-pattern] Decider Decide() signature divergence for scene aggregate

**File:** `scene/decider.go:142`

```go
func Decide(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision
```

vs all others:
```go
func Decide(state State, cmd command.Command, now func() time.Time) command.Decision
```

Scene takes `map[ids.SceneID]State` instead of a single `State`. This is documented and
intentional (cross-scene commands need multi-entity reads). The doc comment in
`scene/doc.go` explains this clearly.

**Assessment:** Justified deviation, well-documented. No action needed.

---

### 9. [missing best practice] Session decider does not check `state.Started` for most commands

**Files:** `session/decider_gate.go`, `session/decider_spotlight.go`,
`session/decider_ooc.go`, `session/decider_session_authority.go`,
`session/decider_ai_turn.go`

Session's `decideStart` checks `state.Started` (reject if already started) and
`decideEnd` checks `!state.Started` (reject if not started). However, gate, spotlight,
OOC, scene-activate, GM-authority, and AI-turn commands do NOT check `state.Started`
before proceeding. This means a gate could theoretically be opened, a spotlight set, or
OOC opened on a session that has not started or has ended.

Compare with campaign: every non-create handler checks `state.Created`.

**Proposal:** Add a `requireSessionActive(state)` guard (checking `state.Started &&
!state.Ended`) to gate, spotlight, OOC, scene-activate, GM-authority, and AI-turn
deciders. The `ActiveSession` command definition constraint may handle this at a higher
layer, but defense-in-depth at the decider level matches the campaign pattern.

---

### 10. [contributor friction] Payload validation depth varies across aggregates

**Files:**
- Campaign `validateAIBindPayload`: checks `ai_agent_id` is non-empty (deep validation)
- Campaign `validateAIAuthRotatePayload`: checks `reason` non-empty (deep validation)
- Campaign `validateForkPayload`: checks required IDs (deep validation)
- Campaign `validateCreatePayload`: unmarshal-only (shallow)
- Session all `validate*Payload`: unmarshal-only (shallow)
- Participant all `validate*Payload`: unmarshal-only (shallow)
- Character all `validate*Payload`: unmarshal-only (shallow)
- Scene all `validate*Payload`: unmarshal-only (shallow)
- Action `validateRollResolvePayload`: checks request_id and roll_seq (deep validation)
- Action `validateOutcomeApplyPayload`: deep validation of effects
- Session `validateGateOpenedPayload`: normalizes gate type + metadata (deep)

Some registry-level validators do deep field validation (campaign AI, action roll/outcome,
session gate), while most only verify the JSON can unmarshal. This creates two levels of
enforcement: registry validators catch schema issues, while deciders catch business rules.
The inconsistency means some invariants are enforced twice (decider + registry) while
others only in the decider.

**Assessment:** This is a judgment call -- deep validation in registry enables earlier
rejection. The risk is duplicated validation logic that can drift. Recommend documenting
the convention: registry validates shape, decider validates business rules.

---

### 11. [correctness risk / minor] Fold update field appliers silently skip unknown fields

**Files:**
- `participant/fold.go:163-164`: `applyParticipantUpdateFields` skips unknown keys with
  `continue`
- `character/fold.go:132-135`: `applyCharacterUpdateFields` skips unknown keys with
  `continue`
- `campaign/fold.go:62-77`: `foldUpdated` switch with no default -- silently ignores
  unknown keys

The deciders properly reject unknown update fields, so this is defense-in-depth
behavior for replay. However, the fold silently skipping means a corrupt event with an
unknown field key would not surface an error during replay. This is likely intentional
for forward-compatibility (old fold seeing new fields from a newer version), but it is
undocumented.

**Proposal:** Add a code comment documenting the intentional silent-skip behavior as a
forward-compatibility strategy.

---

### 12. [contributor friction] Rejection code prefix conventions

All aggregates consistently prefix rejection codes with their domain name:
- Campaign: `CAMPAIGN_*`
- Session: `SESSION_*`
- Participant: `PARTICIPANT_*`
- Character: `CHARACTER_*`
- Scene: `SCENE_*`
- Action: uses unprefixed codes like `REQUEST_ID_REQUIRED`, `ROLL_SEQ_REQUIRED`,
  `OUTCOME_ALREADY_APPLIED`, `NOTE_CONTENT_REQUIRED`

**File:** `action/decider.go:21-27`

Action's codes lack a domain prefix, breaking the convention. `REQUEST_ID_REQUIRED`
could conflict with a hypothetical request aggregate.

**Proposal:** Prefix action rejection codes with `ACTION_` (e.g.,
`ACTION_REQUEST_ID_REQUIRED`, `ACTION_ROLL_SEQ_REQUIRED`).

---

### 13. [contributor friction] `CommandTypeCreateWithParticipants` registered but not handled by Decide()

**File:** `campaign/decider.go:76-97`, `campaign/registry.go:32-36`

`CommandTypeCreateWithParticipants` is registered in the command registry but is not
present in the `Decide()` switch. The doc comment in `campaign/doc.go` explains this
lives in the `campaignbootstrap` workflow package. However, any caller routing this
command type through the campaign decider would get `COMMAND_TYPE_UNSUPPORTED`, which
could be confusing.

**Assessment:** This is documented and intentional -- the command is registered for
schema validation only, handled by a different workflow. Consider adding a code comment
in `Decide()` noting the intentional omission.

---

### 14. [missing best practice] State types use value semantics consistently

All six aggregates use value-type (not pointer) State:
- `campaign.State` - value type, no pointer fields
- `session.State` - value type, has `map[ids.ParticipantID]bool` (OOCReadyParticipants)
  and `[]byte` (GateMetadataJSON)
- `participant.State` - value type, no pointer fields
- `character.State` - value type, has `[]string` (Aliases)
- `scene.State` - value type, has maps and slices (Characters, PlayerPhaseSlots, etc.)
- `action.State` - value type, has maps (Rolls, AppliedOutcomes)

**Assessment:** Consistent. However, note that session and scene State structs contain
maps which are reference types. Fold handlers that mutate maps (e.g.,
`foldOOCReadyMarked`, `foldCharacterAdded`) correctly initialize nil maps before use.
This is handled well.

---

### 15. [anti-pattern] Campaign uses dual error systems (apperrors + rejection codes)

**Files:**
- `campaign/errors.go` - defines `ErrEmptyName`, `ErrInvalidGmMode`,
  `ErrInvalidGameSystem`, `ErrInvalidCampaignStatusTransition` using `apperrors.New()`
- `campaign/policy.go` - defines `ErrCampaignStatusDisallowsOperation` using
  `apperrors.New()`
- `campaign/decider.go` - uses rejection codes (string constants) in `command.Rejection`

Campaign has both `apperrors.Error` values (for use by transport/handler callers) AND
rejection code strings (for use by deciders). The deciders never reference the
`apperrors.Error` values. The `errors.go` file appears to serve transport-layer callers
who call `NormalizeCreateInput()` and similar pre-decider functions.

Action also has this pattern: `action/errors.go` defines `ErrOutcomeAlreadyApplied` but
the decider uses a rejection code string. The decider does reference
`ErrOutcomeAlreadyApplied.Error()` for the message (action/decider_outcome.go:49).

Session, participant, character, and scene do NOT have `errors.go` files.

**Assessment:** This dual-system is architectural -- `apperrors` for transport-layer
callers, rejection codes for event-sourced deciders. It is not problematic per se, but
the relationship should be documented.

---

### 16. [missing best practice] Campaign and action have `errors.go`; others do not

**Files:**
- `campaign/errors.go` - defines 4 sentinel errors
- `action/errors.go` - defines 1 sentinel error + outcome field constants

Session, participant, character, and scene have no `errors.go`. Their error handling is
entirely within rejection codes.

**Assessment:** The `errors.go` files serve pre-decider validation (normalize functions
used by transport). The four aggregates without them do not expose normalize functions
that need to return errors to transport callers. This is consistent usage.

---

### 17. [contributor friction] Fold function signature consistency

All six aggregates use the same fold signature:
```go
func Fold(state State, evt event.Event) (State, error)
```

All use `fold.CoreFoldRouter[State]` for dispatch. All register handlers in `init`-time
`newFoldRouter()` functions. This is fully consistent.

**Assessment:** No issues.

---

### 18. [missing best practice] Campaign has `normalize.go` and `policy.go`; others lack equivalent

**Files:**
- Campaign: `normalize.go` (NormalizeCreateInput), `policy.go`
  (ValidateCampaignOperation)
- Session, participant, character, scene, action: no equivalent

Campaign is the only aggregate with a standalone `policy.go` file that provides
status-based operation gating. Other aggregates handle their policy logic inline in
decider functions (e.g., `scene/decider_shared.go:requireActiveScene()`).

Campaign's `normalize.go` provides `NormalizeCreateInput()` for transport-layer callers
to pre-validate input before building a command. Other aggregates handle normalization
entirely within decider functions.

**Assessment:** This is justifiable by campaign's richer lifecycle state machine. No
action needed unless other aggregates develop similar complexity.

---

## Cross-cutting Refactoring Proposals

### A. Standardize `now` normalization

Move `now = command.NowFunc(now)` to the top of every `Decide()` dispatcher function.
Currently campaign and participant do it at sub-function level.

**Affected:** campaign/decider.go, participant/decider.go

### B. Standardize registry contract types

Rename participant's `commandRegistration`/`eventProjectionRegistration` to
`commandContract`/`eventProjectionContract`. Add wrapper structs to action.

**Affected:** participant/registry.go, action/registry.go

### C. Fix action rejection code constant usage

Replace hardcoded `"COMMAND_TYPE_UNSUPPORTED"` with `command.RejectionCodeCommandTypeUnsupported`.

**Affected:** action/decider.go:62

### D. Add domain prefix to action rejection codes

Prefix all action rejection codes with `ACTION_`.

**Affected:** action/decider.go (constants)

### E. Fix `validateEmptyPayload`

Replace string comparison with proper JSON validation that accepts `null`, `{}`,
`{ }`, and similar valid empty forms.

**Affected:** campaign/registry.go:304-310

### F. Explicitly set `Intent` on all event definitions

Add `Intent: event.IntentProjectionAndReplay` to campaign, participant, and character
event definitions.

**Affected:** campaign/registry.go, participant/registry.go, character/registry.go

### G. Add session state guards to non-lifecycle deciders

Add `requireSessionActive()` check to gate, spotlight, OOC, scene-activate,
GM-authority, and AI-turn handlers.

**Affected:** session/decider_gate.go, session/decider_spotlight.go,
session/decider_ooc.go, session/decider_session_authority.go,
session/decider_ai_turn.go
