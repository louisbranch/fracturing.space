# P07: Game Tools and Daggerheart-Specific AI Seams

This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.

This document must be maintained in accordance with `PLANS.md`.

## Purpose / Big Picture

Review the game-tool layer and Daggerheart-specific AI logic to determine whether system-specific behavior is isolated cleanly from generic orchestration, whether the tool catalog is maintainable, and whether new-system contributor work would be straightforward.

Primary scope:

- `internal/services/ai/orchestration/gametools`
- `internal/services/ai/orchestration/daggerheart`

## Progress

- [x] (2026-03-23 03:59Z) Reviewed tool registry, session/resource access, and Daggerheart-specific AI helpers.
- [x] (2026-03-23 04:07Z) Recorded findings and proposed target generic-vs-system boundary.
- [x] (2026-03-23 04:12Z) Validation complete: `go test ./internal/services/ai/orchestration/gametools ./internal/services/ai/orchestration/daggerheart` passed.
- [x] (2026-03-23 04:13Z) Checked AI integration target note: `go test ./internal/test/integration -run 'TestAIDirectSessionDaggerheart'` returned `[no tests to run]`, so this pass cannot currently rely on that command as meaningful coverage.

## Surprises & Discoveries

- `gametools` is materially broader than its package name implies. It is not just generic direct-session plumbing; it also owns interaction tools, artifact/reference helpers, resource routing, and a large Daggerheart-specific tool family.
- The hidden turn-progression contract found in P06 is reinforced here: interaction tools emit JSON fields such as `ai_turn_ready_for_completion` and `next_step_hint`, and the orchestration runner interprets them out-of-band.
- `internal/services/ai/orchestration/daggerheart` is already the right kind of system-specific package. The problem is that too much Daggerheart authority still lives outside it.
- The tool catalog is larger than it first appears: the production registry currently assembles 43 tools from one package-level registry path.
- The existing tests do preserve one useful contributor contract: tool descriptions and names are treated as stable AI-facing semantics, not incidental strings.

## Decision Log

- Decision: Preserve `internal/services/ai/orchestration/daggerheart` as the system-specific prompt/context seam.
  Rationale: The package already isolates Daggerheart prompt assembly better than the rest of the tool stack. Later refactors should move more Daggerheart policy toward this style rather than fold it back into generic orchestration.
  Date/Author: 2026-03-23 / Codex

- Decision: Treat `internal/services/ai/orchestration/gametools` as a shrink target, not as the permanent home for system-specific tools.
  Rationale: The current package mixes generic transport/session helpers with Daggerheart-specific read surfaces and mechanics flows. A new system should not require edits across generic registry, session, and resource-routing files.
  Date/Author: 2026-03-23 / Codex

- Decision: Align future tool-result refactors with P06 by replacing JSON completion hints with an explicit turn-progression seam.
  Rationale: `gametools` and `runner.go` currently share behavior through undocumented JSON payload fields. This is brittle for both tests and contributor onboarding.
  Date/Author: 2026-03-23 / Codex

## Outcomes & Retrospective

Findings are grouped by review goal and classified as missing best practice, anti-pattern, or refactor candidate.

### Maintainability

1. Anti-pattern: `gametools` is not actually a generic package.
   Evidence:
   - `session.go` embeds a Daggerheart client in the generic `Clients` bundle.
   - `resources.go` switches directly on `daggerheart://` resource URIs.
   - `tools.go` registers a large Daggerheart tool family inside one production registry.
   Why it matters:
   - Generic runtime helpers and system-specific behavior are changing together, which raises the cost of every refactor.
   Refactor direction:
   - Keep generic direct-session and dialer plumbing in a narrow package.
   - Move system-specific tool families and read surfaces behind explicit family builders or subpackages.

2. Missing best practice: one monolithic production registry owns too many concerns.
   Evidence:
   - `newProductionToolRegistry()` assembles the full catalog from one file.
   - The current registry contains 43 tool definitions.
   Why it matters:
   - Adding, removing, or reviewing one tool requires editing and mentally parsing a central catalog with unrelated concerns mixed together.
   Refactor direction:
   - Compose the registry from tool-family builders such as interaction, artifacts, references, and game-system families.
   - Keep the composition root responsible for assembling families, not one package-global builder.

3. Refactor candidate: resource loading for prompt-building is routed through generic URI dispatch with system-specific branches.
   Evidence:
   - `resources.go` owns direct `daggerheart://...` handling.
   Why it matters:
   - A new system currently extends the generic resource reader rather than owning its own read surface.
   Refactor direction:
   - Introduce explicit system-specific resource providers or typed read helpers instead of growing the generic switch.

### Testability

4. Anti-pattern: orchestration progression depends on undocumented JSON output fields from interaction tools.
   Evidence:
   - Interaction tool outputs include `ai_turn_ready_for_completion` and `next_step_hint`.
   - `runner.go` parses those fields from tool-result JSON to decide turn progression.
   Why it matters:
   - Tests need to reproduce a stringly typed cross-package contract rather than assert behavior through a typed seam.
   Refactor direction:
   - Define an explicit completion/progression result contract shared between tool execution and orchestration runtime.

5. Missing best practice: Daggerheart context sources re-read JSON resources and unmarshal them into local structs.
   Evidence:
   - `orchestration/daggerheart/context_sources.go` reads interaction, scene, and character-sheet resources from `gametools`, then re-parses JSON locally.
   Why it matters:
   - Prompt-context tests are coupled to incidental JSON shape instead of a smaller typed provider seam.
   Refactor direction:
   - Introduce a typed Daggerheart prompt-data provider so context sources consume domain-shaped data, not ad hoc resource payloads.

6. Missing best practice: the direct tool invocation path is still JSON round-trip heavy.
   Evidence:
   - `DirectSession.CallTool` marshals generic `any` args to JSON and handlers unmarshal again.
   Why it matters:
   - This makes tool tests harder to focus and hides tool argument contracts behind late decoding.
   Refactor direction:
   - Move toward typed argument decoders owned by each registry entry or a tighter invocation contract at tool-family boundaries.

7. Refactor candidate: the documented integration command for this seam is currently ineffective.
   Evidence:
   - `go test ./internal/test/integration -run 'TestAIDirectSessionDaggerheart'` returned `[no tests to run]`.
   Why it matters:
   - The pass cannot currently rely on a named end-to-end contract test for Daggerheart direct-session behavior.
   Refactor direction:
   - Either add a real integration contract for the Daggerheart direct-session seam or remove/update the stale validation guidance in later refactor work.

### Contributor Clarity

8. Missing best practice: new-system onboarding is spread across too many generic files.
   Evidence:
   - A new system would currently need edits in generic session client wiring, resource routing, and the central registry, not only in a system-local package.
   Why it matters:
   - Contributors cannot follow a clear “add a system here” path, which raises accidental-coupling risk.
   Refactor direction:
   - Make system onboarding require one system-specific prompt package, one system-specific tool family package, and one explicit registration point in the composition root.

9. Positive seam to preserve: Daggerheart prompt logic already has a recognizable home.
   Evidence:
   - `internal/services/ai/orchestration/daggerheart` cleanly groups Daggerheart prompt/context work.
   Why it matters:
   - This is the right contributor-facing pattern for system-specific AI behavior.
   Preservation note:
   - Later refactors should move more Daggerheart-specific policy toward this seam, not away from it.

10. Positive seam to preserve: tool descriptions are tested as part of the AI contract.
    Evidence:
    - `tools_test.go` validates catalog semantics, not only handler mechanics.
    Why it matters:
    - Contributors get immediate feedback when changing AI-facing tool descriptions or names.
    Preservation note:
    - Keep description and naming checks, but attach them to smaller family registries after the catalog split.

Target boundary after refactor:

- Generic orchestration owns tool execution plumbing, dialers, and explicit runtime contracts.
- Generic tool families own cross-system capabilities such as interaction flow or artifact/reference access only when they are truly system-agnostic.
- `orchestration/daggerheart` and future system packages own system prompt/context assembly plus system-specific tool families.
- The composition root assembles tool families explicitly instead of relying on one global default registry.

Concrete refactor slices for a later implementation batch:

1. Extract a typed turn-progression result contract shared between tool execution and `runner.go`, then delete JSON completion-hint parsing.
2. Split the production registry into family-level builders and remove the package-global monolith.
3. Move Daggerheart resource readers and tool families behind a Daggerheart-owned boundary, then shrink generic `session.go` and `resources.go`.
4. Introduce a typed Daggerheart prompt-data provider so context sources stop re-parsing generic resource JSON.
5. Add or repair an integration-level contract test for the Daggerheart direct-session seam.

Tests to add, move, or delete in the refactor phase:

- Add package tests around the typed turn-progression seam once extracted.
- Add Daggerheart-owned tests for prompt-data providers rather than asserting JSON resource parsing indirectly.
- Keep description/name contract tests, but move them next to split family registries.
- Add a real integration test covering Daggerheart direct-session execution, or delete the stale named test target from validation guidance.

Docs to update in the refactor phase:

- `docs/architecture/platform/campaign-ai-orchestration.md`
- `docs/architecture/platform/campaign-ai-agent-system.md`
- `internal/services/ai/orchestration/gametools/doc.go`
- contributor-oriented AI tool/system extension docs if the registry split changes the onboarding path

## Context and Orientation

Use:

- `docs/architecture/platform/campaign-ai-orchestration.md`
- `docs/architecture/platform/campaign-ai-agent-system.md`
- `docs/architecture/platform/campaign-ai-mechanics-quality.md`
- `internal/services/ai/orchestration/gametools/doc.go`

## Plan of Work

Inspect:

- tool registry sprawl and naming consistency
- generic tool helpers vs Daggerheart-specific logic
- tool descriptions and contributor discoverability
- session/resource access patterns
- board/mechanics helper organization
- new-system onboarding cost

## Concrete Steps

1. Inventory the tool catalog and its registration path.
2. Identify Daggerheart-specific logic that lives in generic packages or vice versa.
3. Review tests for system intent, not just helper mechanics.
4. Record likely extraction or deletion candidates.

## Validation and Acceptance

- `go test ./internal/services/ai/orchestration/gametools ./internal/services/ai/orchestration/daggerheart`
- `go test ./internal/test/integration -run 'TestAIDirectSessionDaggerheart'`

Acceptance:

- boundary between generic runtime and Daggerheart logic is explicit
- tool catalog organization is either validated or marked for cleanup
- contributor path for adding/changing a system tool is documented

## Idempotence and Recovery

- Favor a clean generic/system split over compatibility helpers that preserve current package confusion.

## Artifacts and Notes

- Tool families that should move together in a future refactor slice:
  - interaction progression tools
  - artifact/reference tools
  - Daggerheart read-surface helpers
  - Daggerheart mechanics/combat tools

## Interfaces and Dependencies

Track any proposed changes to:

- tool registry contracts
- session/resource helper APIs
- system-specific extension points
- hidden turn-progression contracts between tool outputs and `runner.go`
