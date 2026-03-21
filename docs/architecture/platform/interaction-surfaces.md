---
title: "Interaction surfaces"
parent: "Platform surfaces"
nav_order: 4
status: canonical
owner: engineering
last_reviewed: "2026-03-21"
---

# Interaction Surfaces

## Purpose

Define the authoritative browser and API contract for active play now that
scene interaction is modeled as explicit session and scene state instead of
chat streams, personas, and legacy handoff workflows.

`web` owns the launcher route (`/app/campaigns/{id}/game`), but `play` owns the
browser-facing active-play surface after handoff. `game.v1.InteractionService`
remains the authority either way.

## Core model

- `session` owns table-level interaction state.
  - One `active_scene_id` is authoritative at a time.
  - A session-level OOC overlay can pause scene play for participant/GM
    discussion and rulings.
- `scene` owns in-character phase state.
  - The GM opens a player phase by committing a `gm_interaction` and selecting
    the acting character set.
  - Acting participants are derived from the owners of those characters.
  - Each acting participant owns one committed post and one revokable yield.
  - When all acting participants yield, authority returns to the GM.
- `gm_interaction` owns authored GM content.
  - One immutable interaction is committed for each GM-owned moment.
  - Interactions contain ordered beats such as fiction, prompt, resolution,
    consequence, and guidance.
- Player authority is participant-scoped.
  - UI may present portraits and character-specific bubbles.
  - Rules-affecting writes remain participant-owned, even when one participant
    controls multiple characters.

## Public service surface

`game.v1.InteractionService` is the authoritative read/write surface for active
play.

Reads:

- `GetInteractionState`
  - viewer participant
  - active session
  - active scene and roster
  - current scene GM interaction
  - scene GM interaction history
  - current player phase
  - session OOC overlay

Writes:

- `SetActiveScene`
- `CommitSceneGMInteraction`
- `StartScenePlayerPhase`
- `SubmitScenePlayerPost`
- `YieldScenePlayerPhase`
- `UnyieldScenePlayerPhase`
- `EndScenePlayerPhase`
- `ResolveScenePlayerPhaseReview`
- `PauseSessionForOOC`
- `PostSessionOOC`
- `MarkOOCReadyToResume`
- `ClearOOCReadyToResume`
- `ResumeFromOOC`

The browser surface consumes interaction state directly. Chat-style stream and
persona state is no longer the authoritative game-surface contract.

## Event and projection ownership

- Session events own:
  - active scene changes
  - OOC pause lifecycle
  - OOC posts
  - ready-to-resume tracking
- Scene events own:
  - GM interaction commits
  - player phase start/end
  - participant posts
  - yield and unyield transitions
- Projections expose:
  - session interaction state
  - scene phase control state
  - scene GM interaction history

Typing indicators, draft text, and voice transport are not part of the domain
event model. They remain transport concerns.

## Browser responsibilities

The active-play browser surface is served by `play`.

The browser game route should render state in terms of scenes, phases, and OOC
status rather than transcript routing controls.

Expected behavior:

- Show the active scene and which characters are currently acting.
- Show the latest GM interaction for the active scene.
- Show the latest committed post per acting participant.
- Allow inspection of prior GM interactions for the scene.
- Show yielded participants and OOC ready-to-resume state.
- Keep OOC display distinct from in-character scene display.

Deliberate non-behavior:

- No browser-owned authority inference from transcript text.
- No stream tabs or persona selectors on the main active-play path.
- No automatic AI/GM handoff triggered only by browser state.

## Chat sidecar status

Session chat may remain as an optional human websocket/transcript sidecar, but
it is not an active-play authority surface. Any future transcript work must
consume interaction state as input rather than trying to infer scene flow from
free-form chat.

Typing indicators, human chat fanout, and reconnect cursors are `play`
transport concerns, not `game` domain authority.
