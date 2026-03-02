---
title: "Replay operations"
parent: "Running"
nav_order: 10
status: canonical
owner: engineering
last_reviewed: "2026-03-02"
---

# Replay Operations

Operational runbook for replay execution, projection repair, and integrity
troubleshooting.

For replay architecture and invariants, start with
[Event replay](../architecture/foundations/event-replay.md).

## When to use this runbook

Use replay operations when:

- projection state is stale or inconsistent with event journal sequence
- adapter changes require deterministic read-model rebuild
- integrity verification detects replay-path failures

## Replay modes in operations

## Full replay

Use for maximum confidence rebuilds after major projection or adapter changes.

Tradeoff: highest runtime cost.

## Snapshot-accelerated replay

Use for routine catch-up and faster recovery when snapshot/checkpoint state is
trusted.

Tradeoff: relies on snapshot/checkpoint correctness.

## Partial replay

Use when replay scope is known and bounded by sequence.

Tradeoff: requires confidence in start sequence and campaign scope.

## Operator workflow

1. Identify affected campaign IDs and failure symptoms.
2. Confirm latest checkpoint and expected sequence head.
3. Choose replay mode (full/snapshot/partial).
4. Execute replay run.
5. Validate projection parity and checkpoint progression.
6. Re-run affected integration/smoke checks if change was broad.

## Detecting projection gaps

Gap indicators:

- non-contiguous projection sequence markers
- checkpoint stagnation despite new events
- known entity state mismatch with latest event-derived facts

Primary checks:

- compare campaign event head sequence vs projection/checkpoint sequences
- verify adapter routing coverage for event types in affected interval

## Repairing projection gaps

1. stop unsafe writes for affected scope if required
2. run replay in chosen mode
3. verify contiguous sequence application through target head
4. compare critical projection entities against expected event outcomes
5. restore normal writes after parity checks pass

## Post-persist fold/apply failures

If event append succeeded but fold/apply failed:

- treat journal event as authoritative
- fix failing adapter/folder path
- rerun replay to reconcile derived state

Do not delete authoritative events to "repair" projections.

## Integrity checks and constraints

- sequence continuity is mandatory
- hash/signature verification failures are blocking
- unknown system module/adapter routing is fail-fast
- replay operations must not bypass canonical apply paths

## Practical safeguards

- run in smaller campaign batches when diagnosing failures
- capture before/after checkpoint and head sequences per campaign
- log reason codes and failing event types for repeated failures
- keep rollback plan for operational windows that include write pause

## Related docs

- [Event replay architecture](../architecture/foundations/event-replay.md)
- [Event system reference](../reference/event-system-reference.md)
- [Integration tests](integration-tests.md)
