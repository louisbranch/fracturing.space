---
title: "Chat Service"
parent: "Project"
nav_order: 12
---

# Chat Service Specification

## Purpose

Define the chat subsystem architecture before implementation so the team can
ship a minimal real-time session chat first, then add mode-aware and
audience-aware behavior safely.

This document focuses on:

- Service interfaces with existing services.
- Domain invariants and constraints.
- A phased rollout plan from simple to advanced scenarios.

## Scope and Non-goals

In scope:

- Campaign session chat over WebSocket.
- One logical room per active campaign session.
- Mode-aware actor identity (participant, character, GM, system).
- Selective fanout where not all messages are broadcast to all participants.
- Slash commands and mode-aware mentions (`@participant` / `@character`).

Out of scope (initially):

- Voice/video/media transport.
- Cross-campaign channels.
- End-to-end encrypted chat.
- Fully generic chat platform features (threads, reactions, rich embeds).

## Service Boundary

The chat subsystem is a service boundary under `internal/services/chat/` and
follows project architecture conventions:

- Transport: WebSocket endpoint(s) and optional read APIs.
- Application: orchestration, authorization checks, command routing.
- Domain: room, actor mode, audience policy, message/command semantics.
- Storage: append-only chat event journal plus derived projections.

Service ownership is lane-based:

- In-game lane: chat is transport/orchestration only; game service is the
  event authority and persists canonical in-game chat facts.
- Off-game lane: chat service owns persistence and delivery for messages that
  are explicitly out of game scope.

Chat must not write game projections/tables directly in any lane.

## Message Lanes

To prevent boundary drift, chat traffic is explicitly separated:

- In-game messages:
  - part of gameplay context and campaign timeline,
  - governed by game session/membership rules,
  - persisted by game service event authority.
- Off-game messages:
  - social or coordination chatter outside campaign canon,
  - not written to game event journal,
  - owned fully by chat service lifecycle and retention policy.

Phase 1 currently targets only the in-game lane through chat transport.

## Domain Model

Core entities:

- Room: session-scoped channel identified by `campaign_id` + `session_id`.
- Message: immutable chat fact appended to chat event journal.
- Actor context: sender identity as one of `participant`, `character`, `gm`, or
  `system`.
- Audience: visibility scope for delivery and history reads.
- Mention: resolved reference to participant/character identity at send time.

Recommended message shape:

- `message_id` (server-generated UUID/ULID)
- `room_id` (`campaign_id/session_id`)
- `sequence_id` (monotonic per room)
- `sent_at`
- `kind` (`text`, `command`, `notification`, `system`)
- `actor_context` (type + id + display label snapshot)
- `mode` (authoring mode used for this message)
- `body` (raw text or structured payload)
- `mentions[]` (resolved ids + display snapshot)
- `audience` (scope + explicit targets when applicable)
- `client_message_id` (idempotency key from client)

## Interfaces and Dependencies

## Chat <-> Web Client

WebSocket protocol (initial envelope):

- Client to server:
  - `chat.join`
  - `chat.send`
  - `chat.history.before`
  - `chat.autocomplete`
  - `chat.command.preview` (optional)
- Server to client:
  - `chat.joined`
  - `chat.message`
  - `chat.ack`
  - `chat.error`
  - `chat.presence` (later phase)

Client-supplied mode/audience fields are hints only. The server computes allowed
actor context and effective audience.

## Chat <-> Game Service

Chat depends on game service as the authority for campaign/session/seat state.
Required capabilities:

- Resolve active session for a campaign.
- Resolve participant membership and role for a user.
- Resolve character control mapping for a participant.
- Resolve GM privilege and campaign access policy.
- Accept in-game chat intents/events from chat transport (write authority remains
  in game service).

If command handlers trigger game actions, chat calls game APIs; only game emits
game events.

## Chat <-> Auth Service

Chat validates connection credentials and obtains user identity claims. Auth is
the identity authority; chat only consumes validated identity context.

## Chat <-> MCP/Admin

- MCP parity is a later phase (read/send operations aligned with WebSocket
  semantics).
- Admin may consume chat projections for moderation and diagnostics.

## Invariants

1. Event authority:
   - in-game lane writes are owned by game service events,
   - off-game lane writes are owned by chat service events.
2. Session gating: room posting is allowed only while campaign session is
   active.
3. Server-side authorization: server computes effective actor mode and audience;
   clients cannot escalate visibility.
4. Deterministic visibility: each message stores its effective audience so
   history replay and reconnect fanout are stable.
5. Ordered delivery cursor: every room message has monotonic `sequence_id` for
   reconnect (`last_sequence_id`) and pagination.
6. Command isolation: chat commands must not mutate game storage directly; game
   mutations must go through game service APIs.
7. Idempotent sends: duplicate `client_message_id` in the same room returns the
   original ack/message identity.
8. Lane isolation: off-game messages never reach game service event journal.

## Constraints and Operational Requirements

- One logical channel per active session, with selective fanout inside it.
- Expected delivery model: at-least-once over WebSocket with idempotent client
  handling.
- Reconnect behavior: resume stream from last acknowledged `sequence_id`.
- Rate limiting and payload limits are mandatory to prevent abuse.
- Message history access must enforce per-message audience visibility.
- System-generated notifications share the same room timeline but can target
  restricted audiences.

## Rollout Plan

## Phase 1: Minimal Session Chat

Goal: ship useful chat quickly with low risk.

Scope:

- WebSocket join/send/receive.
- Active-session room binding.
- Participant mode only.
- Broadcast to all session participants.
- Plain text messages and recent history fetch.

Acceptance checks:

- Participants in an active session can exchange messages in real time.
- No posting when session is inactive.
- Reconnect with `last_sequence_id` catches up missing messages.

## Phase 2: Mode and Audience Policy

Goal: support selective visibility and richer actor identity.

Scope:

- Add actor modes: participant, character, gm.
- Add audience policy: `all`, `gm_only`, explicit targets.
- Enforce mode permissions from game service role/control data.
- Add timeline indicators for message visibility/mode.

Acceptance checks:

- Unauthorized mode/audience hints are rejected or downgraded by server.
- GM-only messages are not delivered to non-GM clients.
- History queries return only messages visible to requester.

## Phase 3: Slash Commands and Mentions

Goal: support command and targeting workflows.

Scope:

- Slash command parser and registry with permission matrix.
- Mode-aware autocomplete for `@` entities (participants or characters).
- Mention resolution at send time; store resolved IDs in event payload.

Acceptance checks:

- Command permissions enforce role/mode constraints.
- Mention suggestions and resolution match active mode and visibility rules.
- Unknown commands and invalid mention targets return typed errors.

## Phase 4: System Notifications and Game Integration

Goal: integrate chat with gameplay signals without breaking boundaries.

Scope:

- System-generated notifications (session status, event summaries).
- Optional command adapters that invoke game APIs.
- Consistent provenance in timeline (`system`, `gm`, `participant`,
  `character`).

Acceptance checks:

- Notifications appear with correct provenance and audience.
- Game-affecting commands emit game changes through game APIs only.

## Phase 5: Hardening and Parity

Goal: make chat operationally robust and cross-surface consistent.

Scope:

- Moderation hooks and admin tooling.
- Presence/typing improvements.
- MCP parity for send/read flows.
- Retention and archival policy enforcement.

Acceptance checks:

- Operational limits hold under load and reconnect churn.
- Admin/moderation paths can trace message provenance and audience decisions.

## Open Questions and Decision Gates

1. Journal strategy:
   - Keep chat in a dedicated chat event journal (recommended) or merge into the
     game event journal.
2. Retention policy:
   - Full retention, per-campaign policy, or TTL-based pruning for chat events.
3. Audience defaults:
   - What should default audience be per mode (especially GM mode)?
4. Character mode semantics:
   - Can a participant speak as multiple characters in one active session?
5. Command ownership:
   - Which commands remain chat-local vs delegated to game service.
6. MCP parity timing:
   - Ship WebSocket first, then MCP parity, or dual-surface from day one.

## Implementation Notes

- Start with a custom chat UI composer/timeline rather than waiting for a full
  third-party interface library to match mode/audience semantics.
- Use project domain language consistently: event journal authority, projections
  as derived state, and service-owned boundaries.
