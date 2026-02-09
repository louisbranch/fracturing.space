# Fracturing.Space ![Coverage](../../raw/badges/coverage.svg)

Fracturing.Space is an open-source system for modeling tabletop RPG campaigns as deterministic, event-sourced state machines with persistent and forkable history.

Everything that occurs in a campaign—mechanical actions, narrative developments, and participant changes—is represented as a structured event. Together, these events form a complete, inspectable history of play.

## What this project is addressing

Tabletop RPG campaigns are traditionally ephemeral systems. State is distributed across human memory, notes, and ad hoc judgment, and campaign history is linear and irreversible.

Fracturing.Space treats a campaign as an authoritative state machine with reproducible transitions. Because outcomes are deterministic and all changes are recorded as ordered events, campaign history can be replayed, inspected, and branched. Any point in time can serve as the root for an alternate future.

This makes it possible to treat play not as a single unfolding narrative, but as exploration of a structured space of possible outcomes.

## Motivation

Long-running tabletop RPG campaigns often collapse due to scheduling issues, fragmented record‑keeping, and reliance on a single GM to maintain narrative and mechanical continuity. At the same time, modern systems—including programmatic and AI-driven agents—are now capable of reasoning over structured state and long-lived context.

Fracturing.Space provides a neutral, open foundation for these experiments by managing rules, state, and history in a way that is authoritative, reproducible, and inspectable.

## Why now / use cases

- Programmatic or AI-driven agents operating over authoritative mechanics and reproducible outcomes
- Persistent, long-lived campaigns that do not rely on a single person or tool
- Forkable timelines for playtesting, alternate outcomes, and experimentation
- Solo or collaborative play that can revisit and revise the past

## How it works (brief)

Fracturing.Space models a campaign as a timeline of events.

- Creating or joining a campaign is an event
- Characters, participants, and sessions are events
- Mechanical outcomes and narrative developments are events

Events are appended to an immutable history that represents everything that has occurred so far.

Because event resolution is deterministic, this history can be replayed at any time. Any event boundary can serve as the starting point for a new timeline, allowing alternate outcomes or interpretations to be explored without modifying the original history.

The system organizes state into three layers:

- Campaign: configuration and setup
- Snapshot: materialized projections for replay and performance
- Session: moment-to-moment gameplay events

## Systems

The initial rules system implements Duality Dice, a deterministic resolution mechanic built around paired dice and outcome categories. System details live in the docs.

Duality Dice exists primarily to validate the event model and deterministic resolution pipeline. The architecture is designed to support additional mechanics engines.

### What it does today

The current implementation focuses narrowly on establishing the core model correctly and reliably.

At a high level, it supports:

- Creation and management of campaigns, sessions, participants, and characters
- An event-driven campaign model where all state changes are recorded as ordered events
- Deterministic, server-side resolution of game mechanics
- An initial integrated rules system based on Duality Dice mechanics
- Programmatic access via gRPC and MCP (JSON-RPC) interfaces, suitable for custom clients and automated agents

Higher-level features are intentionally minimal at this stage.

### What it does not provide (yet)

The following are intentionally out of scope for the current phase:

- A built-in user interface or virtual tabletop
- Integrated chat, voice, or media playback
- Turnkey AI narration or content generation
- A hosted, multi-tenant deployment

These are considered medium- to long-term extensions and are expected to evolve as the core platform matures. The current repository focuses on the underlying engine rather than end-user experience.

### Non-goals

- Emulating human improvisation or narrative judgment
- Encoding subjective narrative quality into the core system
- Providing a complete end-user game experience at this stage

Fracturing.Space prioritizes correctness, reproducibility, and inspectable state. Higher-level narrative tooling is expected to live above the core.

### Project status

Fracturing.Space is in an early and experimental stage.

The core architecture and APIs are usable, but the project is not yet a complete, end-to-end game platform. Interfaces may change as additional systems, integrations, and deployment models are explored.

### Documentation (canonical)

The docs are the canonical source of truth for architecture, APIs, and usage. Start with [docs](/docs/index.md) or browse the published site at [GitHub Pages](https://louisbranch.github.io/fracturing.space/).

### Services and boundaries (brief)

Fracturing.Space is organized into service boundaries under `internal/services/`:

- Game service: authoritative rules + campaign state over gRPC; owns the SQLite database
- MCP service: JSON-RPC adapter that forwards to the game service
- Admin service: HTTP dashboard that queries the game service
- Auth service: domain logic only; transport/API surface is planned

Shared utilities (for example, RNG seed generation) live in domain/platform code and are not separate services. See `docs/project/architecture.md` for details.

### Getting involved

This project is open source and contributions are welcome.

Ways to get involved include:

- Improving or extending the core platform
- Integrating additional rules systems
- Designing example campaigns or event models
- Improving documentation and developer onboarding
- Exploring client, UI, or system-driven integrations on top of the platform

See [CONTRIBUTING.md](CONTRIBUTING.md).

### Systems, content, and licensing

This repository focuses on infrastructure and mechanics, not on distributing copyrighted game content.

- Rules systems are implemented as mechanics only, without copyrighted text
- Names and mechanics are referenced for compatibility and interoperability purposes
- Contributors are responsible for ensuring that any submitted content complies with applicable licenses

The long-term goal is to support user-provided and community-maintained systems and content packs, with clear attribution and licensing, without bundling proprietary material directly into the core project.

