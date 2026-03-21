---
title: "AI orchestration tools"
parent: "Reference"
nav_order: 2
last_reviewed: "2026-03-18"
---

# AI orchestration tools

GM-safe tools available during campaign-turn orchestration.

Broader bootstrap or dev-only registrations are intentionally omitted here.

## Campaign context

- `campaign_artifact_list`
- `campaign_artifact_get`
- `campaign_artifact_upsert`
- `campaign_memory_section_read`
- `campaign_memory_section_update`

## Scene lifecycle

- `scene_create`
- `scene_update`
- `scene_end`
- `scene_transition`
- `scene_add_character`
- `scene_remove_character`

## Interaction

- `interaction_activate_scene`
- `interaction_open_scene_player_phase`
- `interaction_resolve_scene_player_review`
- `interaction_record_scene_gm_interaction`
- `interaction_session_ooc_resolve`
- `interaction_open_session_ooc`
- `interaction_post_session_ooc`
- `interaction_mark_ooc_ready_to_resume`
- `interaction_clear_ooc_ready_to_resume`
- `interaction_ooc_resume`

## Rules and system reference

- `duality_rules_version`
- `duality_action_roll`
- `duality_outcome`
- `duality_explain`
- `duality_probability`
- `roll_dice`
- `system_reference_search`
- `system_reference_read`

## Not in the production profile

- campaign lifecycle and fork tools
- participant and character CRUD tools
- session lifecycle tools
- event-list/admin-style tooling

Integration harnesses may enable test-only context bootstrap tooling, but that
surface is not part of the runtime contract described here.

## Verification

For implementation, inspect:

- `internal/services/ai/orchestration/gametools/tools.go`
- `internal/services/ai/orchestration/tool_policy.go`
- `internal/services/ai/orchestration/gametools/tools.go`
- `internal/services/ai/orchestration/gametools/`
