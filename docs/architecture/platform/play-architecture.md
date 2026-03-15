---
title: "Play architecture"
parent: "Platform surfaces"
nav_order: 14
status: canonical
owner: engineering
last_reviewed: "2026-03-14"
---

# Play architecture

Concise architecture contract for the browser-facing active-play service.

## Purpose

`play` owns browser-facing active-play transport after `web` hands the user off
to `/campaigns/{campaign_id}`. It serves the SPA shell, validates the browser
handoff, maintains the play-session cookie, owns websocket fanout, and stores
human transcript history.

`play` does not become gameplay authority. Authoritative interaction state
remains in `game.v1.InteractionService`.

## Package boundaries

- `internal/cmd/play`
  - owns process composition, gRPC connection setup, transcript store
    construction, and shutdown ordering
- `internal/services/play/app`
  - owns HTTP/websocket runtime behavior, browser transport contracts, session
    handoff logic, authenticated browser request mapping, and browser-state
    assembly for bootstrap/history/realtime refresh paths
- `internal/services/play/protocol`
  - owns the browser-facing JSON contract shared by bootstrap, history, and
    websocket payloads
- `internal/services/play/transcript`
  - owns the canonical transcript store contract, including transcript scope,
    append idempotency input, and history pagination defaults
- `internal/services/play/storage/sqlite`
  - owns SQLite transcript persistence, migrations, and concurrent-writer retry
    behavior for the transcript contract
- `internal/services/play/ui`
  - owns the bundled SPA, runtime state/transport seams, renderer view-model
    assembly, and system-specific presentation

## Rules

- `play/app` must not construct gRPC connections or open SQLite stores.
- `play/app` consumes injected collaborators, the canonical `transcript.Store`
  contract, and the shared `play/protocol` browser payload types.
- Browser transport should stay split by responsibility: shell/handoff flow,
  authenticated API request mapping, interaction mutation transport, and
  realtime orchestration should not collapse back into one handler bucket.
- Campaign-path parsing and play-session authentication should flow through a
  dedicated request-context seam inside `play/app`; transport files should not
  re-implement cookie-to-user or campaign-path validation ad hoc.
- The full browser route surface should stay indexed from one route catalog,
  with the interaction mutation subset broken out into its own descriptor list.
  Contributors should be able to find the entire HTTP/WS surface without
  reverse-engineering handler registration flow.
- Bootstrap, history, and realtime refresh assembly should flow through one
  application seam inside `play/app`; route wiring should not manually rebuild
  gRPC auth, transcript queries, and snapshot assembly for each transport path.
- The interaction mutation surface should stay indexed from one descriptor list
  so contributors can see the full browser-facing route set without scanning
  multiple transport helpers.
- The browser runtime should keep IO separate from state transitions: fetch and
  websocket clients belong in dedicated transport modules, while state updates
  belong in pure runtime-state helpers that tests can exercise without browser
  setup.
- System renderers should consume typed view models instead of interpreting raw
  protocol enums and fallback labels inside components. Transport-shaped
  snapshots may enter the runtime boundary, but numeric status normalization and
  renderer-specific display labels belong in dedicated UI view-model helpers.
- Transcript normalization, validation, and history pagination defaults must
  live in `internal/services/play/transcript`; adapters and handlers should
  consume those request/query types instead of open-coding trim/default logic.
- Human chat and typing indicators are `play` transport concerns, not `game`
  domain authority.
- Browser payload contracts should be defined in `internal/services/play/protocol`
  and mirrored in `internal/services/play/ui/src/protocol.ts`; do not redefine
  ad hoc transport structs inside handlers or realtime orchestration.
- Realtime orchestration must keep time/retry behavior explicit and testable.

## Minimum checks

When changing `internal/services/play/**` or `internal/cmd/play/**`, run:

- `go test ./internal/services/play/... ./internal/cmd/play/...`
- `go test -race ./internal/services/play/app ./internal/services/play/storage/sqlite`
- `make play-architecture-check`
- `make smoke` when the browser/runtime path changed

## Related docs

- [Interaction surfaces](interaction-surfaces.md)
- [Small services topology](../../reference/small-services-topology.md)
- [Play contributor map](../../reference/play-contributor-map.md)
