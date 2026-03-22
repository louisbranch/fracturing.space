---
title: "Integration tests"
parent: "Running"
nav_order: 7
status: canonical
owner: engineering
last_reviewed: "2026-03-17"
---

# Integration Tests

## Overview

Integration tests exercise the full request path through the game server and
SQLite storage using real processes and transports. These tests increase trust
in end-to-end behavior across the game gRPC service. Contributors should use
the public runtime Make targets rather than older integration/scenario-specific
aliases.

## Goals

- Validate game gRPC traffic through direct client calls.
- Verify server and storage wiring in one run.
- Keep tests deterministic by avoiding or normalizing random output.
- Support local runs and CI execution.

## Non-goals

- Performance or load testing.
- Cross-platform process orchestration beyond standard CI runners.

## Execution Model

1. Start the game and auth servers in-process on ephemeral ports.
2. Dial gRPC clients for each service (campaign, participant, character, session, fork, etc.).
3. Exercise service operations and assert responses.

The integration harness creates a shared fixture stack and provides per-test
suites with gRPC clients and user identity. AI-scoped fixtures exercise the
full orchestration path including direct tool dispatch.

## Determinism and Randomness

- Prefer deterministic endpoints for assertions (example: duality_outcome).
- For responses with IDs, timestamps, or seeds, validate structure and reuse
  values across steps instead of matching exact strings.
- Parse timestamps as RFC3339 and assert non-empty IDs.

## Tagging and CI

- Integration tests use the build tag: integration.
- Local run:

```sh
go test -tags=integration ./...
```

### Live AI capture

The GM bootstrap fixture can also be refreshed from a live model run. This is a
manual lane, not part of normal CI, and it exists to prove that a real model can
use the exposed tools before the resulting exchange is replayed
deterministically.

Run the live lane with:

```sh
INTEGRATION_OPENAI_API_KEY=... \
go test -tags='integration liveai' ./internal/test/integration \
  -run TestAIGMCampaignContextLiveCaptureBootstrap -count=1
```

For Daggerheart capability/mechanics guidance changes, also run the live
character-capability lane so the recording proves the model can inspect a sheet
before committing a mechanics-aware beat:

```sh
INTEGRATION_OPENAI_API_KEY=... \
go test -tags='integration liveai' ./internal/test/integration \
  -run TestAIGMCampaignContextLiveCaptureCapabilityLookup -count=1
```

For authoritative Daggerheart mechanics-tool changes, also run the live review
lane so the recording proves the model can combine sheet lookup, action
resolution, and GM review resolution in one turn:

```sh
INTEGRATION_OPENAI_API_KEY=... \
go test -tags='integration liveai' ./internal/test/integration \
  -run TestAIGMCampaignContextLiveCaptureMechanicsReview -count=1
```

For Daggerheart combat-procedure changes, also run the live attack-review lane
so the recording proves the model can combine sheet lookup, combat-board
inspection, and the full attack-flow tool during GM review:

```sh
INTEGRATION_OPENAI_API_KEY=... \
go test -tags='integration liveai' ./internal/test/integration \
  -run TestAIGMCampaignContextLiveCaptureAttackReview -count=1
```

For Daggerheart reaction-procedure changes, also run the live reaction-review
lane so the recording proves the model can combine sheet lookup and the
reaction-flow tool during GM review:

```sh
INTEGRATION_OPENAI_API_KEY=... \
go test -tags='integration liveai' ./internal/test/integration \
  -run TestAIGMCampaignContextLiveCaptureReactionReview -count=1
```

For Daggerheart playbook/reference changes, also run the live playbook attack
lane so the recording proves the model can discover a repo-owned playbook via
`system_reference_search/read` before resolving combat:

```sh
INTEGRATION_OPENAI_API_KEY=... \
go test -tags='integration liveai' ./internal/test/integration \
  -run TestAIGMCampaignContextLiveCapturePlaybookAttackReview -count=1
```

For Daggerheart board-control changes, also run the live spotlight-board review
lane so the recording proves the model can discover the spotlight/countdown
playbook guidance, mutate adversary and countdown state, and then re-read the
board before opening the next beat:

```sh
INTEGRATION_OPENAI_API_KEY=... \
go test -tags='integration liveai' ./internal/test/integration \
  -run TestAIGMCampaignContextLiveCaptureSpotlightBoardReview -count=1
```

For Daggerheart countdown-trigger lifecycle changes, also run the live
countdown-trigger review lane so the recording proves the model can create a
scene countdown, advance it to `TRIGGER_PENDING`, resolve the trigger, and
re-read the board before opening the next beat:

```sh
INTEGRATION_OPENAI_API_KEY=... \
go test -tags='integration liveai' ./internal/test/integration \
  -run TestAIGMCampaignContextLiveCaptureCountdownTriggerReview -count=1
```

For Daggerheart GM Fear placement changes, also run the live GM-move placement
lane so the recording proves the model can create an adversary, spend Fear
through `daggerheart_gm_move_apply`, and re-read the board before reopening the
scene:

```sh
INTEGRATION_OPENAI_API_KEY=... \
go test -tags='integration liveai' ./internal/test/integration \
  -run TestAIGMCampaignContextLiveCaptureGMMovePlacementReview -count=1
```

For Daggerheart adversary combat-procedure changes, also run the live
adversary-attack review lane so the recording proves the model can inspect the
board, resolve an adversary attack, and then reopen play:

```sh
INTEGRATION_OPENAI_API_KEY=... \
go test -tags='integration liveai' ./internal/test/integration \
  -run TestAIGMCampaignContextLiveCaptureAdversaryAttackReview -count=1
```

For Daggerheart group-action and tag-team tooling changes, also run the live
group-action and tag-team lanes so the recording proves the model can read the
relevant character sheets before using the coordinated combat tools:

```sh
INTEGRATION_OPENAI_API_KEY=... \
go test -tags='integration liveai' ./internal/test/integration \
  -run 'TestAIGMCampaignContextLiveCapture(GroupActionReview|TagTeamReview)$' -count=1
```

To run the full Daggerheart live mechanics suite added on this branch in one
batch:

```sh
INTEGRATION_OPENAI_API_KEY=... \
go test -tags='integration liveai' ./internal/test/integration \
  -run 'TestAIGMCampaignContextLiveCapture(CapabilityLookup|MechanicsReview|AttackReview|ReactionReview|PlaybookAttackReview|SpotlightBoardReview|CountdownTriggerReview|GMMovePlacementReview|AdversaryAttackReview|GroupActionReview|TagTeamReview)$' \
  -count=1
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
- Each successful live capture also writes a sibling `.summary.json` file with
  scenario, model, result class, tool names, tool-error count, reference-search
  counts, token usage, and the related raw/markdown artifact names.
- The committed replay fixture is updated only when
  `INTEGRATION_AI_WRITE_FIXTURE=1` is set.
- Failed live runs do not overwrite the committed fixture.

For the current checked-in Daggerheart mechanics comparison table built from
those summaries, see
[daggerheart-live-mechanics-matrix.md](../reference/daggerheart-live-mechanics-matrix.md).

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
