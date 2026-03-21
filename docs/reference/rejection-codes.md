# Rejection Codes Reference

Command rejection codes are returned by domain deciders when a command violates
business rules. They appear in `command.Rejection.Code` and are surfaced to
clients through gRPC error details.

## Naming Convention

All codes use `SCREAMING_SNAKE_CASE` with a domain prefix matching the aggregate
or concern that owns the rule:

- `CAMPAIGN_` -- campaign aggregate
- `PARTICIPANT_` -- participant aggregate
- `CHARACTER_` -- character aggregate
- `SESSION_` -- session aggregate
- `SCENE_` -- scene aggregate
- `INVITE_` -- invite aggregate
- `SESSION_READINESS_` -- readiness evaluation (not a decider rejection)

System-specific codes use the system's own vocabulary (e.g., `GM_FEAR_`,
`DAMAGE_`, `ADVERSARY_` for Daggerheart).

## Shared Codes

Defined in `domain/command/`:

| Code | When |
|------|------|
| `PAYLOAD_DECODE_FAILED` | Command payload JSON unmarshalling failed |
| `COMMAND_TYPE_UNSUPPORTED` | No handler registered for the command type |

## Campaign

Defined in `domain/campaign/`:

| Code | When |
|------|------|
| `CAMPAIGN_ALREADY_EXISTS` | Duplicate campaign create |
| `CAMPAIGN_NOT_CREATED` | Operation on a campaign that hasn't been created |
| `CAMPAIGN_NAME_EMPTY` | Name is blank after trimming |
| `CAMPAIGN_INVALID_GAME_SYSTEM` | Unrecognized game system identifier |
| `CAMPAIGN_INVALID_GM_MODE` | Unrecognized GM mode |
| `CAMPAIGN_UPDATE_EMPTY` | Update with no changed fields |
| `CAMPAIGN_INVALID_STATUS` | Unrecognized status value |
| `CAMPAIGN_INVALID_STATUS_TRANSITION` | Illegal lifecycle transition |
| `CAMPAIGN_UPDATE_FIELD_INVALID` | Unknown field key in update payload |
| `CAMPAIGN_LOCALE_INVALID` | Unrecognized locale string |
| `CAMPAIGN_COVER_ASSET_INVALID` | Invalid cover asset identifier |
| `CAMPAIGN_COVER_SET_INVALID` | Invalid cover set identifier |
| `CAMPAIGN_AI_AGENT_ID_REQUIRED` | AI bind without agent ID |
| `CAMPAIGN_PARTICIPANTS_REQUIRED` | Operation requires participants |
| `CAMPAIGN_PARTICIPANT_DUPLICATE` | Duplicate participant in batch |

## Participant

Defined in `domain/participant/`:

| Code | When |
|------|------|
| `PARTICIPANT_ALREADY_JOINED` | Duplicate join |
| `PARTICIPANT_NOT_JOINED` | Operation on absent participant |
| `PARTICIPANT_ID_REQUIRED` | Missing participant ID |
| `PARTICIPANT_NAME_EMPTY` | Name is blank |
| `PARTICIPANT_INVALID_ROLE` | Unrecognized role |
| `PARTICIPANT_INVALID_CONTROLLER` | Unrecognized controller type |
| `PARTICIPANT_INVALID_CAMPAIGN_ACCESS` | Unrecognized access level |
| `PARTICIPANT_INVALID_AVATAR_SET` | Invalid avatar set ID |
| `PARTICIPANT_INVALID_AVATAR_ASSET` | Invalid avatar asset ID |
| `PARTICIPANT_UPDATE_EMPTY` | No changed fields |
| `PARTICIPANT_UPDATE_FIELD_INVALID` | Unknown field key |
| `PARTICIPANT_ALREADY_CLAIMED` | Seat already owned |
| `PARTICIPANT_USER_ID_REQUIRED` | Missing user ID |
| `PARTICIPANT_USER_ID_MISMATCH` | Claim by wrong user |
| `PARTICIPANT_AI_ROLE_REQUIRED` | AI participant missing role |
| `PARTICIPANT_AI_ACCESS_REQUIRED` | AI participant missing access |
| `PARTICIPANT_AI_USER_ID_FORBIDDEN` | AI participant with user ID |
| `PARTICIPANT_AI_IDENTITY_LOCKED` | AI participant identity change |

## Character

Defined in `domain/character/`:

| Code | When |
|------|------|
| `CHARACTER_ALREADY_EXISTS` | Duplicate character create |
| `CHARACTER_ID_REQUIRED` | Missing character ID |
| `CHARACTER_NAME_EMPTY` | Name is blank |
| `CHARACTER_KIND_INVALID` | Unrecognized kind |
| `CHARACTER_INVALID_AVATAR_SET` | Invalid avatar set |
| `CHARACTER_INVALID_AVATAR_ASSET` | Invalid avatar asset |
| `CHARACTER_NOT_CREATED` | Operation on absent character |
| `CHARACTER_UPDATE_EMPTY` | No changed fields |
| `CHARACTER_UPDATE_FIELD_INVALID` | Unknown field key |
| `CHARACTER_ALIASES_INVALID` | Malformed alias list |
| `CHARACTER_OWNER_PARTICIPANT_ID_REQUIRED` | Missing owner |

## Session

Defined in `domain/session/`:

| Code | When |
|------|------|
| `SESSION_ID_REQUIRED` | Missing session ID |
| `SESSION_ALREADY_STARTED` | Duplicate session start |
| `SESSION_NOT_STARTED` | Operation on inactive session |
| `SESSION_GATE_ID_REQUIRED` | Missing gate ID |
| `SESSION_GATE_TYPE_REQUIRED` | Missing gate type |
| `SESSION_GATE_PARTICIPANT_REQUIRED` | Missing gate participant |
| `SESSION_GATE_ALREADY_OPEN` | Duplicate gate open |
| `SESSION_GATE_METADATA_INVALID` | Malformed gate metadata |
| `SESSION_GATE_NOT_OPEN` | Operation on closed gate |
| `SESSION_GATE_MISMATCH` | Response to wrong gate |
| `SESSION_GATE_RESPONSE_INVALID` | Malformed gate response |
| `SESSION_SPOTLIGHT_TYPE_REQUIRED` | Missing spotlight type |

## Scene

Defined in `domain/scene/`:

| Code | When |
|------|------|
| `SCENE_ID_REQUIRED` | Missing scene ID |
| `SCENE_NAME_REQUIRED` | Name is blank |
| `SCENE_CHARACTERS_REQUIRED` | Scene created without characters |
| `SCENE_NOT_FOUND` | Operation on absent scene |
| `SCENE_NOT_ACTIVE` | Operation on ended scene |
| `SCENE_GATE_ID_REQUIRED` | Missing scene gate ID |
| `SCENE_GATE_TYPE_REQUIRED` | Missing scene gate type |
| `SCENE_GATE_ALREADY_OPEN` | Duplicate scene gate open |
| `SCENE_GATE_NOT_OPEN` | Response on closed scene gate |
| `SCENE_CHARACTER_ID_REQUIRED` | Missing character ID in scene op |
| `SCENE_CHARACTER_ALREADY_IN_SCENE` | Duplicate character add |
| `SCENE_CHARACTER_NOT_IN_SCENE` | Remove absent character |
| `SCENE_SPOTLIGHT_TYPE_REQUIRED` | Missing scene spotlight type |
| `SCENE_SPOTLIGHT_NOT_SET` | Clear when no spotlight active |
| `SCENE_SOURCE_SCENE_ID_REQUIRED` | Missing source in move |
| `SCENE_TARGET_SCENE_ID_REQUIRED` | Missing target in move |
| `SCENE_NEW_SCENE_ID_REQUIRED` | Missing new scene in split |

## Invite

Defined in `domain/invite/`:

| Code | When |
|------|------|
| `INVITE_ALREADY_EXISTS` | Duplicate invite create |
| `INVITE_ID_REQUIRED` | Missing invite ID |
| `INVITE_PARTICIPANT_ID_REQUIRED` | Missing participant ID |
| `INVITE_NOT_CREATED` | Operation on absent invite |
| `INVITE_STATUS_INVALID` | Illegal status transition |
| `INVITE_USER_ID_REQUIRED` | Missing user ID |
| `INVITE_JTI_REQUIRED` | Missing JWT identifier |

## Action

Defined in `domain/action/`:

| Code | When |
|------|------|
| `REQUEST_ID_REQUIRED` | Missing action request ID |
| `ROLL_SEQ_REQUIRED` | Missing roll sequence |
| `OUTCOME_ALREADY_APPLIED` | Duplicate outcome application |
| `OUTCOME_EFFECT_SYSTEM_OWNED_FORBIDDEN` | Effect type reserved for system |
| `OUTCOME_EFFECT_TYPE_FORBIDDEN` | Unrecognized effect type |

## Engine

Defined in `domain/engine/`:

| Code | When |
|------|------|
| `CAMPAIGN_ACTIVE_SESSION_LOCKED` | Write blocked by active session lock |
| `SESSION_GATE_OPEN` | Write blocked by open session gate |
| `SCENE_GATE_OPEN` | Write blocked by open scene gate |

## Session Readiness

Defined in `domain/readiness/`. These are evaluation codes, not decider
rejections -- they describe blockers for session start:

| Code | When |
|------|------|
| `SESSION_READINESS_CAMPAIGN_STATUS_DISALLOWS_START` | Campaign status prevents starting |
| `SESSION_READINESS_ACTIVE_SESSION_EXISTS` | Another session is already active |
| `SESSION_READINESS_AI_AGENT_REQUIRED` | AI GM mode but no agent bound |
| `SESSION_READINESS_AI_GM_PARTICIPANT_REQUIRED` | AI GM mode but no AI participant |
| `SESSION_READINESS_GM_REQUIRED` | No GM participant |
| `SESSION_READINESS_PLAYER_REQUIRED` | No player participants |
| `SESSION_READINESS_PLAYER_CHARACTER_REQUIRED` | Player has no character |
| `SESSION_READINESS_CHARACTER_CONTROLLER_REQUIRED` | Character has no controller |
| `SESSION_READINESS_CHARACTER_SYSTEM_REQUIRED` | Character missing system data |

## Daggerheart System

Defined in `domain/systems/daggerheart/`:

| Code | When |
|------|------|
| `GM_FEAR_AFTER_REQUIRED` | Fear update missing target value |
| `GM_FEAR_AFTER_OUT_OF_RANGE` | Fear value outside bounds |
| `GM_FEAR_UNCHANGED` | Fear update with same value |
| `CHARACTER_STATE_PATCH_NO_MUTATION` | Patch with no actual changes |
| `CONDITION_CHANGE_NO_MUTATION` | Condition update with no changes |
| `CONDITION_CHANGE_REMOVE_MISSING` | Remove condition not present |
| `COUNTDOWN_UPDATE_NO_MUTATION` | Countdown update with no changes |
| `COUNTDOWN_BEFORE_MISMATCH` | Optimistic lock failure on countdown |
| `DAMAGE_BEFORE_MISMATCH` | Optimistic lock failure on damage |
| `DAMAGE_ARMOR_SPEND_LIMIT` | Armor spend exceeds available |
| `ADVERSARY_DAMAGE_BEFORE_MISMATCH` | Optimistic lock on adversary damage |
| `ADVERSARY_CONDITION_NO_MUTATION` | Adversary condition no changes |
| `ADVERSARY_CONDITION_REMOVE_MISSING` | Remove absent adversary condition |
| `ADVERSARY_CREATE_NO_MUTATION` | Adversary create with no data |
| `GOLD_INVALID` | Invalid gold value |
| `DOMAIN_CARD_ACQUIRE_INVALID` | Invalid domain card acquisition |
| `EQUIPMENT_SWAP_INVALID` | Invalid equipment swap |
| `CONSUMABLE_INVALID` | Invalid consumable operation |
| `DOWNTIME_MOVE_LIMIT_HIT` | Downtime move limit exceeded |
