package daggerheart

import event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"

// EventType constants for Daggerheart projection and adapter event handling.
const (
	EventTypeDamageApplied                  event.Type = "sys.daggerheart.damage_applied"
	EventTypeRestTaken                      event.Type = "sys.daggerheart.rest_taken"
	EventTypeDowntimeMoveApplied            event.Type = "sys.daggerheart.downtime_move_applied"
	EventTypeLoadoutSwapped                 event.Type = "sys.daggerheart.loadout_swapped"
	EventTypeCharacterStatePatched          event.Type = "sys.daggerheart.character_state_patched"
	EventTypeConditionChanged               event.Type = "sys.daggerheart.condition_changed"
	EventTypeGMFearChanged                  event.Type = "sys.daggerheart.gm_fear_changed"
	EventTypeCountdownCreated               event.Type = "sys.daggerheart.countdown_created"
	EventTypeCountdownUpdated               event.Type = "sys.daggerheart.countdown_updated"
	EventTypeCountdownDeleted               event.Type = "sys.daggerheart.countdown_deleted"
	EventTypeCharacterTemporaryArmorApplied event.Type = "sys.daggerheart.character_temporary_armor_applied"

	EventTypeAdversaryCreated          event.Type = "sys.daggerheart.adversary_created"
	EventTypeAdversaryConditionChanged event.Type = "sys.daggerheart.adversary_condition_changed"
	EventTypeAdversaryDamageApplied    event.Type = "sys.daggerheart.adversary_damage_applied"
	EventTypeAdversaryUpdated          event.Type = "sys.daggerheart.adversary_updated"
	EventTypeAdversaryDeleted          event.Type = "sys.daggerheart.adversary_deleted"
)
