---
title: "Play realtime protocol"
parent: "Platform surfaces"
nav_order: 20
status: canonical
owner: engineering
last_reviewed: "2026-03-26"
---

# Play realtime protocol

WebSocket protocol specification for the play service realtime surface.

## Connection lifecycle

1. Browser fetches `/api/campaigns/{id}/bootstrap` over HTTP, which returns a
   `RealtimeConfig` with `url: "/realtime"` and `protocol_version: 1`.
2. Browser opens a WebSocket to `/realtime`. The play session cookie
   (`play_session`) authenticates the upgrade.
3. Browser sends a `play.connect` frame with the campaign ID and last-known
   sequence cursors. The server responds with `play.ready` containing a full
   room snapshot.
4. Server pushes `play.interaction.updated` whenever game projection state
   changes. Browser sends `play.chat.send` and `play.typing` during the
   session.
5. On disconnect, the server cleans up: stops typing timer, broadcasts
   typing-inactive, and removes the session from its campaign room.

## Frame format

All frames are newline-delimited JSON objects:

```json
{
  "type": "play.<frame_type>",
  "request_id": "<optional correlation id>",
  "payload": { ... }
}
```

`request_id` is echoed back on response frames (`play.ready`, `play.error`,
`play.pong`) for request/response correlation.

## Frame types

### Client to server

| Frame type | Payload | Description |
| --- | --- | --- |
| `play.connect` | `{campaign_id, last_game_seq?, last_chat_seq?}` | Join a campaign room. Server responds with `play.ready`. |
| `play.chat.send` | `{client_message_id?, body}` | Send a human chat message. Broadcast to room as `play.chat.message`. |
| `play.typing` | `{active}` | Typing indicator. Broadcast to room. Auto-expires after typing TTL. |
| `play.ping` | `{}` | Keepalive. Server responds with `play.pong`. |

### Server to client

| Frame type | Payload | Description |
| --- | --- | --- |
| `play.ready` | `RoomSnapshot` | Initial room state after connect. |
| `play.interaction.updated` | `RoomSnapshot` | Game projection changed; full refreshed state. |
| `play.chat.message` | `{message: ChatMessage}` | New chat message (broadcast to all room sessions). |
| `play.typing` | `{session_id, participant_id, name, active}` | Typing indicator update (broadcast to all room sessions). |
| `play.ai_debug.turn.updated` | `AIDebugTurnUpdate` | AI debug turn delta (summary + appended entries). |
| `play.resync` | `{reason}` | Server cannot maintain state; client should reload. |
| `play.pong` | `{timestamp}` | Response to `play.ping`. |
| `play.error` | `{error: {code, message, retryable?, details?}}` | Error response. |

## Error codes

Error codes in `play.error` frames mirror gRPC status code names:

| Code | Meaning |
| --- | --- |
| `invalid_argument` | Malformed frame, missing field, payload too large |
| `resource_exhausted` | Rate limit exceeded (disconnects) |
| `failed_precondition` | Action requires prior state (e.g., connect before chat) |
| `unavailable` | Server shutting down or upstream failure |

## Rate limits and constraints

| Constraint | Value |
| --- | --- |
| Max frame payload size | 32 KB |
| Max frames per second per connection | 50 |
| Max decode errors before disconnect | 3 |
| Max chat message body | 12,000 runes |
| Max client message ID length | 128 characters |
| Typing indicator TTL | 3 seconds |

## Projection subscription

Each campaign room maintains a single gRPC subscription to
`game.v1.EventService.SubscribeCampaignUpdates`. When a `PROJECTION_APPLIED`
update arrives, the room fetches fresh `InteractionState` and broadcasts a
`play.interaction.updated` frame to all connected sessions.

Subscription failures use exponential backoff: 1s initial, 2x multiplier, 30s
cap. A successful connection resets the backoff.

## AI debug subscription

Each campaign room may maintain one AI debug subscription per active session,
via `ai.v1.CampaignDebugService.SubscribeCampaignDebugUpdates`. Turn deltas
are broadcast as `play.ai_debug.turn.updated` frames. The subscription is
reconciled whenever the active session changes.

## Related docs

- [Play architecture](play-architecture.md)
- [Play contributor map](../../reference/play-contributor-map.md)
