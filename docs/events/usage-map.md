# Event Usage Map

## Regenerating
This usage map was assembled by reading `docs/events/event-catalog.md` and scanning code for emitters and appliers. To regenerate the catalog first:

```bash
go generate ./internal/services/game/domain/campaign/event
```

Then rebuild this usage map by rechecking emitters (event appends) and appliers (projection + system adapters).

## Core Events
- `campaign.created`
  - Emitters: `internal/services/game/api/grpc/game/campaign_service.go:111`, `internal/services/game/api/grpc/game/fork_service.go:132`
  - Applier: `internal/services/game/domain/campaign/projection/applier.go:66`
- `campaign.forked`
  - Emitters: `internal/services/game/api/grpc/game/fork_service.go:169`
  - Applier: `internal/services/game/domain/campaign/projection/applier.go:108`
- `campaign.updated`
  - Emitters: `internal/services/game/api/grpc/game/campaign_service.go:334`, `internal/services/game/api/grpc/game/campaign_service.go:412`, `internal/services/game/api/grpc/game/campaign_service.go:483`, `internal/services/game/api/grpc/game/session_service.go:114`
  - Applier: `internal/services/game/domain/campaign/projection/applier.go:108`
- `participant.joined`
  - Emitters: `internal/services/game/api/grpc/game/campaign_service.go:183`, `internal/services/game/api/grpc/game/participant_service.go:138`
  - Applier: `internal/services/game/domain/campaign/projection/applier.go:149`
- `participant.left`
  - Emitters: `internal/services/game/api/grpc/game/participant_service.go:357`
  - Applier: `internal/services/game/domain/campaign/projection/applier.go:319`
- `participant.updated`
  - Emitters: `internal/services/game/api/grpc/game/participant_service.go:271`
  - Applier: `internal/services/game/domain/campaign/projection/applier.go:222`
- `character.created`
  - Emitters: `internal/services/game/api/grpc/game/character_service.go:115`
  - Applier: `internal/services/game/domain/campaign/projection/applier.go:352`
- `character.updated`
  - Emitters: `internal/services/game/api/grpc/game/character_service.go:334`
  - Applier: `internal/services/game/domain/campaign/projection/applier.go:413`
- `character.deleted`
  - Emitters: `internal/services/game/api/grpc/game/character_service.go:422`
  - Applier: `internal/services/game/domain/campaign/projection/applier.go:490`
- `character.profile_updated`
  - Emitters: `internal/services/game/api/grpc/game/character_service.go:189`, `internal/services/game/api/grpc/game/character_service.go:887`
  - Applier: `internal/services/game/domain/campaign/projection/applier.go:667`
- `session.started`
  - Emitters: `internal/services/game/api/grpc/game/session_service.go:137`
  - Applier: `internal/services/game/domain/campaign/projection/applier.go:715`
- `session.ended`
  - Emitters: `internal/services/game/api/grpc/game/session_service.go:297`
  - Applier: `internal/services/game/domain/campaign/projection/applier.go:748`
- `invite.created`
  - Emitters: `internal/services/game/api/grpc/game/invite_service.go:121`
  - Applier: `internal/services/game/domain/campaign/projection/applier.go:635`
- `invite.updated`
  - Emitters: `internal/services/game/api/grpc/game/invite_service.go:284`
  - Applier: `internal/services/game/domain/campaign/projection/applier.go:667`
- `action.roll_resolved`
  - Emitters: none found
  - Applier: none in `internal/services/game/domain/campaign/projection/applier.go`
- `action.outcome_applied`
  - Emitters: `internal/services/game/storage/sqlite/store.go:1220`
  - Applier: none in `internal/services/game/domain/campaign/projection/applier.go`
- `action.outcome_rejected`
  - Emitters: none found
  - Applier: none in `internal/services/game/domain/campaign/projection/applier.go`
- `action.note_added`
  - Emitters: none found
  - Applier: none in `internal/services/game/domain/campaign/projection/applier.go`

## Daggerheart Events
- `action.damage_applied`
  - Emitters: `internal/services/game/api/grpc/systems/daggerheart/actions.go:114`
  - Applier: `internal/services/game/domain/systems/daggerheart/adapter.go:65`
- `action.rest_taken`
  - Emitters: `internal/services/game/api/grpc/systems/daggerheart/actions.go:234`
  - Applier: `internal/services/game/domain/systems/daggerheart/adapter.go:76`
- `action.downtime_move_applied`
  - Emitters: `internal/services/game/api/grpc/systems/daggerheart/actions.go:378`
  - Applier: `internal/services/game/domain/systems/daggerheart/adapter.go:105`
- `action.loadout_swapped`
  - Emitters: `internal/services/game/api/grpc/systems/daggerheart/actions.go:498`
  - Applier: `internal/services/game/domain/systems/daggerheart/adapter.go:116`
- `action.character_state_patched`
  - Emitters: `internal/services/game/api/grpc/game/character_service.go:220`, `internal/services/game/api/grpc/game/snapshot_service.go:218`, `internal/services/game/storage/sqlite/store.go:1171`
  - Applier: `internal/services/game/domain/systems/daggerheart/adapter.go:132`
- `action.gm_fear_changed`
  - Emitters: `internal/services/game/api/grpc/game/snapshot_service.go:313`, `internal/services/game/storage/sqlite/store.go:1074`
  - Applier: `internal/services/game/domain/systems/daggerheart/adapter.go:143`
- `action.stress_spent`
  - Emitters: `internal/services/game/api/grpc/systems/daggerheart/actions.go:532`
  - Applier: none in `internal/services/game/domain/systems/daggerheart/adapter.go`
- `action.hope_spent`
  - Emitters: none found
  - Applier: none in `internal/services/game/domain/systems/daggerheart/adapter.go`
