# Scenario Missing Mechanics

This document tracks mechanics gaps discovered while running the scenario suite. It focuses on game-system behavior that is missing or incorrect even when the DSL bindings exist.

## Status

Mechanics inventory reflects the first full scenario run after disabling comment validation.

## General Mechanics Gaps

(None currently detected.)

## Daggerheart-Specific Mechanics Gaps

Scenario annotations still call out missing DSL/mechanics. Items are grouped by theme and ordered by priority within each bucket (highest impact first):

### Core roll and damage resolution
- Apply Hope gain, Stress clear, and choose a bonus effect — `internal/test/game/scenarios/action_roll_critical_success.lua`.
- Apply Hope gain and record a narrative complication — `internal/test/game/scenarios/action_roll_failure_with_hope.lua`.
- Apply multiple advantage/disadvantage sources to a single roll — `internal/test/game/scenarios/advantage_cancellation.lua`.
- Apply the d6 advantage die to the action roll — `internal/test/game/scenarios/advantage_disguise_roll.lua`.
- Spend Hope on help and apply the advantage die — `internal/test/game/scenarios/help_advantage_roll.lua`.
- Spend Hope and apply an Experience bonus to the roll — `internal/test/game/scenarios/experience_spend_modifier.lua`.
- Force the adversary attack roll to equal Evasion — `internal/test/game/scenarios/evasion_tie_hit.lua`.
- Apply max-dice bonus before rolling damage — `internal/test/game/scenarios/critical_damage_maximum.lua`.
- Assert tier mapping and HP marked for each tier — `internal/test/game/scenarios/damage_thresholds_example.lua`.
- Force the damage dice to 3, 5, and 6 — `internal/test/game/scenarios/damage_roll_modifier.lua`.
- Force the damage dice to 3 and 7 — `internal/test/game/scenarios/damage_roll_proficiency.lua`.

### Combat and adversary actions
- Set the adversary hit, damage total, and armor slot spend — `internal/test/game/scenarios/fear_spotlight_armor_mitigation.lua`.
- Spend adversary stress and resolve a multi-target adversary attack — `internal/test/game/scenarios/sweeping_attack_all_targets.lua`.
- Represent group attack roll and shared damage — `internal/test/game/scenarios/orc_dredge_group_attack.lua`.
- Resolve group attack damage aggregation — `internal/test/game/scenarios/minion_group_attack_rats.lua`.
- Apply Minion (3) overflow and select extra targets — `internal/test/game/scenarios/minion_overflow_damage.lua`.
- Apply Minion (4) overflow and stress marking — `internal/test/game/scenarios/wild_flame_minion_blast.lua`.
- Apply Minion (8) overflow — `internal/test/game/scenarios/minion_high_threshold_imps.lua`.
- Apply the Opportunist doubling and armor mitigation — `internal/test/game/scenarios/orc_archer_opportunist.lua`.
- Assert per-target outcomes and damage tiers — `internal/test/game/scenarios/fireball_orc_pack_multi.lua`.
- Adversary reaction roll with an experience bonus — `internal/test/game/scenarios/fireball_golum_reaction.lua`.
- Apply reactive damage and cooldown on the reaction — `internal/test/game/scenarios/ranged_warding_sphere.lua`.
- Apply group reaction rolls and Vulnerable condition — `internal/test/game/scenarios/ranged_snowblind_trap.lua`.
- Apply disadvantage to the attack and reduce damage severity — `internal/test/game/scenarios/ranged_take_cover.lua`.
- Apply stress spend and advantage die — `internal/test/game/scenarios/ranged_steady_aim.lua`.
- Move the adversary and spend Stress — `internal/test/game/scenarios/ranged_battle_teleport.lua`.
- Apply scene-wide reaction rolls and half damage on success — `internal/test/game/scenarios/ranged_arcane_artillery.lua`.
- Apply area hazard, reaction roll, and forced movement — `internal/test/game/scenarios/ranged_eruption_hazard.lua`.
- Apply advantage die, stress cost, and reroll a 1 — `internal/test/game/scenarios/sam_critical_broadsword.lua`.
- Apply movement, stress spend, and knockback — `internal/test/game/scenarios/skulk_swift_claws.lua`.
- Apply Hidden and swap in 1d6+6 damage on advantage — `internal/test/game/scenarios/skulk_cloaked_backstab.lua`.
- Apply disadvantage to attacks beyond Very Close range — `internal/test/game/scenarios/skulk_reflective_scales.lua`.
- Apply group attack resolution and Restrained condition — `internal/test/game/scenarios/skulk_icicle_barb.lua`.
- Use Better Surrounded to hit all targets in range — `internal/test/game/scenarios/improvised_fear_move_bandit_chain.lua`.
- Apply group attack damage to the target — `internal/test/game/scenarios/improvised_fear_move_bandit_chain.lua`.
- Move allies to cover and apply Hidden until they attack — `internal/test/game/scenarios/leader_into_bramble.lua`.
- Apply Difficulty increase after HP loss — `internal/test/game/scenarios/leader_ferocious_defense.lua`.
- Reduce HP loss and spend Stress on reaction — `internal/test/game/scenarios/leader_brace_reaction.lua`.
- Apply advantage to archer attacks while the countdown runs — `internal/test/game/scenarios/head_guard_on_my_signal.lua`.
- Model the Rally Guards action effect — `internal/test/game/scenarios/head_guard_rally_guards.lua`.

### Group and shared outcomes
- Map individual outcomes to a shared consequence — `internal/test/game/scenarios/airship_group_roll.lua`.
- Encode per-supporter outcomes and group bonuses — `internal/test/game/scenarios/group_finesse_sneak.lua`.
- Assert each participant outcome and any fear/hope changes — `internal/test/game/scenarios/group_action_escape.lua`.

### Resource economy and state changes
- Apply group Hope loss and GM Fear gain — `internal/test/game/scenarios/terrifying_hope_loss.lua`.
- Apply temporary armor bonus and clear Armor Slots on rest — `internal/test/game/scenarios/temporary_armor_bonus.lua`.

### Spellcasting
- Reject a Spellcast roll that attempts an out-of-scope effect — `internal/test/game/scenarios/spellcast_scope_limit.lua`.
- Spend Hope to cast and apply the Fear gain to the GM — `internal/test/game/scenarios/spellcast_hope_cost.lua`.
- Enforce that narration doesn't modify damage — `internal/test/game/scenarios/spellcast_flavor_limits.lua`.
- Roll two Fear dice and take the higher on Spellcast — `internal/test/game/scenarios/environment_mirkwood_blight_chaos_magic.lua`.

### GM moves and improvised fear moves
- Encode the specific GM move type and consequence — `internal/test/game/scenarios/gm_move_examples.lua`.
- Represent item theft and chase trigger — `internal/test/game/scenarios/gm_move_artifact_chase.lua`.
- Encode the narrative fear move effect — `internal/test/game/scenarios/improvised_fear_move_shadow.lua`.
- Encode the improvised fear move effect — `internal/test/game/scenarios/improvised_fear_move_noble_escape.lua`.
- Apply a temporary damage bonus feature — `internal/test/game/scenarios/improvised_fear_move_rage_boost.lua`.

### Social scenes
- Apply hospitality ban, Hope loss, Stress, and unconscious condition — `internal/test/game/scenarios/social_village_elder_peace.lua`.
- Apply Preferential Treatment and The Runaround effects — `internal/test/game/scenarios/social_merchant_haggling.lua`.

### Countdowns and progress systems
- Tick countdown by result tier and award Hope/Fear changes — `internal/test/game/scenarios/progress_countdown_climb.lua`.
- Implement looping countdown and cold gear advantage — `internal/test/game/scenarios/environment_caradhras_pass_icy_winds.lua`.
- Tick a long-term countdown by 1d4 — `internal/test/game/scenarios/environment_gondor_court_imperial_decree.lua`.
- Set fear cap to 15 and activate long-term countdown (8) — `internal/test/game/scenarios/environment_dark_tower_usurpation_final_preparations.lua`.
- Activate siege countdown (10) and tick based on fear/major damage — `internal/test/game/scenarios/environment_dark_tower_usurpation_beginning_of_end.lua`.
- Tie Fear outcomes to countdown ticks and summon on trigger — `internal/test/game/scenarios/environment_isengard_ritual_summoning.lua`.
- Apply Hope die size change and countdown clearance — `internal/test/game/scenarios/environment_isengard_ritual_desecrated_ground.lua`.
- Activate consequence countdown and shift to Helms Deep Siege — `internal/test/game/scenarios/environment_helms_deep_siege_siege_weapons.lua`.
- Apply progress countdown (8) and stress on failure — `internal/test/game/scenarios/environment_shadow_realm_impossible_architecture.lua`.
- Loop countdown and reduce highest trait or mark stress — `internal/test/game/scenarios/environment_shadow_realm_everything_you_are.lua`.
- Loop countdown and apply direct damage with half on success — `internal/test/game/scenarios/environment_mirkwood_blight_choking_ash.lua`.
- Apply tick deltas by outcome tiers — `internal/test/game/scenarios/environment_misty_ascent_progress.lua`.
- Intercept failure and apply stress in place of countdown penalty — `internal/test/game/scenarios/environment_misty_ascent_pitons.lua`.
- Tie action rolls to the escape countdown while hazards unfold — `internal/test/game/scenarios/environment_osgiliath_ruins_apocalypse_then.lua`.

### Environment hazards, damage, and conditions
- Apply movement and damage severity on failure — `internal/test/game/scenarios/environment_caradhras_pass_avalanche.lua`.
- Apply movement check and hazard damage — `internal/test/game/scenarios/environment_prancing_pony_bar_fight.lua`.
- Apply river movement and conditional stress on success — `internal/test/game/scenarios/environment_bruinen_ford_undertow.lua`.
- Tie failure with Fear to immediate undertow action — `internal/test/game/scenarios/environment_bruinen_ford_dangerous_crossing.lua`.
- Apply advantage, bonus damage, or Relentless feature — `internal/test/game/scenarios/environment_isengard_ritual_blasphemous_might.lua`.
- Redirect an attack to the ally by marking Stress — `internal/test/game/scenarios/environment_isengard_ritual_complete.lua`.
- Roll 1d4 stress on failure with Fear — `internal/test/game/scenarios/environment_dark_tower_usurpation_ritual_nexus.lua`.
- Clear 2 HP and increase the usurper's stats after the action — `internal/test/game/scenarios/environment_dark_tower_usurpation_godslayer.lua`.
- Apply direct damage on failure and stress on success — `internal/test/game/scenarios/environment_shadow_realm_unmaking.lua`.
- Deduct Hope on Fear outcome and grant GM Fear if it was last Hope — `internal/test/game/scenarios/environment_shadow_realm_disorienting_reality.lua`.
- Convert a Fear outcome to Hope and mark Stress — `internal/test/game/scenarios/environment_rivendell_sanctuary_relentless_hope.lua`.
- Apply Restrained + Vulnerable, escape roll damage, and Hope loss — `internal/test/game/scenarios/environment_mirkwood_blight_grasping_vines.lua`.
- Apply area reaction roll and half damage on success — `internal/test/game/scenarios/environment_mirkwood_blight_charcoal_constructs.lua`.
- Defer the damage until a follow-up action fails to save — `internal/test/game/scenarios/environment_misty_ascent_fall.lua`.
- Apply extra Hope cost to healing or rest effects — `internal/test/game/scenarios/environment_moria_ossuary_no_place_living.lua`.
- Roll d4 and distribute healing across undead — `internal/test/game/scenarios/environment_moria_ossuary_aura_of_death.lua`.
- Apply reaction roll and 4d8+8 damage on failure — `internal/test/game/scenarios/environment_moria_ossuary_skeletal_burst.lua`.
- Apply damage, Restrained condition, and escape checks — `internal/test/game/scenarios/environment_old_forest_grove_barbed_vines.lua`.
- Apply physical resistance and stress-based movement — `internal/test/game/scenarios/environment_osgiliath_ruins_ghostly_form.lua`.
- Apply area reaction roll, damage, and stress on failure — `internal/test/game/scenarios/environment_pelennor_battle_war_magic.lua`.
- Restrict movement without a successful Agility roll — `internal/test/game/scenarios/environment_pelennor_battle_adrift.lua`.
- Apply the fear loss and advantage to the first strike — `internal/test/game/scenarios/environment_waylayers_where_did_they_come_from.lua`.
- Award Fear to the GM and immediate spotlight shift — `internal/test/game/scenarios/environment_waylayers_surprise.lua`.
- Award Fear to the GM and shift spotlight — `internal/test/game/scenarios/environment_waylaid_surprise.lua`.

### Environment spawns and scene setup
- Spawn two eagles at Very Far range — `internal/test/game/scenarios/environment_caradhras_pass_raptor_nest.lua`.
- Spawn multiple adversaries based on party size — `internal/test/game/scenarios/environment_bree_outpost_wrong_place.lua`.
- Place the adversary and trigger its action — `internal/test/game/scenarios/environment_bruinen_ford_patient_hunter.lua`.
- Spawn adversaries based on party size and spotlight the knight — `internal/test/game/scenarios/environment_helms_deep_siege_reinforcements.lua`.
- Summon 1d4+2 troops and trigger their group attack — `internal/test/game/scenarios/environment_dark_tower_usurpation_defilers_abound.lua`.
- Spawn 1 abomination, 1 corruptor, and 2d6 thralls — `internal/test/game/scenarios/environment_shadow_realm_predators.lua`.
- Spawn multiple adversaries near the priest — `internal/test/game/scenarios/environment_rivendell_sanctuary_divine_censure.lua`.
- Summon 1d6 rotted zombies, two perfected, or a legion — `internal/test/game/scenarios/environment_moria_ossuary_they_keep_coming.lua`.
- Spawn adversaries equal to party size and shift spotlight — `internal/test/game/scenarios/environment_old_forest_grove_not_welcome.lua`.
- Spawn the elemental near a chosen PC and shift spotlight — `internal/test/game/scenarios/environment_old_forest_grove_defiler.lua`.
- Spawn new adversaries and spotlight the knight — `internal/test/game/scenarios/environment_pelennor_battle_reinforcements.lua`.

### Environment social, economy, and information
- Apply advantage on dispel after critical knowledge success — `internal/test/game/scenarios/environment_caradhras_pass_engraved_sigils.lua`.
- Model item loss and chase triggers — `internal/test/game/scenarios/environment_bree_market_sticky_fingers.lua`.
- Introduce a quest item and its non-gold cost — `internal/test/game/scenarios/environment_bree_market_unexpected_find.lua`.
- Separate the PC from the group and apply positioning — `internal/test/game/scenarios/environment_bree_market_crowd_closes_in.lua`.
- Spend gold and apply advantage die — `internal/test/game/scenarios/environment_bree_market_tip_the_scales.lua`.
- Represent the ongoing social pressure from the society — `internal/test/game/scenarios/environment_bree_outpost_broken_compass.lua`.
- Represent rivalry hooks and competitive pressures — `internal/test/game/scenarios/environment_bree_outpost_rival_party.lua`.
- Represent the narrative prompt and resulting tension — `internal/test/game/scenarios/environment_bree_outpost_shakedown.lua`.
- Map outcome to rumor selection and stress on failure — `internal/test/game/scenarios/environment_bree_outpost_rumors.lua`.
- Model the NPC hook and immediate agenda — `internal/test/game/scenarios/environment_prancing_pony_someone_comes_to_town.lua`.
- Model the narrative reveal and its hooks — `internal/test/game/scenarios/environment_prancing_pony_mysterious_stranger.lua`.
- Map outcome to number of details and stress choice — `internal/test/game/scenarios/environment_prancing_pony_talk.lua`.
- Roll and apply gold payout vs stress — `internal/test/game/scenarios/environment_prancing_pony_sing.lua`.
- Reveal a secret route with Instinct/Knowledge success — `internal/test/game/scenarios/environment_helms_deep_siege_secret_entrance.lua`.
- Model ongoing social pressure and favor exchanges — `internal/test/game/scenarios/environment_gondor_court_rival_vassals.lua`.
- Apply Presence reaction and stress or acceptance on failure — `internal/test/game/scenarios/environment_gondor_court_gravity_of_empire.lua`.
- Spend Fear to trigger witness and Instinct reaction to notice — `internal/test/game/scenarios/environment_gondor_court_eyes_everywhere.lua`.
- Apply disadvantage to nonconforming Presence rolls — `internal/test/game/scenarios/environment_gondor_court_all_roads.lua`.
- Spend 2 Hope to refresh a limited-use ability — `internal/test/game/scenarios/environment_dark_tower_usurpation_divine_blessing.lua`.
- Apply outcome-based clarity and Hope gain — `internal/test/game/scenarios/environment_rivendell_sanctuary_guidance.lua`.
- Clear all HP on rest in this environment — `internal/test/game/scenarios/environment_rivendell_sanctuary_healing.lua`.
- Map outcome to number of details and stress for extra clue — `internal/test/game/scenarios/environment_mirkwood_blight_indigo_flame.lua`.
- Map outcome to lore details — `internal/test/game/scenarios/environment_moria_ossuary_centuries_of_knowledge.lua`.
- Apply graded information gain and stress option — `internal/test/game/scenarios/environment_old_forest_grove_overgrown.lua`.
- Map outcomes to info/loot and stress on failure — `internal/test/game/scenarios/environment_osgiliath_ruins_buried_knowledge.lua`.
- Model detours, blocked routes, and challenge prompts — `internal/test/game/scenarios/environment_osgiliath_ruins_dead_ends.lua`.
- Apply narrative escalation and objective shifts — `internal/test/game/scenarios/environment_pelennor_battle_raze.lua`.
- Derive environment Difficulty from highest adversary — `internal/test/game/scenarios/environment_waylayers_relative_strength.lua`.
- Derive environment Difficulty from highest adversary — `internal/test/game/scenarios/environment_waylaid_relative_strength.lua`.
