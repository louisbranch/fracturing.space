---
title: "Campaign AI Session Bootstrap"
parent: "Platform surfaces"
nav_order: 12
status: draft
owner: engineering
last_reviewed: "2026-03-19"
---

# Campaign AI Session Bootstrap

This document describes the current MVP bootstrap for AI-controlled GM turns
and the next layer of improvements that should follow once the initial
campaign-scoped orchestration loop is stable.

## MVP now

- A campaign AI turn can be queued from `session.started` even when the session
  has no active scene yet.
- The AI orchestration runner rebuilds a fresh typed session brief on every
  turn from authoritative game resources instead of carrying a private
  transcript cache.
- The brief currently includes:
  - current session context
  - campaign metadata
  - campaign participants
  - campaign characters
  - campaign sessions
  - session scene list
  - interaction state
- The model receives a curated GM tool surface, including `scene_create`,
  `interaction_activate_scene` for existing-scene switches,
  `interaction_open_scene_player_phase`, and
  `interaction_record_scene_gm_interaction`.
- Prompt assembly is split into brief collection and rendering. AI startup
  selects the concrete render policy explicitly, and missing instruction files
  degrade only the affected instruction field instead of disabling the full
  prompt path.
- On a bootstrap turn with no active scene, the AI GM is expected to:
  - understand who is participating and which GM seat it controls
  - choose or create an opening scene
  - rely on `scene_create` to activate a new scene by default, or use
    `interaction_activate_scene` only when reusing an existing scene
- commit the opening GM interaction
- The MVP remains stateless across turns beyond authoritative game state.
  There is no persisted memory store, recap chain, or imported campaign file
  surface yet.

## Future improvements

- Add campaign-owned writable memories for recurring facts, NPC state, table
  preferences, and unresolved hooks.
- Add operator-managed imported source material such as `story.md`,
  `session.md`, encounter notes, or tone briefs through campaign-owned
  resources instead of direct filesystem access.
- Add recap and summarization pipelines so long-running sessions can preload a
  compact summary plus recent deltas instead of replaying large interaction
  payloads every turn.
- Add richer scene lifecycle tools, including scene update, scene transition,
  scene close, and retrieval of recent scene history.
- Add optional operator review gates before AI-published narration or scene
  mutations for hybrid and safety-sensitive modes.
- Add model-facing explanations of tool families and expected GM behavior so
  the opening bootstrap turn is more deliberate and less prompt-fragile.
