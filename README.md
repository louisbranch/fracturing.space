# Fracturing.Space ![Coverage](../../raw/badges/coverage.svg)

Fracturing.Space is an open-source, server-authoritative platform for tabletop RPG campaigns with persistent state, deterministic mechanics, and forkable timelines.

It treats everything in a campaign, mechanical actions, narrative developments, and participant changes, as structured events that form a complete history of play.

## Motivation

Long-running tabletop RPG campaigns often fall apart due to scheduling, fragmented notes, and a single GM holding the narrative thread. At the same time, modern tools (including AI systems) can now reason over structured state and long-lived context.

Fracturing.Space provides a neutral, open foundation for these experiments: a platform that manages rules, state, and history in a way that both humans and AI systems can reliably build upon.

## Why now / use cases

- AI-assisted GMing with authoritative mechanics and reproducible outcomes
- Persistent, long-lived campaigns that do not rely on a single person or tool
- Forkable timelines for playtesting, alternate outcomes, and experimentation

## How it works (brief)

Fracturing.Space models a campaign as a timeline of events.

- Creating or joining a campaign is an event
- Characters, participants, and sessions are events
- Mechanical outcomes and narrative developments are events

These events form an append-only history that represents everything that has happened so far.

That history can be forked. A new group can start from any point in an existing timeline and explore a different path forward, creating alternate outcomes, parallel stories, or entirely new interpretations of the same starting world.

The system organizes state into three layers:

- Campaign: configuration and setup
- Snapshot: materialized projections for replay and performance
- Session: moment-to-moment gameplay events

## Systems

The initial rules system implements Duality Dice, a deterministic resolution mechanic built around paired dice and outcome categories. System details live in the docs.

### What it does today

The project currently provides a backend platform for managing tabletop RPG campaigns with persistent state and deterministic mechanics resolution.

At a high level, it supports:

- Creation and management of campaigns, sessions, participants, and characters
- An event-driven campaign model where all state changes are recorded as ordered events
- Deterministic, server-side resolution of game mechanics
- An initial integrated rules system based on Duality Dice mechanics
- Programmatic access via gRPC and MCP (JSON-RPC) interfaces, suitable for both custom clients and AI agents

The current focus is on establishing a stable core: campaign history, rules execution, and authoritative state management. Higher-level features are intentionally minimal at this stage.

### What it does not provide (yet)

At present, it does not include:

- A built-in user interface or virtual tabletop
- Integrated chat, voice, or media playback
- Turnkey AI narration or content generation
- A hosted, multi-tenant deployment

These are considered medium- to long-term goals and are expected to evolve as the core platform matures. The current repository focuses on the underlying engine rather than end-user experience.

### Project status

Fracturing.Space is in an early and experimental stage.

The core architecture and APIs are usable, but the project is not yet a complete, end-to-end game platform. Interfaces may change as additional systems, integrations, and deployment models are explored.

### Documentation (canonical)

The docs are the canonical source of truth for architecture, APIs, and usage. Start with [docs](/docs/index.md) or browse the published site at [GitHub Pages](https://louisbranch.github.io/fracturing.space/).

### Getting involved

This project is open source and contributions are welcome.

Ways to get involved include:

- Improving or extending the core platform
- Integrating additional rules systems
- Designing example campaigns or event models
- Improving documentation and developer onboarding
- Exploring client, UI, or AI-driven integrations on top of the platform

See [CONTRIBUTING.md](CONTRIBUTING.md).

### Systems, content, and licensing

This repository focuses on infrastructure and mechanics, not on distributing copyrighted game content.

- Rules systems are implemented as mechanics only, without copyrighted text
- Names and mechanics are referenced for compatibility and interoperability purposes
- Contributors are responsible for ensuring that any submitted content complies with applicable licenses

The long-term goal is to support user-provided and community-maintained systems and content packs, with clear attribution and licensing, without bundling proprietary material directly into the core project.
