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

## How `make check` works

`make check` runs:

1. `make check-core`
2. smart focused architecture gates chosen from changed files
3. `make check-runtime`
4. `make check-coverage`

Focused gates are selected from the branch diff against the local
`origin/main` merge-base when available. If `origin/main` is unavailable,
selection falls back to the working tree plus untracked files.

Override the diff base with:

```bash
CHECK_BASE_REF=<ref> make check
```

## Internal-only commands

The repository still contains internal CI/plumbing targets for shard fanout and
reporting. They are not part of the supported contributor workflow and should
not appear in contributor-facing docs.
