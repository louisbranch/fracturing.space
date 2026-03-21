# Daggerheart Reference Lookup

## When to Search

- Before adjudicating any mechanic, subclass feature, domain card, or ancestry trait.
- When a player references a specific ability, item, or rule by name.
- When you are unsure about the exact wording of a game term.

## Search Strategy

1. Use system_reference_search with specific terms (e.g. "Evasion" not "dodge ability").
2. If the first search returns no results, try synonyms or broader terms.
3. Use system_reference_read to get the full document when the search snippet is not enough.
4. Reference IDs from search results can be passed directly to system_reference_read.

## Common Lookup Patterns

- Character abilities: search by ability name or subclass name
- Domain cards: search by card name or domain name
- Ancestry features: search by ancestry name
- Combat rules: search by mechanic name (e.g. "armor", "stress", "hit points")
- Conditions: search by condition name (e.g. "frightened", "restrained")
