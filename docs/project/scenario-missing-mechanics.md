---
title: "Scenario Missing Mechanics"
parent: "Project"
nav_order: 22
---

# Scenario Missing Mechanics

This document tracks mechanics gaps discovered while running the scenario suite. It focuses on game-system behavior that is missing or incorrect even when the DSL bindings exist.

## Status

Mechanics inventory reflects the first full scenario run after disabling comment validation.

Last reconciled against current `-- Missing Mechanic` markers on 2026-02-16.

SRD annotations added 2026-02-17 against Daggerheart SRD 1.0 (September 2025).

CRB annotations updated 2026-02-17 against Daggerheart Core Rulebook (May 2025, 415 pages).

## Event Timeline Contract Requirement

Before implementing or adjusting any missing mechanic in this document, define its
evented write-path mapping first.

Required mapping:

`mechanic -> command type(s) -> emitted event type(s) -> projection targets -> invariants`

For Daggerheart mechanics, use:
[Daggerheart Event Timeline Contract](daggerheart-event-timeline-contract.md).
Priority backlog mappings live under
[Priority Missing-Mechanic Timeline Mappings](daggerheart-event-timeline-contract.md#priority-missing-mechanic-timeline-mappings)
with row IDs (`P1`, `P2`, ...).

Review gate:

1. No implementation PR should close a mechanic gap without a corresponding timeline entry/update.
2. If command/event mapping is unclear, resolve that ambiguity in docs before code.

## Priority Timeline Mapping Index

Use this index to connect scenario gaps to timeline-contract row IDs before
writing implementation code.

| Scenario | Timeline Row ID |
| --- | --- |
| `internal/test/game/scenarios/evasion_tie_hit.lua` | `P1` |
| `internal/test/game/scenarios/critical_damage_maximum.lua` | `P2` |
| `internal/test/game/scenarios/damage_thresholds_example.lua` | `P3` |
| `internal/test/game/scenarios/damage_roll_modifier.lua` | `P3` |
| `internal/test/game/scenarios/damage_roll_proficiency.lua` | `P3` |
| `internal/test/game/scenarios/fear_spotlight_armor_mitigation.lua` | `P4` |
| `internal/test/game/scenarios/sweeping_attack_all_targets.lua` | `P5` |
| `internal/test/game/scenarios/fireball_orc_pack_multi.lua` | `P5` |
| `internal/test/game/scenarios/orc_dredge_group_attack.lua` | `P6` |
| `internal/test/game/scenarios/minion_group_attack_rats.lua` | `P6` |
| `internal/test/game/scenarios/minion_overflow_damage.lua` | `P7` |
| `internal/test/game/scenarios/wild_flame_minion_blast.lua` | `P7` |
| `internal/test/game/scenarios/minion_high_threshold_imps.lua` | `P7` |
| `internal/test/game/scenarios/fireball_golum_reaction.lua` | `P8` |
| `internal/test/game/scenarios/ranged_warding_sphere.lua` | `P9` |
| `internal/test/game/scenarios/ranged_snowblind_trap.lua` | `P10` |
| `internal/test/game/scenarios/ranged_take_cover.lua` | `P11` |
| `internal/test/game/scenarios/ranged_steady_aim.lua` | `P12` |
| `internal/test/game/scenarios/ranged_battle_teleport.lua` | `P13` |
| `internal/test/game/scenarios/ranged_arcane_artillery.lua` | `P14` |
| `internal/test/game/scenarios/ranged_eruption_hazard.lua` | `P15` |
| `internal/test/game/scenarios/sam_critical_broadsword.lua` | `P16` |
| `internal/test/game/scenarios/skulk_swift_claws.lua` | `P17` |
| `internal/test/game/scenarios/skulk_cloaked_backstab.lua` | `P18` |
| `internal/test/game/scenarios/skulk_reflective_scales.lua` | `P19` |
| `internal/test/game/scenarios/skulk_icicle_barb.lua` | `P20` |
| `internal/test/game/scenarios/improvised_fear_move_bandit_chain.lua` | `P21` |
| `internal/test/game/scenarios/leader_into_bramble.lua` | `P22` |
| `internal/test/game/scenarios/leader_ferocious_defense.lua` | `P23` |
| `internal/test/game/scenarios/leader_brace_reaction.lua` | `P24` |
| `internal/test/game/scenarios/head_guard_on_my_signal.lua` | `P25` |
| `internal/test/game/scenarios/head_guard_rally_guards.lua` | `P26` |
| `internal/test/game/scenarios/airship_group_roll.lua` | `P27` |
| `internal/test/game/scenarios/group_finesse_sneak.lua` | `P27` |
| `internal/test/game/scenarios/group_action_escape.lua` | `P27` |
| `internal/test/game/scenarios/terrifying_hope_loss.lua` | `P28` |
| `internal/test/game/scenarios/temporary_armor_bonus.lua` | `P29` |
| `internal/test/game/scenarios/spellcast_scope_limit.lua` | `P30` |
| `internal/test/game/scenarios/spellcast_hope_cost.lua` | `P30` |
| `internal/test/game/scenarios/combat_objectives_ritual_rescue_capture.lua` | `P31` |
| `internal/test/game/scenarios/companion_experience_stress_clear.lua` | `P31` |
| `internal/test/game/scenarios/death_reaction_dig_two_graves.lua` | `P31` |
| `internal/test/game/scenarios/encounter_battle_points_example.lua` | `P31` |
| `internal/test/game/scenarios/environment_caradhras_pass_avalanche.lua` | `P32` |
| `internal/test/game/scenarios/environment_caradhras_pass_icy_winds.lua` | `P32` |
| `internal/test/game/scenarios/environment_helms_deep_siege_collateral_damage.lua` | `P32` |
| `internal/test/game/scenarios/environment_helms_deep_siege_siege_weapons.lua` | `P32` |
| `internal/test/game/scenarios/environment_mirkwood_blight_choking_ash.lua` | `P32` |
| `internal/test/game/scenarios/environment_mirkwood_blight_grasping_vines.lua` | `P32` |
| `internal/test/game/scenarios/environment_mirkwood_blight_indigo_flame.lua` | `P32` |
| `internal/test/game/scenarios/environment_moria_ossuary_skeletal_burst.lua` | `P32` |
| `internal/test/game/scenarios/environment_old_forest_grove_barbed_vines.lua` | `P32` |
| `internal/test/game/scenarios/environment_old_forest_grove_overgrown.lua` | `P32` |
| `internal/test/game/scenarios/environment_pelennor_battle_raze.lua` | `P32` |
| `internal/test/game/scenarios/environment_prancing_pony_bar_fight.lua` | `P32` |
| `internal/test/game/scenarios/environment_dark_tower_usurpation_beginning_of_end.lua` | `P33` |
| `internal/test/game/scenarios/environment_dark_tower_usurpation_final_preparations.lua` | `P33` |
| `internal/test/game/scenarios/environment_dark_tower_usurpation_ritual_nexus.lua` | `P33` |
| `internal/test/game/scenarios/environment_isengard_ritual_blasphemous_might.lua` | `P33` |
| `internal/test/game/scenarios/environment_isengard_ritual_complete.lua` | `P33` |
| `internal/test/game/scenarios/environment_isengard_ritual_summoning.lua` | `P33` |
| `internal/test/game/scenarios/environment_bree_market_tip_the_scales.lua` | `P34` |
| `internal/test/game/scenarios/environment_bree_market_unexpected_find.lua` | `P34` |
| `internal/test/game/scenarios/environment_bree_outpost_broken_compass.lua` | `P34` |
| `internal/test/game/scenarios/environment_bree_outpost_rumors.lua` | `P34` |
| `internal/test/game/scenarios/environment_caradhras_pass_engraved_sigils.lua` | `P34` |
| `internal/test/game/scenarios/environment_gondor_court_all_roads.lua` | `P34` |
| `internal/test/game/scenarios/environment_gondor_court_eyes_everywhere.lua` | `P34` |
| `internal/test/game/scenarios/environment_moria_ossuary_centuries_of_knowledge.lua` | `P34` |
| `internal/test/game/scenarios/environment_osgiliath_ruins_buried_knowledge.lua` | `P34` |
| `internal/test/game/scenarios/environment_prancing_pony_mysterious_stranger.lua` | `P34` |
| `internal/test/game/scenarios/environment_prancing_pony_sing.lua` | `P34` |
| `internal/test/game/scenarios/environment_prancing_pony_talk.lua` | `P34` |
| `internal/test/game/scenarios/environment_bree_outpost_rival_party.lua` | `P35` |
| `internal/test/game/scenarios/environment_caradhras_pass_raptor_nest.lua` | `P35` |
| `internal/test/game/scenarios/environment_dark_tower_usurpation_defilers_abound.lua` | `P35` |
| `internal/test/game/scenarios/environment_helms_deep_siege_reinforcements.lua` | `P35` |
| `internal/test/game/scenarios/environment_mirkwood_blight_charcoal_constructs.lua` | `P35` |
| `internal/test/game/scenarios/environment_moria_ossuary_they_keep_coming.lua` | `P35` |
| `internal/test/game/scenarios/environment_old_forest_grove_defiler.lua` | `P35` |
| `internal/test/game/scenarios/environment_pelennor_battle_reinforcements.lua` | `P35` |
| `internal/test/game/scenarios/environment_prancing_pony_someone_comes_to_town.lua` | `P35` |
| `internal/test/game/scenarios/environment_shadow_realm_predators.lua` | `P35` |
| `internal/test/game/scenarios/environment_bree_market_crowd_closes_in.lua` | `P36` |
| `internal/test/game/scenarios/environment_bruinen_ford_dangerous_crossing.lua` | `P36` |
| `internal/test/game/scenarios/environment_bruinen_ford_undertow.lua` | `P36` |
| `internal/test/game/scenarios/environment_helms_deep_siege_secret_entrance.lua` | `P36` |
| `internal/test/game/scenarios/environment_misty_ascent_fall.lua` | `P36` |
| `internal/test/game/scenarios/environment_osgiliath_ruins_dead_ends.lua` | `P36` |
| `internal/test/game/scenarios/environment_pelennor_battle_adrift.lua` | `P36` |
| `internal/test/game/scenarios/environment_shadow_realm_disorienting_reality.lua` | `P36` |
| `internal/test/game/scenarios/environment_shadow_realm_impossible_architecture.lua` | `P36` |
| `internal/test/game/scenarios/environment_dark_tower_usurpation_divine_blessing.lua` | `P37` |
| `internal/test/game/scenarios/environment_dark_tower_usurpation_godslayer.lua` | `P37` |
| `internal/test/game/scenarios/environment_isengard_ritual_desecrated_ground.lua` | `P37` |
| `internal/test/game/scenarios/environment_mirkwood_blight_chaos_magic.lua` | `P37` |
| `internal/test/game/scenarios/environment_moria_ossuary_aura_of_death.lua` | `P37` |
| `internal/test/game/scenarios/environment_moria_ossuary_no_place_living.lua` | `P37` |
| `internal/test/game/scenarios/environment_osgiliath_ruins_apocalypse_then.lua` | `P37` |
| `internal/test/game/scenarios/environment_osgiliath_ruins_ghostly_form.lua` | `P37` |
| `internal/test/game/scenarios/environment_pelennor_battle_war_magic.lua` | `P37` |
| `internal/test/game/scenarios/environment_rivendell_sanctuary_divine_censure.lua` | `P37` |
| `internal/test/game/scenarios/environment_rivendell_sanctuary_guidance.lua` | `P37` |
| `internal/test/game/scenarios/environment_rivendell_sanctuary_healing.lua` | `P37` |
| `internal/test/game/scenarios/environment_rivendell_sanctuary_relentless_hope.lua` | `P37` |
| `internal/test/game/scenarios/environment_shadow_realm_everything_you_are.lua` | `P37` |
| `internal/test/game/scenarios/environment_shadow_realm_unmaking.lua` | `P37` |
| `internal/test/game/scenarios/environment_bree_outpost_wrong_place.lua` | `P38` |
| `internal/test/game/scenarios/environment_bruinen_ford_patient_hunter.lua` | `P38` |
| `internal/test/game/scenarios/environment_old_forest_grove_not_welcome.lua` | `P38` |
| `internal/test/game/scenarios/environment_waylaid_relative_strength.lua` | `P38` |
| `internal/test/game/scenarios/environment_waylaid_surprise.lua` | `P38` |
| `internal/test/game/scenarios/environment_waylayers_relative_strength.lua` | `P38` |
| `internal/test/game/scenarios/environment_waylayers_surprise.lua` | `P38` |
| `internal/test/game/scenarios/environment_waylayers_where_did_they_come_from.lua` | `P38` |
| `internal/test/game/scenarios/environment_bree_market_sticky_fingers.lua` | `P39` |
| `internal/test/game/scenarios/gm_move_artifact_chase.lua` | `P39` |
| `internal/test/game/scenarios/gm_move_examples.lua` | `P39` |
| `internal/test/game/scenarios/improvised_fear_move_noble_escape.lua` | `P40` |
| `internal/test/game/scenarios/improvised_fear_move_rage_boost.lua` | `P40` |
| `internal/test/game/scenarios/improvised_fear_move_shadow.lua` | `P40` |
| `internal/test/game/scenarios/orc_archer_opportunist.lua` | `P41` |
| `internal/test/game/scenarios/environment_misty_ascent_pitons.lua` | `P42` |
| `internal/test/game/scenarios/environment_misty_ascent_progress.lua` | `P42` |
| `internal/test/game/scenarios/progress_countdown_climb.lua` | `P42` |
| `internal/test/game/scenarios/environment_bree_outpost_shakedown.lua` | `P43` |
| `internal/test/game/scenarios/environment_gondor_court_gravity_of_empire.lua` | `P43` |
| `internal/test/game/scenarios/environment_gondor_court_imperial_decree.lua` | `P43` |
| `internal/test/game/scenarios/environment_gondor_court_rival_vassals.lua` | `P43` |
| `internal/test/game/scenarios/social_merchant_haggling.lua` | `P43` |
| `internal/test/game/scenarios/social_village_elder_peace.lua` | `P43` |
| `internal/test/game/scenarios/spellcast_flavor_limits.lua` | `P44` |

## Glossary of Common Mechanics

Use these definitions to interpret repeated terms in the gaps list without re-reading the Core Rulebook (CRB).

- Spotlight: the table's active focus. Whoever has the spotlight is the only entity that acts; after the action resolves, the spotlight moves to whoever the fiction or a triggered mechanic indicates.
- Spotlight shift: control passes from the current actor to the next actor. If a shift puts the spotlight on the GM, they immediately take a GM move and may choose which adversary or environment is spotlighted next (spending Fear if required).
- Immediate spotlight: a feature that forces the next action to occur right away (usually the summoned or chosen adversary acts immediately as the next spotlighted entity).
- GM move: the GM's narrative/mechanical response to a roll or opportunity. Moves can introduce attacks, stress, threats, or scene changes. Soft moves offer a response window; hard moves apply consequences immediately.
- Action roll outcomes: Success with Hope (gain Hope), Success with Fear (GM gains Fear), Failure with Hope (gain Hope, GM move), Failure with Fear (GM gains Fear, GM move). Critical Success (matching Duality Dice) is an automatic success with bonus effects.
- Difficulty: the target number a roll must meet or beat. Attacks against PCs use the target's Evasion; attacks against adversaries use the adversary's Difficulty stat.
- Reaction roll: a defensive roll against an incoming attack or hazard. It does not generate Hope or Fear and cannot be aided with Help an Ally.
- Advantage / disadvantage: add (advantage) or subtract (disadvantage) a d6 to the roll total. They cancel one-for-one within the same dice pool. Help an Ally adds a separate advantage die; only the highest helper die applies.
- Help an Ally: spend 1 Hope to roll an advantage die for another PC's action roll; the acting PC adds the highest helper die.
- Hope / Fear: PC metacurrency (Hope) and GM metacurrency (Fear). Hope fuels features; Fear fuels GM moves and adversary/environment Fear Feature(s).
- Resources: Hope, Fear, Stress, Armor Slots, and gold; these are spent, marked, or cleared by features and moves.
- Armor Slots and thresholds: Armor Slots reduce damage severity by one threshold when marked; only 1 Armor Slot can be marked per instance of damage. Damage thresholds determine whether 1, 2, or 3 HP are marked based on final damage after reductions. Max Armor Score is 12.
- Proficiency: the number of weapon damage dice rolled. It increases dice count only and does not multiply flat damage modifiers.
- Resistance / immunity: resistance halves damage of a type before thresholds; immunity ignores damage of that type.
- Direct damage: damage that cannot be reduced by marking Armor Slots.
- Cover: when you take cover behind something that makes attacking you more difficult but not impossible, attack rolls against you are made with disadvantage. Full cover (behind a wall) prevents targeting entirely.
- Conditions: Hidden (rolls against you have disadvantage until you are seen or attack), Restrained (cannot move), Vulnerable (rolls targeting you have advantage). Conditions cannot stack on one target. Temporary conditions can be cleared by a successful action roll to escape.
- Unconscious: a state triggered by the Avoid Death death move. The PC cannot move or act, cannot be targeted by attacks, and returns to consciousness when an ally clears 1+ of their HP or the party finishes a long rest.
- Scars: gained when choosing Avoid Death and rolling your Hope Die equal to or under your level. Cross out a Hope slot permanently. If your last Hope slot is crossed out, the character's journey ends.
- Stress overflow: when forced to mark Stress but all slots are full, mark 1 HP instead. Marking your last Stress makes you Vulnerable until you clear at least 1 Stress.
- Movement and range: range bands are Melee, Very Close, Close, Far, Very Far. During an action, a PC can move within Close range. Moving Far or Very Far requires an Agility Roll. Adversaries move within Close range as part of their action; moving Far/Very Far uses their entire action but requires no roll.
- Countdowns: timers that tick down to trigger an effect. Standard countdowns tick on each action roll. Dynamic progress/consequence countdowns tick based on roll outcomes. Loop countdowns reset to their start value when they trigger. Increasing/decreasing countdowns change their starting value by 1 each loop. Chase countdowns pair a progress and consequence countdown. Long-term countdowns advance on rests (short rest: 1 tick, long rest: 2 ticks) or specific triggers.
- Standard attack: an adversary's primary attack. It uses d20 + attack modifier vs Evasion and deals the listed damage on a hit.
- Fear Feature: a high-impact adversary or environment action that costs Fear in addition to any other costs.
- Spellcasting rules: Spellcast rolls use the subclass Spellcast trait; spells only do what their text says and end per their duration rules.
- Spawn/summon: create one or more adversaries or NPCs at a stated range and add them to the scene; if the feature says to spotlight them, they act immediately.
- Repositioning: the GM updates ranges/line of sight without a roll when a feature explicitly moves or separates characters.
- Environment shift: replace the active environment with another; the GM then uses the new environment's features and may spotlight it immediately.
- Chase: a paired Progress Countdown (PCs) vs Consequence Countdown (opposition). The chase ends when either countdown triggers.
- Social constraint: an ongoing pressure that justifies disadvantage on certain rolls or adds a cost/complication to success.
- Tag Team Roll: once per session per player, spend 3 Hope to make a coordinated attack with another PC. Both make separate action rolls and choose one to apply. On a successful attack, both roll damage and combine totals. Counts as a single action roll for countdowns.
- GM Critical Success: when the GM rolls a natural 20, the roll automatically succeeds. On a critical attack, start with max dice value then roll normally and add both totals.
- GM Fear economy: GM starts with Fear equal to number of PCs. Max 12 Fear. Fear carries between sessions. Fear can be spent to interrupt PCs, make additional GM moves, activate Fear Features, or add adversary Experience to rolls.

## Build Order Summary

This ordering shows which system primitives should be implemented before scenario-specific mechanics.

- Core rolls and outcomes: action rolls, reaction rolls, action roll outcomes, Difficulty/Evasion, Hope/Fear gain.
- Dice modifiers: advantage/disadvantage, Help an Ally, Experience modifiers, rerolls.
- Resources: Hope/Fear/Stress use, Armor Slots, gold changes.
- Damage pipeline: damage rolls, Proficiency, resistance/immunity, Armor Slots, thresholds, direct damage.
- Conditions: Hidden, Restrained, Vulnerable; temporary condition clearing.
- Movement and range: range bands, forced movement, repositioning.
- Spotlight and GM moves: spotlight shifts, GM move triggers, Fear spend to interrupt or add moves.
- Countdowns: standard, dynamic, loop, long-term; progress vs consequence.
- Adversary actions: standard attacks, multi-target attacks, reactions, fear features.
- Environment actions: hazards, scene-wide effects, spawns, environment shifts.
- Social and economy: social constraints, discounts/penalties, gold tracking, rumor/knowledge rolls.
- Spellcasting rules: Spellcast rolls, spell scope limits, spell duration/termination.
## General Mechanics Gaps

(None currently detected.)

## Daggerheart-Specific Mechanics Gaps

### Scenario fixture placeholders requiring clarification

- `internal/test/game/scenarios/combat_objectives_ritual_rescue_capture.lua` — Placeholder for parallel combat objectives (ritual completion, rescue progress, capture progress) with missing rules for updating multiple objective countdowns from action outcomes.
  > **CRB (p.162):** Linked progress and consequence countdowns simultaneously advance according to the same action roll outcomes. Dynamic countdown advancement: Critical Success ticks progress 3/consequence 0; Success with Hope ticks progress 2/consequence 0; Success with Fear ticks progress 1/consequence 1; Failure with Hope ticks progress 0/consequence 2; Failure with Fear ticks progress 0/consequence 3. Multiple parallel countdowns are not explicitly addressed — the CRB covers linked pairs, not three-way objectives. **[CRB: UNCLEAR — parallel multi-objective countdown interaction not specified; only linked pairs are defined]**

- `internal/test/game/scenarios/companion_experience_stress_clear.lua` — Placeholder for companion return cadence where an experience completion triggers stress clear; exact invocation and timing semantics are not specified.
  > **CRB (p.39):** "When your companion would take any amount of damage, they mark a Stress. When they mark their last Stress, they drop out of the scene (by hiding, fleeing, or a similar action). They remain unavailable until the start of your next long rest, where they return with 1 Stress cleared." "When you choose a downtime move that clears Stress on yourself, your companion clears an equal number of Stress." **[CRB: UNCLEAR — "experience completion" as a stress-clear trigger is not in the CRB; companion stress clearing is tied to rests and PC downtime moves only]**

- `internal/test/game/scenarios/death_reaction_dig_two_graves.lua` — Placeholder for a death-triggered adversary reaction (damage + Hope loss) with missing deterministic sequencing and scope rules.
  > **CRB (p.106):** Death moves (Blaze of Glory, Avoid Death, Risk It All) are PC mechanics triggered when marking their last HP. The CRB does not define a general "death reaction" for adversaries. Individual stat blocks may include death-triggered reactions. **[CRB: NOT FOUND — no generic adversary death reaction mechanic; individual stat blocks define their own death-triggered reactions]**

- `internal/test/game/scenarios/encounter_battle_points_example.lua` — Placeholder for encounter battle-point budgeting and encounter composition, requiring clear conversion from points to adversary mix and scaling rules.
  > **CRB (p.196):** Building Balanced Encounters: start with [(3 × number of PCs in combat) + 2] Battle Points. Spend 1 point per Minion group equal to party size, 1 per Social or Support, 2 per Horde/Ranged/Skulk/Standard, 3 per Leader, 4 per Bruiser, 5 per Solo. Adjustments: -1 for easier/shorter, -2 for 2+ Solos, -2 for +1d4 or +2 to all damage dice, +1 for lower-tier adversary, +1 for no Bruisers/Hordes/Leaders/Solos, +2 for harder/longer.

- `internal/test/game/scenarios/environment_helms_deep_siege_collateral_damage.lua` — Placeholder for environment collateral-damage fallout after an adversary falls; requires explicit reaction-roll, damage, and stress-outcome mechanics.
  > **CRB (p.249, Castle Siege — Collateral Damage):** When an adversary is defeated, you can spend a Fear to have a stray attack from a siege weapon hit a point on the battlefield. All targets within Very Close range of that point must make an Agility Reaction Roll. Targets who fail take 3d8+3 physical or magic damage and must mark a Stress. Targets who succeed must mark a Stress.

Scenario annotations still call out missing DSL/mechanics. Items are grouped by theme and ordered by priority within each bucket (highest impact first):

### Core roll and damage resolution
Requires: Core rolls and outcomes; Dice modifiers; Resources; Damage pipeline.
- Force the adversary attack roll to equal Evasion — `internal/test/game/scenarios/evasion_tie_hit.lua`. Trigger: adversary attack roll total equals the target's Evasion. Effects: attack succeeds on a tie. Requires: See section Requires. Timeline: `P1`. Notes: adversary attack roll uses d20 + attack modifier vs Evasion.
  > **CRB (p.160):** "Roll the d20 and add the adversary's attack bonus. If the roll result meets or beats the target PC's Evasion, the attack succeeds and hits." Ties go to the attacker. GM critical (natural 20) auto-succeeds and deals extra damage (max dice value + normal roll).

- Apply max-dice bonus before rolling damage — `internal/test/game/scenarios/critical_damage_maximum.lua`. Trigger: critical success on an attack roll (matching Duality Dice). Effects: roll damage normally, then add the maximum possible result of the damage dice to the total. Requires: See section Requires. Timeline: `P2`. Notes: flat modifiers are not doubled; apply resistance/armor after total damage is known.
  > **CRB (p.98):** "If your attack roll critically succeeds, your attack deals extra damage! Start with the highest possible value the damage dice can roll, and then make a damage roll as usual, adding it to that value." The flat modifier is added once; only the dice maximum is added as a bonus.

- Assert tier mapping and HP marked for each tier — `internal/test/game/scenarios/damage_thresholds_example.lua`. Trigger: damage is applied after a successful attack. Effects: compare final damage to Major/Severe thresholds to mark 1, 2, or 3 HP; if damage is reduced to 0 or less, mark no HP. Requires: See section Requires. Timeline: `P3`. Notes: tiers map to levels (Tier 1: level 1; Tier 2: levels 2-4; Tier 3: levels 5-7; Tier 4: levels 8-10); apply resistance and other reductions before thresholds.
  > **CRB (p.91):** "Severe damage is equal to or above your Severe threshold; you mark 3 HP. Major damage is equal to or above your Major threshold but below Severe; you mark 2 HP. Minor damage is anything below your Major threshold; you mark 1 HP. If you ever reduce incoming damage to 0 or less, you don't mark any HP." A character's level is added to their armor's base damage thresholds. Optional Massive Damage rule (p.91): damage equal to double your Severe threshold marks 4 HP.

- Force the damage dice to 3, 5, and 6 — `internal/test/game/scenarios/damage_roll_modifier.lua`. Trigger: a damage roll is resolved with fixed or modified die results. Effects: sum the damage dice results and add flat modifiers once. Requires: See section Requires. Timeline: `P3`. Notes: damage dice count equals Proficiency (weapons) or Spellcast trait (spell damage); rerolls or die-setting effects replace the rolled value and the new result must be used.
  > **CRB (p.98):** "Your Proficiency determines how many damage dice you roll on a successful attack with a weapon." Damage roll = sum of [Proficiency] dice + flat modifier. Proficiency does not affect flat modifiers. Rerolls: "When a feature allows you to reroll a die, you always take the new result" (p.107).

- Force the damage dice to 3 and 7 — `internal/test/game/scenarios/damage_roll_proficiency.lua`. Trigger: a weapon damage roll uses Proficiency to determine dice count. Effects: roll a number of dice equal to Proficiency, sum results, then add the flat modifier once. Requires: See section Requires. Timeline: `P3`. Notes: Proficiency increases dice count only; it does not multiply flat modifiers.
  > **CRB (p.98):** "You start at 1 Proficiency and can increase this value to a maximum of 6. Your Proficiency determines how many damage dice you roll on a successful attack with a weapon. This value is not weapon-specific, and does not change or reset when you equip a new weapon." Damage dice are determined by the weapon; flat modifiers are added once regardless of Proficiency.

### Combat and adversary actions
Requires: Adversary actions; Spotlight and GM moves; Damage pipeline; Conditions; Resources; Dice modifiers.
- Set the adversary hit, damage total, and armor slot spend — `internal/test/game/scenarios/fear_spotlight_armor_mitigation.lua`. Trigger: GM spends Fear to seize spotlight and make a GM move or spotlight an adversary. Effects: adversary attack uses d20 + attack modifier vs Evasion; on success roll listed damage; target may mark 1 Armor Slot to reduce severity by one threshold. Requires: See section Requires. Timeline: `P4`. Notes: Armor Slots available only if Armor Score > 0; marking Armor Slots happens after total damage is known.
  > **CRB (p.114):** "When your character takes damage, you can negate some (or all) of it by marking an available Armor Slot, then reducing the severity of the damage by one threshold. Each time your character takes damage, you can only mark 1 Armor Slot." **CRB (p.154):** GM can spend Fear to interrupt PCs and spotlight an adversary. **CRB (p.160):** Adversary attacks: d20 + Attack Modifier vs Evasion; on success deal the listed damage.

- Spend adversary stress and resolve a multi-target adversary attack — `internal/test/game/scenarios/sweeping_attack_all_targets.lua`. Trigger: adversary action that costs Stress and targets multiple creatures. Effects: mark the adversary's Stress cost, make one attack roll, compare to each target's Evasion, and apply damage to each target independently. Requires: See section Requires. Timeline: `P5`. Notes: multi-target adversary actions use one attack roll; damage resolution is per target.
  > **CRB (p.160):** "When an adversary's action lets you make an attack against multiple targets, you make one attack roll and ask if it hits any of the targets." **CRB (p.196):** Feature stress costs come from the adversary whose feature is being activated.

- Represent group attack roll and shared damage — `internal/test/game/scenarios/orc_dredge_group_attack.lua`. Trigger: Minion group attack action. Effects: spend Fear, spotlight the Minion group (all Minions within Close range as a single spotlighted entity), make one shared attack roll, and on success each Minion contributes their listed damage; combine damage into a single total. Requires: See section Requires. Timeline: `P6`. Notes: combined damage is treated as a single source before thresholds.
  > **CRB (p.196):** Minion Group Attack pattern (e.g., Skeleton Dredge, p.217): "Spend a Fear to choose a target and spotlight all [Minions] within Close range of them. Those Minions move into Melee range of the target and make one shared attack roll. On a success, they deal [X] physical damage each. Combine this damage."

- Resolve group attack damage aggregation — `internal/test/game/scenarios/minion_group_attack_rats.lua`. Trigger: group attack succeeds. Effects: add each Minion's damage together, then apply resistance/armor/thresholds once to determine HP marked. Requires: See section Requires. Timeline: `P6`. Notes: combined damage is one source for threshold comparison.
  > **CRB (p.211, Giant Rat):** "Group Attack: Spend a Fear to choose a target and spotlight all Giant Rats within Close range of them. Those Minions move into Melee range of the target and make one shared attack roll. On a success, they deal 1 physical damage each. Combine this damage."

- Apply Minion (3) overflow and select extra targets — `internal/test/game/scenarios/minion_overflow_damage.lua`. Trigger: a Minion takes any damage. Effects: Minion is defeated; for every 3 damage dealt, defeat an additional Minion within range the attack would succeed against. Requires: See section Requires. Timeline: `P7`. Notes: overflow targets must be within the original attack's valid range/line of sight.
  > **CRB (p.196):** "Minion (X) - Passive: This adversary is defeated when they take any damage. For every X damage a PC deals to this adversary, defeat an additional Minion within range the attack would succeed against." For Minion (3), see e.g. Jagged Knife Lackey (p.213), Giant Rat (p.211).

- Apply Minion (4) overflow and stress marking — `internal/test/game/scenarios/wild_flame_minion_blast.lua`. Trigger: Minion (4) passive on damage; and any linked effect that marks Stress. Effects: defeat additional Minions for every 4 damage; apply any feature-specified Stress marking to affected targets. Requires: See section Requires. Notes: overflow still respects attack range and targeting rules.
  > **CRB (p.196):** Minion (X) passive as above. For Minion (4), see e.g. Tangle Bramble (p.218): "Minion (4) - Passive: … For every 4 damage a PC deals to the Bramble, defeat an additional Minion within range the attack would succeed against."

- Apply Minion (8) overflow — `internal/test/game/scenarios/minion_high_threshold_imps.lua`. Trigger: Minion (8) passive on damage. Effects: defeat additional Minions for every 8 damage dealt to the initial Minion. Requires: See section Requires. Notes: overflow targets must be within the original attack's valid targeting.
  > **CRB (p.196):** Minion (X) passive as above. **[CRB: NOT FOUND — no Minion (8) stat block exists in the CRB. The CRB has Minion (3) through Minion (13). Minion (8) is scenario-specific or from a stat block not in the CRB.]**

- Apply the Opportunist doubling and armor mitigation — `internal/test/game/scenarios/orc_archer_opportunist.lua`. Trigger: Opportunist passive when two or more adversaries are within Very Close range of a target. Effects: double damage dealt by the Opportunist to that target, then apply armor mitigation and thresholds. Requires: See section Requires. Notes: doubling happens before applying Armor Slots and thresholds.
  > **CRB (p.217, Skeleton Archer):** "Opportunist - Passive: When two or more adversaries are within Very Close range of a creature, all damage the Archer deals to that creature is doubled."

- Assert per-target outcomes and damage tiers — `internal/test/game/scenarios/fireball_orc_pack_multi.lua`. Trigger: multi-target attack roll. Effects: one attack roll and one damage roll, then apply damage and thresholds for each target individually; each target may use armor/resistance separately. Requires: See section Requires. Notes: attack roll is compared to each target's Difficulty/Evasion.
  > **CRB (p.160):** "When an adversary's action lets you make an attack against multiple targets, you make one attack roll and ask if it hits any of the targets." For PC multi-target attacks (p.96): "roll once and apply that result to all of the adversaries the attack can hit. The attack is successful against all targets for which the attack roll result meets or exceeds their Difficulty."

- Adversary reaction roll with an experience bonus — `internal/test/game/scenarios/fireball_golum_reaction.lua`. Trigger: adversary makes a reaction roll to avoid an effect. Effects: roll d20; if GM spends Fear, add a relevant Experience; compare to the effect's Difficulty. Requires: See section Requires. Timeline: `P8`. Notes: a natural 20 reaction roll automatically succeeds but grants no extra benefit.
  > **CRB (p.161):** "When PC moves force an adversary to make a reaction roll, roll a d20. If it meets or exceeds the Difficulty, the NPC succeeds." **CRB (p.154):** "The GM can spend a Fear to add an adversary's relevant Experience to raise their attack roll or increase the Difficulty of a roll made against them." A natural 20 on an adversary reaction roll has no added benefit.

- Apply reactive damage and cooldown on the reaction — `internal/test/game/scenarios/ranged_warding_sphere.lua`. Trigger: reaction such as Warding Sphere when the adversary takes damage within Close range. Effects: deal listed reactive damage to the attacker; reaction is unavailable until refreshed by the specified action. Requires: See section Requires. Notes: reaction triggers regardless of spotlight but obeys its own cooldown rule.
  > **CRB (p.202, War Wizard):** "Warding Sphere - Reaction: When the Wizard takes damage from an attack within Close range, deal 2d6 magic damage to the attacker. This reaction can't be used again until the Wizard uses their 'Refresh Warding Sphere' action." Reactions trigger regardless of spotlight but obey their own cooldown rules.

- Apply group reaction rolls and Vulnerable condition — `internal/test/game/scenarios/ranged_snowblind_trap.lua`. Trigger: area effect that calls for reaction rolls from multiple targets. Effects: each target rolls a reaction roll; on failure apply Vulnerable (rolls against them have advantage) and any listed damage/effects; on success apply reduced effect if specified. Requires: See section Requires. Notes: reaction rolls do not generate Hope or Fear.
  > **CRB (p.102):** "When you gain the Vulnerable condition, you're in a difficult position within the fiction. While you are Vulnerable, all rolls targeting you have advantage." **CRB (p.99):** "Reaction rolls work similarly to action rolls, except they don't generate Hope, Fear, or additional GM moves."

- Apply disadvantage to the attack and reduce damage severity — `internal/test/game/scenarios/ranged_take_cover.lua`. Trigger: attacks made through cover or a feature that imposes disadvantage and reduces damage. Effects: apply a d6 disadvantage die to the attack roll; if the feature reduces damage severity, downgrade severity by one threshold after damage is totaled. Requires: See section Requires. Notes: disadvantage cancels with advantage in the same pool.
  > **CRB (p.104):** "When you take cover behind something that makes attacking you more difficult (but not impossible), attack rolls against you are made with disadvantage." **CRB (p.100):** Disadvantage subtracts a d6; advantage and disadvantage cancel one-for-one. **CRB (p.114):** Armor Slots reduce severity by one threshold.

- Apply stress spend and advantage die — `internal/test/game/scenarios/ranged_steady_aim.lua`. Trigger: feature that costs Stress to gain advantage. Effects: mark Stress, then add a d6 advantage die to the roll total. Requires: See section Requires. Notes: must be declared before rolling.
  > **CRB (p.100):** "When you roll with advantage, you add a d6 advantage die to your total." Feature stress costs come from the adversary (e.g., Hobbling Shot on Archer Guard, p.212, or Double Strike on Giant Scorpion, p.211).

- Move the adversary and spend Stress — `internal/test/game/scenarios/ranged_battle_teleport.lua`. Trigger: adversary feature like Battle Teleport. Effects: mark Stress to teleport within the stated range, then act as allowed (often before/after a standard attack). Requires: See section Requires. Notes: teleport distance obeys range bands.
  > **CRB (p.202, War Wizard):** "Battle Teleport - Passive: Before or after making a standard attack, mark a Stress to teleport to a location within Far range." **CRB (p.103):** Movement uses range bands (Melee, Very Close, Close, Far, Very Far).

- Apply scene-wide reaction rolls and half damage on success — `internal/test/game/scenarios/ranged_arcane_artillery.lua`. Trigger: area action that targets all creatures in the scene. Effects: each target makes a reaction roll; on failure take full damage, on success take half. Requires: See section Requires. Notes: reaction rolls do not generate Hope/Fear.
  > **CRB (p.202, War Wizard):** "Arcane Artillery - Action: Spend a Fear to unleash a precise hail of magical blasts. All targets in the scene must make an Agility Reaction Roll. Targets who fail take 2d12 magic damage. Targets who succeed take half damage."

- Apply area hazard, reaction roll, and forced movement — `internal/test/game/scenarios/ranged_eruption_hazard.lua`. Trigger: area hazard action such as Eruption. Effects: targets in the area make reaction rolls; on failure take damage and are forced out of the area; on success take half damage and do not move. Requires: See section Requires. Notes: forced movement uses range bands and does not require additional rolls.
  > **CRB (p.202, War Wizard):** "Eruption - Action: Spend a Fear and choose a point within Far range. A Very Close area around that point erupts into impassable terrain. All targets within that area must make an Agility Reaction Roll (14). Targets who fail take 2d10 physical damage and are thrown out of the area. Targets who succeed take half damage and aren't moved."

- Apply advantage die, stress cost, and reroll a 1 — `internal/test/game/scenarios/sam_critical_broadsword.lua`. Trigger: feature that grants advantage and allows damage die rerolls. Effects: mark Stress to gain advantage; rerolling dice replaces the original result and the new result must be used. Requires: See section Requires. Notes: rerolls apply only to dice specified by the feature (e.g., damage dice showing 1).
  > **CRB (p.100):** Advantage adds a d6 to the roll. **CRB (p.107):** "At the GM's discretion, most effects can stack. However, you can't stack conditions, advantage or disadvantage, or other effects that say you can't." **CRB (p.107):** "When a feature allows you to reroll a die, you always take the new result."

- Apply movement, stress spend, and knockback — `internal/test/game/scenarios/skulk_swift_claws.lua`. Trigger: action that includes movement and knockback with a Stress cost. Effects: move within the allowed range, make the attack, and on success knock the target back to the specified range. Requires: See section Requires. Notes: movement under pressure normally allows up to Close as part of an action; greater distances require an Agility roll unless the feature overrides.
  > **CRB (p.103):** "When you make an action roll, you can also move to a location within Close range as part of that action. If you want to move farther than your Close range, you'll need to succeed on an Agility Roll." Knockback is feature-specific (e.g., Overwhelming Force on Bear, p.210).

- Apply Hidden and swap in 1d6+6 damage on advantage — `internal/test/game/scenarios/skulk_cloaked_backstab.lua`. Trigger: Cloaked/Hidden state and an attack made with advantage. Effects: while Hidden, attacks against you have disadvantage; if your attack has advantage, apply the Backstab damage upgrade (e.g., 1d6+6) instead of standard damage. Requires: See section Requires. Notes: Hidden ends when you are seen or after you attack; advantage must be present on the attack roll.
  > **CRB (p.214, Jagged Knife Shadow):** "Backstop - Passive: When the Shadow succeeds on a standard attack that has advantage, they deal 1d6+6 physical damage instead of their standard damage. Cloaked - Action: Become Hidden until after the Shadow's next attack. Attacks made while Hidden from this feature have advantage." **CRB (p.102):** "While Hidden, any rolls against you have disadvantage. After an adversary moves to where they would see you, you move into their line of sight, or you make an attack, you are no longer Hidden."

- Apply disadvantage to attacks beyond Very Close range — `internal/test/game/scenarios/skulk_reflective_scales.lua`. Trigger: attacks made beyond Very Close range when a feature imposes disadvantage. Effects: add a d6 disadvantage die to the roll. Requires: See section Requires. Notes: cancels with advantage in the same pool.
  > **CRB (p.100):** Disadvantage subtracts a d6 from the roll total. Cancels with advantage one-for-one. Range-based disadvantage is feature-specific. **[CRB: NOT FOUND — no generic "disadvantage at range" rule; this is a stat-block-specific passive]**

- Apply group attack resolution and Restrained condition — `internal/test/game/scenarios/skulk_icicle_barb.lua`. Trigger: group attack action that applies Restrained. Effects: shared attack roll; on success, apply damage and Restrained to the target if the feature specifies. Requires: See section Requires. Notes: Restrained prevents movement but allows actions; temporary conditions clear via a successful action roll against the condition.
  > **CRB (p.102):** "When you gain the Restrained condition, you can't move until this condition is cleared, but you can still take actions from your current position." **CRB (p.196):** Group Attack pattern: shared attack roll, combined damage. Restrained application is feature-specific (e.g., Bear's Bite, p.210: "the target is Restrained until they break free with a successful Strength Roll").

- Use Better Surrounded to hit all targets in range — `internal/test/game/scenarios/improvised_fear_move_bandit_chain.lua`. Trigger: fear move that upgrades a group attack to hit all targets in range. Effects: make one attack roll and apply it to all targets in range; roll damage once and apply per target if the move specifies shared damage. Requires: See section Requires. Notes: multi-target attack rules apply; spend Fear if the move requires it.
  > **CRB (p.160):** Multi-target attacks: one attack roll compared to each target's Evasion separately. **[CRB: NOT FOUND — "Better Surrounded" is not a named CRB mechanic; this is a scenario-specific improvised fear move]**

- Apply group attack damage to the target — `internal/test/game/scenarios/improvised_fear_move_bandit_chain.lua`. Trigger: group attack success. Effects: each Minion contributes listed damage; combine and apply as one source. Requires: See section Requires. Notes: apply armor/resistance after combining.
  > **CRB (p.196):** Group Attack pattern: "On a success, they deal [X] physical damage each. Combine this damage." Combined damage is one source for threshold comparison.

- Move allies to cover and apply Hidden until they attack — `internal/test/game/scenarios/leader_into_bramble.lua`. Trigger: feature that moves allies into cover and grants Hidden. Effects: allies reposition within the stated range; while Hidden, attacks against them have disadvantage and the condition ends when they attack or are seen. Requires: See section Requires. Notes: cover implies disadvantage on attacks through partial obstruction.
  > **CRB (p.102):** "While Hidden, any rolls against you have disadvantage." **CRB (p.104):** "When you take cover behind something that makes attacking you more difficult (but not impossible), attack rolls against you are made with disadvantage." Cover is now a codified mechanic distinct from Hidden.

- Apply Difficulty increase after HP loss — `internal/test/game/scenarios/leader_ferocious_defense.lua`. Trigger: feature that escalates Difficulty after the adversary marks HP. Effects: increase Difficulty for future rolls against that adversary by the specified amount. Requires: See section Requires. Notes: Difficulty applies to both attacks and action rolls against the adversary.
  > **CRB (p.196):** "Difficulty: The Difficulty of any roll made against the adversary, unless otherwise noted." Difficulty can be modified by features. Example: Knight of the Realm's Chevalier passive (p.224) grants +2 to Difficulty while mounted.

- Reduce HP loss and spend Stress on reaction — `internal/test/game/scenarios/leader_brace_reaction.lua`. Trigger: reaction that mitigates damage by spending Stress. Effects: mark Stress to reduce the number of HP marked (e.g., reduce severity by one threshold or prevent 1 HP) as specified by the feature. Requires: See section Requires. Notes: mitigation happens after damage is totaled but before HP is marked.
  > **CRB (p.114):** "When your character takes damage, you can negate some (or all) of it by marking an available Armor Slot, then reducing the severity of the damage by one threshold. Each time your character takes damage, you can only mark 1 Armor Slot." Stress-based mitigation is feature-specific (e.g., Guardian's Valor domain: "When you mark an Armor Slot to reduce incoming damage, you can mark a Stress to mark an additional Armor Slot").

- Apply advantage to archer attacks while the countdown runs — `internal/test/game/scenarios/head_guard_on_my_signal.lua`. Trigger: On My Signal countdown (5) that ticks when PCs make attack rolls. Effects: when the countdown triggers, all Archer Guards within Far range make standard attacks with advantage against the nearest target; combine damage if multiple succeed on the same target. Requires: See section Requires. Notes: countdown ticks on PC attack rolls; advantage adds a d6.
  > **CRB (p.212, Head Guard):** "On My Signal - Reaction: Countdown (5). When the Head Guard is in the spotlight for the first time, activate the countdown. It ticks down when a PC makes an attack roll. When it triggers, all Archer Guards within Far range make a standard attack with advantage against the nearest target within their range. If any attacks succeed on the same target, combine their damage."

- Model the Rally Guards action effect — `internal/test/game/scenarios/head_guard_rally_guards.lua`. Trigger: Rally Guards action. Effects: spend 2 Fear to spotlight the Head Guard, then immediately shift the spotlight in turn to up to 2d4 allies within Far range, each of whom takes one action this GM turn. Requires: See section Requires. Notes: these allies act via immediate spotlight shifts as part of the same GM turn; no extra spotlight cost beyond the 2 Fear for this action.
  > **CRB (p.212, Head Guard):** "Rally Guards - Action: Spend 2 Fear to spotlight the Head Guard and up to 2d4 allies within Far range."

### Group and shared outcomes
Requires: Core rolls and outcomes; Dice modifiers; Spotlight and GM moves.
- Map individual outcomes to a shared consequence — `internal/test/game/scenarios/airship_group_roll.lua`. Trigger: group action roll where one PC leads and others make reaction rolls. Effects: leader gains +1 for each successful reaction roll and -1 for each failed reaction roll; the leader's result determines the shared outcome. Requires: See section Requires. Notes: reaction rolls do not generate Hope or Fear.
  > **CRB (p.97):** "The action's leader makes an action roll as usual, while the other players make a reaction roll using whichever traits they and the GM decide fit best. The leader's action roll gains a +1 bonus for each reaction roll that succeeds and a −1 penalty for each reaction roll that fails."

- Encode per-supporter outcomes and group bonuses — `internal/test/game/scenarios/group_finesse_sneak.lua`. Trigger: group action roll with multiple supporters. Effects: each supporter makes a reaction roll using an agreed trait; successes and failures modify the leader's roll total as above. Requires: See section Requires. Notes: only the leader's action roll can generate Hope or Fear.
  > **CRB (p.97):** Group Action Rolls as above. Only the leader's action roll generates Hope or Fear; supporters use reaction rolls which do not.

- Assert each participant outcome and any fear/hope changes — `internal/test/game/scenarios/group_action_escape.lua`. Trigger: group action roll resolution. Effects: apply Hope/Fear gain based on the leader's roll only; supporters do not generate Hope/Fear on reaction rolls. Requires: See section Requires. Notes: if the leader rolls with Fear or fails, the spotlight swings to the GM.
  > **CRB (p.97):** Only the leader's action roll generates Hope or Fear. **CRB (p.94):** On a result with Fear (whether success or failure), the GM gains a Fear.

### Resource economy and state changes
Requires: Resources; Adversary actions; Spotlight and GM moves.
- Apply group Hope loss and GM Fear gain — `internal/test/game/scenarios/terrifying_hope_loss.lua`. Trigger: Terrifying or similar feature on a successful adversary attack. Effects: all PCs in the stated range lose the listed Hope; GM gains a Fear. Requires: See section Requires. Notes: if a PC has no Hope to lose, apply any alternate effect specified by the feature.
  > **CRB (p.217, Skeleton Knight):** "Terrifying - Passive: When the Knight makes a successful attack, all PCs within Close range lose a Hope and you gain a Fear." Same pattern on many adversaries.

- Apply temporary armor bonus and clear Armor Slots on rest — `internal/test/game/scenarios/temporary_armor_bonus.lua`. Trigger: effect temporarily increases Armor Score. Effects: increase available Armor Slots by the same amount while the effect lasts; clear Armor Slots during rest via Repair Armor (short rest) or Repair All Armor (long rest). Requires: See section Requires. Notes: when the temporary effect ends, the extra Armor Slots vanish.
  > **CRB (p.114):** "Your character's Armor Score, with all bonuses included, can never exceed 12. If an effect gives your character a temporary Armor Score, you can mark that many additional Armor Slots while the temporary armor is active. When the temporary armor ends, clear a number of Armor Slots equal to the temporary Armor Score." **CRB (p.105):** Short rest: "Repair Armor: clear a number of Armor Slots equal to 1d4 + your tier." Long rest: "Repair All Armor: clear all Armor Slots."

### Spellcasting
Requires: Spellcasting rules; Core rolls and outcomes; Resources.
- Reject a Spellcast roll that attempts an out-of-scope effect — `internal/test/game/scenarios/spellcast_scope_limit.lua`. Trigger: player proposes a spell effect outside the card's text. Effects: GM disallows or reframes the action; no Spellcast roll for effects not supported by the spell. Requires: See section Requires. Notes: rulings over rules; spell effects are bounded by card text.
  > **CRB (p.96):** "You can't make a Spellcast Roll unless you use a spell that calls for one, and the action you're trying to perform must be within the scope of the spell. You can't just make up magic effects that aren't on your character sheet or cards." Flavor doesn't grant access to new effects.

- Spend Hope to cast and apply the Fear gain to the GM — `internal/test/game/scenarios/spellcast_hope_cost.lua`. Trigger: spell or feature requires Hope to cast and the roll is resolved. Effects: spend required Hope; if the roll succeeds or fails with Fear, GM gains a Fear. Requires: See section Requires. Notes: Hope spent must be declared before the roll; Hope gained from a roll with Hope can be spent on the same feature.
  > **CRB (p.101):** Domain cards have a Recall Cost (Stress to swap from vault) and many have Hope costs in their text. **CRB (p.90):** "When using a Hope Feature, if you rolled with Hope for that action, the Hope you gain from that roll can be spent on that feature." **CRB (p.94):** Roll with Hope = gain a Hope; roll with Fear = GM gains a Fear.

- Enforce that narration doesn't modify damage — `internal/test/game/scenarios/spellcast_flavor_limits.lua`. Trigger: narration attempts to change mechanical damage. Effects: damage remains as defined by the spell or feature; flavor does not alter damage dice or modifiers. Requires: See section Requires. Notes: only explicit mechanics change rolls or damage.
  > **CRB (p.96):** "You can always flavor your magic to match the character you're playing, but that flavor won't give you access to new effects." Mechanical effects come from card text, not narration.

- Roll two Fear dice and take the higher on Spellcast — `internal/test/game/scenarios/environment_mirkwood_blight_chaos_magic.lua`. Trigger: Chaos Magic Locus or similar effect modifies Spellcast rolls. Effects: roll two Fear dice and keep the higher result when determining Hope/Fear. Requires: See section Requires. Notes: other modifiers still apply normally.
  > **CRB (p.248, Burning Heart of the Woods):** "Chaos Magic Locus - Passive: When a PC makes a Spellcast Roll, they must roll two Fear Dice and take the higher result."

### GM moves and improvised fear moves
Requires: Spotlight and GM moves; Resources; Adversary actions.
- Encode the specific GM move type and consequence — `internal/test/game/scenarios/gm_move_examples.lua`. Trigger: a player rolls with Fear, fails a roll, or gives a golden opportunity. Effects: GM makes a move (soft on Hope, hard on Fear) such as an attack, new threat, or stress mark. Requires: See section Requires. Notes: GM can spend Fear to interrupt and take additional moves.
  > **CRB (p.148-153):** GM makes moves when: a player rolls with Fear, fails a roll, does something with consequences, gives a golden opportunity, or the GM spends Fear to interrupt. Softer moves on Hope failures; harder moves on Fear results. Example GM moves (p.152): show world reaction, reveal danger, spotlight an adversary, force the group to split, mark Stress, capture something important, take away an opportunity permanently.

- Represent item theft and chase trigger — `internal/test/game/scenarios/gm_move_artifact_chase.lua`. Trigger: GM move introduces theft in a scene. Effects: item is lost unless the PCs engage a chase; use a Progress Countdown to recover the item and a Consequence Countdown for the thief escaping. Requires: See section Requires. Notes: countdowns tick on action rolls as specified by the chase setup.
  > **CRB (p.162):** Countdowns track a coming event by setting a die to a value and ticking it down to 0. **CRB (p.243, Bustling Marketplace):** "Sticky Fingers: A thief tries to steal something from a PC. The PC must succeed on an Instinct Roll to notice the thief or lose an item of the GM's choice... a Progress Countdown (6) to chase down the thief before the thief completes a Consequence Countdown (4) and escapes to their hideout."

- Encode the narrative fear move effect — `internal/test/game/scenarios/improvised_fear_move_shadow.lua`. Trigger: GM spends Fear to invoke a narrative fear move. Effects: apply the move's stated consequence (stress, damage, separation, etc.) and shift spotlight as appropriate. Requires: See section Requires. Notes: Fear spend is required for Fear Features and for interrupting the PCs.
  > **CRB (p.154):** "The GM can spend Fear to: Interrupt the players to make a move. Make an additional GM move. Use an adversary's Fear Feature. Use an environment's Fear Feature. Add an adversary's Experience to a roll." Fear spending per scene guide (p.155): Incidental 0-1, Minor 1-3, Standard 2-4, Major 4-8, Climactic 6-12.

- Encode the improvised fear move effect — `internal/test/game/scenarios/improvised_fear_move_noble_escape.lua`. Trigger: GM spends Fear for an improvised move. Effects: establish the escape or complication as a concrete change in the scene, then return spotlight to the PCs. Requires: See section Requires. Notes: use soft/hard move guidance based on roll outcome.
  > **CRB (p.156):** Improvised Fear moves should "redefine a scene, change the terms, raise the stakes, modify or move the location." Fear moves follow the same guidance as standard GM moves but cost Fear. Common elements: introducing new adversaries, a powerful transformation, or a strong negative environment effect.

- Apply a temporary damage bonus feature — `internal/test/game/scenarios/improvised_fear_move_rage_boost.lua`. Trigger: move grants a temporary bonus to damage. Effects: add the specified bonus to damage rolls for the duration; stacking is allowed unless stated otherwise. Requires: See section Requires. Notes: bonuses apply before thresholds and resistance.
  > **CRB (p.107):** "At the GM's discretion, most effects can stack. However, you can't stack conditions, advantage or disadvantage, or other effects that say you can't." Temporary damage bonuses are feature-specific. **[CRB: UNCLEAR — there is no generic "rage boost" mechanic; the stacking rule confirms bonuses stack, but the specific bonus is scenario-invented]**

### Social scenes
Requires: Social and economy; Resources; Conditions; Spotlight and GM moves.
- Apply hospitality ban, Hope loss, Stress, and unconscious condition — `internal/test/game/scenarios/social_village_elder_peace.lua`. Trigger: social consequence from a failed roll or GM move. Effects: remove access to hospitality, apply Hope loss and Stress as specified, and if rendered unconscious, the PC cannot move or act and cannot be targeted by attacks until an ally clears at least 1 HP or the party finishes a long rest. Requires: See section Requires. Notes: unconsciousness is a temporary incapacitation state, not a death move.
  > **CRB (p.106):** Unconscious IS defined as part of the Avoid Death death move: "Your character avoids death and faces the consequences. They temporarily drop unconscious. Your character can't move or act while unconscious, they can't be targeted by an attack, and they return to consciousness when an ally clears 1 or more of their marked Hit Points or when the party finishes a long rest." After falling unconscious, roll your Hope Die — if its value is equal to or under your character's level, they gain a scar (cross out a Hope slot permanently).

- Apply Preferential Treatment and The Runaround effects — `internal/test/game/scenarios/social_merchant_haggling.lua`. Trigger: Presence roll against a Merchant. Effects: on success, gain a discount on purchases; on a roll of 14 or lower, mark Stress from The Runaround; on failure, pay more and have disadvantage on future Presence rolls against the Merchant. Requires: See section Requires. Notes: discounts and penalties persist as long as the Merchant remains in the scene or as specified.
  > **CRB (p.204, Merchant):** "Preferential Treatment - Passive: A PC who succeeds on a Presence Roll against the Merchant gains a discount on purchases. A PC who fails on a Presence Roll against the Merchant must pay more and has disadvantage on future Presence Rolls against the Merchant. The Runaround - Passive: When a PC rolls a 14 or lower on a Presence Roll made against the Merchant, they must mark a Stress."

### Countdowns and progress systems
Requires: Countdowns; Core rolls and outcomes; Spotlight and GM moves; Resources.
- Tick countdown by result tier and award Hope/Fear changes — `internal/test/game/scenarios/progress_countdown_climb.lua`. Trigger: Progress Countdown tied to a climb. Effects: tick down by roll outcome (Critical: 3, Success with Hope: 2, Success with Fear: 1, Failure with Hope: 0, Failure with Fear: tick up 1) and resolve Hope/Fear on the roll itself. Requires: See section Requires. Notes: countdown triggers when it reaches 0; Failure with Fear may also trigger GM moves.
  > **CRB (p.162):** Dynamic Countdown Advancement table: Progress — Critical Success: tick down 3, Success with Hope: tick down 2, Success with Fear: tick down 1, Failure with Hope: no advancement, Failure with Fear: no advancement. Consequence — Critical: no advancement, Success with Hope: no advancement, Success with Fear: tick down 1, Failure with Hope: tick down 2, Failure with Fear: tick down 3. **[CRB: UNCLEAR — the scenario's "tick up 1 on Failure with Fear" for a progress countdown differs from the CRB's "no advancement"; the CRB uses a separate consequence countdown for negative ticks]**

- Implement looping countdown and cold gear advantage — `internal/test/game/scenarios/environment_caradhras_pass_icy_winds.lua`. Trigger: Loop Countdown (4) for Icy Winds. Effects: when it triggers, each PC makes a Strength reaction roll or marks Stress; if wearing cold gear, gain advantage; countdown resets and continues. Requires: See section Requires. Notes: loop countdowns reset to their starting value after triggering.
  > **CRB (p.162):** "Loop countdowns that reset to their starting value after their countdown effect is triggered." **CRB (p.247, Mountain Pass):** "Icy Winds - Reaction: Countdown (Loop 4). When the PCs enter the mountain pass, activate the countdown. When it triggers, all characters traveling through the pass must succeed on a Strength Reaction Roll or mark a Stress. A PC wearing clothes appropriate for extreme cold gains advantage on these rolls."

- Tick a long-term countdown by 1d4 — `internal/test/game/scenarios/environment_gondor_court_imperial_decree.lua`. Trigger: Imperial Decree action. Effects: spend Fear and tick a long-term countdown by 1d4; if it triggers, the decree is executed. Requires: See section Requires. Notes: long-term countdowns advance after rests or when specified, not every action roll.
  > **CRB (p.162):** "Long-term countdowns that advance after rests instead of action rolls." **CRB (p.251, Imperial Court):** "Imperial Decree - Action: Spend a Fear to tick down a long-term countdown related to the empire's agenda by 1d4."

- Set fear cap to 15 and activate long-term countdown (8) — `internal/test/game/scenarios/environment_dark_tower_usurpation_final_preparations.lua`. Trigger: Final Preparations when the environment first takes spotlight. Effects: designate the usurper, start Long-Term Countdown (8), and raise Fear cap to 15 while the environment remains active. Requires: See section Requires. Notes: countdown triggers the next phase when it reaches 0.
  > **CRB (p.250, Divine Usurpation):** "Final Preparations - Passive: When the environment first takes the spotlight, designate one adversary as the Usurper… Activate a Long-Term Countdown (8) as the Usurper assembles what they need to conduct the ritual. When it triggers, spotlight this environment to use the 'Beginning of the End' feature. While this environment remains in play, you can hold up to 15 Fear."

- Activate siege countdown (10) and tick based on fear/major damage — `internal/test/game/scenarios/environment_dark_tower_usurpation_beginning_of_end.lua`. Trigger: Final Preparations countdown triggers. Effects: activate Divine Siege Countdown (10); tick down by 1 when spotlighting the usurper; tick up by 1 when the usurper takes Major or greater damage; on trigger, the ritual succeeds. Requires: See section Requires. Notes: tick adjustments happen at the specified triggers only.
  > **CRB (p.250, Divine Usurpation):** "Beginning of the End - Reaction: When the 'Final Preparations' long-term countdown triggers, the usurper begins hammering on the gates of the Hallows Above. Activate a Divine Siege Countdown (10). Spotlight the Usurper to describe the Usurper's power. If the Usurper takes Major or greater damage, tick up the countdown by 1. When it triggers, the Usurper shatters the barrier between the Mortal Realm and the Hallows Above."

- Tie Fear outcomes to countdown ticks and summon on trigger — `internal/test/game/scenarios/environment_isengard_ritual_summoning.lua`. Trigger: Summoning ritual countdown. Effects: countdown ticks down when a PC rolls with Fear; on trigger, summon the specified adversary at the leader's position. Requires: See section Requires. Notes: if the leader is defeated, the countdown ends with no effect.
  > **CRB (p.246, Cult Ritual):** "The Summoning - Reaction: Countdown (6). When the PCs enter the scene or the cult begins the ritual to summon a demon, activate the countdown. The countdown ticks down when a PC rolls with Fear. When it triggers, summon a Minor Demon within Very Close range of the ritual's leader. If the leader is defeated, the countdown ends with no effect as the ritual fails."

- Apply Hope die size change and countdown clearance — `internal/test/game/scenarios/environment_isengard_ritual_desecrated_ground.lua`. Trigger: Desecrated Ground environmental effect. Effects: reduce Hope die size (e.g., to d10) while in the area; remove the effect by completing the specified Progress Countdown; on completion, restore normal Hope die. Requires: See section Requires. Notes: apply to all action rolls in the environment.
  > **CRB (p.246, Cult Ritual):** "Desecrated Ground - Passive: Cultists dedicated this place to the Fallen Gods, and their foul influence seeps into it. Reduce the PCs' Hope Die to a d10 while in this environment. The desecration can be removed with a Progress Countdown (6)."

- Activate consequence countdown and shift to Helms Deep Siege — `internal/test/game/scenarios/environment_helms_deep_siege_siege_weapons.lua`. Trigger: Siege Weapons action. Effects: start Consequence Countdown (6); when it triggers, breach the fortifications, gain 2 Fear, shift to the siege/battle environment, and spotlight it. Requires: See section Requires. Notes: if the countdown is dynamic, tick it by roll outcome (Failure with Fear: tick down by 3, Failure with Hope: tick down by 2, Success with Fear: tick down by 1); otherwise tick it once per action roll.
  > **CRB (p.249, Castle Siege):** "Siege Weapons (Environment Change) - Action: Consequence Countdown (6). The attacking force deploys siege weapons to try to raze the defenders' fortifications. Activate the countdown when the siege begins… When it triggers, the defenders' fortifications have been breached and the attackers flood inside. You gain 2 Fear, then shift to the Pitched Battle environment and spotlight it." **CRB (p.162):** Dynamic consequence advancement: Failure with Fear ticks down 3, Failure with Hope ticks down 2, Success with Fear ticks down 1, Success with Hope and Critical: no advancement.

- Apply progress countdown (8) and stress on failure — `internal/test/game/scenarios/environment_shadow_realm_impossible_architecture.lua`. Trigger: Impossible Architecture traversal. Effects: Progress Countdown (8) must be advanced to traverse; on failure, mark Stress in addition to other consequences. Requires: See section Requires. Notes: for dynamic progress countdowns, tick 2 on Success with Hope, 1 on Success with Fear, 3 on Critical Success, and no advancement on failures.
  > **CRB (p.250, Chaos Realm):** "Impossible Architecture - Passive: Up is down, down is right, right is left. Gravity and directionality themselves are in flux… requiring a Progress Countdown (8). On a failure, a PC must mark a Stress in addition to the roll's other consequences." **CRB (p.162):** Dynamic progress advancement as listed above.

- Loop countdown and reduce highest trait or mark stress — `internal/test/game/scenarios/environment_shadow_realm_everything_you_are.lua`. Trigger: Loop Countdown (1d4) for trait loss. Effects: on trigger, each PC makes a Presence reaction roll or reduces their highest trait by 1d4 unless they mark Stress equal to the reduction; loop resets and continues. Requires: See section Requires. Notes: lost trait points return on critical success or after escaping the realm.
  > **CRB (p.250, Chaos Realm):** "Everything You Are This Place Will Take From You - Action: … activate the countdown. When it triggers, all PCs must succeed on a Presence Reaction Roll or their highest trait is temporarily reduced by 1d4 unless they mark a number of Stress equal to its value. Any lost trait points are regained if the PC critically succeeds or escapes the Chaos Realm." **[CRB: UNCLEAR — the CRB says the countdown is activated by this action but does not specify the starting value as 1d4; the scenario fixture uses 1d4 as a randomized starting value]**

- Loop countdown and apply direct damage with half on success — `internal/test/game/scenarios/environment_mirkwood_blight_choking_ash.lua`. Trigger: Choking Ash loop countdown. Effects: on trigger, each PC makes a Strength or Instinct reaction roll; failure takes full direct damage, success takes half. Requires: See section Requires. Notes: direct damage cannot be reduced by Armor Slots.
  > **CRB (p.248, Burning Heart of the Woods):** "Choking Ash - Reaction: Countdown (Loop 6). When the PCs enter the Burning Heart of the Woods, activate the countdown. When it triggers, all characters must make a Strength or Instinct Reaction Roll. Targets who fail take 4d6+5 direct physical damage. Targets who succeed take half damage. Protective masks or clothes give advantage on the reaction roll."

- Apply tick deltas by outcome tiers — `internal/test/game/scenarios/environment_misty_ascent_progress.lua`. Trigger: dynamic progress countdown tied to action rolls. Effects: tick down 3 on critical, 2 on success with Hope, 1 on success with Fear; no advancement on failures. Requires: See section Requires. Notes: consequences may still apply on failures.
  > **CRB (p.162):** Dynamic Countdown Advancement — Progress: Critical Success tick down 3, Success with Hope tick down 2, Success with Fear tick down 1, Failure with Hope no advancement, Failure with Fear no advancement.

- Intercept failure and apply stress in place of countdown penalty — `internal/test/game/scenarios/environment_misty_ascent_pitons.lua`. Trigger: failure on climb while using pitons. Effects: mark Stress instead of ticking the countdown up (or applying the failure penalty). Requires: See section Requires. Notes: requires access to the pitons feature to replace the failure consequence.
  > **CRB (p.244, Cliffside Ascent):** "Pitons Left Behind - Passive: Previous climbers left behind large metal rods that climbers can use to aid their ascent. If a PC using the pitons fails an action roll to climb, they can mark a Stress instead of ticking the countdown."

- Tie action rolls to the escape countdown while hazards unfold — `internal/test/game/scenarios/environment_osgiliath_ruins_apocalypse_then.lua`. Trigger: action rolls during an apocalypse replay. Effects: progress countdown advances per outcome while hazards resolve as GM moves; when countdown triggers, the PCs escape the disaster. Requires: See section Requires. Notes: countdown advances on action rolls; hazards are additional GM moves.
  > **CRB (p.247, Haunted City):** "Apocalypse Then - Action: Spend a Fear to manifest the echo of a past disaster that ravaged the city. Activate a Progress Countdown (5) as the disaster replays around the PCs. To complete the countdown and escape the catastrophe, the PCs must overcome threats… while recalling history and finding clues to escape the inevitable."

### Environment hazards, damage, and conditions
Requires: Environment actions; Core rolls and outcomes; Damage pipeline; Conditions; Movement and range; Resources; Dice modifiers.
- Apply movement and damage severity on failure — `internal/test/game/scenarios/environment_caradhras_pass_avalanche.lua`. Trigger: Avalanche action. Effects: targets make Agility or Strength reaction rolls; on failure, they are moved down the slope to Far range, take listed damage, and mark Stress; on success, mark Stress. Requires: See section Requires. Notes: climbing gear grants advantage; forced movement uses range bands.
  > **CRB (p.247, Mountain Pass):** "Avalanche - Action: Spend a Fear to carve the mountain with an icy torrent, causing an avalanche. All PCs in its path must succeed on an Agility or Strength Reaction Roll or be bowed over and carried down the mountain. A PC using rope, pitons, or other climbing gear gains advantage on this roll. Targets who fail are knocked down the mountain to Far range, take 2d20 physical damage, and must mark a Stress. Targets who succeed must mark a Stress."

- Apply movement check and hazard damage — `internal/test/game/scenarios/environment_prancing_pony_bar_fight.lua`. Trigger: Bar Fight hazard when moving through the tavern. Effects: PC makes Agility or Presence roll to move; on failure, take hazard damage (1d6+2) from collateral. Requires: See section Requires. Notes: hazard persists until the scene ends or is stopped.
  > **CRB (p.244, Local Tavern):** "Bar Fight - Action: Spend a Fear to have a bar fight erupt in the tavern. When a PC tries to move through the tavern while the fight persists, they must succeed on an Agility or Presence Roll or take 1d6+2 physical damage from a wild swing or thrown object."

- Apply river movement and conditional stress on success — `internal/test/game/scenarios/environment_bruinen_ford_undertow.lua`. Trigger: Undertow action. Effects: target makes Agility reaction roll; on failure take damage and become Vulnerable while in the river; on success mark Stress. Requires: See section Requires. Notes: Vulnerable ends when they escape the river or clear the condition.
  > **CRB (p.245, Raging River):** "Undertow - Action: Spend a Fear to catch a PC in the undertow. They must make an Agility Reaction Roll. On a failure, they take 1d6+1 physical damage and are moved a Close distance down the river, becoming Vulnerable until they get out of the river. On a success, they must mark a Stress."

- Tie failure with Fear to immediate undertow action — `internal/test/game/scenarios/environment_bruinen_ford_dangerous_crossing.lua`. Trigger: Dangerous Crossing progress countdown; a failure with Fear. Effects: immediate Undertow action without spending Fear. Requires: See section Requires. Notes: progress countdown advances per action roll outcomes.
  > **CRB (p.245, Raging River):** "Dangerous Crossing - Passive: Crossing the river requires the party to complete a Progress Countdown (4). A PC who rolls a failure with Fear is immediately targeted by the 'Undertow' action without requiring a Fear to be spent on the feature."

- Apply advantage, bonus damage, or Relentless feature — `internal/test/game/scenarios/environment_isengard_ritual_blasphemous_might.lua`. Trigger: Blasphemous Might action that imbues an adversary. Effects: grant advantage on attacks, add extra damage, or grant Relentless (2) as specified; if Fear spent, apply all benefits. Requires: See section Requires. Notes: Relentless allows additional spotlights per GM turn.
  > **CRB (p.246, Cult Ritual):** "Blasphemous Might - Action: A portion of the ritual's power is diverted into a cult member to fight off interlopers. Choose one adversary to become Imbued with terrible magic until the scene ends or they're defeated. An Imbued adversary immediately takes the spotlight and gains one of the following benefits, or all three if you spend a Fear: They gain advantage on all attacks. They deal an extra 1d10 damage on a successful attack. They gain the following feature: Relentless (2) - Passive: This adversary can be spotlighted up to two times per GM turn. Spend Fear as usual to spotlight them."

- Redirect an attack to the ally by marking Stress — `internal/test/game/scenarios/environment_isengard_ritual_complete.lua`. Trigger: Complete the Ritual reaction. Effects: an ally within Very Close range can mark Stress to become the new target of an incoming attack or spell against the ritual leader. Requires: See section Requires. Notes: applies only when leader is targeted; range restriction applies.
  > **CRB (p.246, Cult Ritual):** "Complete the Ritual - Reaction: If the ritual's leader is targeted by an attack or spell, an ally within Very Close range of them can mark a Stress to be targeted by that attack or spell instead."

- Roll 1d4 stress on failure with Fear — `internal/test/game/scenarios/environment_dark_tower_usurpation_ritual_nexus.lua`. Trigger: failure with Fear against the usurper. Effects: mark 1d4 Stress from backlash. Requires: See section Requires. Notes: stress marking follows standard rules; if no Stress slots, mark HP instead.
  > **CRB (p.250, Divine Usurpation):** "Ritual Nexus - Reaction: On any failure with Fear against the Usurper, the PC must mark 1d4 Stress from the backlash of magical power."

- Clear 2 HP and increase the usurper's stats after the action — `internal/test/game/scenarios/environment_dark_tower_usurpation_godslayer.lua`. Trigger: Godslayer action after the siege countdown triggers. Effects: usurper clears 2 HP and gains a stat boost (Difficulty, damage, attack modifier, or a new feature). Requires: See section Requires. Notes: requires spending Fear to activate.
  > **CRB (p.250, Divine Usurpation):** "Godslayer - Action: If the Divine Siege Countdown has triggered, you can immediately use the 'Godslayer' feature without spending Fear to make an additional GM move. The Usurper clears 2 HP, increase their Difficulty, damage, attack modifier, or give them a new feature from the slain god."

- Apply direct damage on failure and stress on success — `internal/test/game/scenarios/environment_shadow_realm_unmaking.lua`. Trigger: Unmaking action. Effects: target makes Strength reaction roll; on failure take direct magic damage, on success mark Stress. Requires: See section Requires. Notes: direct damage ignores Armor Slots.
  > **CRB (p.250, Chaos Realm):** "Unmaking - Action: Spend a Fear to force a PC to make a Strength Reaction Roll. On a failure, they take 4d10 direct magic damage. On a success, they must mark a Stress." Direct damage bypasses Armor Slots per the glossary definition.

- Deduct Hope on Fear outcome and grant GM Fear if it was last Hope — `internal/test/game/scenarios/environment_shadow_realm_disorienting_reality.lua`. Trigger: roll with Fear in Disorienting Reality. Effects: PC loses 1 Hope; if it was their last Hope, GM gains 1 Fear. Requires: See section Requires. Notes: applies only on Fear outcomes in the environment.
  > **CRB (p.250, Chaos Realm):** "Disorienting Reality - Reaction: On a result with Fear, you can ask the PC to describe which of their fears the Chaos Realm evokes as a vision of reality unmakes and reconstitutes itself to the PC. The PC loses a Hope. If it is their last Hope, you gain a Fear."

- Convert a Fear outcome to Hope and mark Stress — `internal/test/game/scenarios/environment_rivendell_sanctuary_relentless_hope.lua`. Trigger: Relentless Hope reaction. Effects: PC can mark Stress to change a Fear result to a Hope result. Requires: See section Requires. Notes: once per scene per PC; reaction rolls are still exempt from Hope/Fear gain.
  > **CRB (p.246, Hallowed Temple):** "Relentless Hope - Reaction: Once per scene, each PC can mark a Stress to turn a result with Fear into a result with Hope."

- Apply Restrained + Vulnerable, escape roll damage, and Hope loss — `internal/test/game/scenarios/environment_mirkwood_blight_grasping_vines.lua`. Trigger: Grasping Vines action. Effects: on failed reaction roll, target becomes Restrained and Vulnerable; escaping via action roll deals listed damage and costs Hope. Requires: See section Requires. Notes: temporary conditions clear on a successful escape roll or by dealing enough damage to the vines.
  > **CRB (p.248, Burning Heart of the Woods):** "Grasping Vines - Action: Animate vines bristling with thorns whip out from the underbrush to ensnare the PCs. A target must succeed on an Agility Reaction Roll or become Restrained and Vulnerable until they break free, clearing both conditions, with a successful Finesse or Strength Roll or by dealing 10 damage to the vines. When the target makes a roll to escape, they take 1d8+4 physical damage and lose a Hope."

- Apply area reaction roll and half damage on success — `internal/test/game/scenarios/environment_mirkwood_blight_charcoal_constructs.lua`. Trigger: Charcoal Constructs action. Effects: targets in the area make Agility reaction rolls; on failure take full damage, on success take half. Requires: See section Requires. Notes: area is within Close range of the chosen point.
  > **CRB (p.248, Burning Heart of the Woods):** "Charcoal Constructs - Action: Warped animals wreathed in indigo flame trample through a point of your choice. All targets within Close range of that point must make an Agility Reaction Roll. Targets who fail take 3d12+3 physical damage. Targets who succeed take half damage instead."

- Defer the damage until a follow-up action fails to save — `internal/test/game/scenarios/environment_misty_ascent_fall.lua`. Trigger: fall setup after a failed climb. Effects: damage is deferred until the next action; if the follow-up action fails, apply fall damage. Requires: See section Requires. Notes: use falling damage guidance (Very Close: 1d10+3; Close: 1d20+5; Far/Very Far: 1d100+15 or death at GM discretion).
  > **CRB (p.244, Cliffside Ascent):** "Fall - Action: Spend a Fear to have a PC's handhold fail, plummeting them toward the ground. If they aren't saved on their next action, they hit the ground and tick up the countdown by 2. The PC takes 1d12 physical damage if the countdown is between 8 and 12, 2d12 between 4 and 7, and 3d12 at 3 or lower." **CRB (p.168):** "A fall from Very Close range deals 1d10+3 physical damage. Close range: 1d20+5. Far or Very Far: 1d100+15 or death at the GM's discretion."

- Apply extra Hope cost to healing or rest effects — `internal/test/game/scenarios/environment_moria_ossuary_no_place_living.lua`. Trigger: healing or rest effects in a no-place-for-the-living environment. Effects: spend an additional Hope to clear HP or use a healing feature; if it already costs Hope, add +1 Hope. Requires: See section Requires. Notes: applies to healing effects and rest moves that clear HP.
  > **CRB (p.251, Necromancer's Ossuary):** "No Place for the Living - Passive: A feature or action that clears HP requires spending a Hope to use. If it already costs Hope, a PC must spend an additional Hope."

- Roll d4 and distribute healing across undead — `internal/test/game/scenarios/environment_moria_ossuary_aura_of_death.lua`. Trigger: Aura of Death action. Effects: roll a d4; each undead within range clears HP and Stress equal to the result, distributed as they choose. Requires: See section Requires. Notes: once per scene if specified by the feature.
  > **CRB (p.251, Necromancer's Ossuary):** "Aura of Death - Action: Once per scene, roll a d4. Each undead within Far range of the Necromancer can clear HP and Stress equal to the result divided between HP and Stress however they choose."

- Apply reaction roll and 4d8+8 damage on failure — `internal/test/game/scenarios/environment_moria_ossuary_skeletal_burst.lua`. Trigger: Skeletal Burst action. Effects: targets in Close range make Agility reaction rolls; on failure take 4d8+8 physical damage, on success take half if specified. Requires: See section Requires. Notes: area of effect within Close range.
  > **CRB (p.251, Necromancer's Ossuary):** "Skeletal Burst - Action: All targets within Close range of a point must succeed on an Agility Reaction Roll or take 4d8+8 physical damage from skeletal shrapnel as part of the ossuary detonates around them."

- Apply damage, Restrained condition, and escape checks — `internal/test/game/scenarios/environment_old_forest_grove_barbed_vines.lua`. Trigger: Barbed Vines action. Effects: on failed reaction roll, target takes damage and becomes Restrained; escape via Finesse or Strength roll or by dealing listed damage to vines. Requires: See section Requires. Notes: Restrained prevents movement; escape is a separate action roll.
  > **CRB (p.243, Abandoned Grove):** "Barbed Vines - Action: Pick a point within the grove. All targets within Very Close range of that point must succeed on an Agility Reaction Roll or take 1d8+3 physical damage and become Restrained by barbed vines. Restrained lasts until they're freed with a successful Finesse or Strength roll or by dealing at least 6 damage to the vines."

- Apply physical resistance and stress-based movement — `internal/test/game/scenarios/environment_osgiliath_ruins_ghostly_form.lua`. Trigger: Ghostly Form environment. Effects: adversaries gain resistance to physical damage and may mark Stress to move through solid objects within Close range. Requires: See section Requires. Notes: resistance halves physical damage before thresholds.
  > **CRB (p.247, Haunted City):** "Ghostly Form - Passive: Adversaries who appear here are of a ghostly form. They have resistance to physical damage and can mark a Stress to move up to Close range through solid objects."

- Apply area reaction roll, damage, and stress on failure — `internal/test/game/scenarios/environment_pelennor_battle_war_magic.lua`. Trigger: War Magic action. Effects: targets within the area make Agility reaction rolls; on failure take damage and mark Stress, on success take half damage and may still mark Stress if specified. Requires: See section Requires. Notes: GM spends Fear to activate the action.
  > **CRB (p.249, Pitched Battle):** "War Magic - Action: Spend a Fear… Pick a point on the battlefield within Very Far range of the magic. All targets within Close range of that point must make an Agility Reaction Roll. Targets who fail take 3d12+8 magic damage and must mark a Stress."

- Restrict movement without a successful Agility roll — `internal/test/game/scenarios/environment_pelennor_battle_adrift.lua`. Trigger: Adrift on a Sea of Steel movement rule. Effects: PC must succeed on an Agility roll to move; if an adversary is within Melee range, they must mark Stress to attempt the roll. Requires: See section Requires. Notes: movement is limited to Close range on a success.
  > **CRB (p.249, Pitched Battle):** "Adrift on a Sea of Steel - Passive: Traversing a battlefield during active combat is extremely dangerous. A PC must succeed on an Agility Roll to move at all, and can only go up to Close range on a success. If an adversary is within Melee range of them, they must mark a Stress to make an Agility Roll to move."

- Apply the fear loss and advantage to the first strike — `internal/test/game/scenarios/environment_waylayers_where_did_they_come_from.lua`. Trigger: surprise/ambush feature. Effects: GM gains Fear and the first PC attack roll gains advantage. Requires: See section Requires. Notes: advantage adds a d6; ambush is an environment action or reaction.
  > **CRB (p.243, Ambushers):** "Where Did They Come? - Reaction: When a PC starts the ambush on unsuspecting adversaries, you lose 2 Fear and the first attack roll a PC makes has advantage." Note: this is the PC-ambushes-adversaries version. **[CRB: UNCLEAR — the scenario may be referencing the inverse (adversaries ambush PCs) which is the Ambushed environment's "Surprise!" action]**

- Award Fear to the GM and immediate spotlight shift — `internal/test/game/scenarios/environment_waylayers_surprise.lua`. Trigger: surprise action. Effects: GM gains 2 Fear and spotlight shifts to an ambushing adversary. Requires: See section Requires. Notes: surprise is an environment action that initiates combat.
  > **CRB (p.243, Ambushed):** "Surprise! - Action: The ambushers reveal themselves to the party, you gain 2 Fear, and the spotlight immediately shifts to one of the ambushing adversaries."

- Award Fear to the GM and shift spotlight — `internal/test/game/scenarios/environment_waylaid_surprise.lua`. Trigger: surprise action. Effects: GM gains Fear and spotlight shifts to an adversary. Requires: See section Requires. Notes: spotlight shift means the GM now acts and spotlights the next actor (often the ambusher).
  > **CRB (p.243, Ambushed):** "Surprise!" as above. Same mechanic, different scenario fixture name.

### Environment spawns and scene setup
Requires: Environment actions; Spotlight and GM moves; Adversary actions; Movement and range; Resources.
- Spawn two eagles at Very Far range — `internal/test/game/scenarios/environment_caradhras_pass_raptor_nest.lua`. Trigger: Raptor Nest reaction. Effects: two Giant Eagles appear at Very Far range of a chosen PC and enter the scene. Requires: See section Requires. Notes: range bands determine immediate engagement distance.
  > **CRB (p.247, Mountain Pass):** "Raptor Nest - Reaction: When the PCs enter the raptors' hunting grounds, two Giant Eagles appear at Very Far range of a chosen PC, identifying the PCs as likely prey."

- Spawn multiple adversaries based on party size — `internal/test/game/scenarios/environment_bree_outpost_wrong_place.lua`. Trigger: Wrong Place, Wrong Time reaction. Effects: spawn a kneebreaker, lackeys equal to party size, and a lieutenant; larger parties add a hexer or sniper; they appear at Close range. Requires: See section Requires. Notes: spend Fear to trigger the encounter.
  > **CRB (p.245, Outpost Town):** "Wrong Place, Wrong Time - Reaction: At night, or when the party is alone in a back alley, you can spend a Fear to introduce a group of thieves who try to rob them. The thieves appear at Close range of a chosen PC and include a Jagged Knife Kneebreaker, as many Lackeys as there are PCs, and a Lieutenant. For a larger party, add a Hexer or Sniper."

- Place the adversary and trigger its action — `internal/test/game/scenarios/environment_bruinen_ford_patient_hunter.lua`. Trigger: Patient Hunter action. Effects: spend Fear to summon a Glass Snake within Close range and immediately spotlight it to use its action. Requires: See section Requires. Notes: spotlighting allows the adversary to act immediately.
  > **CRB (p.245, Raging River):** "Patient Hunter - Action: Spend a Fear to summon a Glass Snake within Close range of a chosen PC. The Snake appears in or near the river and immediately takes the spotlight to use their 'Spinning Serpent' action."

- Spawn adversaries based on party size and spotlight the knight — `internal/test/game/scenarios/environment_helms_deep_siege_reinforcements.lua`. Trigger: Reinforcements action. Effects: summon a Knight of the Realm, Tier 3 Minions equal to party size, and two additional adversaries; the Knight is immediately spotlighted. Requires: See section Requires. Notes: reinforcements appear within Far range of a chosen PC.
  > **CRB (p.249, Castle Siege):** "Reinforcements! - Action: Summon a Knight of the Realm, a number of Tier 3 Minions equal to the number of PCs, and two adversaries of your choice within Far range of a chosen PC as reinforcements. The Knight of the Realm immediately takes the spotlight."

- Summon 1d4+2 troops and trigger their group attack — `internal/test/game/scenarios/environment_dark_tower_usurpation_defilers_abound.lua`. Trigger: Defilers Abound action. Effects: spend 2 Fear to summon 1d4+2 Fallen Shock Troops within Close range of the usurper and immediately spotlight them to use Group Attack. Requires: See section Requires. Notes: group attack uses shared roll and combined damage.
  > **CRB (p.250, Divine Usurpation):** "Defilers Abound - Action: Spend 2 Fear to summon 1d4+2 Fallen Shock Troops that appear within Close range of the Usurper to assist their divine siege. Immediately spotlight the Shock Troops to use a 'Group Attack' action."

- Spawn 1 abomination, 1 corruptor, and 2d6 thralls — `internal/test/game/scenarios/environment_shadow_realm_predators.lua`. Trigger: Outer Realms Predators action. Effects: spend Fear to summon the listed adversaries at Close range of a chosen PC; immediately spotlight one, with optional Fear to auto-succeed its standard attack. Requires: See section Requires. Notes: auto-success applies only to the spotlighted adversary's standard attack.
  > **CRB (p.250, Chaos Realm):** "Outer Realms Predators - Action: Spend a Fear to summon an Outer Realms Abomination, an Outer Realms Corruptor, and 2d6 Outer Realms Thralls, who appear at Close range of a chosen PC in defiance of logic and causality. Immediately spotlight one of these adversaries, and you can spend an additional Fear to automatically succeed on that adversary's standard attack."

- Spawn multiple adversaries near the priest — `internal/test/game/scenarios/environment_rivendell_sanctuary_divine_censure.lua`. Trigger: Divine Censure reaction. Effects: spend Fear to summon a High Seraph and 1d4 Bladed Guards within Close range of the senior priest. Requires: See section Requires. Notes: spawn occurs only after trespass or blasphemy.
  > **CRB (p.246, Hallowed Temple):** "Divine Censure - Reaction: When the PCs have trespassed, blasphemed, or offended the clergy, you can spend a Fear to summon a High Seraph and 1d4 Bladed Guards within Close range of the senior priest to reinforce their will."

- Summon 1d6 rotted zombies, two perfected, or a legion — `internal/test/game/scenarios/environment_moria_ossuary_they_keep_coming.lua`. Trigger: They Just Keep Coming! action. Effects: spend Fear to summon the specified undead at Close range of a chosen PC. Requires: See section Requires. Notes: choice of spawn is per the feature.
  > **CRB (p.251, Necromancer's Ossuary):** "They Just Keep Coming! - Action: Spend a Fear to summon 1d6 Rotted Zombies, two Perfected Zombies, or a Zombie Legion, who appear at Close range of a chosen PC."

- Spawn adversaries equal to party size and shift spotlight — `internal/test/game/scenarios/environment_old_forest_grove_not_welcome.lua`. Trigger: You Are Not Welcome Here action. Effects: summon a Young Dryad, two Sylvan Soldiers, and Minor Treants equal to party size; spotlight shifts to a guardian adversary. Requires: See section Requires. Notes: the GM immediately takes a turn with one of the summoned guardians as the next actor before any PC acts.
  > **CRB (p.243, Abandoned Grove):** "You Are Not Welcome Here - Action: A Young Dryad, two Sylvan Soldiers, and a number of Minor Treants equal to the number of PCs appear to confront the party for their intrusion."

- Spawn the elemental near a chosen PC and shift spotlight — `internal/test/game/scenarios/environment_old_forest_grove_defiler.lua`. Trigger: Defiler action. Effects: spend Fear to summon a Minor Chaos Elemental within Far range of a chosen PC and immediately spotlight it. Requires: See section Requires. Notes: spotlighted adversary acts immediately.
  > **CRB (p.243, Abandoned Grove):** "Defiler - Action: Spend a Fear to summon a Minor Chaos Elemental drawn to the echoes of violence and discord. They appear within Far range of a chosen PC and immediately take the spotlight."

- Spawn new adversaries and spotlight the knight — `internal/test/game/scenarios/environment_pelennor_battle_reinforcements.lua`. Trigger: Reinforcements action in a pitched battle. Effects: summon a Knight of the Realm, Tier 3 Minions equal to party size, and two adversaries; spotlight the Knight. Requires: See section Requires. Notes: reinforcements appear within Far range of a chosen PC.
  > **CRB (p.249, Pitched Battle):** "Reinforcements! - Action: Summon a Knight of the Realm, a number of Tier 3 Minions equal to the number of PCs, and two adversaries of your choice within Far range of a chosen PC as reinforcements. The Knight of the Realm immediately takes the spotlight."

### Environment social, economy, and information
Requires: Social and economy; Core rolls and outcomes; Resources; Dice modifiers; Spotlight and GM moves; Countdowns.
- Apply advantage on dispel after critical knowledge success — `internal/test/game/scenarios/environment_caradhras_pass_engraved_sigils.lua`. Trigger: Knowledge roll about Engraved Sigils. Effects: on critical success, learn sigil details and gain advantage on a dispel roll. Requires: See section Requires. Notes: advantage adds a d6; dispel is a separate action roll.
  > **CRB (p.247, Mountain Pass):** "Engraved Sigils - Passive: Large markings and engravings have been made in the mountainside. A PC with a relevant background or Experience identifies them as weather magic increasing the power of the icy winds. A PC who succeeds on a Knowledge Roll can recall information about the sigils… If a PC critically succeeds, they recognize the sigils are of a style created by ridgeborne enchanters and they gain advantage on a roll to dispel the sigils."

- Model item loss and chase triggers — `internal/test/game/scenarios/environment_bree_market_sticky_fingers.lua`. Trigger: Sticky Fingers action. Effects: PC must succeed on an Instinct roll to notice the theft; on failure, lose an item and start a chase using Progress Countdown (6) vs Consequence Countdown (4). Requires: See section Requires. Notes: countdowns tick on action rolls during the chase.
  > **CRB (p.243, Bustling Marketplace):** "Sticky Fingers - Action: A thief tries to steal something from a PC. The PC must succeed on an Instinct Roll to notice the thief or lose an item of the GM's choice to a Close distance. To retrieve the stolen item, the PCs must complete a Progress Countdown (6) to chase down the thief before the thief completes a Consequence Countdown (4) and escapes to their hideout."

- Introduce a quest item and its non-gold cost — `internal/test/game/scenarios/environment_bree_market_unexpected_find.lua`. Trigger: Unexpected Find action. Effects: reveal a rare item and establish a non-gold cost or favor required to obtain it. Requires: See section Requires. Notes: cost is resolved via a social or quest action.
  > **CRB (p.243, Bustling Marketplace):** "Unexpected Find - Action: Reveal to the PCs that one of the merchants has something they want or need, such as food from their home, a rare book, magical components, a dubious treasure map, or a magical key."

- Separate the PC from the group and apply positioning — `internal/test/game/scenarios/environment_bree_market_crowd_closes_in.lua`. Trigger: Crowd Closes In reaction when a PC splits from the group. Effects: PC is moved to a new position and separated from allies, affecting range and line of sight. Requires: See section Requires. Notes: repositioning: GM updates ranges/line of sight without a roll.
  > **CRB (p.243, Bustling Marketplace):** "Crowd Closes In - Reaction: When one of the PCs splits from the group, the crowds shift and cut them off from the party."

- Spend gold and apply advantage die — `internal/test/game/scenarios/environment_bree_market_tip_the_scales.lua`. Trigger: Tip the Scales passive. Effects: spend a handful of gold to gain advantage on a Presence roll. Requires: See section Requires. Notes: advantage adds a d6; gold is tracked in handfuls/bags/chests.
  > **CRB (p.243, Bustling Marketplace):** "Tip the Scales - Passive: PCs can gain advantage on a Presence Roll by offering a handful of gold as part of the interaction."

- Represent the ongoing social pressure from the society — `internal/test/game/scenarios/environment_bree_outpost_broken_compass.lua`. Trigger: Society of the Broken Compass passive. Effects: ongoing social pressure and rivalry context that shapes rolls and GM moves. Requires: See section Requires. Notes: social constraint: impose disadvantage on relevant social rolls or add a complication when the PCs ignore the society.
  > **CRB (p.245, Outpost Town):** "Society of the Broken Compass - Passive: An adventuring society that maintains a chapterhouse here, where heroes trade boasts and rumors, drink to their imagined successes, and scheme to undermine their rivals."

- Represent rivalry hooks and competitive pressures — `internal/test/game/scenarios/environment_bree_outpost_rival_party.lua`. Trigger: Rival Party passive. Effects: establish a rival group with a hook tied to a PC and maintain competitive pressure in social scenes. Requires: See section Requires. Notes: rivalry influences future rolls and choices.
  > **CRB (p.245, Outpost Town):** "Rival Party - Passive: Another adventuring party is here, seeking the same treasure or leads as the PCs."

- Represent the narrative prompt and resulting tension — `internal/test/game/scenarios/environment_bree_outpost_shakedown.lua`. Trigger: It'd Be a Shame If Something Happened to Your Store action. Effects: introduce a shakedown with immediate tension and a choice to intervene. Requires: See section Requires. Notes: consequences depend on player response and GM move.
  > **CRB (p.245, Outpost Town):** "It'd Be a Shame If Something Happened to Your Store - Action: The PCs witness as agents of a local crime boss shake down a general goods store."

- Map outcome to rumor selection and stress on failure — `internal/test/game/scenarios/environment_bree_outpost_rumors.lua`. Trigger: Rumors Abound passive. Effects: Presence roll outcome determines how many rumors are learned; failures require marking Stress to gain one rumor. Requires: See section Requires. Notes: Stress marking rules apply; the GM chooses rumor content.
  > **CRB (p.245, Outpost Town):** "Rumors Abound - Passive: Gossip is the fastest-traveling currency in the realm. A PC can inquire about local events, rumors, and potential work with a Presence Roll. What they learn depends on the outcome: Critical Success: Learn about two major events… Success with Hope: Learn about two events… Success with Fear: Learn an alarming rumor… Any Failure: The locals respond poorly… The PC must mark a Stress to learn one relevant rumor."

- Model the NPC hook and immediate agenda — `internal/test/game/scenarios/environment_prancing_pony_someone_comes_to_town.lua`. Trigger: Someone Comes to Town action. Effects: introduce an NPC with a job offer or background tie; establish immediate agenda. Requires: See section Requires. Notes: no roll required unless the PCs challenge the introduction.
  > **CRB (p.244, Local Tavern):** "Someone Comes to Town - Action: Introduce a significant NPC who wants to hire the party for something or who relates to a PC's background."

- Model the narrative reveal and its hooks — `internal/test/game/scenarios/environment_prancing_pony_mysterious_stranger.lua`. Trigger: Mysterious Stranger action. Effects: reveal a concealed NPC and provide hooks or questions for the party. Requires: See section Requires. Notes: social rolls may be used to learn more.
  > **CRB (p.244, Local Tavern):** "Mysterious Stranger - Action: Reveal a stranger concealing their identity, lurking in a shaded booth."

- Map outcome to number of details and stress choice — `internal/test/game/scenarios/environment_prancing_pony_talk.lua`. Trigger: What's the Talk of the Town passive. Effects: Presence roll sets number of details learned; on failure, mark Stress to learn one detail. Requires: See section Requires. Notes: if the PC has no Stress slots left, they cannot take the extra detail option.
  > **CRB (p.244, Local Tavern):** "What's the Talk of the Town? - Passive: A PC can ask the bartender, staff, or patrons about local events, rumors, and potential work with a Presence Roll. On a success, they can pick two of the below details to learn—or three if they critically succeed. On a failure, they can pick one and mark a Stress."

- Roll and apply gold payout vs stress — `internal/test/game/scenarios/environment_prancing_pony_sing.lua`. Trigger: Sing For Your Supper passive. Effects: Presence roll to perform; on success gain 1d4 handfuls of gold (2d4 on crit); on failure mark Stress. Requires: See section Requires. Notes: gold is tracked in handfuls.
  > **CRB (p.244, Local Tavern):** "Sing For Your Supper - Passive: A PC can perform one time for the guests by making a Presence Roll. On a success, they earn 1d4 handfuls of gold (2d4 if they critically succeed). On a failure, they mark a Stress."

- Reveal a secret route with Instinct/Knowledge success — `internal/test/game/scenarios/environment_helms_deep_siege_secret_entrance.lua`. Trigger: Secret Entrance passive. Effects: on successful Instinct or Knowledge roll, discover a hidden path into the castle. Requires: See section Requires. Notes: roll Difficulty is environment Difficulty unless otherwise specified.
  > **CRB (p.249, Castle Siege):** "Secret Entrance - Passive: A PC can find or recall a secret way into the castle with a successful Instinct or Knowledge Roll."

- Model ongoing social pressure and favor exchanges — `internal/test/game/scenarios/environment_gondor_court_rival_vassals.lua`. Trigger: Rival Vassals passive. Effects: establish vassal factions and favor exchange pressures that inform social rolls and GM moves. Requires: See section Requires. Notes: social constraint: apply disadvantage when a PC acts against court norms, or introduce a cost/favor requirement on success.
  > **CRB (p.251, Imperial Court):** "Rival Vassals - Passive: The PCs can find imperial subjects, vassals, and supplicants in the court, each vying for favor, seeking proximity to power, exchanging favors for loyalty, and elevating their status above others'."

- Apply Presence reaction and stress or acceptance on failure — `internal/test/game/scenarios/environment_gondor_court_gravity_of_empire.lua`. Trigger: Gravity of Empire action. Effects: target makes Presence reaction roll; on failure mark all Stress or accept the offer; on success mark 1d4 Stress. Requires: See section Requires. Notes: if already at max Stress, the target must accept or exile.
  > **CRB (p.251, Imperial Court):** "The Gravity of Empire - Action: Spend a Fear to present a PC with a golden opportunity or offer to satisfy a major goal in exchange for obeying or supporting the empire. The target must make a Presence Reaction Roll. On a failure, they must mark all their Stress or accept the offer. If they have already marked all their Stress, they must accept the offer or exile themselves from the empire. On a success, they must mark 1d4 Stress as they're taxed by temptation."

- Spend Fear to trigger witness and Instinct reaction to notice — `internal/test/game/scenarios/environment_gondor_court_eyes_everywhere.lua`. Trigger: Eyes Everywhere reaction. Effects: spend Fear to introduce a witness; PC must make Instinct reaction roll to notice and intercept. Requires: See section Requires. Notes: failure may lead to exposure or further consequences.
  > **CRB (p.251, Imperial Court):** "Eyes Everywhere - Reaction: On a result with Fear, you can spend a Fear to have someone loyal to the empire overhear seditious talk within the court. A PC must succeed on an Instinct Reaction Roll to notice that the group has been overheard so they can try to intercept the witness before the PCs are exposed."

- Apply disadvantage to nonconforming Presence rolls — `internal/test/game/scenarios/environment_gondor_court_all_roads.lua`. Trigger: All Roads Lead Here passive. Effects: Presence rolls that oppose imperial norms are made with disadvantage. Requires: See section Requires. Notes: disadvantage adds a d6 penalty and cancels with advantage in the same pool.
  > **CRB (p.251, Imperial Court):** "All Roads Lead Here - Passive: While in the Imperial Court, a PC has disadvantage on Presence Rolls made to take actions that don't fit the imperial way of life or support the empire's dominance."

- Spend 2 Hope to refresh a limited-use ability — `internal/test/game/scenarios/environment_dark_tower_usurpation_divine_blessing.lua`. Trigger: Divine Blessing passive on critical success. Effects: PC may spend 2 Hope to refresh a limited-use feature (once per rest/session). Requires: See section Requires. Notes: only available on critical success.
  > **CRB (p.250, Divine Usurpation):** "Divine Blessing - Passive: When a PC critically succeeds, they can spend 2 Hope to refresh an ability normally limited by uses (such as once per rest, once per session)."

- Apply outcome-based clarity and Hope gain — `internal/test/game/scenarios/environment_rivendell_sanctuary_guidance.lua`. Trigger: Divine Guidance passive. Effects: Instinct roll outcome determines clarity; on critical success gain 1d4 Hope distributed among the party. Requires: See section Requires. Notes: if the deity is unwelcome, roll with disadvantage.
  > **CRB (p.246, Hallowed Temple):** "Divine Guidance - Passive: A PC who prays to a deity while in the Hallowed Temple can make an Instinct Roll to receive answers. If the god they beseech isn't welcome in this temple, the roll is made with disadvantage. Critical Success: The PCs gain clear information. Additionally, they gain 1d4 Hope, which can be distributed between the party. Success with Hope: The PC receives clear information. Success with Fear: The PC receives brief flashes of insight and an emotional impression conveying an answer. Any Failure: The PC receives only vague flashes. They can mark a Stress to receive one clear image without context."

- Clear all HP on rest in this environment — `internal/test/game/scenarios/environment_rivendell_sanctuary_healing.lua`. Trigger: A Place of Healing passive on rest. Effects: any PC resting in the environment clears all HP. Requires: See section Requires. Notes: applies to rests taken in the environment.
  > **CRB (p.246, Hallowed Temple):** "A Place of Healing - Passive: A PC who takes a rest in the Hallowed Temple automatically clears all HP."

- Map outcome to number of details and stress for extra clue — `internal/test/game/scenarios/environment_mirkwood_blight_indigo_flame.lua`. Trigger: Indigo Flame investigation roll. Effects: Knowledge roll outcome sets number of details learned; on failure mark Stress to learn one and gain advantage on the next investigation roll. Requires: See section Requires. Notes: advantage is a single d6 added to the next investigation roll; Stress can only be marked if slots remain.
  > **CRB (p.248, Burning Heart of the Woods):** "The Indigo Flame - Passive: PCs who approach the central tree can make a Knowledge Roll to try to identify the magic that consumed this environment. On a success: They learn three of the below details. On a success with Fear, they learn two. On a failure: They can mark a Stress to learn one and gain advantage on the next action roll to investigate this environment."

- Map outcome to lore details — `internal/test/game/scenarios/environment_moria_ossuary_centuries_of_knowledge.lua`. Trigger: Knowledge roll in the library. Effects: outcome determines the amount of arcana/history information gained. Requires: See section Requires. Notes: use environment Difficulty unless specified.
  > **CRB (p.251, Necromancer's Ossuary):** "Centuries of Knowledge - Passive: A PC can investigate the library and laboratory and make a Knowledge Roll to learn information related to arcana, local history, and the Necromancer's plans."

- Apply graded information gain and stress option — `internal/test/game/scenarios/environment_old_forest_grove_overgrown.lua`. Trigger: Overgrown Battlefield passive. Effects: Instinct roll outcome determines number of details learned; on failure mark Stress to learn one and gain advantage on the next investigation roll. Requires: See section Requires. Notes: relevant Experience can grant an extra detail.
  > **CRB (p.243, Abandoned Grove):** "Overgrown Battlefield - Passive: There has been a battle here. A PC can make an Instinct Roll to identify evidence of that fight. On a success with Hope, learn all three pieces of information below. On a success with Fear, learn two. On a failure, a PC can mark a Stress to learn one and gain advantage on the next action roll to investigate this environment."

- Map outcomes to info/loot and stress on failure — `internal/test/game/scenarios/environment_osgiliath_ruins_buried_knowledge.lua`. Trigger: Buried Knowledge passive. Effects: Instinct or Knowledge roll outcome yields information and possibly loot; on failure mark Stress to find a lead. Requires: See section Requires. Notes: loot and details are selected by the GM.
  > **CRB (p.247, Haunted City):** "Buried Knowledge - Passive: The city has countless mysteries to unfold. A PC who seeks knowledge about the fallen city can make an Instinct or Knowledge Roll to learn about this place and discover (potentially haunted) loot. Critical Success: Gain valuable information and a related useful item. Success with Hope: Gain valuable information. Success with Fear: Uncover vague or incomplete information. Any Failure: Mark a Stress to find a lead after an exhaustive search."

- Model detours, blocked routes, and challenge prompts — `internal/test/game/scenarios/environment_osgiliath_ruins_dead_ends.lua`. Trigger: Dead Ends action. Effects: ghosts or echoes block a path and force a detour or challenge, reshaping the route. Requires: See section Requires. Notes: resolve the detour with an action roll or a progress countdown, then return to normal scene flow.
  > **CRB (p.247, Haunted City):** "Dead Ends - Action: The ghosts of an earlier era manifest scenes from their bygone era… blocking the way behind them, forcing a detour, or presenting them with a challenge."

- Apply narrative escalation and objective shifts — `internal/test/game/scenarios/environment_pelennor_battle_raze.lua`. Trigger: Raze and Pillage action. Effects: raise stakes by lighting fires, stealing assets, kidnapping, or killing the populace; objectives shift accordingly. Requires: See section Requires. Notes: a GM move with immediate narrative impact.
  > **CRB (p.249, Pitched Battle):** "Raze and Pillage - Action: The attacking force raises the stakes by lighting a fire, stealing a valuable asset, kidnapping an important person, or killing the populace."

- Derive environment Difficulty from highest adversary — `internal/test/game/scenarios/environment_waylayers_relative_strength.lua`. Trigger: Relative Strength passive. Effects: set environment Difficulty equal to the highest Difficulty among present adversaries. Requires: See section Requires. Notes: recalculate if the highest-Difficulty adversary changes.
  > **CRB (p.243, Ambushed):** "Relative Strength - Passive: The Difficulty of this environment equals that of the adversary with the highest Difficulty."

- Derive environment Difficulty from highest adversary — `internal/test/game/scenarios/environment_waylaid_relative_strength.lua`. Trigger: Relative Strength passive. Effects: set environment Difficulty equal to the highest Difficulty among present adversaries. Requires: See section Requires. Notes: recalculate if the highest-Difficulty adversary changes.
  > **CRB (p.243, Ambushers):** "Relative Strength - Passive: The Difficulty of this environment equals that of the adversary with the highest Difficulty."

### Mechanics present in the CRB but missing from the scenario doc

The following core mechanics are defined in the Core Rulebook but were not covered by any scenario gap above. These should be considered for future scenario coverage.

- **Tag Team Rolls (p.97):** Once per session, each player can spend 3 Hope to initiate a Tag Team Roll with another PC. Both make separate action rolls and choose one to apply for both results. On a roll with Hope, all PCs involved gain a Hope. On a roll with Fear, the GM gains a Fear for each PC involved. On a successful Tag Team attack, both roll damage and add totals together; if damage types differ, choose which to deal. A Tag Team Roll counts as a single action roll for countdowns/features. Trigger: two PCs declare a coordinated action. Effects: spend 3 Hope; both roll, choose one result; on attack success combine damage. Requires: Core rolls and outcomes; Resources; Damage pipeline. Notes: you can only initiate one per session, but can be involved in multiple.
  > **CRB (p.97):** "Once per session, each player can choose to spend 3 Hope and initiate a Tag Team Roll between their character and another PC. You both make separate action rolls, but before resolving the roll's outcome, choose one of the rolls to apply for both of your results. When you and a partner succeed on a Tag Team Roll attack, you both roll damage and add the totals together."

- **Death Moves (p.106):** When a PC marks their last Hit Point, they must choose one of three death moves. **Blaze of Glory:** take one action (at the GM's discretion) which critically succeeds, then cross through the veil of death. **Avoid Death:** the PC temporarily drops unconscious — can't move or act, can't be targeted by attacks, returns to consciousness when an ally clears 1+ HP or the party finishes a long rest. Roll the Hope Die: if its value is equal to or under the character's level, they gain a scar (cross out a Hope slot permanently; if the last Hope slot, the character's journey ends). **Risk It All:** roll Duality Dice. If the Hope Die is higher, stay on your feet and clear HP or Stress equal to the Hope Die value (divide however you prefer). If the Fear Die is higher, cross through the veil of death. If you critically succeed, stay up and clear all HP and Stress. Trigger: PC marks their last HP. Effects: choose Blaze of Glory, Avoid Death, or Risk It All. Requires: Core rolls and outcomes. Notes: death moves halt normal play until resolved.
  > **CRB (p.106):** "Blaze of Glory: Take one action (at the GM's discretion), which critically succeeds, then cross through the veil of death. Avoid Death: They temporarily drop unconscious… return to consciousness when an ally clears 1 or more of their marked Hit Points or when the party finishes a long rest. After your character falls unconscious, roll your Hope Die. If its value is equal to or under your character's level, they gain a scar. Risk It All: Roll your Duality Dice. Hope Die higher = stay up, clear HP/Stress equal to Hope Die value. Fear Die higher = death. Critical = clear all HP and Stress."

- **Falling and Collision Damage (p.168):** If a character falls to the ground, use the following damage guide: Very Close range: 1d10+3 physical damage. Close range: 1d20+5 physical damage. Far or Very Far range: 1d100+15 physical damage, or death at the GM's discretion. If a character collides with an object or another character at a dangerous speed, they take 1d20+5 direct physical damage. Damage dice can be increased or decreased to fit the story. Trigger: character falls or collides at speed. Effects: apply damage based on fall distance. Requires: Damage pipeline. Notes: collision damage is always direct (ignores armor).
  > **CRB (p.168):** "A fall from Very Close range deals 1d10+3 physical damage. A fall from Close range deals 1d20+5 physical damage. A fall from Far or Very Far range deals 1d100+15 physical damage, or death at the GM's discretion. If a character collides with an object or another character at a dangerous speed, they take 1d20+5 direct physical damage."

- **Underwater Combat (p.168):** By default, attack rolls made while the attacker is underwater have disadvantage (unless the creature can naturally fight underwater). For creatures that can't breathe underwater, use a countdown (starting value 3 or higher). Tick down once whenever any PC takes an action underwater. If an action roll is a failure or with Fear, the GM can tick it down an additional time (if both failure and Fear, tick twice). Once the countdown ends, a PC underwater must mark a Stress whenever they take an action. Trigger: combat or actions taken underwater. Effects: disadvantage on attacks; breath countdown (3); Stress on actions after breath runs out. Requires: Core rolls and outcomes; Dice modifiers; Countdowns. Notes: aquatic creatures are exempt.
  > **CRB (p.168):** "Attack rolls underwater have disadvantage unless the creature can naturally fight underwater. For creatures that cannot breathe underwater, use a countdown (starting value 3 or higher). Tick down once whenever any PC takes an action underwater. If an action roll is a failure or with Fear, the GM can tick it down an additional time."

- **Fate Rolls (p.168):** For moments outside PC influence where the GM wants chance to decide. Ask a player to roll only their Hope or Fear Die (choice doesn't affect outcome, just flavor). The roll does not grant Hope or Fear. The GM declares what event occurs within certain number ranges or scales outcome by result. Trigger: GM calls for a random outcome. Effects: single d12 roll, GM interprets. Requires: Core rolls and outcomes. Notes: fate rolls do not generate Hope or Fear; they are purely narrative.
  > **CRB (p.168):** "Ask a player to roll only their Hope or Fear Die. The roll does not grant Hope or Fear. The GM declares what event occurs within certain number ranges or scales the outcome by how high or low the result is."

- **Conflict Between PCs (p.168):** Before rolling, discuss with both players how to resolve the conflict (a roll might not be necessary). Attack roll against a PC: the attacker rolls against the defender's Evasion, just like an adversary. Any other action: the instigator makes an action roll and the target makes a reaction roll; to succeed, the instigator must beat a Difficulty equal to the total value of the reaction roll. Trigger: PCs in conflict. Effects: attack vs Evasion or action roll vs reaction roll. Requires: Core rolls and outcomes. Notes: both players roll; this is a narrative tool, not a combat optimization.
  > **CRB (p.168):** "On an attack roll against a PC, the attacker rolls against the defender's Evasion, just like an adversary. On any other kind of action roll, the instigator makes an action roll and the target makes a reaction roll. To succeed, the instigator must beat a Difficulty equal to the total value of the reaction roll."

- **Extended Downtime (p.181):** When fast-forwarding the story across multiple days or longer, the GM gains 1d6 Fear per PC and can advance long-term countdowns as appropriate. Trigger: time skip between sessions or scenes. Effects: gain 1d6 Fear per PC; advance long-term countdowns. Requires: Resources; Countdowns. Notes: this represents narrative time passage, not a single rest.
  > **CRB (p.181):** "For fast-forwarding across multiple days or longer, the GM gains 1d6 Fear per PC and can advance long-term countdowns as appropriate."

- **Stacking Effects (p.107):** At the GM's discretion, most effects can stack. However, you can't stack conditions, advantage or disadvantage, or other effects that say you can't. If two or more effects apply and the rules don't specify order, the controlling player (including GM) can apply them in any order. Trigger: multiple effects apply to the same target or roll. Effects: most bonuses/penalties accumulate; conditions/advantage/disadvantage do not stack. Requires: Dice modifiers. Notes: this is a core rule that affects all other mechanics.
  > **CRB (p.107):** "At the GM's discretion, most effects can stack. For example, if two bards give you a Rally Die, you can spend both of them on the same roll. However, you can't stack conditions, advantage or disadvantage, or other effects that say you can't."

- **Adversary Advantage and Disadvantage (p.160):** NPCs roll an additional d20 and pick the highest (advantage) or lowest (disadvantage) result. Some PC abilities impose disadvantage on NPC rolls. Trigger: adversary has advantage or disadvantage. Effects: roll extra d20, take higher (advantage) or lower (disadvantage). Requires: Dice modifiers. Notes: this differs from PC advantage/disadvantage (which uses d6); adversaries use d20.
  > **CRB (p.160):** "NPCs can also roll with advantage (or disadvantage), but when they do, the GM rolls an additional d20 and picks the highest (or lowest) result."

- **Adversary Experience (Optional) (p.154):** The GM can spend a Fear to add an adversary's relevant Experience to raise their attack roll or increase the Difficulty of a roll made against them. Trigger: GM spends Fear during an adversary roll. Effects: add Experience modifier to attack roll or Difficulty. Requires: Resources; Adversary actions. Notes: optional per GM discretion.
  > **CRB (p.154):** "Add an adversary's Experience to a roll: When rolling for an adversary, spend a Fear to add the adversary's relevant Experience modifier to the roll."

- **Scars (p.106):** When a PC gains a scar (from Avoid Death), cross out one Hope slot permanently. That slot can never be used again. The narrative impact (physical scar, painful memory, deep fear) is up to the player. Scars can be healed at the GM's discretion as a downtime project or quest reward. If the last Hope slot is crossed out, the character's journey ends — work with the GM for a farewell at session's end, then create a new character at the party's current level. Trigger: Avoid Death death move when Hope Die ≤ character level. Effects: permanently lose one Hope slot. Requires: Death moves. Notes: scars are the only mechanic that permanently reduces a character's Hope capacity.
  > **CRB (p.106):** "If you do, cross out one of your Hope slots. You can't use this slot anymore. Scars are permanent, but can be healed at the GM's discretion as a downtime project, a reward for a quest focused on healing that scar, or something with similar narrative weight. If you ever cross out your last Hope slot, it's time to end your character's journey."

- **Stress Overflow (p.92):** When forced to mark 1+ Stress but all Stress slots are already full, mark 1 Hit Point instead (regardless of how much Stress was required). Partial overflow: if some slots remain, fill them first, then mark 1 HP for the remainder. Trigger: Stress mark required when Stress is full or partially full. Effects: mark 1 HP per overflow instance. Requires: Resources; HP tracking. Notes: this creates a path from resource exhaustion to death spiral.
  > **CRB (p.92):** "If you're ever forced to mark 1 or more Stress but your slots are already full, you must instead mark 1 Hit Point. If you would take 2 Stress from an enemy and you have 1 Stress left, you would mark 1 Stress and 1 Hit Point."

- **Last Stress → Vulnerable (p.92):** When a character marks their last Stress slot, they immediately gain the Vulnerable condition until they clear at least 1 Stress. Trigger: marking the last empty Stress slot. Effects: gain Vulnerable condition. Requires: Conditions; Resources. Notes: Vulnerable means taking +1 more HP damage when you mark HP.
  > **CRB (p.92):** "When you mark your last Stress, you become Vulnerable (see the 'Conditions' section on page 102) until you clear at least 1 Stress."

- **Unarmored Characters (p.114):** A character without armor has Armor Score 0, Major damage threshold equal to their level, and Severe damage threshold equal to twice their level. With Armor Score 0, you cannot mark Armor Slots. Temporary armor (e.g., spells) grants a temporary Armor Score and the ability to mark that many additional Armor Slots; when the temporary armor ends, clear Armor Slots equal to the temporary score. Trigger: character has no equipped armor. Effects: set Armor Score 0, Major = level, Severe = 2× level, no Armor Slot marking. Requires: Damage pipeline; Equipment. Notes: temporary armor stacks on top of 0 base.
  > **CRB (p.114):** "Going unarmored does not give your character any bonuses or penalties, but while unarmored, they have an Armor Score of 0, their Major threshold is equal to their level, and their Severe threshold is equal to twice their level. If your character has an Armor Score of 0, you can't mark Armor Slots."

- **Maximum Armor Score (p.114):** A character's Armor Score, with all bonuses included, can never exceed 12. Trigger: equipping armor or gaining armor bonuses. Effects: cap total Armor Score at 12. Requires: Equipment. Notes: applies after all bonuses are summed.
  > **CRB (p.114):** "Your character's Armor Score, with all bonuses included, can never exceed 12."

- **Throwing Weapons (p.112):** A character can throw any weapon that could theoretically be thrown (e.g., dagger, axe) at a target within Very Close range, making an attack roll using Finesse. On success, deal damage as usual. The weapon is lost after throwing — it cannot be attacked with or benefit from its features until retrieved. Trigger: character throws a weapon at Very Close range. Effects: Finesse attack roll, normal weapon damage, weapon lost. Requires: Core rolls; Equipment. Notes: there is no formal "throwable" trait; the GM decides what can be thrown.
  > **CRB (p.112):** "When you're using a weapon that you could theoretically throw (such as a dagger or an axe), you can throw it at a target within Very Close range, making an attack roll using Finesse. On a success, deal damage as usual for that weapon. Once thrown, you lose that weapon."

- **Switching Weapons (p.112):** In a dangerous situation, mark a Stress to swap an Inventory Weapon into the Active Weapon slot (previous Active Weapon moves to Inventory). In a calm situation or during a rest, swap freely with no Stress cost. Trigger: character changes equipped weapon. Effects: mark Stress if in danger; no cost if calm/resting. Requires: Resources; Equipment. Notes: this adds tactical cost to mid-combat weapon changes.
  > **CRB (p.112):** "When your character is in a dangerous situation, you can mark a Stress to equip an Inventory Weapon, moving their previous Active Weapon into the Inventory Weapon section. If your character is in a calm situation or preparing during a rest, they can swap weapons with no Stress cost."

- **GM Critical Success (p.148):** When the GM rolls a natural 20 on the d20, the roll automatically succeeds. On a critical attack roll, deal extra damage: start with the maximum possible value of the damage dice, then roll damage as usual and add it to that value. A critical success on an adversary reaction roll has no added benefit. Trigger: GM rolls nat 20. Effects: auto-success; on attack, max damage + rolled damage. Requires: Adversary actions; Damage pipeline. Notes: this is the NPC equivalent of PC critical success.
  > **CRB (p.148):** "Whenever you roll a 20 on the d20, your roll automatically succeeds. If you critically succeed on an attack roll, you also deal extra damage. Start with the highest possible value the damage dice can roll, and then make a damage roll as usual, adding it to that value. A critical success on a reaction roll does not have any added benefit for an adversary."

- **GM Fear Starting Pool and Maximum (p.154):** The GM starts a campaign with Fear equal to the number of PCs. Fear is gained from: PC rolls with Fear, downtime, certain PC abilities/spells, and adversary features. Maximum Fear is 12. Fear carries over between sessions. Trigger: campaign start; Fear gains during play. Effects: track Fear pool, cap at 12. Requires: Resources. Notes: Fear should be visible to players; track with tokens or dice.
  > **CRB (p.154):** "When you start a campaign, you begin with an amount of Fear equal to the number of PCs. You can hold up to a maximum of 12 Fear. Fear carries over between sessions, so note how many Fear you have at the end of each session and begin the next time with that same pool."

- **Short Rest Limit (p.105):** A party can take up to three short rests before their next rest must be a long rest. If a short rest is interrupted (e.g., by an attack), the characters don't gain its benefits. If a long rest is interrupted, the characters gain short rest benefits instead (even if three short rests have already been taken). Trigger: party initiates a rest. Effects: track short rest count (max 3); reset on long rest. Requires: Resources; Downtime. Notes: interrupted long rest still grants short rest benefits.
  > **CRB (p.105):** "A party can take up to three short rests before their next rest must be a long rest. If a short rest is interrupted (such as by an adversary's attack), characters don't gain its benefits. If a long rest is interrupted, the characters instead gain the benefits of a short rest (even if they've already had three short rests)."

- **Downtime Consequences (p.105):** On a short rest, the GM gains 1d4 Fear. On a long rest, the GM gains 1d4 + number of PCs Fear, and can advance a long-term countdown. Trigger: party completes a rest. Effects: GM gains Fear; long rest may advance countdowns. Requires: Resources; Countdowns. Notes: the world doesn't stop when you rest — this is the cost of downtime.
  > **CRB (p.105):** "On a short rest, they gain 1d4 Fear. On a long rest, they gain an amount of Fear equal to 1d4 + the number of PCs, and they can advance a long-term countdown."

- **Refreshing Features During Downtime (p.105):** Effects and features that last "until your next rest" or are usable "per rest" refresh on either short or long rest. Effects and features that last "until your next long rest" or are usable "per long rest" only refresh on long rest. "Once per session" features don't refresh during rests — they refresh at the start of the next session (or during a break in long sessions, at GM discretion). Trigger: rest or session boundary. Effects: refresh applicable features; end applicable temporary effects. Requires: Downtime. Notes: distinguishing "per rest" vs "per long rest" vs "per session" is critical for feature tracking.
  > **CRB (p.105):** "Any effects that last until your next rest end when your character finishes either a long or a short rest. Likewise, any features that can be used a number of times per rest refresh when your character finishes either a long or a short rest. Any effects that last until your next long rest end when your character finishes a long rest. Some features say you can use them 'once per session.' These features don't refresh during rests, but instead are available again at the start of your next Daggerheart session."

- **Working on a Project (p.105, p.181):** If a PC wants to pursue a long-term project during downtime, they discuss it with the GM. Projects usually involve a Progress Countdown. Each time a PC takes the "Work on a Project" downtime move during a long rest, they either automatically tick down their countdown or make an action roll to gauge progress. This is a long rest–only downtime move. Trigger: PC spends long rest downtime on a project. Effects: tick down project countdown (auto or via roll). Requires: Countdowns; Downtime. Notes: examples include deciphering ancient text, crafting weapons, healing scars.
  > **CRB (p.105):** "If a PC wants to pursue a project that would take them a substantial amount of time but that they can work on during a long rest, they should first discuss it with the GM. Projects usually involve a Progress Countdown. Each time a PC takes the Work on a Project downtime move during a long rest, they either automatically tick down their countdown, or the GM might ask them to make an action roll to gauge their progress."

- **Critical Reaction Roll (p.99):** On a critical success reaction roll, you don't clear a Stress or gain a Hope (unlike normal critical successes), but you do ignore any effects that would still impact you on a success (such as taking damage or marking Stress). Trigger: PC rolls matching Duality Dice on a reaction roll. Effects: ignore success-tier side effects; no Hope/Stress benefit. Requires: Core rolls and outcomes. Notes: this makes critical reactions purely defensive — they negate rather than reward.
  > **CRB (p.99):** "If you critically succeed on a reaction roll, you don't clear a Stress or gain a Hope, but you do ignore any effects that would still impact you on a success (such as taking damage or marking Stress)."

- **Multiple Resistance (p.99):** If multiple features grant the same kind of resistance, they only count as one source — incoming damage is still only halved. Immunity negates all damage of that type. Apply resistance/immunity before other damage reduction (e.g., Armor Slots). If an attack deals both physical and magic damage, you only benefit from resistance/immunity if you are resistant/immune to both types. Trigger: multiple resistance sources apply to the same damage. Effects: resistance doesn't stack; apply before Armor Slots. Requires: Damage pipeline. Notes: this prevents resistance stacking exploits.
  > **CRB (p.99):** "If multiple features grant the same kind of resistance, they only count as one source of resistance. Apply the resistance or immunity first. You can then use other methods to reduce the damage further. If an attack deals both physical and magic damage, you can only benefit from resistance or immunity if you are resistant or immune to both damage types."

- **Simultaneous and Stacking Effects (p.107):** If two or more effects can apply to a situation and the rules don't specify order, the controlling player (including the GM) can apply them in any order. If you want to apply multiple effects, they must both be able to successfully resolve to be used together — otherwise, choose one. Trigger: multiple triggered effects apply simultaneously. Effects: controller chooses resolution order; effects must all be valid to stack. Requires: Core rolls and outcomes. Notes: prevents exploits like triggering on Fear then converting the roll to Hope.
  > **CRB (p.107):** "If two or more effects can apply to a situation, and the rules don't tell you which order to apply them in, the player controlling the effects (including the GM) can apply the effects in any order. If you want to apply two or more effects, they must both be able to successfully resolve to be used together."

- **Increasing and Decreasing Countdowns (p.163):** Some countdowns loop but change their starting value each cycle. An increasing countdown's starting value increases by 1 each time it triggers and resets (e.g., "Countdown (Increasing 8)" goes 8 → 9 → 10 → ...). A decreasing countdown's starting value decreases by 1 each cycle (e.g., "Countdown (Decreasing 8)" goes 8 → 7 → 6 → ...). When a decreasing countdown reaches 0, a major event triggers. Trigger: countdown triggers and resets. Effects: starting value shifts by ±1 each cycle. Requires: Countdowns. Notes: useful for escalating or de-escalating tension across a scene.
  > **CRB (p.163):** "Each time an increasing countdown triggers and resets, its starting value increases by 1. Similarly, each time a decreasing countdown triggers, its starting value decreases by 1. Once a decreasing countdown reaches 0, a major event triggers."

- **Chase Countdowns (p.163):** Track a chase with two countdowns: one for the pursuers, one for the escapees. Pick a die for the pursuers' countdown at its highest value; set the escapees' countdown at a lower value to reflect their lead (1 lower = small lead, 3 = decent, 5 = substantial). PC action rolls advance both countdowns: success ticks down the PC's side (progress), failure or success with Fear ticks down the other side (consequence). The first countdown to reach 0 determines the outcome. Trigger: chase scene begins. Effects: dual countdowns tracked simultaneously; action rolls advance both. Requires: Countdowns; Core rolls and outcomes. Notes: works whether PCs are pursuing or escaping.
  > **CRB (p.163):** "When the chase begins, set two countdowns: one for the pursuing party and one for the escaping party. First pick a die for the pursuers' countdown—the more time you want the chase to take, the higher the starting value should be—then set that die at its highest value. Next, select another die with the same starting value for the escapee's countdown, but set that die at a lower value to reflect how much of a lead they have."
