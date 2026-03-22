---
id: "playbook-action-roll-and-outcomes"
title: "Action Rolls and Outcome Consequences"
kind: "playbook"
aliases: ["action roll", "duality outcome", "hope fear outcome"]
---

# Action Rolls and Outcome Consequences

## Query Use

Use this when the GM needs to resolve a standard Daggerheart action roll and
carry the outcome into authoritative state.

## When To Use

- A character is attempting a risky action with meaningful stakes.
- The turn is not a full attack, reaction, group action, or tag-team
  procedure.
- The GM needs authoritative Hope, Fear, Stress, or complication updates.

## Required Reads

- Read the acting character with `character_sheet_read` when the move depends
  on a named trait, item, armor interaction, domain card, class feature, or
  subclass feature.
- Read the exact rule text with `system_reference_search` and
  `system_reference_read` when the move name or consequence wording matters.

## Tool Sequence

1. Confirm the current capability with `character_sheet_read`.
2. Resolve the authoritative procedure with `daggerheart_action_roll_resolve`.
3. Use the returned `action_roll` and `roll_outcome` data to frame the next GM
   beat or review resolution.

## Character Sheet Fields That Matter

- `daggerheart.traits`
- `daggerheart.resources.hope`
- `daggerheart.resources.stress`
- `daggerheart.equipment`
- `daggerheart.domain_cards`
- `daggerheart.active_class_features`
- `daggerheart.active_subclass_features`
- `daggerheart.conditions`

## Consequence Heuristics

- Hope results usually create momentum or recovery.
- Fear results usually add pressure, complication, or changed GM authority.
- Critical results often improve both momentum and the fiction-facing effect.
- If the returned outcome says a complication is required, the next GM beat
  should make that cost explicit instead of silently hand-waving it away.

## Scenario Examples

- `internal/test/game/scenarios/systems/daggerheart/action_roll_outcomes.lua`
- `internal/test/game/scenarios/systems/daggerheart/action_roll_critical_success.lua`
- `internal/test/game/scenarios/systems/daggerheart/action_roll_failure_with_hope.lua`

## Common Failure Modes

- Calling a combat flow tool for a plain action roll.
- Skipping `character_sheet_read` before invoking a named card or feature.
- Narrating a Fear complication without committing the authoritative outcome
  first.
