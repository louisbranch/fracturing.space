---
title: "Daggerheart Event Timeline Contract"
parent: "Project"
nav_order: 23
---

# Daggerheart Event Timeline Contract

This document maps high-traffic Daggerheart mechanics onto the canonical write path:

`request -> command -> decider -> event append -> projection apply policy`

Use this as the onboarding contract for new mechanics and for review of existing paths.

This document is intentional/mechanic mapping guidance, not a generated type
inventory. For exact payload fields and emitter references, use:

- [Event catalog](../events/event-catalog.md)
- [Command catalog](../events/command-catalog.md)
- [Usage map](../events/usage-map.md)

## Command/Event Timeline Map

| Mechanic | Command Type(s) | Emitted Event Type(s) | Projection Targets | Apply policy notes | Required invariants |
| --- | --- | --- | --- | --- | --- |
| Action roll resolution | `action.roll.resolve` | `action.roll_resolved` | Event journal (no direct Daggerheart projection mutation) | Request path records event; projection apply is skipped for this envelope | Campaign/session valid; roll payload valid; command must emit event |
| Roll outcome finalization | `action.outcome.apply` | `action.outcome_applied` | Event journal (plus follow-on Daggerheart/system commands) | Request path records outcome event; projection apply is skipped for this envelope | Roll event exists and matches session; no duplicate/bypass apply |
| Outcome-driven GM Fear update | `sys.daggerheart.gm_fear.set` | `sys.daggerheart.gm_fear_changed` | Daggerheart snapshot (`gm_fear`) | Inline apply depends on runtime mode; outbox mode must not inline apply in request path | Fear bounds and spend/gain checks enforced |
| Outcome-driven character state patch | `sys.daggerheart.character_state.patch` | `sys.daggerheart.character_state_patched` | Daggerheart character state | Inline apply mode-controlled | Patch payload must include meaningful deltas |
| Outcome-driven condition change | `sys.daggerheart.condition.change` | `sys.daggerheart.condition_changed` | Daggerheart character conditions | Inline apply mode-controlled | Normalized set diff; no empty/invalid conditions |
| Session gate for GM consequence | `session.gate_open`, `session.spotlight_set` | `session.gate_opened`, `session.spotlight_set` | Session gate + spotlight projections | Inline apply mode-controlled | One open gate at a time; request/session correlation |
| Character damage apply | `sys.daggerheart.damage.apply` | `sys.daggerheart.damage_applied` | Daggerheart character HP/armor | Inline apply mode-controlled | Campaign system is Daggerheart; damage payload valid; emits event |
| Multi-target damage apply | `sys.daggerheart.multi_target_damage.apply` | N × `sys.daggerheart.damage_applied` | Per-target Daggerheart character HP/armor | Inline apply mode-controlled | All targets validated atomically; emits N damage_applied events in single batch via DecideFuncMulti |
| Adversary damage apply | `sys.daggerheart.adversary_damage.apply` | `sys.daggerheart.adversary_damage_applied` | Daggerheart adversary HP/armor | Inline apply mode-controlled | Adversary exists in session; payload valid; emits event |
| Rest | `sys.daggerheart.rest.take` | `sys.daggerheart.rest_taken`, optional `sys.daggerheart.countdown_updated` (when `long_term_countdown` is present) | Daggerheart snapshot and targeted character state, plus long-term countdown state | Inline apply mode-controlled | Rest type valid; campaign/session mutate gates pass; rest + optional countdown update emit atomically from one command decision |
| Downtime move | `sys.daggerheart.downtime_move.apply` | `sys.daggerheart.downtime_move_applied` | Daggerheart character state | Inline apply mode-controlled | Move is valid; resulting resource bounds valid |
| Temporary armor apply | `sys.daggerheart.character_temporary_armor.apply` | `sys.daggerheart.character_temporary_armor_applied` | Daggerheart temporary armor buckets and armor totals | Inline apply mode-controlled | Source/duration/amount validation; emits event |
| Loadout swap and associated resource mutation | `sys.daggerheart.loadout.swap`, `sys.daggerheart.stress.spend` | `sys.daggerheart.loadout_swapped`, `sys.daggerheart.character_state_patched` | Daggerheart character loadout-facing stress/state | Inline apply mode-controlled for Daggerheart events | Recall cost bounds; stress spend consistency |
| Character conditions apply endpoint | `sys.daggerheart.condition.change`, `sys.daggerheart.character_state.patch` (life state updates) | `sys.daggerheart.condition_changed`, `sys.daggerheart.character_state_patched` | Character conditions/life state | Inline apply mode-controlled | No-op updates rejected; roll correlation checked when provided |
| Adversary condition changes | `sys.daggerheart.adversary_condition.change` | `sys.daggerheart.adversary_condition_changed` | Adversary conditions | Inline apply mode-controlled | No-op updates rejected; normalized set required |
| Countdown create/update/delete | `sys.daggerheart.countdown.create`, `sys.daggerheart.countdown.update`, `sys.daggerheart.countdown.delete` | `sys.daggerheart.countdown_created`, `sys.daggerheart.countdown_updated`, `sys.daggerheart.countdown_deleted` | Daggerheart countdown projections | Inline apply mode-controlled | Countdown bounds/rules validated before command |
| Adversary create/update/delete | `sys.daggerheart.adversary.create`, `sys.daggerheart.adversary.update`, `sys.daggerheart.adversary.delete` | `sys.daggerheart.adversary_created`, `sys.daggerheart.adversary_updated`, `sys.daggerheart.adversary_deleted` | Daggerheart adversary projections | Inline apply mode-controlled | Session-scoped adversary integrity and payload validation |

## ApplyRollOutcome sequencing contract

`ApplyRollOutcome` must preserve this command order for replay-safe ownership:

1. optional `sys.daggerheart.gm_fear.set`
2. per-target optional `sys.daggerheart.character_state.patch`
3. per-target optional `sys.daggerheart.condition.change`
4. final `action.outcome.apply`

Invariants:

- `action.outcome.apply` is journal-facing and must not include system-owned
  effects in `pre_effects`/`post_effects`.
- Daggerheart state mutation is expressed only through explicit `sys.daggerheart.*`
  commands/events.
- Session-side follow-up effects (for example gate open + spotlight set) remain
  core-owned post-effects on `action.outcome.apply`.

### Known Gap: Consequence Atomicity

`ApplyRollOutcome` applies consequence commands sequentially. If command 3 of
5 fails, commands 1-2 are already persisted. This is acceptable because:

1. Each consequence command independently produces valid state — there is no
   intermediate "half-applied" state that violates domain invariants.
2. Idempotency guards prevent double-application on retry.
3. `action.outcome.apply` at the end serves as a completion marker — its
   absence signals that the consequence set is incomplete, enabling retry.
4. Replay recovers intermediate state deterministically from the event journal.

If true multi-command atomicity is needed in the future, follow the
`rest.take` precedent: a single command whose decider emits multiple events
from one decision, all batch-appended atomically.

## Priority Missing-Mechanic Timeline Mappings

Use these row IDs in `scenario-missing-mechanics.md` while backfilling the full
mechanic backlog. If a mechanic cannot be represented with existing command/event
types, add the new type to this contract before implementation.

| Row ID | Mechanic Gap (Scenario) | Command Type(s) | Emitted Event Type(s) | Projection Targets | Apply Policy Notes | Required Invariants |
| --- | --- | --- | --- | --- | --- | --- |
| P1 | Attack tie hits target Evasion (`evasion_tie_hit.lua`) | `action.roll.resolve`, `sys.daggerheart.damage.apply` | `action.roll_resolved`, `sys.daggerheart.damage_applied` | Event journal + character HP/armor state | `action.roll_resolved` remains journal-only; damage apply mode-controlled | Attack succeeds when total equals Evasion; damage event emitted only on hit |
| P2 | Critical max-damage bonus (`critical_damage_maximum.lua`) | `action.roll.resolve`, `sys.daggerheart.damage.apply` | `action.roll_resolved`, `sys.daggerheart.damage_applied` | Event journal + character HP/armor state | Same as P1 | Critical damage bonus is max dice value + rolled damage before resistance/armor/thresholds |
| P3 | Damage dice/proficiency/threshold pipeline (`damage_thresholds_example.lua`, `damage_roll_modifier.lua`, `damage_roll_proficiency.lua`) | `action.roll.resolve`, `sys.daggerheart.damage.apply` | `action.roll_resolved`, `sys.daggerheart.damage_applied` | Event journal + character HP/armor state | Same as P1 | Proficiency affects dice count only; thresholds apply after mitigation; no HP marked when final damage <= 0 |
| P4 | Fear spotlight + armor mitigation (`fear_spotlight_armor_mitigation.lua`) | `sys.daggerheart.gm_fear.set`, `session.spotlight_set`, `action.roll.resolve`, `sys.daggerheart.damage.apply` | `sys.daggerheart.gm_fear_changed`, `session.spotlight_set`, `action.roll_resolved`, `sys.daggerheart.damage_applied` | GM fear snapshot + session spotlight + event journal + character HP/armor state | Session/core applies mode-controlled; roll event stays journal-only | Fear spend bounds enforced; spotlight/session correlation enforced; max one armor slot marked per damage instance |
| P5 | Multi-target adversary damage resolution (`sweeping_attack_all_targets.lua`, `fireball_orc_pack_multi.lua`) | `action.roll.resolve`, `sys.daggerheart.damage.apply` (per target) | `action.roll_resolved`, `sys.daggerheart.damage_applied` (per target) | Event journal + per-target character HP/armor state | Same as P1 | One attack roll can fan out; hit/miss and mitigation resolve per target |
| P6 | Minion shared/group attack aggregation (`orc_dredge_group_attack.lua`, `minion_group_attack_rats.lua`) | `action.roll.resolve`, `sys.daggerheart.damage.apply` | `action.roll_resolved`, `sys.daggerheart.damage_applied` | Event journal + character HP/armor state | Same as P1 | Group attack uses one shared attack roll; combined minion damage treated as one source before thresholds |
| P7 | Minion overflow defeats (`minion_overflow_damage.lua`, `wild_flame_minion_blast.lua`, `minion_high_threshold_imps.lua`) | `sys.daggerheart.adversary_damage.apply`, `sys.daggerheart.adversary.delete` | `sys.daggerheart.adversary_damage_applied`, `sys.daggerheart.adversary_deleted` | Adversary state projection | Inline apply mode-controlled | Overflow defeats additional minions by threshold ratio; overflow targets remain within valid attack scope |
| P8 | Adversary reaction roll with optional experience spend (`fireball_golum_reaction.lua`) | `action.roll.resolve`, `sys.daggerheart.gm_fear.set` (when spending Fear) | `action.roll_resolved`, `sys.daggerheart.gm_fear_changed` (when spent) | Event journal + GM fear snapshot | Roll event journal-only; GM fear apply mode-controlled | Nat 20 reaction auto-succeeds with no extra bonus; experience bonus requires explicit Fear spend |
| P9 | Reactive damage + cooldown reaction (`ranged_warding_sphere.lua`) | `sys.daggerheart.adversary_damage.apply`, `sys.daggerheart.adversary.update` | `sys.daggerheart.adversary_damage_applied`, `sys.daggerheart.adversary_updated` | Adversary HP/stress and feature-cooldown state | Inline apply mode-controlled | Reaction only triggers when available; cooldown toggles deterministically after trigger and refresh action |
| P10 | Group reactions applying Vulnerable (`ranged_snowblind_trap.lua`) | `action.roll.resolve`, `sys.daggerheart.condition.change`, `sys.daggerheart.damage.apply` | `action.roll_resolved`, `sys.daggerheart.condition_changed`, `sys.daggerheart.damage_applied` | Event journal + character conditions + character HP/armor | Roll event journal-only; system events mode-controlled | Reaction rolls generate no Hope/Fear; Vulnerable applies only on failure branches |
| P11 | Cover disadvantage and severity downgrade (`ranged_take_cover.lua`) | `action.roll.resolve`, `sys.daggerheart.damage.apply` | `action.roll_resolved`, `sys.daggerheart.damage_applied` | Event journal + character HP/armor | Same as P1 | Disadvantage is applied before hit resolution; severity downgrade applies before thresholds |
| P12 | Stress-for-advantage setup (`ranged_steady_aim.lua`) | `sys.daggerheart.stress.spend`, `action.roll.resolve` | `sys.daggerheart.character_state_patched`, `action.roll_resolved` | Character stress state + event journal | Stress spend mode-controlled; roll event journal-only | Stress spend must happen before roll and only once per declared feature use |
| P13 | Stress-cost teleport/reposition (`ranged_battle_teleport.lua`) | `sys.daggerheart.adversary.update`, `action.roll.resolve` | `sys.daggerheart.adversary_updated`, `action.roll_resolved` | Adversary projection + event journal | Adversary update mode-controlled; roll event journal-only | Teleport range bounds enforced; stress/resource cost captured in update payload |
| P14 | Scene-wide reaction with half damage on success (`ranged_arcane_artillery.lua`) | `action.roll.resolve`, `sys.daggerheart.damage.apply` | `action.roll_resolved`, `sys.daggerheart.damage_applied` | Event journal + per-target character HP/armor | Same as P5 | One hazard branch fans out per target; success branch applies half-damage contract |
| P15 | Area hazard + forced movement (`ranged_eruption_hazard.lua`) | `action.roll.resolve`, `sys.daggerheart.damage.apply`, `action.outcome.apply` | `action.roll_resolved`, `sys.daggerheart.damage_applied`, `action.outcome_applied` | Event journal + character HP/armor + outcome metadata | Roll/outcome remain journal-first; system apply mode-controlled | Forced movement and damage branches remain consistent per target reaction outcome |
| P16 | Stress advantage plus reroll replacement (`sam_critical_broadsword.lua`) | `sys.daggerheart.stress.spend`, `action.roll.resolve` | `sys.daggerheart.character_state_patched`, `action.roll_resolved` | Character stress state + event journal | Same as P12 | Reroll replaces original die result; advantage/disadvantage cancellation rules preserved |
| P17 | Move/attack/knockback with stress spend (`skulk_swift_claws.lua`) | `action.roll.resolve`, `sys.daggerheart.stress.spend`, `sys.daggerheart.damage.apply` | `action.roll_resolved`, `sys.daggerheart.character_state_patched`, `sys.daggerheart.damage_applied` | Event journal + character stress + character HP/armor | Roll journal-only; system apply mode-controlled | Stress spend validated before feature outcome; knockback only applies on successful branch |
| P18 | Hidden + advantage-based backstab damage (`skulk_cloaked_backstab.lua`) | `sys.daggerheart.condition.change`, `action.roll.resolve`, `sys.daggerheart.damage.apply` | `sys.daggerheart.condition_changed`, `action.roll_resolved`, `sys.daggerheart.damage_applied` | Character/adversary conditions + event journal + HP/armor state | Condition/damage mode-controlled; roll journal-only | Hidden state lifecycle is deterministic; damage upgrade only applies when attack has advantage |
| P19 | Range-based disadvantage gate (`skulk_reflective_scales.lua`) | `action.roll.resolve` | `action.roll_resolved` | Event journal only | Journal-only apply path | Disadvantage modifier applied only beyond Very Close and recorded in roll metadata |
| P20 | Group attack + Restrained condition (`skulk_icicle_barb.lua`) | `action.roll.resolve`, `sys.daggerheart.damage.apply`, `sys.daggerheart.condition.change` | `action.roll_resolved`, `sys.daggerheart.damage_applied`, `sys.daggerheart.condition_changed` | Event journal + character HP/armor + character conditions | Roll journal-only; system events mode-controlled | Shared attack roll branch is deterministic; Restrained applies only when feature branch is successful |
| P21 | Fear move chain hitting all targets (`improvised_fear_move_bandit_chain.lua`) | `sys.daggerheart.gm_fear.set`, `action.roll.resolve`, `sys.daggerheart.damage.apply` | `sys.daggerheart.gm_fear_changed`, `action.roll_resolved`, `sys.daggerheart.damage_applied` | GM fear snapshot + event journal + per-target HP/armor | GM fear mode-controlled; roll journal-only | Fear spend required before upgraded targeting; shared roll semantics preserved across all targets |
| P22 | Ally reposition + Hidden application (`leader_into_bramble.lua`) | `sys.daggerheart.condition.change`, `action.outcome.apply` | `sys.daggerheart.condition_changed`, `action.outcome_applied` | Character conditions + event journal outcome metadata | Condition mode-controlled; outcome journal-only | Hidden condition lifecycle remains deterministic after reposition side effects |
| P23 | Difficulty escalation after HP loss (`leader_ferocious_defense.lua`) | `sys.daggerheart.adversary.update` | `sys.daggerheart.adversary_updated` | Adversary projection | Inline apply mode-controlled | Difficulty increase occurs only after qualifying HP-loss event; no duplicate escalations |
| P24 | Reaction mitigation via stress spend (`leader_brace_reaction.lua`) | `sys.daggerheart.adversary.update`, `sys.daggerheart.adversary_damage.apply` | `sys.daggerheart.adversary_updated`, `sys.daggerheart.adversary_damage_applied` | Adversary stress/HP projection | Inline apply mode-controlled | Stress mitigation cannot over-apply; final marks respect mitigated severity rules |
| P25 | Countdown-triggered archer volley (`head_guard_on_my_signal.lua`) | `sys.daggerheart.countdown.update`, `action.roll.resolve`, `sys.daggerheart.damage.apply` | `sys.daggerheart.countdown_updated`, `action.roll_resolved`, `sys.daggerheart.damage_applied` | Countdown projection + event journal + per-target HP/armor | Countdown/damage mode-controlled; roll journal-only | Countdown triggers on qualifying PC attack rolls only; triggered volley enforces advantage contract |
| P26 | Rally guards spotlight chain (`head_guard_rally_guards.lua`) | `sys.daggerheart.gm_fear.set`, `session.spotlight_set` | `sys.daggerheart.gm_fear_changed`, `session.spotlight_set` | GM fear snapshot + session spotlight projection | Mode-controlled by runtime policy | Fear spend is exactly 2 for the feature; spotlight fanout bounded by declared ally count |
| P27 | Group action leader/support outcome mapping (`airship_group_roll.lua`, `group_finesse_sneak.lua`, `group_action_escape.lua`) | `action.roll.resolve`, `action.outcome.apply` | `action.roll_resolved`, `action.outcome_applied` | Event journal outcome stream | Journal-only apply path | Only leader action roll generates Hope/Fear; supporter reaction outcomes modify leader total deterministically |
| P28 | Group Hope-loss with GM Fear gain (`terrifying_hope_loss.lua`) | `sys.daggerheart.character_state.patch`, `sys.daggerheart.gm_fear.set` | `sys.daggerheart.character_state_patched`, `sys.daggerheart.gm_fear_changed` | Character hope state + GM fear snapshot | Inline apply mode-controlled | Group hope reduction respects lower bounds; GM fear gain respects cap |
| P29 | Temporary armor lifecycle and rest cleanup (`temporary_armor_bonus.lua`) | `sys.daggerheart.character_temporary_armor.apply`, `sys.daggerheart.rest.take`, `sys.daggerheart.downtime_move.apply` | `sys.daggerheart.character_temporary_armor_applied`, `sys.daggerheart.rest_taken`, `sys.daggerheart.downtime_move_applied` | Temporary armor buckets + character armor state + snapshot rest counters | Inline apply mode-controlled | Temporary armor source/duration validity enforced; rest cleanup removes only matching durations |
| P30 | Spellcast scope and resource side-effects (`spellcast_scope_limit.lua`, `spellcast_hope_cost.lua`) | `action.roll.resolve`, `sys.daggerheart.hope.spend`, `sys.daggerheart.gm_fear.set` | `action.roll_resolved`, `sys.daggerheart.character_state_patched`, `sys.daggerheart.gm_fear_changed` | Event journal + character hope + GM fear snapshot | Roll journal-only; resource updates mode-controlled | Out-of-scope spell effects are rejected pre-command; Hope spend and Fear gain correlate with outcome branch |
| P31 | Clarification-gated placeholder mechanics (`combat_objectives_ritual_rescue_capture.lua`, `companion_experience_stress_clear.lua`, `death_reaction_dig_two_graves.lua`, `encounter_battle_points_example.lua`) | `TBD (see P31 clarification gate)` | `TBD` | `TBD` | Do not implement until command/event contract is resolved in docs | Ambiguous CRB semantics must be resolved before coding; no hidden write-path assumptions |
| P32 | Environment hazard/reaction damage actions (for example `environment_caradhras_pass_avalanche.lua`, `environment_helms_deep_siege_siege_weapons.lua`, `environment_mirkwood_blight_choking_ash.lua`) | `action.roll.resolve`, `sys.daggerheart.damage.apply`, `sys.daggerheart.condition.change` (as needed) | `action.roll_resolved`, `sys.daggerheart.damage_applied`, `sys.daggerheart.condition_changed` | Event journal + per-target HP/armor + conditions | Roll remains journal-only; system state mutations mode-controlled | Per-target reaction outcomes are deterministic; half/full damage branches stay explicit |
| P33 | Ritual/objective countdown pressure flows (for example `environment_dark_tower_usurpation_ritual_nexus.lua`, `environment_isengard_ritual_complete.lua`) | `sys.daggerheart.countdown.update`, `action.outcome.apply`, `sys.daggerheart.gm_fear.set` (as needed) | `sys.daggerheart.countdown_updated`, `action.outcome_applied`, `sys.daggerheart.gm_fear_changed` | Countdown projections + outcome journal + GM fear snapshot | Countdown/fear mode-controlled; outcome journal-only | Countdown tick rules and ritual trigger thresholds are explicit and replay-safe |
| P34 | Environment social/intel/economy discoveries (for example `environment_bree_outpost_rumors.lua`, `environment_prancing_pony_talk.lua`, `environment_moria_ossuary_centuries_of_knowledge.lua`) | `action.roll.resolve`, `action.outcome.apply`, `story.note.add` | `action.roll_resolved`, `action.outcome_applied`, `story.note_added` | Event journal narrative stream | Audit/journal path only | Outcome branch determines discovered info and costs; no direct projection writes for narrative notes |
| P35 | Environment reinforcements/spawn/escalation waves (for example `environment_helms_deep_siege_reinforcements.lua`, `environment_shadow_realm_predators.lua`, `environment_pelennor_battle_reinforcements.lua`) | `sys.daggerheart.adversary.create`, `sys.daggerheart.adversary.update`, `session.spotlight_set` | `sys.daggerheart.adversary_created`, `sys.daggerheart.adversary_updated`, `session.spotlight_set` | Adversary projection + session spotlight | Mode-controlled by runtime policy | Spawn count/range/session scope validated; spotlight transitions remain session-correlated |
| P36 | Environment movement/fall/route-pressure consequences (for example `environment_bruinen_ford_dangerous_crossing.lua`, `environment_misty_ascent_fall.lua`, `environment_osgiliath_ruins_dead_ends.lua`) | `action.roll.resolve`, `sys.daggerheart.damage.apply`, `action.outcome.apply` | `action.roll_resolved`, `sys.daggerheart.damage_applied`, `action.outcome_applied` | Event journal + HP/armor state + outcome metadata | Roll/outcome journal-first; damage mode-controlled | Forced movement/fall effects are encoded as explicit branch outcomes and never inferred post hoc |
| P37 | Persistent aura/condition/resource pressure environments (for example `environment_shadow_realm_unmaking.lua`, `environment_rivendell_sanctuary_healing.lua`, `environment_dark_tower_usurpation_divine_blessing.lua`) | `sys.daggerheart.condition.change`, `sys.daggerheart.character_state.patch`, `sys.daggerheart.gm_fear.set` (as needed) | `sys.daggerheart.condition_changed`, `sys.daggerheart.character_state_patched`, `sys.daggerheart.gm_fear_changed` | Character conditions/resources + GM fear snapshot | Inline apply mode-controlled | Ongoing effects are explicit and bounded; condition/resource changes respect lower/upper caps |
| P38 | Relative-strength difficulty + ambush-reveal patterns (`environment_waylaid_relative_strength.lua`, `environment_waylayers_relative_strength.lua`, `environment_waylayers_where_did_they_come_from.lua`) | `sys.daggerheart.adversary.update`, `action.roll.resolve`, `session.spotlight_set` | `sys.daggerheart.adversary_updated`, `action.roll_resolved`, `session.spotlight_set` | Adversary projection + event journal + spotlight | Mode-controlled by runtime policy | Derived difficulty always recomputes from active adversary set; surprise reveals preserve spotlight integrity |
| P39 | GM move cookbook patterns (`gm_move_examples.lua`, `gm_move_artifact_chase.lua`) | `sys.daggerheart.gm_fear.set`, `session.spotlight_set`, `sys.daggerheart.countdown.update` (as needed) | `sys.daggerheart.gm_fear_changed`, `session.spotlight_set`, `sys.daggerheart.countdown_updated` | GM fear + spotlight + countdown projections | Mode-controlled by runtime policy | Fear spend, spotlight transfer, and countdown pressure remain explicit and ordered |
| P40 | Improvised Fear move variants (`improvised_fear_move_noble_escape.lua`, `improvised_fear_move_rage_boost.lua`, `improvised_fear_move_shadow.lua`) | `sys.daggerheart.gm_fear.set`, `action.roll.resolve`, `sys.daggerheart.damage.apply`, `sys.daggerheart.condition.change` (variant-dependent) | `sys.daggerheart.gm_fear_changed`, `action.roll_resolved`, `sys.daggerheart.damage_applied`, `sys.daggerheart.condition_changed` | GM fear + event journal + HP/armor/conditions | GM fear and state updates mode-controlled; roll journal-only | Improvised move cost must be explicit; variant side effects must map to deterministic event sets |
| P41 | Opportunist passive damage doubling (`orc_archer_opportunist.lua`) | `action.roll.resolve`, `sys.daggerheart.damage.apply` | `action.roll_resolved`, `sys.daggerheart.damage_applied` | Event journal + HP/armor state | Same as P1 | Doubling condition (two+ nearby adversaries) is validated before damage computation |
| P42 | Progress traversal countdown flows (`progress_countdown_climb.lua`, `environment_misty_ascent_progress.lua`, `environment_misty_ascent_pitons.lua`) | `sys.daggerheart.countdown.create`, `sys.daggerheart.countdown.update`, `action.roll.resolve` | `sys.daggerheart.countdown_created`, `sys.daggerheart.countdown_updated`, `action.roll_resolved` | Countdown projections + event journal | Countdown mode-controlled; roll journal-only | Tick rules are deterministic per outcome branch; tooling updates cannot skip countdown events |
| P43 | Social negotiation/conflict outcomes (`social_merchant_haggling.lua`, `social_village_elder_peace.lua`) | `action.roll.resolve`, `action.outcome.apply`, `story.note.add` | `action.roll_resolved`, `action.outcome_applied`, `story.note_added` | Event journal narrative stream | Audit/journal path only | Social success/failure costs are branch-explicit and replayable from the journal |
| P44 | Spellcast flavor/scope guard (`spellcast_flavor_limits.lua`) | `action.roll.resolve`, `action.outcome.reject` | `action.roll_resolved`, `action.outcome_rejected` | Event journal only | Audit/journal path only | Out-of-scope flavor effects are rejected explicitly; no silent coercion into state mutation |

## P31 Clarification Gate (Doc-First)

`P31` remains implementation-blocked until the domain contract questions below are
resolved.

| Scenario | Required clarification questions | Provisional boundary (until clarified) |
| --- | --- | --- |
| `combat_objectives_ritual_rescue_capture.lua` | Does one roll tick multiple objective countdowns, and if so, in what deterministic order? Are objectives modeled as linked pairs only, or as N-way objective groups? Should fanout happen in one command or explicit per-countdown commands? | Keep objective changes explicit with `sys.daggerheart.countdown.update` per objective. No implicit multi-objective fanout command/event contract yet. |
| `companion_experience_stress_clear.lua` | What domain event signals companion "experience completion"? Is companion stress clear allowed outside long-rest and downtime-move rules? Does stress clear target one companion only or all companions tied to the acting PC? | No companion-specific stress-clear command/event is introduced. Scenario coverage should stay on existing rest/downtime stress flows only. |
| `death_reaction_dig_two_graves.lua` | Is death reaction a generic adversary lifecycle rule or stat-block-local behavior only? What is the exact event ordering across adversary defeat, reaction damage, and Hope loss? How are targets selected for Hope loss in multi-PC scenes? | Model only explicit steps (`adversary_attack`, `gm_fear`, `character_state.patch`) with no automatic on-death trigger pipeline. |
| `encounter_battle_points_example.lua` | Is battle-point budgeting runtime state (commands/events) or prep-time guidance only? If runtime, what deterministic conversion rules map point budgets to adversary composition and scaling adjustments? | Treat battle points as documentation/planning metadata only; do not append runtime journal events for encounter budgeting. |

Before implementing `P31`, capture decisions in docs and then add a concrete
timeline row update with command/event types, projection targets, and invariants.

## Source Field Convention

`CharacterStatePatchedPayload.Source` is an optional discriminator set by
transform commands that emit `character_state_patched` events. It enables
journal queries to distinguish the origin of a patch without inspecting field
patterns or introducing separate event types.

| Transform command | Source value |
| --- | --- |
| `sys.daggerheart.hope.spend` | `hope.spend` |
| `sys.daggerheart.stress.spend` | `stress.spend` |
| `sys.daggerheart.character_state.patch` (direct) | _(empty — generic GM/system adjustment)_ |

When adding new transforms that emit `character_state_patched`, set `Source`
to the originating command's short name (the suffix after `sys.daggerheart.`).

## Non-Negotiable Handler Rules

1. Mutating request handlers must use shared orchestration (`executeAndApplyDomainCommand`).
2. Request handlers must not call direct event append APIs.
3. Request handlers must not call direct projection/storage mutation APIs for domain outcomes.
4. Every mutating command path must reject empty decision events unless explicitly audit-only.
5. Inline projection apply behavior must be controlled only by runtime mode policy.

## Required Guard Tests

Use these tests as baseline architecture guardrails:

- `internal/services/game/api/grpc/systems/daggerheart/write_path_arch_test.go`
- `internal/services/game/api/grpc/systems/daggerheart/domain_write_helper_test.go`
- `internal/services/game/api/grpc/game/domain_write_helper_test.go`

When adding a new mutating mechanic, update/add tests so bypass patterns fail fast.

## Mechanics Implementation Priority Tiers

Each priority mechanic (P1-P44) is classified by how much new infrastructure it
requires. Start with Tier 1 (reuse existing types) before tackling higher tiers.

### Tier 1 — Reuse existing types, handler orchestration only

These mechanics compose existing command/event types with no new registrations:

P1, P2, P3, P5, P6, P10, P11, P14, P16, P17, P20, P21, P41

All use existing commands: `action.roll.resolve`, `sys.daggerheart.damage.apply`,
`sys.daggerheart.condition.change`, `sys.daggerheart.gm_fear.set`,
`sys.daggerheart.character_state.patch`.

### Tier 2 — Existing types with new multi-event orchestration

Require new handler choreography (multi-target fanout, countdown-triggered
chains, environment sequences) but no new event types:

P4, P8, P12, P13, P15, P18, P19, P22, P24, P25, P26, P27, P28, P29, P30,
P32-P40, P42, P43

### Tier 3 — May require new command/event types

- **P7** (minion overflow) — batch adversary delete semantics
- **P9** (reaction cooldown) — adversary feature state tracking
- **P23** (difficulty escalation) — adversary stat modification trigger
- **P34** (narrative outcomes) — `story.note.add` / `story.note_added` core type
- **P44** (spellcast rejection) — `action.outcome.reject` / `action.outcome_rejected` core type

### Tier 4 — Blocked on design clarification

**P31** (all sub-scenarios) — see P31 Clarification Gate below.

### Recommended build order

1. **Damage pipeline** (P1-P3): highest-frequency combat mechanics, pure Tier 1
2. **Multi-target and group** (P4-P6): extends damage pipeline to multiple targets
3. **Adversary lifecycle** (P7-P9): surfaces need for new types early
4. **Condition and reaction** (P10-P14, P18-P20, P22): compose existing types
5. **Resource and countdown** (P12, P16, P17, P25, P28, P29): compose existing types
6. **Complex orchestration** (P30, P32-P42): environment and GM move patterns
7. **New types** (P34, P44): add core types, then implement
8. **Deferred** (P31): after clarification gate resolves

## How To Add A New Daggerheart Mechanic

1. Add command and event registrations in the Daggerheart decider/registry.
2. Add a timeline row in this document before implementation.
3. Implement request handler using shared write orchestration only.
4. Implement/update adapter projection handling for emitted event types.
5. Add Red/Green tests:
   - command/event behavior
   - projection/apply behavior
   - architecture bypass guard where relevant
6. Validate runtime mode behavior (`inline_apply_only`, `outbox_apply_only`, `shadow_only`) is explicit for the new path.
