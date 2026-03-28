# Projection Apply Outbox

The projection apply outbox is the background projection path for the game
server. When enabled, it is an event-journal-backed queue that ensures every
persisted event eventually updates the materialized projection views through a
campaign-owned worker schedule. Request-path inline apply still exists as a
separate runtime mode; the outbox worker does not rely on inline projection
mutation.

## Lifecycle Overview

### Enqueue

When `FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_ENABLED=true`, each
`AppendEvent` call atomically inserts a corresponding row in
`projection_apply_outbox` within the same transaction. This guarantees that
every persisted event has a pending outbox entry.

### Worker Modes

Startup resolves one of three projection apply modes:

| Env Flags | Mode | Behavior |
|---|---|---|
| Outbox off | `inline_apply_only` | Writes apply projections inline on the request path. No outbox worker runs. |
| Outbox on + Worker on | `outbox_apply_only` | The apply worker drains queued projection work and request-path inline apply is disabled. |
| Outbox on + Shadow Worker on | `shadow_only` | The shadow worker drains queue ownership and retry semantics without mutating projections. |

### Processing

When the apply worker is enabled, it polls `projection_apply_outbox` at a fixed
interval (default 2s) and claims a batch of due campaigns (default batch size
64 rows, processed sequentially per campaign):

1. **Claim campaign** -- Find campaigns whose earliest outstanding outbox row is
   due or stale, then claim a campaign lease in the event DB.
2. **Claim row** -- For each leased campaign, claim the earliest processable row
   and mark it `processing`.
3. **Load** -- Retrieve the full event from the journal by campaign ID and
   sequence number.
4. **Filter** -- Skip events whose registered intent is `audit_only`.
5. **Apply** -- Call the projection applier within an exactly-once transaction.
6. **Complete or retry** -- On success, delete the outbox row. On failure,
   schedule a retry. Later rows for the same campaign stay blocked behind the
   failed head row.
7. **Release lease** -- Drop the campaign lease when the campaign has no more
   immediately processable rows in the current pass.

### Retry Backoff

Failed rows are retried with exponential backoff:

```
attempt 1 -> 1s
attempt 2 -> 2s
attempt 3 -> 4s
...
attempt N -> min(2^(N-1)s, 5m)
```

The backoff caps at 5 minutes to avoid indefinitely long waits.

### Dead-Letter Threshold

After **8 consecutive failures** (`deadLetterThreshold`), the row transitions
to `dead` status. Dead rows are excluded from normal worker processing and
require explicit operator intervention:

- `RequeueProjectionApplyOutboxRow(campaignID, seq)` -- requeue a single dead
  row by resetting its status to `pending` with attempt count 0.
- `RequeueProjectionApplyOutboxDeadRows(limit)` -- bulk requeue up to `limit`
  dead rows in deterministic order.

### Inspection

The outbox exposes two inspection methods for maintenance and monitoring:

- `GetProjectionApplyOutboxSummary` -- returns counts by status (`pending`,
  `processing`, `failed`, `dead`) and the oldest retry-eligible entry.
- `ListProjectionApplyOutboxRows(status, limit)` -- lists outbox rows
  optionally filtered by status for detailed inspection.

## Transaction Handling

The outbox uses two distinct transaction scopes:

### Campaign Claim Transaction

The scheduler first claims campaigns, not individual rows. A campaign lease
row in the event DB records the current worker owner plus lease expiry. This
prevents multiple workers from projecting the same campaign concurrently, even
across processes.

Within a claimed campaign, the worker then marks the earliest due row as
`processing`. The row-level status remains useful for crash recovery,
inspection, and dead-letter handling, but it is no longer the primary
concurrency boundary. The campaign lease is.

### Apply Transaction (`txStore`)

Projection application uses `ApplyProjectionEventExactlyOnce`, which opens a
separate transaction on the projection database (not the event journal). Inside
this transaction:
- A per-campaign/sequence checkpoint prevents duplicate application.
- All projection mutations happen against `ProjectionApplyTxStore` (the
  transaction-scoped store bundle).
- System adapters are rebound to the transaction store so system-specific
  projection writes participate in the same transaction.
- The projection store makes a single attempt and returns an error. Retry
  policy lives in the outbox worker rather than inside `coreprojection`.

This separation is intentional: the event journal (outbox) and projection
database are distinct SQLite files. The claim transaction operates on the event
journal; the apply transaction operates on the projection database. Atomicity
within each database is guaranteed; cross-database consistency relies on
idempotent replay. Campaign ownership removes same-campaign contention in the
outbox worker, but it does not change SQLite's single-writer rule for the
shared projection database.

## Watermark Integration

Each successful projection apply updates the campaign's projection watermark
via `SaveProjectionWatermark`. Watermarks track the highest applied sequence
and the expected next sequence, enabling gap detection at startup.

When the outbox worker is enabled, the watermark store is required. Startup
asserts this (`assertWatermarkStoreConfigured`) so that misconfiguration fails
fast rather than silently disabling gap detection.

Gap detection at startup (`DetectProjectionGaps`) compares watermarks against
the event journal high-water mark and triggers targeted replay for any
campaigns with unapplied events.
