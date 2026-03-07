---
title: "Event replay"
parent: "Foundations"
nav_order: 5
status: canonical
owner: engineering
last_reviewed: "2026-03-06"
---

# Event Replay and Snapshots

Concise architecture contract for replay, checkpointing, and snapshot use.

## Purpose

Replay rebuilds derived state from the authoritative event journal. Snapshots and
checkpoints reduce rebuild cost; they do not replace journal truth.

## Core concepts

- **Event journal**: append-only source of truth.
- **Projection**: derived read state built from ordered events.
- **Checkpoint**: last successfully applied sequence (`last_seq`).
- **Snapshot**: materialized derived state at a sequence point.

## Replay invariants

1. Events are applied in strict sequence order.
2. Sequence gaps are replay errors.
3. Checkpoint progress advances only after successful apply.
4. Unknown system module/adapter routing is replay-fatal.
5. Replay must be deterministic and idempotent.
6. Services must not bypass replay by writing projection state directly.

## Replay modes

- **Full replay**: rebuild from sequence `0`.
- **Snapshot-accelerated replay**: seed from snapshot, continue from snapshot sequence.
- **Partial replay**: resume after a known sequence boundary.

Mode selection is operational; invariants stay the same.

## Checkpoint and snapshot model

- Snapshot-accelerated replay starts from the snapshot sequence.
- When a checkpoint is ahead of snapshot sequence, replay must cap the
  checkpoint cursor at snapshot sequence so no events are skipped.
- Without a snapshot seed, replay starts from max of configured `after_seq`
  and checkpoint sequence.
- Successful apply advances checkpoint.
- Snapshot writes are optimization artifacts and can be recomputed.
- Snapshot corruption must not block journal-based recovery.

## Failure handling model

- **Post-persist fold/apply failure**: event remains authoritative; replay can recover state.
- **Projection drift**: detected via sequence gap checks and repaired via replay.
- **Adapter not found**: fail fast; do not continue with partial projection state.

## Architecture boundary

This page defines replay architecture only. Operator procedures, repair commands,
and runbook workflows live in running docs.

## Deep references

- [Replay operations runbook](../../running/replay-operations.md)
- [Event system reference](../../reference/event-system-reference.md)
- [Event-driven system](event-driven-system.md)
