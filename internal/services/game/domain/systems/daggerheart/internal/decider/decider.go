package decider

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

// ── Command type constants ─────────────────────────────────────────────

const (
	commandTypeGMMoveApply                  command.Type = commandids.DaggerheartGMMoveApply
	commandTypeGMFearSet                    command.Type = commandids.DaggerheartGMFearSet
	commandTypeCharacterProfileReplace      command.Type = commandids.DaggerheartCharacterProfileReplace
	commandTypeCharacterProfileDelete       command.Type = commandids.DaggerheartCharacterProfileDelete
	commandTypeCharacterStatePatch          command.Type = commandids.DaggerheartCharacterStatePatch
	commandTypeConditionChange              command.Type = commandids.DaggerheartConditionChange
	commandTypeHopeSpend                    command.Type = commandids.DaggerheartHopeSpend
	commandTypeStressSpend                  command.Type = commandids.DaggerheartStressSpend
	commandTypeLoadoutSwap                  command.Type = commandids.DaggerheartLoadoutSwap
	commandTypeRestTake                     command.Type = commandids.DaggerheartRestTake
	commandTypeCountdownCreate              command.Type = commandids.DaggerheartCountdownCreate
	commandTypeCountdownUpdate              command.Type = commandids.DaggerheartCountdownUpdate
	commandTypeCountdownDelete              command.Type = commandids.DaggerheartCountdownDelete
	commandTypeDamageApply                  command.Type = commandids.DaggerheartDamageApply
	commandTypeAdversaryDamageApply         command.Type = commandids.DaggerheartAdversaryDamageApply
	commandTypeCharacterTemporaryArmorApply command.Type = commandids.DaggerheartCharacterTemporaryArmorApply
	commandTypeAdversaryConditionChange     command.Type = commandids.DaggerheartAdversaryConditionChange
	commandTypeAdversaryCreate              command.Type = commandids.DaggerheartAdversaryCreate
	commandTypeAdversaryUpdate              command.Type = commandids.DaggerheartAdversaryUpdate
	commandTypeAdversaryFeatureApply        command.Type = commandids.DaggerheartAdversaryFeatureApply
	commandTypeAdversaryDelete              command.Type = commandids.DaggerheartAdversaryDelete
	commandTypeEnvironmentEntityCreate      command.Type = commandids.DaggerheartEnvironmentEntityCreate
	commandTypeEnvironmentEntityUpdate      command.Type = commandids.DaggerheartEnvironmentEntityUpdate
	commandTypeEnvironmentEntityDelete      command.Type = commandids.DaggerheartEnvironmentEntityDelete
	commandTypeMultiTargetDamageApply       command.Type = commandids.DaggerheartMultiTargetDamageApply
	commandTypeLevelUpApply                 command.Type = commandids.DaggerheartLevelUpApply
	commandTypeClassFeatureApply            command.Type = commandids.DaggerheartClassFeatureApply
	commandTypeSubclassFeatureApply         command.Type = commandids.DaggerheartSubclassFeatureApply
	commandTypeBeastformTransform           command.Type = commandids.DaggerheartBeastformTransform
	commandTypeBeastformDrop                command.Type = commandids.DaggerheartBeastformDrop
	commandTypeCompanionExperienceBegin     command.Type = commandids.DaggerheartCompanionExperienceBegin
	commandTypeCompanionReturn              command.Type = commandids.DaggerheartCompanionReturn
	commandTypeGoldUpdate                   command.Type = commandids.DaggerheartGoldUpdate
	commandTypeDomainCardAcquire            command.Type = commandids.DaggerheartDomainCardAcquire
	commandTypeEquipmentSwap                command.Type = commandids.DaggerheartEquipmentSwap
	commandTypeConsumableUse                command.Type = commandids.DaggerheartConsumableUse
	commandTypeConsumableAcquire            command.Type = commandids.DaggerheartConsumableAcquire
	commandTypeStatModifierChange           command.Type = commandids.DaggerheartStatModifierChange
)

// ── Rejection code constants ───────────────────────────────────────────

const (
	rejectionCodeGMFearAfterRequired               = "GM_FEAR_AFTER_REQUIRED"
	rejectionCodeGMFearOutOfRange                  = "GM_FEAR_AFTER_OUT_OF_RANGE"
	rejectionCodeGMFearUnchanged                   = "GM_FEAR_UNCHANGED"
	rejectionCodeGMMoveKindUnsupported             = "GM_MOVE_KIND_UNSUPPORTED"
	rejectionCodeGMMoveShapeUnsupported            = "GM_MOVE_SHAPE_UNSUPPORTED"
	rejectionCodeGMMoveDescriptionRequired         = "GM_MOVE_DESCRIPTION_REQUIRED"
	rejectionCodeGMMoveFearSpentRequired           = "GM_MOVE_FEAR_SPENT_REQUIRED"
	rejectionCodeGMMoveInsufficientFear            = "GM_MOVE_INSUFFICIENT_FEAR"
	rejectionCodeCharacterStatePatchNoMutation     = "CHARACTER_STATE_PATCH_NO_MUTATION"
	rejectionCodeConditionChangeNoMutation         = "CONDITION_CHANGE_NO_MUTATION"
	rejectionCodeConditionChangeRemoveMissing      = "CONDITION_CHANGE_REMOVE_MISSING"
	rejectionCodeCountdownUpdateNoMutation         = "COUNTDOWN_UPDATE_NO_MUTATION"
	rejectionCodeCountdownBeforeMismatch           = "COUNTDOWN_BEFORE_MISMATCH"
	rejectionCodeDamageBeforeMismatch              = "DAMAGE_BEFORE_MISMATCH"
	rejectionCodeDamageArmorSpendLimit             = "DAMAGE_ARMOR_SPEND_LIMIT"
	rejectionCodeAdversaryDamageBeforeMismatch     = "ADVERSARY_DAMAGE_BEFORE_MISMATCH"
	rejectionCodeAdversaryConditionNoMutation      = "ADVERSARY_CONDITION_NO_MUTATION"
	rejectionCodeAdversaryConditionRemoveMissing   = "ADVERSARY_CONDITION_REMOVE_MISSING"
	rejectionCodeAdversaryCreateNoMutation         = "ADVERSARY_CREATE_NO_MUTATION"
	rejectionCodeAdversaryFeatureApplyNoMutation   = "ADVERSARY_FEATURE_APPLY_NO_MUTATION"
	rejectionCodeEnvironmentEntityCreateNoMutation = "ENVIRONMENT_ENTITY_CREATE_NO_MUTATION"
	rejectionCodeStatModifierChangeNoMutation      = "STAT_MODIFIER_CHANGE_NO_MUTATION"
	rejectionCodePayloadDecodeFailed               = "PAYLOAD_DECODE_FAILED"
	rejectionCodeCommandTypeUnsupported            = "COMMAND_TYPE_UNSUPPORTED"
)

// ── Decider ────────────────────────────────────────────────────────────

// Decider handles Daggerheart system commands.
type Decider struct {
	handledCommands []command.Type
}

// NewDecider creates a Decider that reports the given command types as handled.
func NewDecider(handledCommands []command.Type) Decider {
	return Decider{handledCommands: handledCommands}
}

type decisionHandler func(daggerheartstate.SnapshotState, bool, command.Command, func() time.Time) command.Decision

var decisionHandlers = map[command.Type]decisionHandler{
	commandTypeGMMoveApply:                  decideGMMoveApply,
	commandTypeGMFearSet:                    decideGMFearSet,
	commandTypeCharacterProfileReplace:      wrapDecisionWithoutState(decideCharacterProfileReplace),
	commandTypeCharacterProfileDelete:       wrapDecisionWithoutState(decideCharacterProfileDelete),
	commandTypeCharacterStatePatch:          decideCharacterStatePatch,
	commandTypeConditionChange:              decideConditionChange,
	commandTypeHopeSpend:                    decideHopeSpend,
	commandTypeStressSpend:                  decideStressSpend,
	commandTypeLoadoutSwap:                  decideLoadoutSwap,
	commandTypeRestTake:                     wrapDecisionWithStateNoSnapshotFlag(decideRestTake),
	commandTypeCountdownCreate:              wrapDecisionWithoutState(decideCountdownCreate),
	commandTypeCountdownUpdate:              decideCountdownUpdate,
	commandTypeCountdownDelete:              wrapDecisionWithoutState(decideCountdownDelete),
	commandTypeDamageApply:                  decideDamageApply,
	commandTypeAdversaryDamageApply:         decideAdversaryDamageApply,
	commandTypeCharacterTemporaryArmorApply: wrapDecisionWithoutState(decideCharacterTemporaryArmorApply),
	commandTypeAdversaryConditionChange:     decideAdversaryConditionChange,
	commandTypeAdversaryCreate:              decideAdversaryCreate,
	commandTypeAdversaryUpdate:              wrapDecisionWithoutState(decideAdversaryUpdate),
	commandTypeAdversaryFeatureApply:        decideAdversaryFeatureApply,
	commandTypeAdversaryDelete:              wrapDecisionWithoutState(decideAdversaryDelete),
	commandTypeEnvironmentEntityCreate:      decideEnvironmentEntityCreate,
	commandTypeEnvironmentEntityUpdate:      wrapDecisionWithoutState(decideEnvironmentEntityUpdate),
	commandTypeEnvironmentEntityDelete:      wrapDecisionWithoutState(decideEnvironmentEntityDelete),
	commandTypeMultiTargetDamageApply:       decideMultiTargetDamageApply,
	commandTypeLevelUpApply:                 decideLevelUpApply,
	commandTypeClassFeatureApply:            decideClassFeatureApply,
	commandTypeSubclassFeatureApply:         decideSubclassFeatureApply,
	commandTypeBeastformTransform:           decideBeastformTransform,
	commandTypeBeastformDrop:                decideBeastformDrop,
	commandTypeCompanionExperienceBegin:     decideCompanionExperienceBegin,
	commandTypeCompanionReturn:              decideCompanionReturn,
	commandTypeGoldUpdate:                   decideGoldUpdate,
	commandTypeDomainCardAcquire:            wrapDecisionWithoutState(decideDomainCardAcquire),
	commandTypeEquipmentSwap:                wrapDecisionWithoutState(decideEquipmentSwap),
	commandTypeConsumableUse:                decideConsumableUse,
	commandTypeConsumableAcquire:            decideConsumableAcquire,
	commandTypeStatModifierChange:           decideStatModifierChange,
}

// DeciderHandledCommands returns the command types this decider handles.
func (d Decider) DeciderHandledCommands() []command.Type {
	return d.handledCommands
}

// Decide returns the decision for a system command against current state.
func (d Decider) Decide(state any, cmd command.Command, now func() time.Time) command.Decision {
	snapshotState, hasSnapshot := daggerheartstate.SnapshotOrDefault(state)
	handler, ok := decisionHandlers[cmd.Type]
	if !ok {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCommandTypeUnsupported,
			Message: "command type is not supported by daggerheart decider",
		})
	}
	return handler(snapshotState, hasSnapshot, cmd, now)
}

func wrapDecisionWithoutState(
	handler func(command.Command, func() time.Time) command.Decision,
) decisionHandler {
	return func(_ daggerheartstate.SnapshotState, _ bool, cmd command.Command, now func() time.Time) command.Decision {
		return handler(cmd, now)
	}
}

func wrapDecisionWithStateNoSnapshotFlag(
	handler func(daggerheartstate.SnapshotState, command.Command, func() time.Time) command.Decision,
) decisionHandler {
	return func(snapshotState daggerheartstate.SnapshotState, _ bool, cmd command.Command, now func() time.Time) command.Decision {
		return handler(snapshotState, cmd, now)
	}
}
