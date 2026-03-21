---
title: "Play WebSocket Protocol"
parent: "Reference"
nav_order: 22
status: canonical
owner: engineering
last_reviewed: "2026-03-19"
---

# Play WebSocket Protocol

Reference for the play service real-time WebSocket protocol: frame format,
message types, connection lifecycle, and reconnection behavior.

For play service architecture context, see
[Play contributor map](play-contributor-map.md).

## Connection endpoint

Clients obtain the WebSocket URL from the bootstrap response at
`realtime.url` (default path: `/realtime`). The connection requires an
authenticated session; the server validates identity on the HTTP upgrade.

## Frame format

All frames are JSON text messages with a uniform envelope:

```json
{
  "type": "play.<name>",
  "request_id": "<optional-client-correlation-id>",
  "payload": { ... }
}
```

- `type` is always prefixed with `play.`.
- `request_id` is optional and echoed back on responses when present.
- `payload` contents vary by message type (omitted when empty).

Maximum frame payload size: **8192 bytes**. The server enforces this limit
before JSON decoding (`MaxPayloadBytes`). Frames exceeding this limit are
rejected and the connection may be closed.

## Connection lifecycle

```
Client                            Server
  |                                 |
  |--- WebSocket upgrade ---------->|
  |                                 |
  |--- play.connect --------------->|
  |    { campaign_id,               |
  |      last_game_seq,             |
  |      last_chat_seq }            |
  |                                 |
  |<-------------- play.ready ------|
  |    (full RoomSnapshot)          |
  |                                 |
  |--- play.ping ------------------>|  (every 30s)
  |<-------------- play.pong ------|
  |                                 |
```

1. Client opens a WebSocket connection to the bootstrap `realtime.url`.
2. Client sends `play.connect` with the campaign to join and the last known
   sequence numbers for game events and chat messages.
3. Server responds with `play.ready` containing a full `RoomSnapshot` that
   includes any events the client missed since its last known sequences.
4. Client sends `play.ping` every 30 seconds to keep the connection alive.
5. Server responds with `play.pong`.

After `play.ready`, both sides may send domain messages at any time.

## Client to server messages

### `play.connect`

Initial handshake. Must be the first message after the WebSocket opens.

```json
{
  "type": "play.connect",
  "payload": {
    "campaign_id": "string",
    "last_game_seq": 0,
    "last_chat_seq": 0
  }
}
```

- `campaign_id` -- the campaign room to join.
- `last_game_seq` -- last game event sequence the client has seen (0 for fresh).
- `last_chat_seq` -- last chat message sequence the client has seen (0 for fresh).

### `play.ping`

Keepalive ping. No payload.

```json
{ "type": "play.ping" }
```

### `play.chat.send`

Send a chat message to the room.

```json
{
  "type": "play.chat.send",
  "payload": {
    "client_message_id": "string",
    "body": "string"
  }
}
```

- `client_message_id` -- client-generated idempotency key.
- `body` -- message text content.

### `play.chat.typing`

Chat typing indicator.

```json
{
  "type": "play.chat.typing",
  "payload": {
    "active": true
  }
}
```

### `play.draft.typing`

Draft typing indicator (on-stage content). Same shape as `play.chat.typing`.

```json
{
  "type": "play.draft.typing",
  "payload": {
    "active": true
  }
}
```

## Server to client messages

### `play.ready`

Connection established. Sent exactly once in response to `play.connect`.

```json
{
  "type": "play.ready",
  "payload": {
    "interaction": { ... },
    "participants": [ ... ],
    "character_catalog": [ ... ],
    "chat_snapshot": { ... },
    "last_game_seq": 42
  }
}
```

Payload is a full `RoomSnapshot`: current interaction state, participant list,
character catalog, chat history since the client's last known sequence, and the
latest game event sequence number.

### `play.pong`

Keepalive response.

```json
{
  "type": "play.pong",
  "payload": {
    "timestamp": "2026-03-19T12:00:00Z"
  }
}
```

### `play.chat.message`

A new chat message was posted to the room.

```json
{
  "type": "play.chat.message",
  "payload": {
    "message": {
      "id": "string",
      "participant_id": "string",
      "body": "string",
      "seq": 7,
      "created_at": "2026-03-19T12:00:00Z"
    }
  }
}
```

### `play.chat.typing`

Typing indicator broadcast for chat.

```json
{
  "type": "play.chat.typing",
  "payload": {
    "participant_id": "string",
    "name": "string",
    "active": true
  }
}
```

### `play.draft.typing`

Typing indicator broadcast for on-stage drafts.

```json
{
  "type": "play.draft.typing",
  "payload": {
    "participant_id": "string",
    "name": "string",
    "active": true
  }
}
```

### `play.interaction.updated`

The interaction state changed (new turn, phase transition, etc.). Payload is a
full `RoomSnapshot` with the updated state.

```json
{
  "type": "play.interaction.updated",
  "payload": { ... }
}
```

### `play.resync`

Server-initiated signal that the client should re-bootstrap (server state has
diverged beyond incremental catch-up). No payload.

```json
{ "type": "play.resync" }
```

Clients should respond by closing and re-opening the connection, starting a
fresh `play.connect` handshake.

### `play.error`

Error notification.

```json
{
  "type": "play.error",
  "payload": {
    "code": "string",
    "message": "string"
  }
}
```

## Typing indicator behavior

The server expires typing indicators after a configurable TTL. The default is
3 seconds, communicated to the client via `bootstrap.realtime.typing_ttl_ms`.

To keep a typing indicator active, the client must re-send the typing event
before the TTL expires. When the user stops typing, the client sends
`active: false` to clear the indicator immediately rather than waiting for
server-side expiry.

## Reconnection

When the WebSocket connection drops, the client reconnects with exponential
backoff:

| Attempt | Delay |
|---------|-------|
| 1       | 1 s   |
| 2       | 2 s   |
| 3       | 4 s   |
| 4       | 8 s   |
| 5       | 16 s  |
| 6+      | 30 s (max) |

On reconnect the client re-sends `play.connect` with the latest known
`last_game_seq` and `last_chat_seq` so the server can deliver only missed
events in the `play.ready` snapshot.

The backoff timer resets on a successful `play.ready` response, not on TCP
connection establishment. This prevents tight reconnect loops when the server
accepts TCP but rejects the handshake.

## Related docs

- [Play contributor map](play-contributor-map.md)
