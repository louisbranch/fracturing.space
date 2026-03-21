# Projection Apply Outbox

The projection apply outbox is an event-journal-backed queue that ensures every
event eventually updates the materialized projection views, even when inline
projection apply fails or is disabled.

## Lifecycle Overview

### Enqueue

When `FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_ENABLED=true`, each
`AppendEvent` call atomically inserts a corresponding row in
`projection_apply_outbox` within the same transaction. This guarantees that
every persisted event has a pending outbox entry.

### Worker Modes

Startup validates that exactly one worker mode is enabled when the outbox is
active. The three valid configurations are:

| Env Flags | Mode | Behavior |
|---|---|---|
| Outbox off | `inline_apply_only` | Events are applied inline during command execution. No outbox rows are created. |
| Outbox on + Worker on | `outbox_apply_only` | Inline apply is disabled. The apply worker drains outbox rows and applies projections. |
| Outbox on + Shadow on | `shadow_only` | Inline apply stays on. The shadow worker drains rows with retry scheduling but does not mutate projections. Useful for validating outbox throughput before cutover. |

### Processing

The apply worker polls `projection_apply_outbox` at a fixed interval (default
2s) and claims a batch of due rows (default 64) in a single transaction:

1. **Claim** -- Select rows with `status IN ('pending', 'failed')` whose
   `next_attempt_at <= now`, plus stale `processing` rows whose `updated_at`
   exceeds the processing lease (2 minutes). Mark claimed rows as `processing`.

2. **Load** -- Retrieve the full event from the journal by campaign ID and
   sequence number.

3. **Filter** -- Skip events whose registered intent is `audit_only` (they do
   not need projection application).

4. **Apply** -- Call the projection applier within an exactly-once transaction.
   The `ApplyProjectionEventExactlyOnce` method uses per-campaign/sequence
   checkpointing so replay is idempotent.

5. **Complete** -- On success, delete the outbox row. On failure, schedule a
   retry (see below).

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

### Claim Transaction

The `claimDue` method runs a single read-then-update transaction that:
1. Queries due rows (pending/failed past `next_attempt_at`, or stale
   processing past the 2-minute lease).
2. Updates each candidate to `processing` status.
3. Commits the claim batch atomically.

This two-phase claim within one transaction prevents concurrent workers from
double-processing the same row. The processing lease (2 minutes) ensures rows
claimed by a crashed worker eventually become eligible for reclaim.

### Apply Transaction (`txStore`)

Projection application uses `ApplyProjectionEventExactlyOnce`, which opens a
separate transaction on the projection database (not the event journal). Inside
this transaction:
- A per-campaign/sequence checkpoint prevents duplicate application.
- All projection mutations happen against `ProjectionApplyTxStore` (the
  transaction-scoped store bundle).
- System adapters are rebound to the transaction store so system-specific
  projection writes participate in the same transaction.

This separation is intentional: the event journal (outbox) and projection
database are distinct SQLite files. The claim transaction operates on the event
journal; the apply transaction operates on the projection database. Atomicity
within each database is guaranteed; cross-database consistency relies on
idempotent replay.

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
