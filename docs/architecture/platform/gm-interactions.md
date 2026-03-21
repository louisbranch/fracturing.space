---
title: "GM interactions"
parent: "Platform surfaces"
nav_order: 17
status: canonical
owner: engineering
last_reviewed: "2026-03-21"
---

# GM Interactions

## Purpose
Define the canonical `gm_interaction` model for active play.

This model does not replace scene ownership, player phases, acting sets, yields,
GM review, or OOC control state. It replaces the authored split between
scene-level GM output and phase-level prompt text with one structured
GM-authored interaction.

## Core principle
The GM authors one discrete `gm_interaction` for each GM-owned moment of play.

One interaction may include fictional updates, prompts, resolution,
consequences, and next-action guidance. The GM is not authoring separate
narration and prompt artifacts.

## Boundaries
`gm_interaction` is authored content. Scene/session/phase state remains
explicit control state and answers:

- who may act now
- whether a player phase is open
- whether review is pending
- whether OOC is blocking progress

`gm_interaction` answers:

- what changed in the fiction
- what matters right now
- what the GM is asking for
- what happens next

## Interaction structure
A `gm_interaction` is discrete, ordered, player-readable, machine-readable, and
immutable once committed.

Each interaction contains one or more ordered beats.

### Beat types
- `fiction`: establish or update shared fictional context
- `prompt`: request player intention, choice, order, or commitment
- `resolution`: handle uncertain or consequential action through mechanics
- `consequence`: return resolution results to the fiction
- `guidance`: clarify what is actionable next without re-authoring scene state

## Stored model
`gm_interaction` is stored as structured data, not only rendered text.

Canonical fields:

- `interaction_id`
- `scene_id`
- `phase_id`
- `created_at`
- `participant_id`
- `title`
- `character_ids`
- optional `illustration`
- ordered `beats`

Minimum beat fields:

- `beat_id`
- `type`
- `text`

Suggested shape:

```json
{
  "interaction_id": "gmint_123",
  "scene_id": "scene_1",
  "phase_id": "phase_7",
  "created_at": "2026-03-20T15:04:05Z",
  "participant_id": "gm_ai",
  "title": "Investigating the Docks",
  "character_ids": ["mara", "henric"],
  "illustration": {
    "image_url": "https://res.cloudinary.com/.../lantern_in_the_dark.png",
    "alt": "A storm lantern burning in darkness.",
    "caption": "Optional image caption."
  },
  "beats": [
    {"beat_id": "beat_1", "type": "fiction", "text": "The ship lurches hard and the lantern tears loose from its hook."},
    {"beat_id": "beat_2", "type": "prompt", "text": "Mara and Henric are both close enough to react. Who goes for it first?"}
  ]
}
```

## Phase semantics
Phase state remains authoritative for control.

- `GM`: no player phase is open; the GM owns the next move
- `PLAYERS`: one player phase is open and players respond via slots
- `GM_REVIEW`: the same player phase is awaiting GM review resolution

Review outcomes always produce a new immutable `gm_interaction`:

- `request_revisions`: same `phase_id`, same player phase remains open
- `return_to_gm`: current player phase closes and authority returns to GM
- `advance_to_players`: current phase closes, a new phase opens, and the new
  interaction is linked to the new `phase_id`

Revision loops reuse the same `phase_id`. Opening a truly new beat creates a
new phase.

## Scene and session lifecycle
Administrative lifecycle control is separate from authored GM content:

- `scene.end`
- scene switch
- `session.end`
- OOC pause/resume

Normal narrative closure is:

1. GM resolves review with a final `gm_interaction`
2. phase returns to `GM`
3. GM closes the scene or moves elsewhere

Administrative interruption may bypass narrative completion. A scene or session
may force-close with an open phase and without synthesizing a final
`gm_interaction`.

## UI implications
The browser should consume:

- active scene overview
- explicit phase control state
- `current_interaction`
- `interaction_history`
- participant-owned slots

UI slices like “current prompt” or “latest outcome” are derived from the latest
interaction beat structure. They are not separately authored fields.
