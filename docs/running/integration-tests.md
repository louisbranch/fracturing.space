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

- Make targets:

```sh
make test
make integration
make cover
```

## Checklist

- If event definitions changed, run `go generate ./internal/services/game/domain/campaign/event`
  and confirm `docs/events/event-catalog.md` is updated in the diff.

- CI should run the integration tag via make (for example: make cover).
