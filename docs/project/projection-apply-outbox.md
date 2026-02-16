---
title: "Projection Apply Outbox"
parent: "Project"
nav_order: 10
---

# Projection Apply Outbox Plan

This document defines an implementation plan to durably couple event append and
projection apply.

## Problem

Today, accepted domain events are appended first, and projection apply happens
afterward in-process. If the process crashes between those steps, projections
can lag until replay/repair runs.

## Goal

Guarantee that every appended event is eventually applied to projections through
a durable retry path, with visibility into lag and failure.

## Scope

In scope:

- durable apply work queue keyed by `campaign_id + seq`
- idempotent projection apply worker
- retry with bounded backoff and poison-message handling
- operational metrics and admin inspection hooks

Out of scope:

- replacing replay as authoritative repair
- redesigning projection schemas
- changing event hash/signature model

## Severity and Complexity

| Item | Severity | Complexity | Notes |
| --- | --- | --- | --- |
| Missing durable coupling between append/apply | High | L | Direct consistency risk under crash/restart |
| Lack of projection lag observability | Medium | M | Slows diagnosis and on-call response |
| No standardized retry/dead-letter semantics | Medium | M | Inconsistent recovery behavior |

## Design

## 1. Durable queue table

Add a queue in projection storage (or dedicated operational DB) with:

- `campaign_id` (TEXT, not null)
- `seq` (INTEGER, not null)
- `event_type` (TEXT, not null)
- `status` (`pending|processing|failed|dead`)
- `attempt_count` (INTEGER, default 0)
- `next_attempt_at` (TIMESTAMP, not null)
- `last_error` (TEXT)
- `updated_at` (TIMESTAMP, not null)

Unique key: `(campaign_id, seq)`.

## 2. Append path behavior

When a domain write appends events:

1. append event to authoritative journal
2. enqueue `(campaign_id, seq)` work item transactionally in the same write unit
3. best-effort inline apply remains optional (fast path), but queue is source of
   retry truth

If inline apply succeeds, mark queue item `done` (or delete row).

## 3. Worker behavior

Worker loop:

1. claim due `pending/failed` rows ordered by `next_attempt_at, seq`
2. load authoritative event by `(campaign_id, seq)`
3. run projection applier idempotently
4. on success: mark done
5. on failure: increment attempts, set backoff, store `last_error`
6. when `attempt_count` exceeds threshold: mark `dead`

Use row-level claim semantics to avoid duplicate workers applying same item.

## 4. Idempotency contract

Projection appliers must tolerate duplicate apply attempts for the same event.

Required check:

- applying the same event twice must not corrupt projection state

## 5. Observability

Expose:

- queue depth by status
- oldest pending age (projection lag)
- apply success/failure counts
- dead-letter count

Add admin inspection endpoints/commands for:

- list pending/failed/dead entries
- replay single queue item
- requeue dead entry after fix

## Rollout Plan

1. Schema and queue writer scaffolding behind feature flag.
2. Worker in shadow mode (enqueue + observe, no apply side effects).
3. Enable apply worker for one environment/canary.
4. Enable globally; keep replay tooling as fallback.
5. Remove legacy assumptions that inline apply is the only path.

Current phase-1 scaffold flag:

- `FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_ENABLED=true` enables
  enqueueing `(campaign_id, seq)` rows on event append.
- `FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_SHADOW_WORKER_ENABLED=true`
  enables the background shadow worker (requires enqueue flag) to claim due
  rows and record retry metadata without applying projections yet.
- `FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_WORKER_ENABLED=true` enables
  the background apply worker (requires enqueue flag) to claim due rows, load
  authoritative events, invoke projection apply, and delete rows on success.
- Apply worker retry semantics: failed apply/load attempts are marked `failed`
  with bounded backoff, and rows are marked `dead` after 8 attempts.
- When both worker flags are enabled, the apply worker takes precedence and the
  shadow worker is disabled to avoid competing consumers.

## Test Plan

Unit:

- enqueue on append for every accepted event
- retry backoff transitions
- dead-letter threshold behavior
- idempotent duplicate apply

Integration:

- simulate crash between append and apply, verify worker repairs projection
- simulate transient applier/storage errors, verify retry and eventual success
- verify queue metrics under load

Non-regression:

- replay from journal still rebuilds correct projections from scratch
- write throughput and p95 latency remain within accepted bounds

## Operational Runbook Notes

- If lag rises: inspect queue oldest pending and last errors.
- Use maintenance CLI for quick inspection:
  - `go run ./cmd/maintenance -outbox-report`
  - `go run ./cmd/maintenance -outbox-report -outbox-status failed -outbox-limit 100`
  - add `-json` for machine-readable output.
- Requeue one dead row after fixing apply logic:
  - `go run ./cmd/maintenance -outbox-requeue -outbox-requeue-campaign-id <campaign_id> -outbox-requeue-seq <seq>`
- Requeue dead rows in bounded batches after fixing apply logic:
  - `go run ./cmd/maintenance -outbox-requeue-dead -outbox-requeue-dead-limit 100`
  - rows are requeued in `next_attempt_at, seq` order so the oldest dead work is retried first.
  - add `-json` for machine-readable batch requeue results.
- If dead-letter grows: fix underlying apply bug, then requeue dead items.
- Replay remains the final repair path when queue state is uncertain.
