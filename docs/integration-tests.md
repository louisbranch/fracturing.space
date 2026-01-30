# Integration Tests

## Overview

Integration tests exercise the full request path through the gRPC server, MCP
bridge, and BoltDB storage using real processes and transports. These tests are
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

1. Start the gRPC server in-process on an ephemeral port.
2. Start the MCP server as a subprocess and point it at the gRPC address.
3. Connect an MCP client over the stdio transport and exchange JSON-RPC.
4. Assert responses using strict or normalized expectations.

## Determinism and Randomness

- Prefer deterministic endpoints for assertions (example: duality_outcome).
- For responses with IDs, timestamps, or seeds, validate structure and reuse
  values across steps instead of matching exact strings.
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

- CI should run the integration tag via make (for example: make cover).
