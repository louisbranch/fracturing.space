Change are not backward compatible. No migration needed. Change schema and proto files for logical consistency, not historical compatibility. Assume no live data.

## common proto
- [x] Pronoun is oneof {enum, custom string}
enum: unspecified, she/her, he/him, they/them, it/its
- [x] Pronoun enums are localized
- [x] Pronoun proto is used for social and game messages

## Web service
- [x] campaigns page, add updated_at human time as muted txt at the bottom of each campaign card
- [x] campaign page, change the h1 and title from 'Campaign' => the campaign name
- [x] campaign page overview tab, remove the h2 campaign name
- [x] campaign page overview tab, change the dl to be:
    Campaign Name | Campaign ID
    System | GM mode
    Status | Locale
    Intent | Access Policy
    Theme Prompt
- [x] campaign page participants tab, add participant_count badge
- [x] campaign page characters tab, add character_count badge
- [x] profile settings hides the it/its enum option from selection
- [x] In pt-BR, 'Mode de GM' => 'Modo de MJ', 'Autumn Twilight' => 'Crepúsculo de Outono'
- [x] campaign overview and pronoun labels on campaign/profile views are locale-aware (Status/Locale/Intent/Access + pronoun values)

## Game service
- [x] Default names are localized using campaign's locale
- [x] When a user creates a participant and the user doesn't have a name from social, name the participant 'Mysterious Person' instead of user the user email
- [x] When a user creates a participant and the user doesn't have a pronoun from social, the participant has they/them pronoun
- [x] When an AI participant is created, name it 'Oracle' and assign pronoun it/its
- [x] When a PC character is created by a player participant, assign them as the char controller
- [x] When a NPC character is created by a GM participant, assign them as the controller
- [x] In pt-BR, GM should be 'MJ' and not 'gm' to match GM Mode

## MCP service
- [x] Participant/character update handlers preserve nullable pronoun semantics:
  - omitted `pronouns` => do not update
  - explicit empty `pronouns` => send `PRONOUN_UNSPECIFIED` (clear)
