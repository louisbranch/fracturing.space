---
title: "Campaign AI Mechanics Quality"
parent: "Platform surfaces"
nav_order: 18
status: canonical
owner: engineering
last_reviewed: "2026-03-22"
---
# Campaign AI Mechanics Quality
Durable follow-on design guidance for mechanics-heavy AI GM turns. This note is
grounded in the March 22, 2026 Daggerheart live-mechanics acceptance run, but
the recommendations are framed as AI-service behavior patterns rather than
system-specific one-offs.
Use the live evidence in [Daggerheart Live Mechanics Matrix](../../reference/daggerheart-live-mechanics-matrix.md)
as the factual baseline. Use this note for the next design moves.
## What The Live Run Proved
- The current mechanics tool surface is broad enough for real AI GM play. The
  agent completed accepted live runs for sheet reads, board reads, action
  resolution, combat flows, Fear moves, adversary placement, and scene
  countdown lifecycle.
- Player-facing mechanical outcomes are usually legible in direct resolution
  lanes. The model can state roll outcome, damage, HP, Armor, Hope, and next
  action options in a way players can follow.
- Bounded reference usage is healthier than eager lookup. Board-control lanes
  were clean without consulting the reference corpus, while the explicit
  playbook lane could still succeed with one intentional search/read pair.
- The remaining instability is primarily runtime and recovery behavior. Session
  gates, precondition mismatches, and extra recovery calls now cause more live
  variance than missing mechanics tools.

## Observed Gaps

- Board-control turns still leak raw engine vocabulary too easily. Countdown
  IDs, adversary IDs, and internal state labels are useful for QA and memory,
  but they should not bleed into player-facing beats.
- The current live suite proves tool execution better than intent
  interpretation. Many lanes still name the mechanic family up front instead of
  forcing the model to infer it from the player's phrasing.
- The AI service lacks a dedicated critique surface. We can see when a run
  over-researched, over-recovered, or lacked context, but the production GM
  lane is not the place to ask the model to explain that back to us.
- Reference usage still needs stage awareness. The model should not research
  Fear, spotlight, or countdown procedures before those mechanics are actually
  relevant on the current turn.

## Recommended Next Workstreams
### 1. Mechanics Communication Contract
For mechanics-heavy turns, committed GM interactions should follow one stable
shape:
- `fiction`: what just happened in-world
- `consequence`: the authoritative mechanical result
- `guidance`: what changed in the decision space
- `prompt`: what the player does next

Player-facing `consequence` beats should explicitly name any resource or board
delta that matters to play now:

- HP
- Armor
- Hope
- Stress
- Fear
- spotlight owner
- visible countdown pressure

Keep internal IDs, enum names, and engine state labels in memory or OOC notes
only. The player should hear the game state change, not storage vocabulary.

### 2. Intent-To-Mechanics Eval Ladder

Add a dedicated evaluation track where the player's natural-language intent is
the main mechanic cue. These lanes should not name the required tool family in
the player prompt.

The first ladder should cover:

- explicit Hope spend for a named feature
- named feature use that may or may not be legal from the current sheet
- domain-card use
- equipment-driven action
- ambiguous intent that should trigger clarification instead of a bad tool call
- multi-actor intent that should become group action or tag-team resolution

Use deterministic integration coverage first, then live-agent lanes for the
same cases.

### 3. Diagnostic / Coach Mode

Add a separate, non-authoritative critique path that runs after a live or
replay capture. It should inspect the transcript, tool trace, and summary
artifacts and return:

- the chosen tool path and why it likely happened
- unnecessary reference lookups
- missing context that would have prevented a lookup or failed call
- better tool shapes, prompt policy, or always-on guidance
- player-facing clarity issues in the committed beats

This critique mode should not share the authoritative GM channel. It is a
product and tooling analysis surface, not part of the live scene turn.

### 4. Bounded Reference Strategy

Preserve the current two-layer model and make it explicit:

- short always-on operational primer for common mechanics turns
- on-demand playbooks and reference reads only when exact procedure is unclear

Reference budgets should become part of evaluation policy:

- no reference lookup in board-control lanes that already have clear state and
  an obvious authoritative tool path
- exactly one intentional search/read pair in explicit playbook lanes
- no pre-emptive Fear, spotlight, or countdown research before those mechanics
  are active in the current turn

### 5. Runtime Robustness

Treat session-gate and precondition failures as the primary live reliability
problem now that the mechanics surface is broader.

Recommended direction:

- improve precondition diagnosis before board-sensitive mechanics calls
- prefer stopping cleanly after a failed authoritative mechanic over trying an
  adjacent mechanic family
- keep recovery guidance corrective and narrow when a retry is actually valid

## Acceptance Markers

This work is materially improved when:

- mechanics-heavy beats state resource changes without leaking engine terms
- at least one natural-language intent lane exists for each major mechanic
  family
- critique reports consistently identify unnecessary lookups and missing context
- live summaries show lower unnecessary reference usage
- repeated live runs fail less often due to post-error tool thrash

## Relationship To Existing Docs

- [Campaign AI Orchestration](campaign-ai-orchestration.md) defines the runtime
  turn boundary and tool policy.
- [Campaign AI Agent System](campaign-ai-agent-system.md) defines instruction
  composition, channel discipline, and beat-oriented authoring.
- [Daggerheart Live Mechanics Matrix](../../reference/daggerheart-live-mechanics-matrix.md)
  is the evidence table, not the roadmap.
