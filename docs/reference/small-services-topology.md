---
title: "Small services topology"
parent: "Reference"
nav_order: 14
status: canonical
owner: engineering
last_reviewed: "2026-03-13"
---

# Small Services Topology

Runtime and ownership map for service processes outside `web`, `game`, and
`admin`.

## Runtime boundaries

| Service | Entrypoint | Command config/wiring | Runtime package |
| --- | --- | --- | --- |
| AI | `cmd/ai` | `internal/cmd/ai` | `internal/services/ai/app` |
| Auth | `cmd/auth` | `internal/cmd/auth` | `internal/services/auth/app` |
| Discovery | `cmd/discovery` | `internal/cmd/discovery` | `internal/services/discovery/app` |
| MCP | `cmd/mcp` | `internal/cmd/mcp` | `internal/services/mcp/service` |
| Notifications | `cmd/notifications` | `internal/cmd/notifications` | `internal/services/notifications/app` |
| Play | `cmd/play` | `internal/cmd/play` | `internal/services/play/app` |
| Social | `cmd/social` | `internal/cmd/social` | `internal/services/social/app` |
| Status | `cmd/status` | `internal/cmd/status` | `internal/services/status/app` |
| Userhub | `cmd/userhub` | `internal/cmd/userhub` | `internal/services/userhub/app` |
| Worker | `cmd/worker` | `internal/cmd/worker` | `internal/services/worker/app` |

## Mode and lifecycle notes

- Shared process bootstrap (`signal` handling, config parse flow, service log
  prefix): `internal/platform/cmd`.
- Shared status capability reporting startup helper: `internal/platform/cmd`.
- Notifications runtime modes:
  - `api`: gRPC API server only.
  - `worker`: email-delivery worker only.
  - `all`: API + worker in one process.
- Play runtime role:
  - browser-facing active-play surface
  - serves the SPA shell and embedded play assets (or a configured Vite dev
    server origin in local UI development)
  - validates web-to-play launch grants and issues host-scoped `play_session`
    cookies
  - owns active-play websocket transport, durable human transcript storage, and
    typing/presence fanout without becoming gameplay authority
  - replaces the removed standalone chat runtime; transcript transport now
    lives inside `play`
- MCP runtime role:
  - internal HTTP bridge between AI orchestration and game/AI services
  - not a public integrator or browser-facing surface
  - production tool exposure is intentionally GM-safe

## Common first-edit seams

- Command/flag or startup wiring changes: `internal/cmd/<service>`.
- Runtime lifecycle (`New`, `Serve`, `Close`) changes: `internal/services/<service>/app` or `.../service`.
- gRPC handler/API behavior changes: `internal/services/<service>/api/grpc/<service>`.
- Storage adapter/schema behavior changes: `internal/services/<service>/storage/sqlite`.

## Related docs

- [Contributor map](contributor-map.md)
- [System architecture](../architecture/foundations/architecture.md)
- [Local development](../running/local-dev.md)
