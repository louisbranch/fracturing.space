---
id: "playbook-gm-fear-adversaries-and-spotlight"
title: "GM Fear, Adversaries, and Spotlight"
kind: "playbook"
aliases: ["gm fear", "adversary spotlight", "fear spend", "spotlight combat"]
---

# GM Fear, Adversaries, and Spotlight

## Query Use

Use this when the GM needs to spend Fear, place or inspect threats, or move
combat pressure through spotlight.

## When To Use

- Fear is being spent through a direct move, adversary feature, environment
  feature, or adversary experience.
- A new adversary should be placed on the current scene board.
- Ongoing pressure should be tracked openly as a visible countdown.
- The GM needs to understand which threats are active and who currently owns
  spotlight.

## Required Reads

- Read the current board with `daggerheart_combat_board_read` before spending
  Fear or framing spotlight-sensitive narration.
- Do not consult this playbook until Fear, spotlight, or visible board pressure
  is actually relevant on the turn.
- If the board reports `NO_ACTIVE_SCENE`, diagnose with `interaction_state_read`
  and correct the active scene before improvising new threats.
- Read exact rule wording with `system_reference_search` and
  `system_reference_read` only when the move taxonomy or spotlight procedure needs
  confirmation.

## Tool Sequence

1. Inspect the current board with `daggerheart_combat_board_read`.
2. Place threats with `daggerheart_adversary_create` if the fiction requires a
   new adversary.
3. Update scene-specific threat notes with `daggerheart_adversary_update` when
   the board should reflect a changed threat posture.
4. Create or advance open pressure clocks with `daggerheart_scene_countdown_create`
   and `daggerheart_scene_countdown_advance`.
5. If a countdown reaches `TRIGGER_PENDING`, resolve that pending trigger with
   `daggerheart_scene_countdown_resolve_trigger` before narrating the next beat
   that depends on its looped or cleared state.
6. Spend Fear with `daggerheart_gm_move_apply` when the spend target is known.
7. Re-read the board if the next beat depends on updated spotlight, Fear,
   countdown, or adversary state.
8. If the board shows `TRIGGER_PENDING`, resolve that countdown trigger before
   narrating as though the looped or reset state is already visible.

## Board Fields That Matter

- `gm_fear`
- `spotlight`
- `scene_id`
- `countdowns`
- `adversaries[].id`
- `adversaries[].scene_id`
- `adversaries[].features`
- `adversaries[].spotlight_count`
- `adversaries[].spotlight_gate_id`

## GM Move Heuristics

- Spend Fear to change the situation, not just to tax numbers.
- Prefer named adversary or environment features before improvising a looser
  move.
- Keep spotlight movement legible to the players after each meaningful threat
  activation.

## Scenario Examples

- `internal/test/game/scenarios/systems/daggerheart/gm_fear_adversary_feature.lua`
- `internal/test/game/scenarios/systems/daggerheart/adversary_spotlight.lua`
- `internal/test/game/scenarios/systems/daggerheart/full_example_spotlight_sequence.lua`
- `internal/test/game/scenarios/systems/daggerheart/gm_fear_spend_chain.lua`
- `internal/test/game/scenarios/systems/daggerheart/countdown_lifecycle.lua`
- `internal/test/game/scenarios/systems/daggerheart/countdown_variants.lua`
- `internal/test/game/scenarios/systems/daggerheart/countdown_linked_pair.lua`
- `internal/test/game/scenarios/systems/daggerheart/gm_move_artifact_chase.lua`
- `internal/test/game/scenarios/systems/daggerheart/chase_countdown_ring.lua`
- `internal/test/game/scenarios/systems/daggerheart/progress_countdown_climb.lua`

## Common Failure Modes

- Spending Fear without checking the current board or available feature state.
- Inventing a threat that should have been created as an adversary.
- Leaving a countdown in `TRIGGER_PENDING` and narrating as if the reset or loop
  behavior already happened.
- Forgetting to reflect spotlight ownership in the next narrated prompt.
