---
title: "Daggerheart creation workflow"
parent: "Reference"
nav_order: 6
status: canonical
owner: engineering
last_reviewed: "2026-03-13"
---

# Daggerheart creation workflow and readiness contract

This document defines the durable contract for Daggerheart character creation,
workflow progress, reset behavior, and session-start readiness.

## Scope and posture

- The workflow is a clean-slate contract (no backward-compatibility shims).
- Workflow state is derived from profile fields (no persisted cursor).
- Core APIs are generic, while step semantics are Daggerheart-specific.

## Canonical step model

Daggerheart character creation is a strict 9-step sequence with an intentional
UX ordering choice: the three free-form steps stay at the end so structured
selection and validation happen first.

1. `class_subclass` (`class_id`, `subclass_id`, and any subclass-required setup such as `companion`)
2. `heritage` (`heritage.first_feature_ancestry_id`, `heritage.second_feature_ancestry_id`, `heritage.community_id`, optional `heritage.ancestry_label`)
3. `traits` (`traits_assigned` plus SRD distribution validation)
4. `equipment` (`starting_weapon_ids[]`, `starting_armor_id`, `starting_potion_item_id`)
5. `experiences` (`experiences[]`)
6. `domain_cards` (`domain_card_ids[]`)
7. `details` (`details_recorded` after recording starting details)
8. `background` (`background`, free-form text)
9. `connections` (`connections`, free-form text)

This ordering is accepted product behavior. It is not treated as a rules defect
unless it changes validation or readiness semantics.

## Profile fields and storage shape

The canonical profile contract is carried by
`systems.daggerheart.v1.DaggerheartProfile` and projected into
`daggerheart_character_profiles`.

Required workflow-related fields:

- `class_id`, `subclass_id`
- `subclass_creation_requirements[]`
- `heritage_json`
- `companion_sheet_json` when the selected subclass requires it
- `traits_assigned`
- `details_recorded`
- `starting_weapon_ids_json`, `starting_armor_id`, `starting_potion_item_id`
- `background`
- `experiences_json`
- `domain_card_ids_json`
- `connections`

## Progress evaluation semantics

Progress is computed from profile data via the Daggerheart evaluator.

- Raw field checks determine which requirements are satisfied.
- Reported `steps[].complete` is prefix-gated: later steps are not complete until
  all prior steps are complete.
- `next_step` is the first incomplete step; `0` means ready.
- `ready` is true only when all nine steps are complete.
- `unmet_reasons` reports all missing requirements.

## Generic workflow API

`game.v1.CharacterService` exposes four generic RPCs:

- `GetCharacterCreationProgress`
- `ApplyCharacterCreationStep`
- `ApplyCharacterCreationWorkflow` (atomic bulk apply)
- `ResetCharacterCreationWorkflow`

`ApplyCharacterCreationStep` carries a system-specific oneof payload.
For Daggerheart this is `DaggerheartCreationStepInput`.

`ApplyCharacterCreationWorkflow` carries all step payloads in one system-specific
message (`DaggerheartCreationWorkflowInput`) and applies them in canonical order
as one all-or-nothing write.

## Single pipeline policy

Workflow fields are write-once-through-pipeline semantics:

- Step-by-step: `ApplyCharacterCreationStep`
- Bulk import/test setup: `ApplyCharacterCreationWorkflow`

`PatchCharacterProfile` must not mutate workflow-owned fields. This prevents
bypassing ordering/content validation and keeps readiness invariants centralized
in one pipeline.

## Strict apply gating

Apply requests are strictly ordered.

- The requested step must equal `next_step`.
- Out-of-order writes are rejected with `FailedPrecondition`.
- Content IDs are validated against Daggerheart content stores.
- Heritage selection is resolved into stored feature ids.
  - Single ancestry stores the same ancestry id in both feature slots.
  - Mixed ancestry stores the first ancestry's first feature and the second
    ancestry's second feature, plus a free-form ancestry label.
- Subclass content may declare creation requirements.
  - `subclass-beastbound` currently requires a companion sheet during step 1.
- Trait values are validated via Daggerheart trait validation and SRD starting
  distribution (`+2,+1,+1,+0,+0,-1`).
- Domain cards must belong to one of the selected class domains.
- Starting equipment is validated against tier-1 weapon/armor catalog entries
  and allowed starting potion ids.
- Companion sheets are validated as static creation-time data.
  - `animal_kind`, `name`, `attack_description`, and exactly two experience
    names are required.
  - Experience modifiers normalize to `+2`.
  - Evasion/range/damage die are derived and stored as fixed defaults.
  - Damage type must be `physical` or `magic`.

All successful applies write through system-owned command execution using
`sys.daggerheart.character_profile.replace` (no direct projection mutation in
request handlers).

## Reset semantics

Reset is workflow-destructive by design.

- `ResetCharacterCreationWorkflow` emits
  `sys.daggerheart.character_profile.delete`.
- The Daggerheart adapter handles that event directly and removes the projected
  Daggerheart profile row for the character.
- Post-reset progress returns to step 1 with `ready = false`.

## Readiness enforcement

Readiness is enforced at two boundaries:

1. **Domain session start**: Daggerheart module `CharacterReady` delegates to
   workflow evaluation and blocks `session.start` when incomplete.
2. **Canonical readiness report API**: `GetCampaignSessionReadiness` evaluates
   the same domain readiness report used by session-start command handling and
   returns deterministic blockers for UI/operator consumers.

This keeps transport-level readiness surfaces consistent with domain decisions.

## MCP contract alignment

MCP readiness and import flows use `character_creation_workflow_apply` for
workflow fields. `character_profile_patch` is limited to non-workflow profile
fields. These are broader MCP/domain contracts, not part of the GM-safe
production AI bridge profile.

## Implementation map

- Workflow evaluator: `internal/services/game/domain/systems/daggerheart/creation_workflow.go`
- Workflow provider + service dispatch: `internal/services/game/api/grpc/game/charactertransport/character_workflow.go`
- CharacterService RPC handlers: `internal/services/game/api/grpc/game/charactertransport/character_service.go`
- Profile adapter/reset handling: `internal/services/game/domain/systems/daggerheart/internal/adapter/profile.go`
- Session-start readiness evaluator: `internal/services/game/domain/readiness/session_start.go`
- Campaign readiness RPC: `internal/services/game/api/grpc/game/campaigntransport/campaign_readiness_service.go`
- MCP DTO/handlers: `internal/services/mcp/domain/campaign.go`, `internal/services/mcp/domain/character_handlers.go`
