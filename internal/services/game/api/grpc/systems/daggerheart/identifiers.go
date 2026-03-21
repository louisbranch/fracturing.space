package daggerheart

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

const (
	commandTypeActionOutcomeApply              = commandids.ActionOutcomeApply
	commandTypeActionRollResolve               = commandids.ActionRollResolve
	commandTypeCharacterDelete                 = commandids.CharacterDelete
	commandTypeSessionGateOpen                 = commandids.SessionGateOpen
	commandTypeSessionSpotlightSet             = commandids.SessionSpotlightSet
	commandTypeDaggerheartAdversaryCreate      = commandids.DaggerheartAdversaryCreate
	commandTypeDaggerheartAdversaryDelete      = commandids.DaggerheartAdversaryDelete
	commandTypeDaggerheartAdversaryUpdate      = commandids.DaggerheartAdversaryUpdate
	commandTypeDaggerheartAdversaryCondition   = commandids.DaggerheartAdversaryConditionChange
	commandTypeDaggerheartAdversaryDamageApply = commandids.DaggerheartAdversaryDamageApply
	commandTypeDaggerheartCharacterStatePatch  = commandids.DaggerheartCharacterStatePatch
	commandTypeDaggerheartTemporaryArmorApply  = commandids.DaggerheartCharacterTemporaryArmorApply
	commandTypeDaggerheartConditionChange      = commandids.DaggerheartConditionChange
	commandTypeDaggerheartCountdownCreate      = commandids.DaggerheartCountdownCreate
	commandTypeDaggerheartCountdownDelete      = commandids.DaggerheartCountdownDelete
	commandTypeDaggerheartCountdownUpdate      = commandids.DaggerheartCountdownUpdate
	commandTypeDaggerheartDamageApply          = commandids.DaggerheartDamageApply
	commandTypeDaggerheartGMFearSet            = commandids.DaggerheartGMFearSet
	commandTypeDaggerheartHopeSpend            = commandids.DaggerheartHopeSpend
	commandTypeDaggerheartLoadoutSwap          = commandids.DaggerheartLoadoutSwap
	commandTypeDaggerheartRestTake             = commandids.DaggerheartRestTake
	commandTypeDaggerheartStressSpend          = commandids.DaggerheartStressSpend
	commandTypeDaggerheartLevelUpApply         = commandids.DaggerheartLevelUpApply
	commandTypeDaggerheartGoldUpdate           = commandids.DaggerheartGoldUpdate
	commandTypeDaggerheartDomainCardAcquire    = commandids.DaggerheartDomainCardAcquire
	commandTypeDaggerheartEquipmentSwap        = commandids.DaggerheartEquipmentSwap
	commandTypeDaggerheartConsumableUse        = commandids.DaggerheartConsumableUse
	commandTypeDaggerheartConsumableAcquire    = commandids.DaggerheartConsumableAcquire
)

const (
	eventTypeActionOutcomeApplied           = action.EventTypeOutcomeApplied
	eventTypeActionRollResolved             = action.EventTypeRollResolved
	eventTypeDaggerheartCharacterStatePatch = bridge.EventTypeCharacterStatePatched
	eventTypeDaggerheartConditionChanged    = bridge.EventTypeConditionChanged
	eventTypeDaggerheartGMFearChanged       = bridge.EventTypeGMFearChanged
	eventTypeDaggerheartLevelUpApplied      = bridge.EventTypeLevelUpApplied
	eventTypeDaggerheartGoldUpdated         = bridge.EventTypeGoldUpdated
	eventTypeDaggerheartDomainCardAcquired  = bridge.EventTypeDomainCardAcquired
	eventTypeDaggerheartEquipmentSwapped    = bridge.EventTypeEquipmentSwapped
	eventTypeDaggerheartConsumableUsed      = bridge.EventTypeConsumableUsed
	eventTypeDaggerheartConsumableAcquired  = bridge.EventTypeConsumableAcquired
)
