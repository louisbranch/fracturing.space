---
title: "Verification commands"
parent: "Running"
nav_order: 12
status: canonical
owner: engineering
last_reviewed: "2026-03-24"
---

# Verification commands

Canonical contributor and agent verification workflow.

## Public command surface

Use this 3-command surface for normal development:

| Command | Use case | When to run |
| --- | --- | --- |
| `make test` | Fast unit/domain verification | During normal implementation |
| `make smoke` | Quick runtime confidence across integration and scenario smoke coverage | When runtime paths need quick feedback |
| `make check` | Full local guard including static checks, smart architecture gates, full runtime coverage, and coverage baselines | Immediately before push, PR open, or PR update |

For web changes, use the canonical [Web testing map](../architecture/platform/web-testing-map.md)
to choose the right focused test files before broadening to the public command
surface.

## Focused diagnostics

These commands remain supported, but they are not part of the default command
matrix:

- `make docs-check`
- `make web-architecture-check`
- `make game-architecture-check`
- `make admin-architecture-check`
- `make play-architecture-check`
- `make ai-eval-promptfoo`
- `make ai-eval-promptfoo-core`
- `make ai-eval-promptfoo-decision`
- `make ai-eval-promptfoo-view`
- `make cover`
- `make cover-critical-domain`

Use them when you are debugging a specific surface or when you need focused
coverage output separate from `make check`.

For the Promptfoo commands above:

- Promptfoo phase 2 is complete as a non-gating focused diagnostics surface; use
  the existing `core`, `decision`, and `view` targets rather than adding more
  adoption plumbing during normal implementation
- runtime-invalid rows remain visible, but the generated scorecard computes
  quality pass rate from valid runs only
- per-case live diagnostics now land in `.tmp/ai-live-captures/*.diagnostics.json`
  and are the canonical deep-dive artifact when a live eval fails

`make check` already runs the coverage lane. Do not start `make cover` or
`make cover-critical-domain` in parallel with `make check`; the coverage
commands share artifact paths and are guarded to fail fast on overlap.

## How `make check` works

`make check` runs:

1. `make check-core`
2. smart focused architecture gates chosen from changed files
3. `make check-runtime`
4. `make check-coverage`

`make check-runtime` now covers runtime shape checks plus the scenario suite.
`make check-coverage` owns the full unit+integration repository run as part of
the coverage lane, so `make check` no longer repeats a separate non-coverage
`go test -tags=integration ./...` pass.

Focused gates are selected from the branch diff against the local
`origin/main` merge-base when available. If `origin/main` is unavailable,
selection falls back to the working tree plus untracked files.

Override the diff base with:

```bash
CHECK_BASE_REF=<ref> make check
```

## Live status artifacts

Public verification commands now write live status under `.tmp/test-status/`:

- `make test` updates `.tmp/test-status/test/`
- `make smoke` updates `.tmp/test-status/smoke/`
- `make check` updates `.tmp/test-status/check/`

Long-running `go test -json` lanes also write per-lane status files beneath
those directories, such as scenario or coverage shard runs. Status JSON is
meant to help humans and agents tell whether a lane is still running, which
stage is active, and which package/test is currently hot.

For `make smoke` and `make check`, the top-level `status.json` reports aggregate
stage progress while nested files report lane-specific test activity.

Local verification commands also isolate command-owned temp work under
`.tmp/test-tmp/` and remove those per-run temp roots on exit. New local test
artifacts should not accumulate in system `/tmp` during normal runs.

## Adding a coverage floor

Coverage floors in `docs/reference/coverage-floors.json` prevent regression in
tested packages. When you add meaningful tests to a package, consider adding a
floor entry.

**Measure the current baseline:**

```bash
go test ./internal/services/<path>/... -cover -count=1 -short
```

**Add the entry** to `docs/reference/coverage-floors.json`:

```json
{
  "package": "github.com/louisbranch/fracturing.space/internal/services/<path>",
  "floor": 73.0,
  "description": "One-line description of what the package tests protect."
}
```

Set the floor 2–3% below the measured value. The `allow_drop` tolerance
(currently 0.1%) absorbs tiny fluctuations from unrelated changes.

**Verify** the floor is enforced by running `make check` — the coverage lane
will fail if the package drops below its floor.

## Testing guidance for contributors

### Test error paths, not just happy paths

If a fake supports error injection fields (`.PutErr`, `.GetErr`, `.ListErr`,
`.UpdateErr`, `.EnqueueErr`), add subtests that set those fields and verify the
handler returns the expected gRPC status code. Example:

```go
func TestCreateWidget_StoreError(t *testing.T) {
    store := fakes.NewWidgetStore()
    store.PutErr = errors.New("db write fail")
    svc := newTestService(store)

    _, err := svc.CreateWidget(ctx, req)
    grpcassert.StatusCode(t, err, codes.Internal)
}
```

### Assert response content, not just status codes

Handler tests should verify at least one meaningful property of the response
body beyond the HTTP/gRPC status. For HTML handlers, check a key element or
data attribute. For gRPC handlers, check a response field value.

### Use the shared grpcassert package

Import `internal/test/grpcassert` for gRPC status assertions instead of
writing local helpers:

```go
grpcassert.StatusCode(t, err, codes.NotFound)
grpcassert.StatusMessage(t, err, "campaign not found")
```

Game transport packages that exercise handlers returning domain errors should
use a local `assertStatusCode` wrapper that calls
`grpcerror.HandleDomainError` before delegating to `grpcassert.StatusCode`.

## Internal-only commands

The repository still contains internal CI/plumbing targets for shard fanout and
reporting. They are not part of the supported contributor workflow and should
not appear in contributor-facing docs.
