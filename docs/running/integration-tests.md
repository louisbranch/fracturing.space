---
title: "Integration tests"
parent: "Running"
nav_order: 7
status: canonical
owner: engineering
last_reviewed: "2026-03-24"
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

## Canonical Service Chains

Use integration tests for cross-service runtime chains that need real transport,
storage, and startup wiring:

- `invite -> worker -> notifications -> userhub`: prove invite outbox events
  become inbox notifications and then appear on the dashboard.
- `web -> play -> game`: prove authenticated web launch reaches play and a real
  interaction mutates game state.
- `admin -> game`: prove admin pages and HTMX refresh paths stay aligned with
  live game mutations.
- `discovery -> game`: prove builtin starter reconciliation creates a real
  starter campaign and persists the resulting discovery `source_id`.
- `userhub` degraded mode: prove one optional downstream can fail without
  breaking the whole dashboard.

Keep pure game acceptance behavior in scenario scripts. Do not move game-only
workflows into integration tests just to increase service count.

## Lane Ownership

- `make test`: unit and package-level seams. Keep this fast.
- `make smoke`: one representative test per critical service chain plus the
  scenario smoke manifest.
- `make check`: full local confidence before PR update.

When adding a new runtime feature, prefer extending one canonical service-chain
suite instead of creating another bespoke bootstrap stack.

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
- Each live capture, including failed runs that reached the execution path,
  writes sibling `.summary.json` and `.diagnostics.json` artifacts with the
  structured failure summary, quality-metric status, tool/reference counts,
  token usage, and the related raw/markdown artifact names.
- The committed replay fixture is updated only when
  `INTEGRATION_AI_WRITE_FIXTURE=1` is set.
- Failed live runs do not overwrite the committed fixture.

For the current checked-in Daggerheart mechanics comparison table built from
those summaries, see
[daggerheart-live-mechanics-matrix.md](../reference/daggerheart-live-mechanics-matrix.md).

## Promptfoo evaluation lane

Promptfoo now has a non-gating phase-2 evaluation lane for comparing live AI GM behavior
across models and instruction profiles without replacing the repo-owned Go
orchestration harness.

Run the fast core comparison with:

```sh
INTEGRATION_OPENAI_API_KEY=... make ai-eval-promptfoo-core
```

Run the deeper decision matrix with:

```sh
INTEGRATION_OPENAI_API_KEY=... make ai-eval-promptfoo-decision
```

To inspect recent Promptfoo runs in the local web UI:

```sh
make ai-eval-promptfoo-view
```

If the default Promptfoo viewer port is already occupied, choose another one:

```sh
make ai-eval-promptfoo-view PROMPTFOO_VIEW_PORT=15501 PROMPTFOO_VIEW_ARGS="--no"
```

Notes:

- This evaluation lane is **not** part of `make check`.
- It uses the live AI capture lane through `cmd/aieval`, then emits
  Promptfoo-friendly JSON for matrix comparison and report generation.
- `make ai-eval-promptfoo-core` runs the default `core` scenario set once per
  case for quick engineering iteration.
- `make ai-eval-promptfoo-decision` runs the same `core` set with three repeats
  per case for model or prompt-profile comparison.
- The `core` set focuses on mechanics-fidelity scenarios such as Hope spend +
  experience use, stance capability checks, narrator authority, and subdue
  intent. The `extended` set covers playbook/reference and spotlight-board
  lanes.
- Use `PROMPTFOO_ARGS='...'` to pass filters or output options through to the
  underlying `promptfoo eval`.
- The wrapper uses `promptfoo@latest` by default. Set `PROMPTFOO_NPX_SPEC` if
  you need to force a specific Promptfoo package version for one run.
- Promptfoo persistence is routed to `.tmp/promptfoo-home/` by default so the
  local database, logs, and `view` state stay in a writable repo-local path.
  Override `PROMPTFOO_CONFIG_DIR` only when you intentionally want a different
  Promptfoo home.
- `make ai-eval-promptfoo-view` runs `npx promptfoo@latest view` so you can
  inspect recent eval results, failed assertions, and per-case output details
  in the Promptfoo UI. When Promptfoo does not persist a fresh headless eval on
  its own, the repo wrapper synthesizes `results.json` from captured provider
  case outputs and imports that eval into Promptfoo so the viewer still has a
  fresh local record to open.
- Set `PROMPTFOO_VIEW_PORT` when `15500` is already in use. Use
  `PROMPTFOO_VIEW_ARGS="--no"` when you want the server to start without
  attempting to open a browser.
- Each run writes a stable artifact bundle under `.tmp/promptfoo/<run-id>/`
  with `results.json`, `scorecard.md`, per-case provider captures under
  `cases/`, and any captured harness logs.
- Each Promptfoo case is isolated with a stable case id so concurrent
  model/prompt/repeat runs do not overwrite one another's eval JSON or live
  capture artifacts.
- Promptfoo failures are intentionally compact in the report. Raw `go test`
  stderr/stdout is preserved in artifact logs instead of being embedded inline
  in the Promptfoo error field, while structured live `.diagnostics.json`
  artifacts carry the useful failure description.
- Promptfoo scorecards separate **quality failures** from **invalid runtime
  runs**. Invalid runs stay visible in the report, but they do not count
  against the model-quality pass rate.
- Promptfoo is the comparison/reporting layer only. The live Go harness remains
  the authoritative execution path, and replay fixtures remain the deterministic
  regression surface.

### Phase 2 status

Phase 2 is complete for local operator use:

- `make ai-eval-promptfoo-core`, `make ai-eval-promptfoo-decision`, and
  `make ai-eval-promptfoo-view` are the supported command surface.
- compact failure summaries, per-case diagnostics, and stable artifact bundles
  under `.tmp/promptfoo/<run-id>/` are expected outputs, not optional extras.
- Promptfoo remains non-gating and does not replace replay or live integration
  tests.

### What to do now

Use the existing phase-2 surface for comparison and diagnosis instead of adding
more Promptfoo plumbing for now:

- run `make ai-eval-promptfoo-core` when a model, prompt-profile, or GM-control
  change needs a fast comparison against the canonical scenario set
- run `make ai-eval-promptfoo-decision` before changing the preferred GM model
  or default instruction profile
- inspect `.tmp/promptfoo/<run-id>/scorecard.md` first, then follow artifact
  links into `.summary.json`, `.diagnostics.json`, raw captures, and harness
  logs when a row needs deeper debugging
- treat `metric_status=invalid` rows as runtime diagnostics to fix or rerun,
  not as model-quality evidence

Defer new eval ladders, critique mode, and broader vendor expansion to a later
phase-3 effort.

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
