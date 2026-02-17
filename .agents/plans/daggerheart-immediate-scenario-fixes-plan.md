# Daggerheart Immediate Scenario Fix Plan

## Purpose

Unblock a small set of scenario fixtures that are currently blocked only by missing DSL usage, not by engine capabilities.

Current state constraints:
- Outcome branches are already wired for `apply_roll_outcome` and `apply_reaction_outcome`.
- Roll determinism already supports `total` and `outcome` hints in `action_roll` / `reaction_roll`.
- No new runtime services or domain commands are required for these two selected mechanics.

## Selected Mechanics (Current State Feasible)

### 1) Direct-damage reaction result control

- Goal: force a reaction roll to a specific total for deterministic fixture setup.
- Event: `event.TypeRollResolved` from `reaction_roll` plus `event.TypeReactionResolved` from `apply_reaction_outcome`.
- Projectors affected: actor reaction roll projection for the spotlighted character.
- Proposed DSL:
  - `scene:reaction_roll{ actor = "...", trait = "...", difficulty = ..., total = 19, outcome = "hope" }`
  - `scene:apply_reaction_outcome{}`

### 2) Chase outcome-to-countdown advancement

- Goal: connect action-roll outcomes to progress/consequence countdown updates.
- Event(s):
  - `event.TypeRollResolved` from `action_roll`
  - `event.TypeOutcomeApplied` from `apply_roll_outcome` (or rejected)
  - `daggerheart.EventTypeCountdownUpdated` from nested `countdown_update` branches
- Projectors affected:
  - action roll outcome state
  - countdown state
- Proposed DSL:
  - `scene:apply_roll_outcome{ on_success = { { kind = "countdown_update", name = "PC Progress", delta = 1 } }, on_failure = { { kind = "countdown_update", name = "Thief Escape", delta = 1 } } }`

### 3) Helms Deep collateral-damage outcomes (blocked for now)

- Goal: apply direct-damage / stress outcomes after a reaction outcome in `environment_helms_deep_siege_collateral_damage.lua`.
- Status: currently blocked.
- Blocker: no existing DSL step in this runner to apply reaction-derived stress or direct damage to characters in place at this layer.
- Recommendation: keep scenario marked “unclear / requires DSL extension” until a `apply_direct_damage` + character stress mutation step exists.

## Task List

- [ ] Update `internal/test/game/scenarios/direct_damage_reaction.lua` to remove the missing-mechanics marker and use explicit deterministic `total`.
- [ ] Update `internal/test/game/scenarios/chase_countdown_ring.lua` to use `apply_roll_outcome` branches and remove both missing-mechanics markers.
- [ ] Keep `internal/test/game/scenarios/environment_helms_deep_siege_collateral_damage.lua` placeholder comment until new damage/stress DSL steps are added.
- [ ] Update `docs/project/scenario-missing-mechanics.md` to remove now-implemented mechanics from the “Scenario fixture placeholders requiring clarification” list and keep collateral damage in `Unclear / blocked`.
- [ ] Optional follow-up: add explicit DSL for reaction-result damage + stress mutation and retire the collateral placeholder when available.

## Out of Scope (for this batch)

- New scenario DSL commands for direct damage application from reaction outcomes.
- Combat damage model or stress mutation API changes.
- Bulk removal of unrelated placeholder comments.

## Evidence / Notes

- This batch only changes scenario fixture scripts and the missing-mechanics ledger.
- No production game logic or protocol changes are expected in this pass.
