# Game gRPC Package Split

Status: **in progress** — M0 complete, M1–M7 pending.

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

## Remaining Milestones

Each milestone is one PR. Tests pass at each boundary.

### M1: Extract `game/gametest/`

Shared test infrastructure (fakes, builder, runtime, fixtures) becomes an
exported non-test package so each entity subpackage can import it without
duplicating ~1468 lines of fake stores.

Files to extract:

- `fakes_test.go` → `gametest/fakes.go`
- `testutil_test.go` → `gametest/builder.go`
- `main_test.go` runtime init → `gametest/runtime.go`
- `campaign_fixtures_test.go` → `gametest/campaign_fixtures.go`
- `participant_fixtures_test.go` → `gametest/participant_fixtures.go`

### M2: Extract invite + scene

Most self-contained entities:

- **`invitetransport/`** — all `invite_*.go` files (~11 source + test files).
- **`scenetransport/`** — all `scene_*.go` files (~12 source + test files).

Root `stores.go` gains `InviteDeps()`, `SceneDeps()` factory methods.

### M3: Extract fork + snapshot + event/timeline

- **`forktransport/`** — ~12 files.
- **`snapshottransport/`** — ~11 files.
- **`eventtransport/`** — ~19 files (event + timeline).

### M4: Extract participant + character

- **`participanttransport/`** — ~14 files. Imports `game/authz/` for telemetry.
- **`charactertransport/`** — ~16 files. `characterworkflow/` stays as-is.

### M5: Extract session + communication

- **`sessiontransport/`** — ~22 files. Communication goes here because
  `communicationApplication` embeds `sessionApplication`.

### M6: Extract campaign + authorization service

- **`campaigntransport/`** — ~29 files. Campaign AI service is campaign-scoped.
- **`authorizationtransport/`** — ~8 files. Imports `game/authz/`.

### M7: Root cleanup

Final root contents (~30 files): `stores.go`, domain adapters, system/statistics
/integration services, architecture tests. Update architecture tests for new
directory structure. Remove dead re-exports.

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
