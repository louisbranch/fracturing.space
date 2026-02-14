# Testing Scenarios

Game-only scenarios live under `internal/test/game/scenarios/*.lua`. Each file returns a `Scenario` instance that describes player/GM/adversary interactions at a high level. The runner (in `internal/test/game`) expands each move into gRPC calls and performs minimal auto-assertions (event types + projection deltas).

## Running scenarios

```bash
make scenario
```

## Core API

```lua
local scene = Scenario.new("example")

scene:campaign{ name = "My Campaign", system = "DAGGERHEART", gm_mode = "HUMAN" }
scene:pc("Frodo")
scene:adversary("Nazgul")
scene:start_session("Battlefield")
scene:gm_fear(6)

scene:attack{ actor = "Frodo", target = "Nazgul", trait = "instinct", difficulty = 10, damage_type = "physical" }
scene:gm_spend_fear(1):spotlight("Nazgul")
scene:apply_condition{ target = "Frodo", add = { "VULNERABLE" } }
scene:adversary_attack{ actor = "Nazgul", target = "Frodo", difficulty = 10, damage_type = "physical" }

scene:end_session()
return scene
```

## Moves

- `campaign{ name, system, gm_mode, theme }`
- `start_session(name)` / `end_session()`
- `pc(name, opts)` / `npc(name, opts)` / `prefab(name)`
- `adversary(name, opts)`
- `gm_fear(value)`
- `reaction{ actor, trait, difficulty, modifiers, outcome, seed, expect_hope_delta, expect_stress_delta, expect_target }`
- `gm_spend_fear(amount):spotlight(target)`
- `attack{ actor, target, trait, difficulty, damage_type, outcome, damage_dice, modifiers, resist_physical, resist_magic, immune_physical, immune_magic, direct, massive_damage, expect_hope_delta, expect_stress_delta, expect_target }`
- `multi_attack{ actor, targets, trait, difficulty, outcome, damage_type, damage_dice, modifiers, resist_physical, resist_magic, immune_physical, immune_magic, direct, massive_damage, expect_hope_delta, expect_stress_delta, expect_target }`
- `combined_damage{ target, damage_type, sources, source, resist_physical, resist_magic, immune_physical, immune_magic, direct, massive_damage }`
- `adversary_attack{ actor, target, difficulty, attack_modifier, advantage, disadvantage, damage_type, damage_dice, resist_physical, resist_magic, immune_physical, immune_magic, direct, massive_damage, expect_hope_delta, expect_stress_delta, expect_target }`
- `apply_condition{ target, add, remove, source }`
- `group_action{ leader, leader_trait, difficulty, supporters, leader_modifiers, outcome, expect_hope_delta, expect_stress_delta, expect_target }`
- `tag_team{ first, first_trait, second, second_trait, selected, difficulty, outcome, expect_hope_delta, expect_stress_delta, expect_target }`
- `rest{ type, party_size, interrupted, characters, expect_hope_delta, expect_stress_delta, expect_target }`
- `downtime_move{ target, move, prepare_with_group, expect_hope_delta, expect_stress_delta, expect_target }`
- `death_move{ target, move, hp_clear, stress_clear, expect_hope_delta, expect_stress_delta, expect_target }`
- `blaze_of_glory(target)`
- `swap_loadout{ target, card_id, recall_cost, in_rest }`
- `countdown_create{ name, kind, current, max, direction, looping, countdown_id }`
- `countdown_update{ name, countdown_id, delta, current, reason }`
- `countdown_delete{ name, countdown_id, reason }`
- `action_roll{ actor, trait, difficulty, modifiers, outcome, seed }`
- `reaction_roll{ actor, trait, difficulty, modifiers, outcome, seed }`
- `damage_roll{ actor, damage_dice, modifier, critical, seed }`
- `adversary_attack_roll{ actor, attack_modifier, advantage, disadvantage, seed }`
- `apply_roll_outcome{ roll_seq, target, targets }`
- `apply_attack_outcome{ roll_seq, target, targets }`
- `apply_adversary_attack_outcome{ roll_seq, targets, difficulty }`
- `apply_reaction_outcome{ roll_seq }`
- `mitigate_damage{ target, armor }` (sets armor slots)

## Auto-assertions

For each move, the runner validates:

- Relevant event types were emitted.
- Roll linkage exists when applicable.
- Projection deltas for HP/armor/GM fear/conditions changed in the expected direction.

Optional expectation keys (`expect_hope_delta`, `expect_stress_delta`) assert resource deltas for a single character. Use `expect_target` to override the default character (actor/leader/selected/target).

When `resist_*` or `immune_*` flags are set on character targets, the runner validates the `action.damage_applied` payload flags. Adversary targets apply resist/immune when calculating HP/armor updates.

Outcome hints (e.g. `outcome = "fear"`) influence the action roll seed only. Damage roll results remain deterministic but are not constrained by the hint.

Action roll modifiers can omit `value` when the source is a hope spend (`help`, `experience`, `tag_team`, `hope_feature`). This records the spend without adjusting the total modifier.

Modifier helpers are available via `Modifiers`:

```lua
scene:attack{
  actor = "Frodo",
  target = "Nazgul",
  modifiers = {
    Modifiers.hope("help"),
    Modifiers.mod("training", 2)
  }
}
```

Attack and multi-target attack steps can target adversaries. The runner applies adversary damage by updating HP/armor through the adversary update API. Conditions still target characters only.

## Scenario map

- `internal/test/game/scenarios/basic_flow.lua`
  - Campaign/session lifecycle basics.
- `internal/test/game/scenarios/adversary_spotlight.lua`
  - Spotlight flow, GM fear spend, conditions, adversary attack.
  - SRD: spotlight, GM moves, conditions, attack flow.
- `internal/test/game/scenarios/action_roll_outcomes.lua`
  - Hope/Fear/Critical outcomes on action rolls.
  - SRD: action roll outcomes, critical success.
- `internal/test/game/scenarios/condition_lifecycle.lua`
  - Apply/remove Vulnerable via GM spotlight.
  - SRD: conditions, clearing conditions, GM moves.
- `internal/test/game/scenarios/armor_mitigation.lua`
  - Armor absorption on adversary damage.
  - SRD: armor, damage thresholds.
- `internal/test/game/scenarios/gm_fear_spend_chain.lua`
  - Multiple GM fear spends in sequence.
  - SRD: Fear usage, GM moves.
- `internal/test/game/scenarios/critical_damage.lua`
  - Critical success on attack roll.
  - SRD: critical success, critical damage.
- `internal/test/game/scenarios/armor_depletion.lua`
  - Repeated adversary attacks until armor is depleted.
  - SRD: armor, damage thresholds.
- `internal/test/game/scenarios/group_action.lua`
  - Group action roll with supporters.
  - SRD: group action rolls.
- `internal/test/game/scenarios/tag_team.lua`
  - Tag team roll between two PCs.
  - SRD: tag team rolls.
- `internal/test/game/scenarios/rest_and_downtime.lua`
  - Short rest and downtime move.
  - SRD: downtime, rest rules.
- `internal/test/game/scenarios/death_move.lua`
  - Avoid Death move.
  - SRD: death moves.
- `internal/test/game/scenarios/help_and_resistance.lua`
  - Help an ally hope spend + resistance flags.
  - SRD: help an ally, resistance.
- `internal/test/game/scenarios/adversary_attack_advantage.lua`
  - Adversary attack with advantage and modifiers.
  - SRD: adversary attacks, advantage.
- `internal/test/game/scenarios/multi_target_attack.lua`
  - Single roll applied to multiple targets.
  - SRD: multi-target attacks.
- `internal/test/game/scenarios/combined_damage_sources.lua`
  - Combine damage sources before thresholds.
  - SRD: damage thresholds.
- `internal/test/game/scenarios/gm_move_severity.lua`
  - Fear outcome + GM spends for soft vs hard move examples.
  - SRD: GM moves, fear.
- `internal/test/game/scenarios/condition_stacking_guard.lua`
  - Apply duplicate condition alongside a new one.
  - SRD: conditions.
- `internal/test/game/scenarios/fear_floor.lua`
  - Spend GM fear down to zero.
  - SRD: fear floor.
- `internal/test/game/scenarios/adversary_spotlight_chain.lua`
  - Multiple fear spends to spotlight adversaries.
  - SRD: spotlight, fear spend.
- `internal/test/game/scenarios/reaction_flow.lua`
  - Reaction flow, roll outcomes, and reaction resolution.
  - SRD: reaction rolls.
- `internal/test/game/scenarios/countdown_lifecycle.lua`
  - Countdown creation, update, and deletion.
  - SRD: countdowns.
- `internal/test/game/scenarios/blaze_of_glory.lua`
  - Death move blaze of glory resolution.
  - SRD: death moves, blaze of glory.
- `internal/test/game/scenarios/loadout_swap.lua`
  - Loadout swap with recall cost.
  - SRD: loadout swaps.
- `internal/test/game/scenarios/low_level_rolls.lua`
  - Low-level roll + outcome APIs.
  - SRD: rolls, outcomes, and adversary attacks.

## Additional scenarios (ready now)

See scenario map.

## Minor DSL additions (next tier)

- Player advantage/disadvantage once action roll APIs expose fields (SessionAttackFlowRequest lacks advantage/disadvantage).

## Larger DSL or API additions

None.
