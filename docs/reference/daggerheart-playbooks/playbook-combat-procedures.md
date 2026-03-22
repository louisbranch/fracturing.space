---
id: "playbook-combat-procedures"
title: "Combat Procedures"
kind: "playbook"
aliases: ["attack flow", "reaction flow", "group action", "tag team", "combat procedure"]
---

# Combat Procedures

## Query Use

Use this when the table is in a structured combat procedure and the model
should prefer one authoritative high-level tool over stitching together raw
rolls.

## When To Use

- A player is making a full attack that should carry through damage.
- An adversary is attacking and the board state should update authoritatively.
- The group is using supporter, tag-team, or reaction procedures.
- Armor reactions or spotlight-sensitive threat actions make the sequence easy
  to misorder by hand.

## Required Reads

- Read the acting character with `character_sheet_read` before invoking a
  character-specific attack or reaction.
- Read the current threat board with `daggerheart_combat_board_read` before an
  adversary attack or spotlight-sensitive attack.
- Read exact rule wording with `system_reference_search` and
  `system_reference_read` only when the procedure choice is unclear.

## Procedure Map

- Use `daggerheart_attack_flow_resolve` for a player attack that should include
  roll, attack outcome, damage roll, and damage application.
- Use `daggerheart_adversary_attack_flow_resolve` for an adversary attack that
  should include attack roll, outcome, and damage application.
- Use `daggerheart_reaction_flow_resolve` for true Daggerheart reaction
  procedures rather than a normal action roll.
- Use `daggerheart_group_action_flow_resolve` when supporters contribute to a
  leader's result.
- Use `daggerheart_tag_team_flow_resolve` when two characters roll and one
  combined outcome is selected.

## Character And Board Fields That Matter

- `daggerheart.traits`
- `daggerheart.equipment.primary_weapon`
- `daggerheart.equipment.active_armor`
- `daggerheart.resources.hope`
- `daggerheart.resources.armor`
- `daggerheart.active_class_features`
- `daggerheart.active_subclass_features`
- `spotlight`
- `adversaries`

## Tool Sequences

### Player Attack

1. `character_sheet_read`
2. `daggerheart_combat_board_read` when the target is an adversary or spotlight
   matters
3. `daggerheart_attack_flow_resolve`

### Adversary Attack

1. `daggerheart_combat_board_read`
2. `character_sheet_read` for the defending character when armor or reactions
   matter
3. `daggerheart_adversary_attack_flow_resolve`

### Reaction, Group, and Tag-Team

1. `character_sheet_read` for the involved characters
2. The matching high-level procedure tool
3. Use the returned summaries to frame the next prompt or review result

## Scenario Examples

- `internal/test/game/scenarios/systems/daggerheart/reaction_flow.lua`
- `internal/test/game/scenarios/systems/daggerheart/group_action.lua`
- `internal/test/game/scenarios/systems/daggerheart/tag_team.lua`
- `internal/test/game/scenarios/systems/daggerheart/full_example_spotlight_sequence.lua`

## Common Failure Modes

- Using a low-level roll tool when a canonical combat procedure already exists.
- Failing to inspect the board before an adversary or spotlight-sensitive flow.
- Resolving a reaction as a normal action roll instead of the reaction
  procedure.
