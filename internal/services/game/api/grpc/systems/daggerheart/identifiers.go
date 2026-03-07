package daggerheart

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

const (
	commandTypeActionOutcomeApply              command.Type = "action.outcome.apply"
	commandTypeActionRollResolve               command.Type = "action.roll.resolve"
	commandTypeCharacterDelete                 command.Type = "character.delete"
	commandTypeSessionGateOpen                 command.Type = "session.gate_open"
	commandTypeSessionSpotlightSet             command.Type = "session.spotlight_set"
	commandTypeDaggerheartAdversaryCreate      command.Type = "sys.daggerheart.adversary.create"
	commandTypeDaggerheartAdversaryDelete      command.Type = "sys.daggerheart.adversary.delete"
	commandTypeDaggerheartAdversaryUpdate      command.Type = "sys.daggerheart.adversary.update"
	commandTypeDaggerheartAdversaryCondition   command.Type = "sys.daggerheart.adversary_condition.change"
	commandTypeDaggerheartAdversaryDamageApply command.Type = "sys.daggerheart.adversary_damage.apply"
	commandTypeDaggerheartCharacterStatePatch  command.Type = "sys.daggerheart.character_state.patch"
	commandTypeDaggerheartTemporaryArmorApply  command.Type = "sys.daggerheart.character_temporary_armor.apply"
	commandTypeDaggerheartConditionChange      command.Type = "sys.daggerheart.condition.change"
	commandTypeDaggerheartCountdownCreate      command.Type = "sys.daggerheart.countdown.create"
	commandTypeDaggerheartCountdownDelete      command.Type = "sys.daggerheart.countdown.delete"
	commandTypeDaggerheartCountdownUpdate      command.Type = "sys.daggerheart.countdown.update"
	commandTypeDaggerheartDamageApply          command.Type = "sys.daggerheart.damage.apply"
	commandTypeDaggerheartDowntimeMoveApply    command.Type = "sys.daggerheart.downtime_move.apply"
	commandTypeDaggerheartGMFearSet            command.Type = "sys.daggerheart.gm_fear.set"
	commandTypeDaggerheartHopeSpend            command.Type = "sys.daggerheart.hope.spend"
	commandTypeDaggerheartLoadoutSwap          command.Type = "sys.daggerheart.loadout.swap"
	commandTypeDaggerheartRestTake             command.Type = "sys.daggerheart.rest.take"
	commandTypeDaggerheartStressSpend          command.Type = "sys.daggerheart.stress.spend"
	commandTypeDaggerheartLevelUpApply         command.Type = "sys.daggerheart.level_up.apply"
	commandTypeDaggerheartGoldUpdate           command.Type = "sys.daggerheart.gold.update"
	commandTypeDaggerheartDomainCardAcquire    command.Type = "sys.daggerheart.domain_card.acquire"
	commandTypeDaggerheartEquipmentSwap        command.Type = "sys.daggerheart.equipment.swap"
	commandTypeDaggerheartConsumableUse        command.Type = "sys.daggerheart.consumable.use"
	commandTypeDaggerheartConsumableAcquire    command.Type = "sys.daggerheart.consumable.acquire"
)

const (
	eventTypeActionOutcomeApplied           event.Type = "action.outcome_applied"
	eventTypeActionRollResolved             event.Type = "action.roll_resolved"
	eventTypeDaggerheartCharacterStatePatch event.Type = "sys.daggerheart.character_state_patched"
	eventTypeDaggerheartConditionChanged    event.Type = "sys.daggerheart.condition_changed"
	eventTypeDaggerheartGMFearChanged       event.Type = "sys.daggerheart.gm_fear_changed"
	eventTypeDaggerheartLevelUpApplied      event.Type = "sys.daggerheart.level_up_applied"
	eventTypeDaggerheartGoldUpdated         event.Type = "sys.daggerheart.gold_updated"
	eventTypeDaggerheartDomainCardAcquired  event.Type = "sys.daggerheart.domain_card_acquired"
	eventTypeDaggerheartEquipmentSwapped    event.Type = "sys.daggerheart.equipment_swapped"
	eventTypeDaggerheartConsumableUsed      event.Type = "sys.daggerheart.consumable_used"
	eventTypeDaggerheartConsumableAcquired  event.Type = "sys.daggerheart.consumable_acquired"
)
