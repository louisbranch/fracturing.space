---
title: "Integration tests"
parent: "Running"
nav_order: 7
status: canonical
owner: engineering
last_reviewed: "2026-02-26"
---

# Integration Tests

## Overview

Integration tests exercise the full request path through the game server, MCP
bridge, and SQLite storage using real processes and transports. These tests are
meant to increase trust in end-to-end behavior and backward compatibility.

## Goals

- Validate MCP JSON-RPC traffic over stdio for real client behavior.
- Verify server, MCP, and storage wiring in one run.
- Keep tests deterministic by avoiding or normalizing random output.
- Support local runs and CI execution.

## Non-goals

- Full HTTP transport coverage (planned for a later phase).
- Performance or load testing.
- Cross-platform process orchestration beyond standard CI runners.

## Execution Model

1. Start the game server in-process on an ephemeral port.
2. Start the MCP server as a subprocess and point it at the game address.
3. Connect an MCP client over the stdio transport and exchange JSON-RPC.
4. Assert responses using strict or normalized expectations.

## Scenario Fixture Format

Blackbox tests load action-focused scenario fixtures from
`internal/test/integration/fixtures/blackbox_*.json`. Each fixture describes human
actions (initialize, subscribe, tool calls, resource reads) and the loader
expands them into JSON-RPC requests with IDs, jsonrpc version, and optional
expectations. This keeps scenarios readable while still validating protocol
correctness.

### Blocks (Reusable Steps)

Define reusable blocks in the `blocks` section and reference them with `use` in
the main `steps` array. Blocks are inlined during expansion, so they can be
shared across scenarios without repeating handshakes or setup flows.

### Expectations

Use `expect: ok` (default) to validate jsonrpc/id, `expect: none` to skip the
implicit protocol assertions, or `expect: no_response` for notifications.
`expect_paths` always applies when provided so you can validate specific fields
without the default jsonrpc/id checks.

### Captures

Use `capture` to extract IDs from responses and reuse them in later steps via
`{{capture_name}}` or `{ "ref": "capture_name" }` for direct substitutions. The
loader supports shortcuts like `campaign`, `participant`, `character`, and
`session` to map to common structuredContent ID paths.

Use `expect_sse: true` at the fixture level to assert SSE resource updates for
that scenario. Other fixtures omit SSE checks by default.

Example:

```json
{
  "blocks": {
    "handshake": [
      {"action": "initialize"},
      {"action": "initialized"}
    ]
  },
  "steps": [
    {"use": "handshake"},
    {
      "action": "tool_call",
      "tool": "campaign_create",
      "args": {"name": "Test Campaign", "gm_mode": "HUMAN"},
      "capture": {"campaign_id": "campaign"}
    },
    {
      "action": "tool_call",
      "tool": "participant_create",
      "args": {"campaign_id": {"ref": "campaign_id"}, "display_name": "Player"}
    },
    {
      "action": "read_resource",
      "uri": "campaign://{{campaign_id}}",
      "expect_paths": {
        "result.contents[0].text|json.campaign.id": "{{campaign_id}}"
      }
    }
  ]
}
```

## Determinism and Randomness

- Prefer deterministic endpoints for assertions (example: duality_outcome).
- For responses with IDs, timestamps, or seeds, validate structure and reuse
  values across steps instead of matching exact strings.
- For list resources without stable ordering, use `expect_contains` to assert a
  matching entry exists rather than hard-coding array indices.
- Parse timestamps as RFC3339 and assert non-empty IDs.

## Candidate Test Cases

- List tools: verify expected tool IDs are returned.
- Duality outcome: call with fixed dice and verify exact output.
- Campaign create + list: create a campaign, then read campaigns://list and
  assert the new campaign is present with matching IDs and timestamps.
- Rules metadata: verify duality_rules_version returns stable fields.

## Tagging and CI

- Integration tests use the build tag: integration.
- Local run:

```sh
go test -tags=integration ./...
```

## Command Matrix

Use these commands by audience and intent:

| Audience | Command | Use case | When to run |
| --- | --- | --- | --- |
| Users | `make integration-smoke` | Fast local confidence check | During active feature work before commit |
| Users | `make integration` | Full deterministic integration verification | Before opening/merging runtime-impacting changes |
| Agents | `make integration-smoke` | Short feedback loop in implementation | After each meaningful iteration |
| Agents | `make integration` | Completion gate for integration behavior | Before reporting task done |
| CI (PR) | `make integration-smoke-pr` | Fast PR gate with stdio + HTTP smoke | Every pull request |
| CI (main/nightly) | `INTEGRATION_VERIFY_SHARDS_TOTAL=4 make integration-shard-check` | Ensure shard coverage is complete/non-overlapping | Before shard matrix execution |
| CI (main/nightly) | `INTEGRATION_SHARD_TOTAL=4 INTEGRATION_SHARD_INDEX=<n> make integration-shard` | Parallel full integration fanout | Matrix jobs on non-PR workflows |

Alias behavior:

- `make integration-smoke` routes to `integration-smoke-pr`.
- `make integration` routes to `integration-full` by default.
- If `INTEGRATION_SHARD_TOTAL` and `INTEGRATION_SHARD_INDEX` are set,
  `make integration` routes to `integration-shard`.

Advanced explicit targets:

```sh
make integration-smoke-pr
make integration-smoke-full
make integration-full
make integration-shard
make integration-shard-check
```

- Scenario companion lanes (for game flow contracts):

```sh
make scenario-smoke
make scenario-full
```

- Make targets:

```sh
make test
make integration
make cover
```

## Scenario Sharding

Scenario tests support deterministic sharding for CI fanout:

```sh
SCENARIO_SHARD_TOTAL=6 SCENARIO_SHARD_INDEX=0 make scenario-shard
SCENARIO_VERIFY_SHARDS_TOTAL=6 make scenario-shard-check
```

Each scenario file is assigned by stable hash of its relative path.

## Integration Sharding

Integration tests support deterministic top-level test sharding for CI fanout
and CI should use explicit shard targets:

```sh
INTEGRATION_SHARD_TOTAL=4 INTEGRATION_SHARD_INDEX=0 make integration-shard
INTEGRATION_VERIFY_SHARDS_TOTAL=4 make integration-shard-check
```

Top-level `Test...` names are assigned by stable hash modulo shard total.

CI target guidance:

- Pull requests: `make integration-smoke-pr` + `make scenario-smoke`.
- Main/nightly: `make integration-shard-check` + shard matrix `make integration-shard`.
- Nightly soak: shard runs with `INTEGRATION_SHARED_FIXTURE=true` (non-blocking until promoted).

## Runtime Reporting

Generate runtime artifacts (JSON + CSV) from `go test -json` output:

```sh
bash ./scripts/test-runtime-report.sh smoke
bash ./scripts/test-runtime-report.sh smoke-pr
bash ./scripts/test-runtime-report.sh integration-full
INTEGRATION_SHARD_TOTAL=4 INTEGRATION_SHARD_INDEX=0 bash ./scripts/test-runtime-report.sh integration-shard
SCENARIO_SHARD_TOTAL=6 SCENARIO_SHARD_INDEX=0 bash ./scripts/test-runtime-report.sh scenario-shard
```

Budget thresholds live in `.github/test-runtime-budgets.json`. By default, budget checks emit warnings. Set `RUNTIME_BUDGET_ENFORCE=true` to fail on regressions.

## Checklist

- If event definitions changed, run `go run ./internal/tools/eventdocgen`
  and confirm the [event catalog](../events/event-catalog.md) is updated in the diff.

- CI should run the integration tag via make (for example: make cover).
