# Daggerheart Reference Lookup

## When to Search

- First confirm what the current character can do with character_sheet_read when the question is about that character's actual loadout or state.
- When a player references a specific ability, item, or rule by name.
- When you are unsure about the exact wording of a game term, named feature, or procedural sequence.
- When a turn explicitly requires a playbook consult.

## When Not to Search

- Do not search just because the turn is mechanical.
- Do not search Fear or spotlight guidance before those mechanics are actually relevant on the current turn.
- If the sheet, board, and dedicated mechanics tool already make the path obvious, act without a reference lookup.

## Search Strategy

1. Use character_sheet_read first when you need to confirm the acting character's traits, Hope, equipment, domain cards, or active features.
2. Use system_reference_search with specific terms (e.g. "Evasion" not "dodge ability").
3. If the first search returns no results, try synonyms or broader terms.
4. Use system_reference_read to get the full document when the search snippet is not enough.
5. Reference IDs from search results can be passed directly to system_reference_read.
6. Once you know the rule and the current character capability, use the authoritative mechanics tool for the mutation itself, such as daggerheart_action_roll_resolve, daggerheart_attack_flow_resolve, daggerheart_adversary_attack_flow_resolve, daggerheart_incoming_damage_resolve, daggerheart_reaction_flow_resolve, daggerheart_gm_move_apply, daggerheart_adversary_create, daggerheart_adversary_update, daggerheart_scene_countdown_create, or daggerheart_scene_countdown_advance.
7. When the question is procedural rather than definitional and the correct tool sequence is unclear, prefer a playbook search such as "combat procedures", "gm fear spotlight", or "action roll outcomes".
8. Stop after one search and one read unless those results are clearly insufficient.

## Common Lookup Patterns

- Current capability checks: inspect the sheet for traits, gear, Hope, conditions, domain cards, and class/subclass features
- Character abilities: search by ability name or subclass name
- Domain cards: search by card name or domain name
- Ancestry features: search by ancestry name
- Combat rules: search by mechanic name (e.g. "armor", "stress", "hit points")
- Combat procedures: after lookup, prefer the dedicated attack/reaction/group/tag-team flow tools and daggerheart_incoming_damage_resolve over stitching together lower-level rolls by hand
- Combat defense pauses: if a flow returns `choice_required`, ask the affected player for that one defense or mitigation choice, then retry the same tool with the returned `checkpoint_id` and explicit decision, or use daggerheart_incoming_damage_resolve when the unresolved step is pure incoming damage
- Board pressure: read the board, then use the adversary/countdown tools to externalize spotlight pressure and escalating danger, and re-read the board when the next beat depends on the updated state
- Conditions: search by condition name (e.g. "frightened", "restrained")
