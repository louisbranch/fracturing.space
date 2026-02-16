# Daggerheart Scenario Outcome Mechanics Exec Plan

This file is the living execution plan for mechanics implementation that are now feasible with the current scenario runner and domain services.

## Purpose / Big Picture

Unblock scenario fixture coverage by implementing mechanics that are already supported by existing services and event flow:
- deterministic roll fixture control (`total` on rolls),
- branching behavior after action outcomes,
- branching behavior after reaction outcomes.

These mechanics directly reduce placeholder ambiguity in scenario fixtures without requiring new service bindings.

## Progress

- [x] (2026-02-16 20:10Z) Add deterministic roll-seed inputs using an exact `total` field in `chooseActionSeed` and cache behavior; updated tests in `internal/tools/scenario/runner_helpers_test.go`.
- [x] (2026-02-16 20:12Z) Capture action roll metadata by `roll_seq` in `internal/tools/scenario/runner_types.go`, `internal/tools/scenario/runner.go`, and `internal/tools/scenario/runner_steps.go`.
- [x] (2026-02-16 20:15Z) Implement reusable outcome-branch parsing/evaluation helpers and branch execution in `internal/tools/scenario/runner_helpers.go`.
- [x] (2026-02-16 20:18Z) Wire branch execution into `runApplyRollOutcomeStep` and `runApplyReactionOutcomeStep`; add regression tests in `internal/tools/scenario/runner_steps_test.go`.
- [x] (2026-02-16 20:20Z) Add helper coverage for branch parsing/evaluation (`parseOutcomeBranchSteps`, `resolveOutcomeBranches`, action/reaction evaluators).
- [ ] (pending) Run focused scenario/integration checks and mark concrete fixtures as cleared.
- [ ] (pending) Update remaining scenario placeholder list if newly unblocked mechanics change risk classification.

## Surprises & Discoveries

- No DSL binding gaps were found in `internal/tools/scenario/dsl.go`; argument validation is permissive enough that new `on_*` behavior fields can be interpreted at step execution time.
- `session_reaction_outcome` responses include structured result data, which enables deterministic branching directly from the outcome object.
- `action_roll` and `reaction_roll` can be made deterministic by `total`, but this is distinct from `seed` behavior (it still resolves by replay seed search to match target constraints).

## Decision Log

- Decision: Introduce `rollOutcomes` in `scenarioState` keyed by `roll_seq`.
  - Rationale: outcome branches need stable metadata after the roll step.
  - Date/Author: 2026-02-16 / Codex
- Decision: Add both action and reaction branch evaluators with shared execution helper.
  - Rationale: mechanics are structurally similar and should share parse/ordering behavior.
  - Date/Author: 2026-02-16 / Codex
- Decision: Include both `on_critical` and alias `on_crit`.
  - Rationale: keeps compatibility with fixture naming styles while preserving a canonical branch family.
  - Date/Author: 2026-02-16 / Codex

## Outcomes & Retrospective

- Outcome-driven steps now support executable branches for at least `apply_roll_outcome` and `apply_reaction_outcome`.
- Deterministic fixture reproduction can now target both roll class (`outcome`) and exact `total`.
- Remaining risk is semantics scope: which branch names are expected by the fixture corpus.

## Context and Orientation

- Primary files:
  - `internal/tools/scenario/runner.go`
  - `internal/tools/scenario/runner_types.go`
  - `internal/tools/scenario/runner_helpers.go`
  - `internal/tools/scenario/runner_steps.go`
  - `internal/tools/scenario/runner_helpers_test.go`
  - `internal/tools/scenario/runner_steps_test.go`
- Supporting docs:
  - `docs/project/scenario-dsl-dependencies.md`
  - `docs/project/scenario-missing-mechanics.md`

## Plan of Work

1. Execute deterministic seed control for action/reaction rolls (`total` + optional `outcome` validation).
2. Implement action-roll branch dispatch with explicit outcome predicates.
3. Implement reaction-roll branch dispatch with structured result predicates.
4. Keep branch syntax permissive but predictable through allowed `on_*` keys.
5. Record acceptance and update ambiguity markers in missing-mechanics notes if unblocked scenarios are resolved.

## Concrete Steps

### 1) Deterministic Roll Seeding (`total`)
- What it will do
  - Support exact total constraints in roll steps.
  - Preserve existing `outcome` hint behavior and allow combined `outcome + total` filtering.
- Event created
  - None (deterministic fixture helper only).
- Projects it effects
  - `internal/tools/scenario/runner_helpers.go` (`chooseActionSeed`, `actionSeedKey`)
  - `internal/tools/scenario/runner.go` (`scenarioState` initialization)
- Proposed DSL
```lua
scene:action_roll{
  actor = "Aragorn",
  trait = "agility",
  difficulty = 12,
  total = 18,
}
```

### 2) Action outcome branch execution (`apply_roll_outcome`)
- What it will do
  - Execute branch steps depending on roll outcome class.
  - Execute multiple branches if multiple predicates match.
- Event created
  - `event.TypeOutcomeApplied` (or `event.TypeOutcomeRejected` on rejection)
- Projects it effects
  - `internal/tools/scenario/runner_types.go` (store `actionRollResult` metadata by `roll_seq`)
  - `internal/tools/scenario/runner_steps.go` (`runApplyRollOutcomeStep`)
  - `internal/tools/scenario/runner_helpers.go` (branch parser/evaluator/runner helpers)
- Proposed DSL
```lua
scene:apply_roll_outcome{
  on_success = {
    { kind = "countdown_update", name = "Progress", delta = -1 },
  },
  on_failure = {
    { kind = "countdown_update", name = "Consequence", delta = -1 },
  },
  on_hope = { { kind = "clear_spotlight" } },
  on_critical = { { kind = "set_spotlight", type = "gm" } },
}
```

### 3) Reaction outcome branch execution (`apply_reaction_outcome`)
- What it will do
  - Execute branch steps on reaction success/failure/hope/fear/critical outcomes.
- Event created
  - `daggerheart.EventTypeReactionResolved`
- Projects it effects
  - `internal/tools/scenario/runner_steps.go` (`runApplyReactionOutcomeStep`)
  - `internal/tools/scenario/runner_helpers.go` (reaction outcome parser/evaluator helpers)
- Proposed DSL
```lua
scene:apply_reaction_outcome{
  roll_seq = 0, -- defaults to state.lastRollSeq
  on_success = {
    { kind = "countdown_update", name = "Shield", delta = -1 },
  },
  on_failure = {
    { kind = "apply_damage", target = "Frodo", die_count = 4, die_type = 8, flat = 8 },
  },
}
```

## Validation and Acceptance

- Unit test files already updated:
  - `internal/tools/scenario/runner_helpers_test.go`
  - `internal/tools/scenario/runner_steps_test.go`
- Commands to run for acceptance:
  - `go test ./internal/tools/scenario -run TestChooseActionSeed`
  - `go test ./internal/tools/scenario -run TestRunApplyRollOutcomeStep`
  - `go test ./internal/tools/scenario -run TestRunApplyReactionOutcomeStep`
  - `make integration`
- Evidence required before task closure:
  - commands above pass and no new placeholder behavior regressions are introduced.

## Idempotence and Recovery

- Branch helpers are pure and deterministic for a fixed input map.
- Outcome metadata is keyed by `roll_seq`, so repeating `apply_*` steps with explicit `roll_seq` stays deterministic.
- If a rollback is needed, remove metadata map initialization in `RunScenario` and clear branch wiring in steps.

## Artifacts and Notes

- This plan supersedes prior mechanical ordering from earlier drafts and is aligned with current code state.
- Keep this plan updated with discovery + validation evidence as mechanics are applied to concrete placeholders.

## Interfaces and Dependencies

- `scenario` DSL methods:
  - `apply_roll_outcome`
  - `apply_reaction_outcome`
  - `action_roll`
  - `reaction_roll`
- Event/event-type contracts:
  - `event.TypeRollResolved`
  - `event.TypeOutcomeApplied`
  - `event.TypeOutcomeRejected`
  - `daggerheart.EventTypeReactionResolved`
- Scenario artifacts likely to be unblocked:
  - `internal/test/game/scenarios/direct_damage_reaction.lua`
  - `internal/test/game/scenarios/chase_countdown_ring.lua`
  - `internal/test/game/scenarios/environment_helms_deep_siege_collateral_damage.lua`
