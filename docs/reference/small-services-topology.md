---
title: "Small services topology"
parent: "Reference"
nav_order: 14
status: canonical
owner: engineering
last_reviewed: "2026-03-07"
---

# Small Services Topology

Runtime and ownership map for service processes outside `web`, `game`, and
`admin`.

## Runtime boundaries

| Service | Entrypoint | Command config/wiring | Runtime package |
| --- | --- | --- | --- |
| AI | `cmd/ai` | `internal/cmd/ai` | `internal/services/ai/app` |
| Auth | `cmd/auth` | `internal/cmd/auth` | `internal/services/auth/app` |
| Chat | `cmd/chat` | `internal/cmd/chat` | `internal/services/chat/app` |
| Discovery | `cmd/discovery` | `internal/cmd/discovery` | `internal/services/discovery/app` |
| MCP | `cmd/mcp` | `internal/cmd/mcp` | `internal/services/mcp/service` |
| Notifications | `cmd/notifications` | `internal/cmd/notifications` | `internal/services/notifications/app` |
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
- MCP transport modes:
  - `stdio`: local MCP host integration.
  - `http`: HTTP/SSE transport boundary.

## Common first-edit seams

- Command/flag or startup wiring changes: `internal/cmd/<service>`.
- Runtime lifecycle (`New`, `Serve`, `Close`) changes: `internal/services/<service>/app` or `.../service`.
- gRPC handler/API behavior changes: `internal/services/<service>/api/grpc/<service>`.
- Storage adapter/schema behavior changes: `internal/services/<service>/storage/sqlite`.

## Related docs

- [Contributor map](contributor-map.md)
- [System architecture](../architecture/foundations/architecture.md)
- [Local development](../running/local-dev.md)
