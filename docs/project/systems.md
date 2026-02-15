---
title: "Systems checklist"
parent: "Project"
nav_order: 6
---

# Systems checklist

Use this as the high-level design checklist for a new rules system. For the
architecture and implementation steps, see [game systems](game-systems.md).

## System surface

- Ruleset identity: name, version, dice model
- Outcome taxonomy: result categories players/GM must reason about
- Resource model: player/GM currencies, caps, defaults
- State scope: profile (static) vs state (dynamic) vs snapshot (campaign-level)

## Deterministic resolution

- Seeded randomness with explicit inputs and outputs
- Pure outcome evaluation functions
- Explainability surface for debugging/audit

## State and projections

- Profile schema: traits, thresholds, static modifiers
- State schema: mutable resources and combat state
- Snapshots for campaign-level state
- Projections derived only from events

## Core mechanics

- Attack resolution and difficulty targets
- Damage rules, thresholds, severity mapping
- Mitigation: resistance, immunity, armor
- Critical success rules

## Recovery and downtime

- Rest cadence and interruption rules
- Downtime move set
- Refresh model for per-rest features

## Abilities and loadouts

- Ability types and common fields
- Loadout rules (active vs vaulted)
- Swap constraints and costs

## Validation and guardrails

- Caps and ranges enforced at domain and projection layers
- Event safety: reject invalid payloads
- Versioning and compatibility

## Surfaces and sequencing

- Domain mechanics first
- Transport APIs after mechanics stabilize
- MCP and other interfaces last

## Where to go next

- [Game system architecture and implementation](game-systems.md)
- [Event sourcing and replay](event-replay.md)
- [Domain language](domain-language.md)
