# Scenario DSL Dependencies

This document records planned scenario DSL work inferred from scenario comments and the dependencies that are not yet met. It is meant to guide implementation order and clarify which scenarios are blocked by which missing primitives.

## Missing DSL Primitives

### Outcome Assertions
Ability to assert roll outcomes for action, reaction, attack, adversary attack, and critical results.

Blocked scenarios:
- `internal/test/game/scenarios/action_roll_outcomes.lua`
- `internal/test/game/scenarios/critical_damage.lua`
- `internal/test/game/scenarios/gm_move_severity.lua`
- `internal/test/game/scenarios/adversary_spotlight.lua`

### Resource Delta Assertions
Ability to assert deltas for hope, fear, stress, HP, armor, and similar resources.

Blocked scenarios:
- `internal/test/game/scenarios/help_and_resistance.lua`
- `internal/test/game/scenarios/group_action.lua`
- `internal/test/game/scenarios/action_roll_outcomes.lua`
- `internal/test/game/scenarios/armor_mitigation.lua`
- `internal/test/game/scenarios/armor_depletion.lua`
- `internal/test/game/scenarios/gm_move_severity.lua`
- `internal/test/game/scenarios/fear_floor.lua`

### Damage Assertions
Ability to assert damage totals, armor spend, severity tiers, and critical bonuses.

Blocked scenarios:
- `internal/test/game/scenarios/combined_damage_sources.lua`
- `internal/test/game/scenarios/armor_mitigation.lua`
- `internal/test/game/scenarios/armor_depletion.lua`
- `internal/test/game/scenarios/adversary_attack_advantage.lua`
- `internal/test/game/scenarios/critical_damage.lua`
- `internal/test/game/scenarios/adversary_spotlight.lua`

### Per-Target Damage Assertions
Ability to assert damage outcomes for each target in a multi-target attack.

Blocked scenarios:
- `internal/test/game/scenarios/multi_target_attack.lua`

### Spotlight and Fear Flow Assertions
Ability to assert spotlight ownership changes, fear pool increases/spends, and fear floor behavior.

Blocked scenarios:
- `internal/test/game/scenarios/adversary_spotlight.lua`
- `internal/test/game/scenarios/gm_move_severity.lua`
- `internal/test/game/scenarios/fear_floor.lua`

### Recovery and Lifecycle Assertions
Ability to assert recovery outcomes for rest/downtime and death-move consequences.

Blocked scenarios:
- `internal/test/game/scenarios/rest_and_downtime.lua`
- `internal/test/game/scenarios/death_move.lua`

### Tag Team Outcome Selection
Ability to assert the chosen final roller and the outcome selection in tag-team actions.

Blocked scenarios:
- `internal/test/game/scenarios/tag_team.lua`

## Recommended Implementation Order

1. Outcome assertions
2. Resource delta assertions
3. Damage assertions
4. Per-target damage assertions
5. Spotlight and fear flow assertions
6. Recovery and lifecycle assertions
7. Tag team outcome selection
