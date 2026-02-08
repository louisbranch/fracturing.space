# Event Replay and Snapshots

This guide explains how the event journal, projections, and snapshots work
together, and how to replay events to rebuild derived state.

## Concepts

- **Event journal**: the append-only source of truth for a campaign.
- **Projection**: a derived view built by applying events in order.
- **Snapshot**: a materialized projection derived from the event journal at a
  specific sequence to speed up replay. Snapshots are not authoritative.

Snapshots never replace the event journal; they only accelerate rebuilds.

## Replay modes

### Full replay

Rebuild projections from the beginning of the campaign journal. Use this after
schema or projection changes to re-derive state from first principles.

### Snapshot-accelerated replay

Start from the latest snapshot and apply events after the snapshot sequence.
This is the default for most recovery and rebuild workflows.

### Partial replay

Replay a bounded window of events (after-seq / until-seq). This is useful for
targeted checks or backfills without reprocessing the full history.

## What snapshots contain

Snapshots capture projection state needed for fast rebuilds without replaying
the full event journal.
Today this includes Daggerheart character state and GM fear. Snapshots do not
contain story content, telemetry, or other non-canonical data.

## Admin CLI workflows

The maintenance CLI can scan, validate, replay, or check integrity for a campaign.

```bash
# Scan snapshot-related events without applying projections
cmd/maintenance -campaign-id camp_123 -dry-run

# Validate snapshot event payloads
cmd/maintenance -campaign-id camp_123 -validate

# Replay snapshot-related events and apply projections
cmd/maintenance -campaign-id camp_123

# Integrity check (replay into scratch store and compare)
cmd/maintenance -campaign-id camp_123 -integrity

# Batch and JSON output
cmd/maintenance -campaign-ids camp_123,camp_456 -validate -json
```

Warnings are capped by default (`-warnings-cap 25`). Set `-warnings-cap 0` to
disable the cap.

## Operational notes

- Event order is authoritative; projections assume sequential application.
- Validation can fail if payloads are malformed or out of bounds.
- Integrity checks compare stored projections against a clean replay and exit
  non-zero on mismatches.

## Best practices

- Run `-integrity` after migrations or bulk backfills.
- Use `-validate` before replaying in production to surface invalid events.
- Prefer snapshot-accelerated replay for routine rebuilds.
