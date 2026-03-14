---
title: "Communication surfaces next steps"
parent: "Platform surfaces"
nav_order: 4
status: proposed
owner: engineering
last_reviewed: "2026-03-09"
---

# Communication Surfaces Next Steps

## Purpose

Describe the next intended behavior for the `chat` service and the browser game
surface now that game-owned communication context, gate workflows, and derived
workflow policy state exist.

This document is intentionally forward-looking. It describes target behavior
that should guide the next implementation slice; it does not claim that all of
the behaviors below exist today.

## Current implemented baseline

- `game` owns communication context, stream visibility, persona eligibility,
  gate state, spotlight state, gate response progress, and derived workflow
  policy summaries.
- `chat` relays transcript traffic and game-backed room state over websocket.
- `web` renders a game surface that consumes the game-backed communication
  context instead of inferring routing or authority from transcript text.

## Chat next responsibilities

`chat` should continue to stay transport-focused. The next expected work is:

- Preserve game-owned workflow state as pass-through room state.
  - `chat` should not reinterpret `resolution_state`, `suggested_decision`, or
    other workflow progress fields.
- Add durable transcript behavior.
  - Store transcript history outside process memory.
  - Support per-stream resume/history cursors.
  - Keep transcript bodies non-authoritative unless game explicitly promotes
    them elsewhere.
- Support workflow-aware control payload entry, not workflow authority.
  - `gate.open` should carry structured metadata for workflows such as vote
    options and eligible participants.
  - `chat` should validate only transport shape and leave workflow semantics to
    `game`.
- Keep AI pacing separate from participant workflow policy.
  - AI handoff remains its own control flow.
  - `chat` must not auto-submit AI turns merely because a `vote` or
    `ready_check` reaches a derived resolution state.

## Web game surface next responsibilities

The browser game surface should become a workflow-aware consumer of
communication state.

Expected next behaviors:

- Render workflow-specific gate summaries.
  - `ready_check`: show ready count, wait count, pending participants, and
    whether the workflow is blocked or ready to resolve.
  - `vote`: show options, tallies, pending participants, current leader, and tie
    state.
- Use derived policy summaries to drive controls.
  - `resolution_state=ready_to_resolve` should enable a clear resolve action.
  - `resolution_state=blocked` should show why progress is blocked.
  - `resolution_state=manual_review` should signal that the GM must decide.
- Separate message identity from workflow authority in the UI.
  - Persona switching changes how messages are presented.
  - Gate responses remain participant-scoped actions even if the participant is
    currently chatting as a character persona.
- Replace generic gate forms with workflow-specific forms.
  - `ready_check` should expose ready/wait participation, not a free-form
    decision box.
  - `vote` should expose explicit choice controls and option summaries.
  - Opening a `vote` gate should collect the options list and, when needed, the
    eligible participant set.
- Keep transcript rendering contextual.
  - Distinguish stream type, speaker persona, workflow/system state, and
    scene/session context without asking users to infer those from message text.

## Deliberate non-behavior for the next slice

The following should remain out of scope until intentionally designed:

- Automatic gate resolution writes triggered only by browser logic.
- Automatic AI GM turns triggered only by workflow completion.
- Persona-scoped vote counting.
- Transcript parsing that tries to infer gameplay decisions from free-form chat.

## Protocol evolution next step

The next communication architecture step is a contract cleanup, not another
authority move.

`game` already owns communication context, session-gate workflow state, derived
progress, and GM handoff behavior. The remaining gap is that the public game
contract still exposes gate metadata, progress, resolution, and response
payloads through `google.protobuf.Struct`. That keeps `chat` and `web` on
dynamic-map adapters even though the game domain and projection path are now
typed internally.

Target direction for the next slice:

- Replace generic gate payloads with typed workflow envelopes.
- Define explicit workflow messages for `ready_check`, `vote`, and
  `gm_handoff`.
- Use typed fields for open metadata, participant responses, derived progress,
  and resolution payloads.
- Keep a generic fallback only if unknown/custom gate types remain a real
  product requirement.

Consumer expectations after that cut:

- `chat` stays transport-focused and forwards typed workflow state without
  reinterpreting game policy.
- `web` renders workflow-specific controls and summaries from typed fields,
  not dynamic `Struct` decoding.
- Persona selection remains message-presentation state.
- Workflow responses remain participant-scoped unless a future game contract
  explicitly changes that rule.

Deliberate non-behavior for that slice:

- No transcript parsing to infer workflow decisions.
- No browser-only automatic gate resolution writes.
- No chat-owned workflow semantics.
- No persona-scoped vote counting.

## Sequencing guidance

Recommended order for the next implementation phase:

1. Define typed workflow messages in the game/session communication proto.
2. Cut game transport mappers to the typed workflow envelope.
3. Cut chat and web adapters to typed workflow mapping.
4. Delete generic-map workflow handling on the main path.
5. Add durable transcript storage and resume/history behavior in `chat`.
6. Decide whether any derived workflow states should produce automatic
   game-owned resolution writes, or remain advisory-only for GM control.
