---
title: "Integration tests"
parent: "Running"
nav_order: 7
status: canonical
owner: engineering
last_reviewed: "2026-03-14"
---

# Integration Tests

## Overview

Integration tests exercise the full request path through the game server, MCP
bridge, and SQLite storage using real processes and transports. These tests are
meant to increase trust in end-to-end behavior across the internal AI bridge.
Contributors should use the public runtime Make targets rather than the older
integration/scenario-specific aliases.

## Goals

- Validate MCP JSON-RPC traffic over the internal HTTP bridge.
- Verify server, MCP, and storage wiring in one run.
- Keep tests deterministic by avoiding or normalizing random output.
- Support local runs and CI execution.

## Non-goals

- Performance or load testing.
- Cross-platform process orchestration beyond standard CI runners.

## Execution Model

1. Start the game server in-process on an ephemeral port.
2. Start the MCP server as a subprocess and point it at the game address.
3. Connect an MCP client over the HTTP bridge and exchange JSON-RPC.
4. Assert responses using strict or normalized expectations.

The general integration harness defaults to a non-production MCP harness
profile so legacy context-dependent coverage can bootstrap campaign/session
identity with a test-only `set_context` tool. The generic HTTP blackbox
fixtures also run against that harness profile; production AI-bridge coverage
is handled separately by the AI-scoped fixtures and orchestration tests.

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

Use fixture-level transport assertions only when the scenario depends on
streaming update behavior.

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
- GM-safe scene/interaction flow: create a scene, drive interaction state, and
  verify authoritative resources update as expected.
- Rules metadata: verify duality_rules_version returns stable fields.

## Tagging and CI

- Integration tests use the build tag: integration.
- Local run:

```sh
go test -tags=integration ./...
```

### Live AI capture

The GM bootstrap fixture can also be refreshed from a live model run. This is a
manual lane, not part of normal CI, and it exists to prove that a real model can
use the exposed MCP tools before the resulting exchange is replayed
deterministically.

Run the live lane with:

```sh
INTEGRATION_OPENAI_API_KEY=... \
go test -tags='integration liveai' ./internal/test/integration \
  -run TestAIGMCampaignContextLiveCaptureBootstrap -count=1
```

Optional environment variables:

- `INTEGRATION_AI_MODEL`: model name to use; defaults to `gpt-5.4`
- `INTEGRATION_AI_REASONING_EFFORT`: Responses API reasoning effort; defaults to `medium`
- `INTEGRATION_OPENAI_RESPONSES_URL`: alternate OpenAI-compatible Responses URL
- `INTEGRATION_AI_WRITE_FIXTURE=1`: allow the test to overwrite the committed
  replay fixture after a successful live run

Behavior:

- Raw live provider captures are always written under `.tmp/ai-live-captures/`
  for local inspection.
- The committed replay fixture is updated only when
  `INTEGRATION_AI_WRITE_FIXTURE=1` is set.
- Failed live runs do not overwrite the committed fixture.

## Supported verification commands

For the supported contributor workflow, use the canonical
[Verification commands](verification.md) surface. Raw
`go test -tags=integration ./...` remains useful for low-level debugging, but it
is not the default contributor path.

## Scenario Sharding

Scenario tests support deterministic sharding for CI fanout. Treat shard entry
points as internal CI plumbing; contributors should use `make test`,
`make smoke`, and `make check`.

## Integration Sharding

Integration tests support deterministic top-level test sharding for CI fanout
and CI may invoke shard-specific targets internally.

Top-level `Test...` names are assigned by stable hash modulo shard total.

CI target guidance:

- Pull requests should use the public `make check` surface locally.
- Main/nightly workflows may shard full runtime lanes internally for fanout.
- Nightly soak runs may enable shared-fixture variants as internal workflow detail.

## Runtime Reporting

Runtime reports are generated from `go test -json` output by CI/internal
automation, and the public local verification commands now also emit live status
artifacts under `.tmp/test-status/`. Treat the shard scripts and report
generation helpers as internal plumbing; the supported public surface remains
`make test`, `make smoke`, and `make check`.

## Checklist

- If event definitions changed, run `go run ./internal/tools/eventdocgen`
  and confirm the [event catalog](../events/event-catalog.md) is updated in the diff.

- Use the public Make verification surface above; avoid depending on retired
  shard/plumbing targets in contributor-facing docs.
