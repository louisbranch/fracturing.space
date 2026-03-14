---
title: "Verification commands"
parent: "Running"
nav_order: 12
status: canonical
owner: engineering
last_reviewed: "2026-03-10"
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

## Focused diagnostics

These commands remain supported, but they are not part of the default command
matrix:

- `make docs-check`
- `make web-architecture-check`
- `make game-architecture-check`
- `make admin-architecture-check`
- `make cover`
- `make cover-critical-domain`

Use them when you are debugging a specific surface or when you need focused
coverage output separate from `make check`.

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

## Internal-only commands

The repository still contains internal CI/plumbing targets for shard fanout and
reporting. They are not part of the supported contributor workflow and should
not appear in contributor-facing docs.
