# Game gRPC Package Split

Status: **complete** — all milestones (M0–M7) done.

## Motivation

`internal/services/game/api/grpc/game/` had 143 source files in a single Go
package. This is the single biggest structural barrier to contributor onboarding
and makes unrelated entity concerns share a single compilation unit.

The application layer is already internally decomposed — each entity's
application struct holds per-entity store subsets and scoped policy deps. The
extraction moves existing boundaries to package boundaries, not inventing new
ones.

## Completed

### M0: Extract `game/authz/` and `game/handler/`

Two foundation packages that cross-cutting code depends on:

- **`game/authz/`** — authorization policy enforcement, actor resolution,
  evaluator, telemetry, mapping helpers, target evaluator, constants.
- **`game/handler/`** — domain write helpers, pagination constants,
  command/event type identifiers, actor mapping, social profile resolver,
  auth username lookup, default names, system ID helpers.

Root dropped from 143 to ~124 source files. 21 old files deleted, ~55 root
callers updated.

### M1: Extract `game/gametest/`

Shared test infrastructure (fakes, builder, runtime, fixtures) moved to an
exported non-test package. ~1468 lines of fake stores, fixtures, and helpers
reused by all entity subpackages.

### M2: Extract invite + scene

- **`invitetransport/`** — invite lifecycle, claim, revoke, list.
- **`scenetransport/`** — scene CRUD, character membership.

### M3: Extract fork + snapshot + event/timeline

- **`forktransport/`** — campaign fork management.
- **`snapshottransport/`** — character snapshot CRUD.
- **`eventtransport/`** — event timeline, replay, append.

### M4: Extract participant + character

- **`participanttransport/`** — participant lifecycle, social, access.
- **`charactertransport/`** — character CRUD, profile management.

### M5: Extract session + communication

- **`sessiontransport/`** — session lifecycle, gates, spotlight, and
  communication context/control. Communication lives here because
  `communicationApplication` embeds `sessionApplication`.

### M6: Extract campaign + authorization service

- **`campaigntransport/`** — 22 source files + 6 test files. Campaign AI service
  is campaign-scoped. `NewClearCampaignAIBindingFunc` exported for participant
  service cross-cutting use.
- **`authorizationtransport/`** — 4 source files + 1 test file + doc.go. Deps
  struct with Campaign, Participant, Character, Audit stores; delegates to
  `authz.Evaluator`.

### M7: Root cleanup

Root reduced to 23 files: `stores*.go` (dependency container), domain/system
adapters, integration/statistics/system services, architecture tests, and
`doc.go`. Removed dead test infrastructure (`testStoresBuilder`, `fakeDomainEngine`,
`testRuntime`, `mustJSON`), deleted empty `main_test.go`, fixed stale
architecture test path (`event_application.go` → `eventtransport/`), and updated
`doc.go` to reflect post-extraction package layout.

## Import DAG

```
domain/  storage/  grpc/internal/{domainwriteexec,grpcerror,...}
    ↑         ↑         ↑
    |         |         |
game/handler/ ─────────┘
game/authz/   ────────────┘
    ↑     ↑
    |     |
game/{entity}transport/   (one per entity, imports handler + authz)
    ↑
    |
game/  (root: Stores, wiring, small services, architecture tests)
    ↑
    |
app/  (bootstrap, service registration)
```

## Verification

- `make test` after each milestone
- `make check` after M6/M7
- Architecture tests updated to scan new subpackage paths
- No import cycles: `go build ./...` confirms clean DAG
