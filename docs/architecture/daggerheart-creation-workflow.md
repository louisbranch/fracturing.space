---
title: "Daggerheart creation workflow"
parent: "Architecture"
nav_order: 8
status: canonical
owner: engineering
last_reviewed: "2026-02-26"
---

# Daggerheart creation workflow and readiness contract

This document defines the durable contract for Daggerheart character creation,
workflow progress, reset behavior, and session-start readiness.

## Scope and posture

- The workflow is a clean-slate contract (no backward-compatibility shims).
- Workflow state is derived from profile fields (no persisted cursor).
- Core APIs are generic, while step semantics are Daggerheart-specific.

## Canonical step model

Daggerheart character creation is a strict SRD-aligned 9-step sequence:

1. `class_subclass` (`class_id` + `subclass_id`)
2. `heritage` (`ancestry_id` + `community_id`)
3. `traits` (`traits_assigned` plus SRD distribution validation)
4. `details` (`details_recorded` after recording starting details)
5. `equipment` (`starting_weapon_ids[]`, `starting_armor_id`, `starting_potion_item_id`)
6. `background` (`background`, free-form text)
7. `experiences` (`experiences[]`)
8. `domain_cards` (`domain_card_ids[]`)
9. `connections` (`connections`, free-form text)

Step 6 and step 9 are intentionally free-form text fields.

## Profile fields and storage shape

The canonical profile contract is carried by
`systems.daggerheart.v1.DaggerheartProfile` and projected into
`daggerheart_character_profiles`.

Required workflow-related fields:

- `class_id`, `subclass_id`, `ancestry_id`, `community_id`
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
- Trait values are validated via Daggerheart trait validation and SRD starting
  distribution (`+2,+1,+1,+0,+0,-1`).
- Domain cards must belong to one of the selected class domains.
- Starting equipment is validated against tier-1 weapon/armor catalog entries
  and allowed starting potion ids.

All successful applies write through core command execution using
`character.profile_update` (no direct projection mutation in request handlers).

## Reset semantics

Reset is workflow-destructive by design.

- `ResetCharacterCreationWorkflow` emits a `character.profile_update` with
  `{"daggerheart":{"reset":true}}`.
- The Daggerheart profile adapter interprets this as a delete operation and
  removes the projected Daggerheart profile row for the character.
- Post-reset progress returns to step 1 with `ready = false`.

## Readiness enforcement

Readiness is enforced at two boundaries:

1. **Domain session start**: Daggerheart module `CharacterReady` delegates to
   workflow evaluation and blocks `session.start` when incomplete.
2. **Projection read model**: campaign `can_start_session` includes Daggerheart
   completeness predicates so list/get campaign views align with runtime
   readiness behavior.

This keeps transport-level readiness surfaces consistent with domain decisions.

## MCP contract alignment

MCP readiness and import flows use `character_creation_workflow_apply` for
workflow fields. `character_profile_patch` is limited to non-workflow profile
fields.

## Implementation map

- Workflow evaluator: `internal/services/game/domain/bridge/daggerheart/creation_workflow.go`
- Workflow provider + service dispatch: `internal/services/game/api/grpc/game/character_workflow.go`
- CharacterService RPC handlers: `internal/services/game/api/grpc/game/character_service.go`
- Profile adapter/reset handling: `internal/services/game/domain/bridge/daggerheart/adapter_profile.go`
- Campaign readiness SQL: `internal/services/game/storage/sqlite/queries/campaigns.sql`
- MCP DTO/handlers: `internal/services/mcp/domain/campaign.go`, `internal/services/mcp/domain/character_handlers.go`
