---
title: "Overview"
parent: "Architecture"
nav_order: 1
status: canonical
owner: engineering
last_reviewed: "2026-02-26"
---

# Project Overview

Canonical orientation page for the current system model.

## Read this first

1. [Architecture](architecture.md)
2. [Domain language](domain-language.md)
3. [Event-driven system](event-driven-system.md)

## If you are changing X, read Y

- Replay/snapshots/recovery behavior: [Event replay](event-replay.md)
- Game-system extension mechanics: [Game systems](game-systems.md)
- Authorization semantics: [Campaign Authorization Model](campaign-authorization-model.md)
- Identity and auth boundaries: [Identity and Recovery](identity.md), [OAuth System](oauth.md)

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
