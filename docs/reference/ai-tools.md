---
title: "AI orchestration tools"
parent: "Reference"
nav_order: 2
last_reviewed: "2026-03-17"
---

# AI orchestration tools

GM-safe tools available during campaign-turn orchestration.

Broader bootstrap or dev-only registrations are intentionally omitted here.

## Campaign context

- `campaign_artifact_list`
- `campaign_artifact_get`
- `campaign_artifact_upsert`

## Scene and interaction

- `scene_create`
- `interaction_active_scene_set`
- `interaction_scene_player_phase_start`
- `interaction_scene_player_phase_accept`
- `interaction_scene_player_revisions_request`
- `interaction_scene_player_phase_end`
- `interaction_scene_gm_output_commit`
- `interaction_ooc_pause`
- `interaction_ooc_post`
- `interaction_ooc_ready_mark`
- `interaction_ooc_ready_clear`
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

- `internal/services/shared/mcpbridge/profile.go`
- `internal/services/ai/orchestration/gametools/tools.go`
- `internal/services/ai/orchestration/gametools/`
