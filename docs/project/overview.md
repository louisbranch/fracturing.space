---
title: "Overview"
parent: "Project"
nav_order: 1
---

# Project Overview

## Read this first

- Event model and write flow: [Event-driven system](event-driven-system.md)
- Replay and recovery: [Event replay](event-replay.md)
- System extension model: [Game systems](game-systems.md)

## New mechanic onboarding path

Use this sequence when adding or reviewing mechanics:

1. System author contract and write-path rules: [Game systems](game-systems.md)
2. Daggerheart mechanic-to-event mapping baseline: [Daggerheart Event Timeline Contract](daggerheart-event-timeline-contract.md)
3. Open scenario/mechanics gaps that need mapping before implementation: [Scenario Missing Mechanics](scenario-missing-mechanics.md)
4. Engine anti-pattern review context: [Event-Driven Engine Review](event-driven-engine-review.md)

## What it is

Fracturing.Space models a tabletop RPG campaign as a deterministic, event-sourced state machine. Every change is an ordered event, enabling full replay, inspection, and branching from any point in history.

## Why it exists

Long-running campaigns suffer from fragmented records and single-GM continuity. A deterministic, authoritative core enables reproducible outcomes, reliable history, and programmatic or AI-driven tooling on top.

## What it supports today

- Campaign, session, participant, and character management
- Deterministic, server-side resolution of mechanics
- Event journal with replayable state
- gRPC and MCP (JSON-RPC) interfaces
- Duality Dice as the initial rules system

## What it does not include (yet)

- A full end-user UI or VTT
- Integrated chat/voice/media
- Hosted multi-tenant deployment
- Turnkey AI narration

## Non-goals

- Emulating human improvisation
- Encoding subjective narrative quality into core rules
- Shipping proprietary game content

## Status

Early and experimental. Interfaces may change as additional systems and deployment models evolve.
