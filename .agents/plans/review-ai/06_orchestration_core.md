# P06: Orchestration Core

This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.

This document must be maintained in accordance with `PLANS.md`.

## Purpose / Big Picture

Review the orchestration core to decide whether the campaign-turn runtime is cleanly layered, policy-driven, and testable, with prompt/context/runtime behavior expressed in maintainable seams rather than ad hoc control flow.

Primary scope:

- `internal/services/ai/orchestration`

## Progress

- [x] (2026-03-23 04:53Z) Reviewed orchestration docs, package entrypoints, runner/prompt/context/observability files, and orchestration tests.
- [x] (2026-03-23 04:55Z) Verified orchestration baseline with `go test ./internal/services/ai/orchestration`.
- [x] (2026-03-23 05:03Z) Synthesized orchestration-core findings, runner-policy cleanup needs, and the target pipeline shape.

## Surprises & Discoveries

- The package split is mostly aligned with the docs: prompt collection, rendering, tool policy, and runner control flow are all easy to locate.
- The main orchestration debt is concentrated in `runner.go`, not spread evenly across the package.
- The test suite is strong on runner behavior and prompt rendering, but that strength partly comes from broad runner tests compensating for implicit policy in the runner itself.
- The typed session-brief seam is good and should be preserved; the main cleanup target is runtime progression policy after prompt assembly.

## Decision Log

- Decision: Keep the collector/renderer split and typed `SessionBrief` model as the orchestration foundation.
  Rationale: The current prompt path matches the architecture docs and is one of the clearer seams in the AI service. The refactor target is the runtime turn-state policy around the runner, not collapsing prompt collection and rendering back together.
  Date/Author: 2026-03-23 / Codex

## Outcomes & Retrospective

P06 is complete for planning purposes. The orchestration package already has the right major building blocks, but the runner encodes too much turn-progression policy through booleans, tool-name constants, JSON hint parsing, and reminder flags. The next clean foundation is to keep the prompt path intact while extracting explicit turn-state and completion policy seams from the runner.

## Context and Orientation

Use:

- `docs/architecture/platform/campaign-ai-orchestration.md`
- `docs/architecture/platform/campaign-ai-agent-system.md`
- `docs/architecture/platform/campaign-ai-mechanics-quality.md`
- `docs/reference/ai-service-lifecycle-terms.md`
- `internal/services/ai/orchestration/doc.go`

## Plan of Work

Inspect:

- runner state machine clarity
- timeout/step/budget policy placement
- prompt-build pipeline and typed brief use
- context-source registry ownership
- observability/span coverage
- failure semantics and reminder/retry logic
- test seam quality for sessions, providers, and prompt components

## Current Findings

### F01: The runner carries an implicit state machine through booleans, reminder flags, and tool-order counters

Category: anti-pattern, maintainability and testability risk

Evidence:

- `internal/services/ai/orchestration/runner.go:163-175` initializes a cluster of mutable runtime state:
  `committedOrResolved`, `readyForCompletion`, `commitReminderUsed`, `completionReminderUsed`, `playerPhaseReminderUsed`, `lastCommitToolOrder`, `lastPlayerHandoffToolOrder`, and `toolOrder`.
- `internal/services/ai/orchestration/runner.go:220-255` branches final-output behavior through those flags and counters.
- `internal/services/ai/orchestration/runner.go:300-315` mutates progression state indirectly from tool results.

Impact:

- The turn-completion rules are real policy, but they are encoded as ad hoc local control flow instead of an explicit runtime model.
- Contributors have to read most of `runner.go` at once to understand what makes a turn “complete”.
- Tests cover the behavior well, but many of them are broad because there is no smaller seam for progression policy.

Refactor direction:

- Extract an explicit turn-progression state/policy type from the runner.
- Give the runtime a named model for:
  - authoritative commit observed
  - player handoff observed
  - completion readiness
  - reminder eligibility
- Keep the outer loop in the runner, but move progression rules into a smaller policy object with package-local tests.

### F02: The generic runner is directly coupled to specific interaction tool names and JSON result hints

Category: anti-pattern, clarity risk

Evidence:

- `internal/services/ai/orchestration/runner.go:19-21` hard-codes `interaction_open_scene_player_phase`, `interaction_resolve_scene_player_review`, and `interaction_session_ooc_resolve`.
- `internal/services/ai/orchestration/runner.go:382-395` bakes those tool names into commit/handoff classification helpers.
- `internal/services/ai/orchestration/runner.go:427-455` parses `toolResultControlHints` out of raw tool-result JSON strings to infer readiness and player handoff.

Impact:

- The file is called `runner.go`, but it is not only a generic provider/tool loop; it also knows specific campaign interaction semantics.
- P07 tool-surface changes will have to reach back into the orchestration core instead of staying behind a narrower control-policy seam.
- New contributors must understand tool payload conventions before they can safely change turn progression.

Refactor direction:

- Move tool-result completion/handoff interpretation behind an explicit orchestration policy seam.
- Candidate shapes:
  - a `TurnProgressPolicy` interface supplied by the composition root or tool profile
  - a provider-neutral classifier over tool calls and tool results
- Keep the core runner responsible for ordering and retries, but stop hard-coding campaign interaction tool names into its generic control loop.

### F03: `NewRunner` silently degrades missing collaborators instead of making runtime policy explicit

Category: missing best practice, contributor-clarity risk

Evidence:

- `internal/services/ai/orchestration/runner.go:48-55` silently defaults `PromptBuilder` and `ToolPolicy`.
- `internal/services/ai/orchestration/prompt_builder.go:25-39` defines `newDegradedPromptBuilder()` as an explicit degraded mode with only core sources and default render policy.
- Architecture docs say AI startup chooses prompt render policy explicitly in the composition root; the runner currently hides missing config by auto-falling back.

Impact:

- Misconfiguration can look like a valid runtime instead of failing fast at construction.
- The live prompt path becomes harder to reason about because the orchestration package has both explicit composition-time policy and silent fallback behavior.
- Contributors may not realize when tests are exercising degraded behavior rather than the configured production path.

Refactor direction:

- Keep degraded builders as explicit test or fallback helpers, but stop making them an implicit default in `NewRunner`.
- Prefer constructor-time validation for required runtime collaborators.
- Let the composition root decide when degraded behavior is intentional.

### F04: `ContextSourceRegistry` has implicit ownership and last-writer-wins behavior for typed interaction state

Category: missing best practice

Evidence:

- `internal/services/ai/orchestration/context_source.go:93-118` maintains an ordered mutable registry and overwrites `brief.InteractionState` whenever a later source returns one.
- There is no source identity or conflict detection when multiple sources contribute typed state.

Impact:

- The typed brief seam is good, but the ownership rule for typed facts is implicit rather than enforced.
- A future system-specific context source could accidentally replace the authoritative interaction-state snapshot.
- Contributors have to know by convention which source is allowed to set typed state.

Refactor direction:

- Make typed-fact ownership explicit.
- Options:
  - reserve `InteractionState` for one named core source only
  - let the registry reject duplicate typed-state contributors
  - give sources stable IDs and explicit merge rules
- Preserve ordered section collection, but tighten typed-state merge semantics.

### F05: Observability is too coarse for the package’s real failure modes

Category: missing best practice, operational clarity risk

Evidence:

- `internal/services/ai/orchestration/runner.go:73`, `:146`, and `:177` create spans for run, prompt build, and provider steps.
- Tool execution inside `sess.CallTool(...)` and context-source collection inside `CollectBrief(...)` do not emit their own spans.
- `internal/services/ai/orchestration/observability.go:13-19` only records error/status on existing spans.

Impact:

- Production debugging can tell that orchestration failed, but not quickly which context source or tool call dominated latency or broke the turn.
- Reminder loops and tool-result truncation emit some events, but the most expensive operations still lack per-step attribution.
- Cross-service issues with game resource reads and tool execution are harder to localize.

Refactor direction:

- Add source-level and tool-call-level observability.
- At minimum:
  - one span or event per context source collection
  - one span around each tool call with tool name and error/truncation metadata
  - clearer run-level attributes for reminder path selection and completion state

### F06: One orchestration seam is already healthy and should be preserved

Category: best practice already employed, preserve as-is

Evidence:

- `internal/services/ai/orchestration/context_source.go:55-72` and `prompt_builder.go:61-67` preserve the typed `SessionBrief` seam between collection and rendering.
- `internal/services/ai/orchestration/prompt_rendering.go:53-106` keeps rendering policy separate from resource collection.
- Package-local tests cover prompt assembly and brief budgeting directly.

Impact:

- Prompt behavior is easier to test than it was in string-reparse designs.
- The package already has one clean architectural seam worth keeping through refactors.

Refactor direction:

- Preserve the collector/renderer split.
- Focus cleanup on runtime progression policy and context-source ownership, not on collapsing prompt assembly back into one path.

## Concrete Steps

1. Map the orchestration flow from input validation through provider/tool loop completion.
2. Identify where policy is encoded in condition chains rather than explicit types.
3. Compare prompt-building and context-source boundaries to the architecture docs.
4. Record refactor slices that would reduce orchestration complexity without weakening safety.

## Target Orchestration Shape

Keep the existing top-level package responsibilities:

- typed brief collection
- prompt rendering
- tool policy filtering
- runner execution
- observability helpers

Refactor the runner side into smaller explicit policy seams:

1. Preserve the prompt path exactly as a collector -> renderer pipeline.
   - `SessionBriefCollector` and `PromptRenderer` remain separate
2. Extract turn progression into an explicit policy/state component.
   - commit detection
   - handoff detection
   - completion readiness
   - reminder transitions
3. Separate generic loop mechanics from campaign-specific completion semantics.
   - the runner owns step ordering, provider calls, tool execution, and result passing
   - a narrower policy seam owns “what counts as complete”
4. Tighten context-source ownership.
   - ordered section contributions remain fine
   - typed-state merges should become explicit and auditable
5. Improve observability at the real work seams.
   - context-source collection
   - provider step execution
   - tool call execution
   - reminder/retry path decisions

## Validation and Acceptance

- `go test ./internal/services/ai/orchestration`
- `go test ./internal/services/ai/...`

Acceptance:

- orchestration responsibilities are clearly separated
- policy-bearing interfaces are named explicitly
- any breaking changes to runner/prompt/context APIs are recorded

## Idempotence and Recovery

- If P07 or P08 finds system/context leakage into orchestration, update this file to absorb the new boundary decision.

## Artifacts and Notes

- Record any control-flow clusters that should become explicit policy/state types.
- P07 should align with any extracted completion-policy seam instead of adding more tool-name conditionals back into `runner.go`.

## Interfaces and Dependencies

Track any proposed changes to:

- `CampaignTurnRunner`
- prompt builder / renderer contracts
- session/context-source interfaces
- tool-result completion signal contracts

## Cutover Order

1. Stop implicit degraded defaults in `NewRunner`; make intentional degraded behavior explicit at construction time.
2. Extract turn-progression state from the runner without changing the outer `CampaignTurnRunner` interface yet.
3. Move tool-name and tool-result completion semantics behind a narrower orchestration policy seam.
4. Tighten `ContextSourceRegistry` typed-state ownership once the progression policy is smaller and easier to reason about.
5. Add source-level and tool-call-level observability after the control-flow seams are explicit.
