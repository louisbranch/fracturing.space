# Daggerheart Progress Implementation

## Purpose
Describe the implemented mechanics outcomes that flow from the PRD and summarize the remaining Phase 2 work.

## Source
- PRD: `docs/product/project-requirement-document.md`

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

## PRD Coverage Table
| PRD section | Status | Notes |
| --- | --- | --- |
| Core Loop Requirements | Partial | Action rolls and outcomes are wired; spotlight model and GM consequence flow are not formalized. |
| Resolution System | Implemented | Duality/action/reaction/attack/damage rolls and advantage/disadvantage plus group action/tag team flows are implemented. |
| Character Model | Partial | Core profile/state, traits, resources, thresholds are implemented; full schema breadth still needs coverage. |
| Conditions | Implemented | Hidden/Restrained/Vulnerable supported with condition change events and service API. |
| Progression | Not started | Levels/tiers/advancements/multiclassing not implemented. |
| Combat and Damage | Implemented | Attack/damage rules, thresholds, mitigation, resist/immunity, massive damage; event flows in place. |
| Rest and Downtime | Implemented | Mechanics and events for rest/downtime, GM Fear tracking, refresh flags. |
| Death and Scars | Implemented | Death move resolution and Blaze of Glory flows with events/state changes. |
| Content Archetypes | Partial | Adversary entity and stats are in place; classes/subclasses/cards/items/environments still missing. |
| GM Mechanics | Implemented | GM Fear tracking, moves, countdowns, and adversary d20 rolls are wired. |
| Optional Rules | Not started | Toggleable rules not implemented. |
| Determinism and Judgment Boundaries | Partial | Deterministic rolls/outcomes implemented; GM inputs/narrative consequences not modeled. |

## References
- PRD: `docs/product/project-requirement-document.md`
