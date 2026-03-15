---
title: "MCP tools"
parent: "Reference"
nav_order: 2
last_reviewed: "2026-03-12"
---

# MCP tools

Exact MCP tool names currently registered by the server.

## Context

- `set_context`

## Campaign

- `campaign_create`
- `campaign_end`
- `campaign_archive`
- `campaign_restore`

## Participants

- `participant_create`
- `participant_update`
- `participant_delete`

## Characters

- `character_create`
- `character_update`
- `character_delete`
- `character_control_set`
- `character_sheet_get`
- `character_profile_patch`
- `character_state_patch`

## Session and outcomes

- `session_start`
- `session_end`
- `session_action_roll`
- `session_roll_outcome_apply`

## Interaction

- `interaction_active_scene_set`
- `interaction_scene_player_phase_start`
- `interaction_scene_player_post_submit`
- `interaction_scene_player_phase_yield`
- `interaction_scene_player_phase_unyield`
- `interaction_scene_player_phase_end`
- `interaction_ooc_pause`
- `interaction_ooc_post`
- `interaction_ooc_ready_mark`
- `interaction_ooc_ready_clear`
- `interaction_ooc_resume`

## Daggerheart utilities

- `duality_rules_version`
- `duality_action_roll`
- `duality_outcome`
- `duality_explain`
- `duality_probability`
- `roll_dice`

## Verification

For implementation and registration details, inspect:

- `internal/services/mcp/service/server.go`
- `internal/services/mcp/service/server_registration.go`
- `internal/services/mcp/domain/`
