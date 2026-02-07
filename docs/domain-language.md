# Domain Language

This document defines the canonical terms for the Fracturing.Space. The language
here drives how we name packages, APIs, and documentation.

## Core Concepts

### Campaign (Repository)

A campaign is the complete history and derived views for a game timeline.
It is analogous to a Git repository:

- The campaign event journal is the commit history.
- Projections are derived state (like working trees or views).
- A fork creates a new campaign with its own timeline.

### Event Journal (Commit History)

The event journal is the immutable, append-only log of everything that happens
in a campaign. Every change to the game is an event. The event journal is the
source of truth.

### Event

An event is an immutable fact that happened in the game. It can describe:

- State changes (campaign, participant, character, session, system state).
- Story changes (GM narration, dialogue, plot beats, scene changes).

Events are organized with a mostly flat namespace. Buckets exist for grouping
but are expected to evolve over time. The current catch-all bucket for
theater-of-the-mind changes is `story` (provisional and subject to rename).

### Story vs Story Events

Story content is reusable, campaign-agnostic narrative material (modules,
scenes, NPC lore, locations). It lives in the narrative/content packages and
does not create events by itself.

Story events are campaign-specific narrative facts (notes added, canonized
details, scene progression). They are written to the event journal when
narrative changes become part of a campaign's history.

Example `story.*` taxonomy (non-exhaustive):

- `story.note_added` (GM/system notes)
- `story.canonized` (facts accepted into canon)
- `story.scene_started` / `story.scene_ended`
- `story.reveal_added` (new info revealed)

### Projection

Projections are derived, queryable views built from events. They are not the
source of truth. Examples include campaign metadata, participant lists,
character state, and session summaries.

### Snapshot

Snapshots are materialized projections captured at a specific event sequence.
They exist for performance and fast replay, not as an authoritative record.

### Fork

A fork creates a new campaign by branching from a specific event sequence in
another campaign. The new campaign has its own event journal and projections.

### Telemetry

Telemetry logs capture non-mutating operations (queries, list operations,
validation failures, and system metrics). Telemetry is stored separately from
the game event journal, even if it shares the same database.

## Naming Principles

- Prefer domain terms over implementation terms (event journal vs event table).
- Keep event type names flat and readable; introduce buckets sparingly.
- Avoid naming that locks in a specific storage or transport layer.

## Non-goals

- Backwards compatibility with legacy event APIs is not a requirement.
- Event namespaces can change as the model evolves.
