---
title: "Daggerheart Progress Implementation"
parent: "Product"
nav_order: 2
---

# Daggerheart Progress Implementation

## Purpose
Describe implemented mechanics, current PRD coverage, and remaining Phase 2 work based on the Daggerheart SRD mechanics analysis.

## Sources
- PRD: [Daggerheart PRD](daggerheart-PRD.md)

## Implemented Outcomes (Phase 0/1)
- Deterministic action roll engine and duality outcome evaluation with explain/probability surfaces.
- Core character profile/state models with caps, validation, and storage projections.
- Advantage/disadvantage and reaction roll semantics in the rules layer.
- Attack and damage rules (thresholds, massive damage, resistance/immunity, armor reduction).
- Resource transactions for Hope/Stress/Armor with caps and constraints.
- Rest/downtime cadence rules (GM Fear gain, refresh flags, downtime move effects).
- Ability loadout/vault framework and recall cost behavior.
- Campaign-scoped adversaries with optional session scoping and stored combat stats.

## Implemented Outcomes (Event-Driven Wiring)
- System-owned Daggerheart action events emitted from system RPCs.
- Session action rolls emit `action.roll_resolved` with roll kind metadata.
- Mandatory roll outcome effects emit `action.outcome_applied` with roll linkage.
- Damage applied events include HP/armor deltas, mitigation, resistance/immunity, roll linkage, and source actor IDs.
- Rest, downtime, and loadout actions emit system events with GM Fear tracking and downtime deltas.
- Hope and stress spend events emit with before/after values and roll linkage where applicable.
- Death move resolution emits `action.death_move_resolved` with life state, scars, and recovery deltas.
- Blaze of Glory completion emits `action.blaze_of_glory_resolved` and removes the character from campaign availability.
- Attack and reaction outcome application emit `action.attack_resolved` and `action.reaction_resolved` for roll follow-up.
- Session damage rolls emit `action.damage_roll_resolved` with dice results and RNG metadata.
- Session attack flow helper chains roll, outcome, damage roll, and apply damage with linkage validation.
- Session reaction flow helper chains reaction rolls with outcome and reaction outcome application.
- Adversary CRUD emits create/update/delete events and projects adversaries into storage.
- Adversary attack flow chains adversary roll, outcome application, damage roll, and apply damage.

## Mechanics Delta Checklist (From SRD Review)
- [x] Critical success: Hope gain and Stress clear on action roll crits; critical damage on attacks.
- [x] Reaction roll crits: ignore success-side effects, no Hope gain, no Stress clear, no Help an Ally.
- [x] Stress overflow: if forced to mark Stress when full, mark 1 HP instead; last Stress applies Vulnerable.
- [x] Armor rules: unarmored thresholds = level / (2 x level); Armor Slot reduces severity by one step.
- [x] Tag Team roll: Hope grants Hope to all participants; Fear grants GM Fear per PC; counts as one action roll.
- [x] Rest consequences: short rest GM Fear 1d4; long rest GM Fear 1d4 + PCs and advance a long-term countdown.
- [x] Underwater rules: attack disadvantage; breath countdown (3) advances on actions, extra ticks on failures.
- [x] Adversary action rolls: default auto-success unless dramatic; optional d20 check with Fear-spent Experience.

## PRD Coverage Table
| PRD section | Status | Notes |
| --- | --- | --- |
| Core Loop Requirements | Implemented | Action rolls/outcomes wired; spotlight model and GM consequence flow formalized. |
| Resolution System | Implemented | Duality/action/reaction/attack/damage rolls and advantage/disadvantage plus group action/tag team flows are implemented. |
| Character Model | Partial | Core profile/state, traits, resources, thresholds are implemented; full schema breadth still needs coverage. |
| Conditions | Implemented | Hidden/Restrained/Vulnerable supported with condition change events and service API. |
| Progression | Not started | Levels/tiers/advancements/multiclassing not implemented. |
| Combat and Damage | Implemented | Attack/damage rules, thresholds, mitigation, resist/immunity, massive damage; event flows in place. |
| Rest and Downtime | Implemented | Mechanics and events for rest/downtime, GM Fear tracking, refresh flags. |
| Death and Scars | Implemented | Death move resolution and Blaze of Glory flows with events/state changes. |
| Content Archetypes | Partial | Adversary entity and stats are in place; classes/subclasses/cards/items/environments still missing. |
| GM Mechanics | Implemented | GM Fear tracking, moves, countdowns, and adversary d20 rolls are wired. |
| Optional Rules | Partial | Underwater/breath and adversary action checks added; toggles/configs still missing. |
| Determinism and Judgment Boundaries | Partial | Deterministic rolls/outcomes implemented; GM inputs/narrative consequences not modeled. |

## References
- PRD: [Daggerheart PRD](daggerheart-PRD.md)
