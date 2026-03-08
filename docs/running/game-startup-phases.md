---
title: "Game server startup phases"
parent: "Running"
nav_order: 11
status: canonical
owner: engineering
last_reviewed: "2026-03-07"
---

# Game Server Startup Phases

The game server starts through a sequence of validated phases. Each phase
registers rollback handlers so failures at any point clean up earlier
resources in reverse order.

Entry point: `server.NewWithAddrContext()` in `internal/services/game/app/bootstrap.go`.

## Phase overview

```mermaid
flowchart TD
    START([NewWithAddrContext]) --> P1

    P1["1. Registries\ncommand/event/system types"]
    P1 -->|fatal| P2
    P2["2. Network\nTCP listener"]
    P2 -->|fatal| P3
    P3["3. Storage\nevents + projections + content DBs"]
    P3 -->|fatal| P4
    P4["4. Domain\nstores, applier, write runtime"]
    P4 -->|fatal| P5
    P5["5. Systems\nparity check, gap repair, policy"]
    P5 -->|fatal| P6
    P6["6. Dependencies\nAuth, Social, AI, Status"]
    P6 -->|"Auth fatal\nothers graceful"| P7
    P7["7. Transport\ngRPC service registration"]
    P7 -->|fatal| P8
    P8["8. Runtime\nprojection mode, workers, status"]
    P8 --> SERVE([Server.Serve])

    P1 & P2 & P3 & P4 & P5 & P6 & P7 & P8 -.->|"on failure"| RB["LIFO Rollback\nclean up in reverse order"]
```

| # | Phase | File | Failure behavior |
|---|-------|------|-----------------|
| 1 | Registries | `bootstrap.go` | Fatal — command/event types misconfigured |
| 2 | Network | `bootstrap.go` | Fatal — port unavailable |
| 3 | Storage | `server_bootstrap.go` | Fatal — database open/migration failure |
| 4 | Domain | `bootstrap.go` | Fatal — store wiring or applier validation failure |
| 5 | Systems | `bootstrap.go`, `system_registration.go` | Fatal — module/adapter/metadata parity violation |
| 6 | Dependencies | `server_bootstrap.go` | Auth fatal; Social/AI/Status graceful |
| 7 | Transport | `bootstrap_service_registration.go` | Fatal — gRPC service registration failure |
| 8 | Runtime | `bootstrap.go` | Fatal — projection mode or worker configuration error |

## Phase details

### 1. Registries

`engine.BuildRegistries(registeredSystemModules()...)`

Initializes command, event, and system registries from declared game system
modules. Validates that command types map to handlers and event types map to
fold functions. Output drives all downstream wiring.

### 2. Network

`net.Listen("tcp", addr)`

Opens the gRPC listener. Registered for rollback cleanup. The listener stays
open until the serve loop starts.

### 3. Storage

`openStorageBundle()` in `server_bootstrap.go`

Opens three SQLite databases:
- **Events** (`FRACTURING_SPACE_GAME_EVENTS_DB_PATH`) — event journal with
  integrity keyring and chain verification
- **Projections** (`FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH`) — materialized
  read models
- **Content** (`FRACTURING_SPACE_GAME_CONTENT_DB_PATH`) — Daggerheart content
  metadata

Each database runs migrations on open and is registered for rollback cleanup.

### 4. Domain

Builds the `Stores` struct and projection `Applier`:
1. Create `WriteRuntime` for in-flight write tracking
2. Build `gamegrpc.Stores` from projection database
3. Attach event registry
4. Configure domain execution layer
5. Validate all stores are wired
6. Extract `projection.Applier`

### 5. Systems

Three substeps:
1. **System metadata registry** — loads game system metadata (Daggerheart)
2. **Parity validation** — ensures module registries, adapter registries, and
   metadata registries all agree
3. **Projection gap repair** — detects campaigns with stale projections and
   replays missing events
4. **Session lock policy validation** — ensures transport interceptor and domain
   policy agree on blocked commands

### 6. Dependencies

Connects to external microservices:
- **Auth** (required) — `FRACTURING_SPACE_AUTH_ADDR`
- **Social** (graceful) — `FRACTURING_SPACE_SOCIAL_ADDR` — logs warning if
  unavailable
- **AI** (graceful) — `FRACTURING_SPACE_AI_ADDR` — logs warning if unavailable
- **Status** (advisory) — `FRACTURING_SPACE_STATUS_ADDR` — accumulates locally
  if unavailable
- **AI session grant config** — loaded from environment

### 7. Transport

`registerServices()` in `bootstrap_service_registration.go`

Builds and mounts gRPC service descriptors:
- 3 Daggerheart services (core, content, assets)
- 13 game core services (Campaign, Participant, Character, Session, etc.)
- Health service with per-service status

### 8. Runtime

Configures background workers and status reporting:
- **Projection apply mode** — resolves `inline_apply_only` (default),
  `outbox_apply_only`, or `shadow_only` from environment
- **Outbox worker** — processes queued projection events (if enabled)
- **Shadow worker** — cleans up processed outbox rows (if enabled)
- **Status reporter** — heartbeat and catalog availability monitoring

## Serve loop

After bootstrap, `Server.Serve(ctx)` starts:
1. Background workers (projection, status, catalog monitor)
2. gRPC server on the listener
3. Shutdown on context cancellation — workers stop, gRPC graceful stop,
   resources closed

## Rollback on failure

All phases register cleanup in a LIFO stack:
1. Listener close
2. Storage bundle close
3. Auth connection close
4. Social connection close
5. AI connection close
6. Status connection close

On error in any phase, cleanup runs in reverse order.
