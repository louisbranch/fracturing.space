---
title: "Chat Phase 1 API"
parent: "Project"
nav_order: 13
---

# Chat Phase 1 WebSocket API Contract

## Purpose

Define the implementation-ready WebSocket contract for Phase 1 of chat:

- one room per active campaign session,
- participant mode only,
- broadcast delivery to all active session participants.

This contract is intentionally narrow and leaves mode-aware audience policies
and slash commands to later phases.

Ownership note:

- This Phase 1 transport targets the in-game lane.
- Event authority for persisted in-game chat facts is expected to live in game
  service; chat service acts as the websocket gateway.

## Scope and Non-goals

In scope:

- WebSocket connect/join/send/history/reconnect.
- Ordered room timeline via monotonic `sequence_id`.
- Idempotent sends via `client_message_id`.
- Typed error envelopes and canonical error codes.

Out of scope:

- Character/GM/system authoring modes.
- Selective audience fanout policies.
- Slash commands and mention autocomplete.
- Presence, typing indicators, reactions, attachments.
- Off-game chat lane persistence and policy.

## Endpoint and Auth

Endpoint (proposed):

- `GET /ws`

Authentication:

- Connection must carry authenticated user context (for example via session
  cookie or bearer token already used by web transport).
- Unauthenticated connections are closed with `UNAUTHENTICATED`.

Connection model:

- A socket may join at most one room in Phase 1.
- Joining a different room requires a new connection.
- This path is expected to be served from a dedicated chat origin (for example
  `chat.example.com/ws`).

## Base Frame Envelope

All frames are UTF-8 JSON objects using this envelope:

```json
{
  "type": "chat.send",
  "request_id": "req_01HXYZ...",
  "payload": {}
}
```

Field contract:

- `type` (string, required): message type discriminator.
- `request_id` (string, optional on server push): client correlation id echoed in
  `chat.ack` and `chat.error`.
- `payload` (object, required): type-specific payload.

## Shared Types

Room identifier:

```json
{
  "campaign_id": "camp_123",
  "session_id": "sess_456"
}
```

Chat message (Phase 1):

```json
{
  "message_id": "msg_01JABC...",
  "campaign_id": "camp_123",
  "session_id": "sess_456",
  "sequence_id": 42,
  "sent_at": "2026-02-15T18:42:10Z",
  "kind": "text",
  "actor": {
    "participant_id": "part_789",
    "display_name": "Ari"
  },
  "body": "hello table",
  "client_message_id": "cli_01JABC..."
}
```

Notes:

- `sequence_id` is strictly monotonic within a room.
- `kind` is fixed to `text` in Phase 1.
- Audience is implicitly all room participants in Phase 1.

## Client -> Server Types

## `chat.join`

Join active session room for a campaign and optionally resume from a cursor.

Payload:

```json
{
  "campaign_id": "camp_123",
  "last_sequence_id": 40
}
```

Rules:

- `campaign_id` required.
- `last_sequence_id` optional; when present server replays messages where
  `sequence_id > last_sequence_id`.
- User must be a participant of the campaign and session must be active.

## `chat.send`

Send a chat message to the joined room.

Payload:

```json
{
  "client_message_id": "cli_01JABC...",
  "body": "hello table"
}
```

Rules:

- `client_message_id` required, max 128 chars.
- `body` required, non-empty after trim.
- Payload size limits are enforced server-side.
- Duplicate `client_message_id` for same room returns idempotent ack.

## `chat.history.before`

Fetch older messages before a known cursor.

Payload:

```json
{
  "before_sequence_id": 41,
  "limit": 50
}
```

Rules:

- `before_sequence_id` required and must be >= 1.
- `limit` optional; default 50, max 200.
- Response order is ascending `sequence_id`.

## Server -> Client Types

## `chat.joined`

Confirms room join and current server cursor.

Payload:

```json
{
  "campaign_id": "camp_123",
  "session_id": "sess_456",
  "latest_sequence_id": 42,
  "server_time": "2026-02-15T18:42:10Z"
}
```

## `chat.message`

Pushes timeline message (live or catch-up/history).

Payload:

```json
{
  "message": {
    "message_id": "msg_01JABC...",
    "campaign_id": "camp_123",
    "session_id": "sess_456",
    "sequence_id": 42,
    "sent_at": "2026-02-15T18:42:10Z",
    "kind": "text",
    "actor": {
      "participant_id": "part_789",
      "display_name": "Ari"
    },
    "body": "hello table",
    "client_message_id": "cli_01JABC..."
  }
}
```

## `chat.ack`

Acknowledges request completion for request/response style actions.

Payload:

```json
{
  "request_id": "req_01HXYZ...",
  "result": {
    "status": "ok",
    "message_id": "msg_01JABC...",
    "sequence_id": 42
  }
}
```

Rules:

- For idempotent duplicate send, `message_id` and `sequence_id` of original
  message are returned.
- For non-send actions, `result` payload shape is action-specific.

## `chat.error`

Returns typed failure for a client request.

Payload:

```json
{
  "request_id": "req_01HXYZ...",
  "error": {
    "code": "SESSION_INACTIVE",
    "message": "campaign session is not active",
    "retryable": false,
    "details": {}
  }
}
```

## Error Codes

Canonical Phase 1 codes:

- `UNAUTHENTICATED`: missing/invalid auth context.
- `FORBIDDEN`: authenticated but not allowed for campaign/room.
- `ROOM_NOT_FOUND`: campaign room cannot be resolved.
- `SESSION_INACTIVE`: campaign has no active session room.
- `INVALID_ARGUMENT`: malformed payload or failed validation.
- `PAYLOAD_TOO_LARGE`: message exceeds max payload/body size.
- `RATE_LIMITED`: send/join/history frequency exceeds limits.
- `CURSOR_OUT_OF_RANGE`: invalid cursor for catch-up/history request.
- `DUPLICATE_CLIENT_MESSAGE_ID`: optional explicit duplicate signal when not
  auto-acked as idempotent success.
- `INTERNAL`: unexpected server failure.

Error handling rules:

- `chat.error` must include `request_id` when request included it.
- `message` is user-safe and non-sensitive.
- `details` is optional structured metadata for client UX.

## Ordering, Replay, and Idempotency

Ordering:

- Server persists message, allocates `sequence_id`, then broadcasts.
- Clients render by `sequence_id`; if out-of-order arrival occurs, client sorts
  and de-duplicates by `message_id`.

Reconnect:

- Client reconnects and sends `chat.join` with `last_sequence_id`.
- Server replays unseen messages before live stream continuation.

Idempotency:

- Uniqueness key: `(room_id, client_message_id)`.
- On duplicate send, server returns original `message_id`/`sequence_id`.

## Validation Constraints

- Max body bytes: implementation-defined config value (document default when
  implemented).
- Max history `limit`: 200.
- Trim-only bodies are invalid.
- Unknown `type` values return `INVALID_ARGUMENT`.

## Example Flow

1. Client connects to `/ws/chat` with authenticated context.
2. Client sends `chat.join` with `campaign_id` and optional `last_sequence_id`.
3. Server sends `chat.joined`.
4. Client sends `chat.send` with `client_message_id` and `body`.
5. Server emits `chat.ack` for sender and `chat.message` to all room
   participants (including sender).
6. Client can request older records with `chat.history.before`.

## Acceptance Checklist

- Envelope and all Phase 1 message types are fully specified.
- Error code behavior is explicit and consistent.
- Ordering/reconnect/idempotency semantics are deterministic.
- Scope excludes advanced mode/audience/command features by design.

## Relation to Higher-level Spec

This document defines only Phase 1 wire semantics and complements
`docs/project/chat-service.md`, which remains the source for phased
architecture and long-term capabilities.
