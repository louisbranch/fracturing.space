# Scenario DSL Coverage and Mechanics Inventory ExecPlan

This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.

This plan must be maintained in accordance with `PLANS.md`.

## Purpose / Big Picture

After this work, every scenario in `internal/test/game/scenarios/*.lua` can be executed without missing DSL bindings, and the team has a clear, documented inventory of missing game mechanics categorized as general or Daggerheart-specific. The results are visible by running the scenario test suite and by reading the two documentation files that list the remaining mechanical gaps and the DSL gaps.

## Progress

- [x] (2026-02-12 18:35Z) Create `plans/scenario-dsl-exec.md` and record this plan.
- [x] (2026-02-12 18:41Z) Inventory DSL usage vs bindings; update `docs/project/scenario-dsl-dependencies.md`.
- [x] (2026-02-12 18:54Z) Inventory missing mechanics and document them in `docs/project/scenario-missing-mechanics.md`.
- [x] (2026-02-12 18:42Z) Implement missing DSL (tests first) and update scenarios to use it.
- [ ] (YYYY-MM-DD HH:MMZ) Validate with scenario tests, unit tests, coverage, and integration.
- [ ] (2026-02-12 18:53Z) Restore or replace scenario comment validation after the mechanics work.

## Surprises & Discoveries

- Observation: Scenario suite fails before execution because many scenario blocks lack leading comments, triggering the comment-first validation.
  Evidence: `scenario block missing comment at .../abstracted_presence_roll.lua:11` and automated scan reported 161 files with missing comment blocks.

## Decision Log

- Decision: Maintain two docs: DSL gaps and mechanics gaps.
  Rationale: DSL and mechanics evolve at different speeds; separating keeps each list focused and actionable.
  Date/Author: 2026-02-12 / OpenCode

## Outcomes & Retrospective

(To be filled after milestones complete.)

## Context and Orientation

Scenario scripts live in `internal/test/game/scenarios/*.lua`. The Lua DSL is defined in two parallel bindings that must remain in lockstep:
- `internal/tools/scenario/dsl.go`
- `internal/test/game/lua_binding_test.go` (build tag `scenario`)

Scenario execution is implemented in:
- `internal/tools/scenario/runner_steps.go`
- `internal/test/game/runner_test.go`

There is an existing, outdated DSL gap list in `docs/project/scenario-dsl-dependencies.md`. This plan will refresh that list and add a new, mechanics-focused doc at `docs/project/scenario-missing-mechanics.md`.

"Missing DSL" means a scenario attempts to call a `scene:...` function that is not bound in the DSL layer or not supported by the runner. "Missing mechanics" means a scenario can call the DSL but the underlying game system behavior is not implemented (e.g., the service rejects it or produces incorrect outcomes).

## Plan of Work

1) Inventory all DSL calls used by scenarios and compare them to the DSL bindings in `internal/tools/scenario/dsl.go` and `internal/test/game/lua_binding_test.go`. Update `docs/project/scenario-dsl-dependencies.md` to list every missing binding and the scenarios that need it.

2) Identify missing mechanics by running the scenario suite and classifying failures. For each failure that is not a DSL binding issue, record the mechanic, the scenario(s), and a plain-language description of what behavior is missing.

3) Create `docs/project/scenario-missing-mechanics.md` to document the mechanics gaps. Each entry must state whether the mechanic appears general or Daggerheart-specific.

4) For each missing DSL binding identified in step 1:
   - Write a failing test first (TDD).
   - Implement the binding in both DSL files.
   - If the runner lacks the step kind, add it and test it.
   - Update scenarios to use the new DSL and remove placeholders.

5) Validate end-to-end by running tests and ensuring the scenario suite passes without missing DSL calls. Report coverage with `make cover`.

6) Revisit scenario comment validation and restore or replace it after the DSL/mechanics work is complete.

## Concrete Steps

Working directory: repository root.

Discovery commands (read-only):
- Enumerate scenario DSL calls:
  - `rg -n "scene:[a-zA-Z_]+" internal/test/game/scenarios/*.lua`
- Enumerate DSL bindings:
  - `rg -n "scenarioMethods" -n internal/tools/scenario/dsl.go`
  - `rg -n "scenarioMethods" -n internal/test/game/lua_binding_test.go`
- Run scenario tests to expose missing mechanics:
  - `go test ./internal/test/game -tags=scenario -run TestScenarioScripts`

When updating docs:
- `docs/project/scenario-dsl-dependencies.md` should contain only DSL gaps (missing bindings, missing step kinds).
- `docs/project/scenario-missing-mechanics.md` should contain only mechanics gaps (general vs Daggerheart-specific).

When implementing DSL:
- Follow TDD: add a failing test, then implement the binding, then refactor.
- Add or update scenario scripts to use the new DSL.
- Keep DSL bindings in both files in sync.

## Validation and Acceptance

Acceptance criteria:
- All scenario scripts execute without missing DSL errors.
- `docs/project/scenario-dsl-dependencies.md` accurately reflects only remaining DSL gaps.
- `docs/project/scenario-missing-mechanics.md` lists all missing mechanics with general vs Daggerheart-specific labels.

Required commands after implementation:
- `make test`
- `make cover` (report coverage impact)
- `make integration`
- `go test ./internal/test/game -tags=scenario -run TestScenarioScripts`

## Idempotence and Recovery

All discovery steps are read-only. If a test fails mid-run, fix the blocking DSL or mechanic and rerun the same commands. For documentation updates, ensure entries include scenario references so they are easy to verify and re-check later.

## Artifacts and Notes

Include short, representative outputs in the plan as evidence when steps are completed, such as:
- A small excerpt of a failing scenario test indicating the missing DSL or mechanic.
- A diff snippet showing a new DSL binding added.

Representative output:
- `load scenario .../internal/test/game/scenarios/abstracted_presence_roll.lua: scenario block missing comment at .../abstracted_presence_roll.lua:11`

## Interfaces and Dependencies

When adding new DSL bindings:
- `internal/tools/scenario/dsl.go` must export a Lua method with a stable name and append a `Step{Kind: ..., Args: ...}`.
- `internal/test/game/lua_binding_test.go` must mirror the same method name and step behavior.
- If a new step kind is used, implement it in:
  - `internal/tools/scenario/runner_steps.go`
  - `internal/test/game/runner_test.go`

Decision note:
When a missing mechanic is discovered, document it first in `docs/project/scenario-missing-mechanics.md` even if implementation will follow later, to keep the inventory accurate.
