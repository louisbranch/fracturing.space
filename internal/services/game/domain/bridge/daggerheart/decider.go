package daggerheart

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
)

const (
	commandTypeGMFearSet                    command.Type = commandids.DaggerheartGMFearSet
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
	commandTypeDowntimeMoveApply            command.Type = commandids.DaggerheartDowntimeMoveApply
	commandTypeCharacterTemporaryArmorApply command.Type = commandids.DaggerheartCharacterTemporaryArmorApply
	commandTypeAdversaryConditionChange     command.Type = commandids.DaggerheartAdversaryConditionChange
	commandTypeAdversaryCreate              command.Type = commandids.DaggerheartAdversaryCreate
	commandTypeAdversaryUpdate              command.Type = commandids.DaggerheartAdversaryUpdate
	commandTypeAdversaryDelete              command.Type = commandids.DaggerheartAdversaryDelete
	commandTypeMultiTargetDamageApply       command.Type = commandids.DaggerheartMultiTargetDamageApply

	rejectionCodeGMFearAfterRequired             = "GM_FEAR_AFTER_REQUIRED"
	rejectionCodeGMFearOutOfRange                = "GM_FEAR_AFTER_OUT_OF_RANGE"
	rejectionCodeGMFearUnchanged                 = "GM_FEAR_UNCHANGED"
	rejectionCodeCharacterStatePatchNoMutation   = "CHARACTER_STATE_PATCH_NO_MUTATION"
	rejectionCodeConditionChangeNoMutation       = "CONDITION_CHANGE_NO_MUTATION"
	rejectionCodeConditionChangeRemoveMissing    = "CONDITION_CHANGE_REMOVE_MISSING"
	rejectionCodeCountdownUpdateNoMutation       = "COUNTDOWN_UPDATE_NO_MUTATION"
	rejectionCodeCountdownBeforeMismatch         = "COUNTDOWN_BEFORE_MISMATCH"
	rejectionCodeDamageBeforeMismatch            = "DAMAGE_BEFORE_MISMATCH"
	rejectionCodeDamageArmorSpendLimit           = "DAMAGE_ARMOR_SPEND_LIMIT"
	rejectionCodeAdversaryDamageBeforeMismatch   = "ADVERSARY_DAMAGE_BEFORE_MISMATCH"
	rejectionCodeAdversaryConditionNoMutation    = "ADVERSARY_CONDITION_NO_MUTATION"
	rejectionCodeAdversaryConditionRemoveMissing = "ADVERSARY_CONDITION_REMOVE_MISSING"
	rejectionCodeAdversaryCreateNoMutation       = "ADVERSARY_CREATE_NO_MUTATION"
	rejectionCodePayloadDecodeFailed             = "PAYLOAD_DECODE_FAILED"
	rejectionCodeCommandTypeUnsupported          = "COMMAND_TYPE_UNSUPPORTED"
)

// Decider handles Daggerheart system commands.
type Decider struct{}

type daggerheartDecisionHandler func(SnapshotState, bool, command.Command, func() time.Time) command.Decision

var daggerheartDecisionHandlers = map[command.Type]daggerheartDecisionHandler{
	commandTypeGMFearSet:                    decideGMFearSet,
	commandTypeCharacterStatePatch:          decideCharacterStatePatch,
	commandTypeConditionChange:              decideConditionChange,
	commandTypeHopeSpend:                    decideHopeSpend,
	commandTypeStressSpend:                  decideStressSpend,
	commandTypeLoadoutSwap:                  wrapDaggerheartDecisionWithoutState(decideLoadoutSwap),
	commandTypeRestTake:                     wrapDaggerheartDecisionWithStateNoSnapshotFlag(decideRestTake),
	commandTypeCountdownCreate:              wrapDaggerheartDecisionWithoutState(decideCountdownCreate),
	commandTypeCountdownUpdate:              decideCountdownUpdate,
	commandTypeCountdownDelete:              wrapDaggerheartDecisionWithoutState(decideCountdownDelete),
	commandTypeDamageApply:                  decideDamageApply,
	commandTypeAdversaryDamageApply:         decideAdversaryDamageApply,
	commandTypeDowntimeMoveApply:            wrapDaggerheartDecisionWithoutState(decideDowntimeMoveApply),
	commandTypeCharacterTemporaryArmorApply: wrapDaggerheartDecisionWithoutState(decideCharacterTemporaryArmorApply),
	commandTypeAdversaryConditionChange:     decideAdversaryConditionChange,
	commandTypeAdversaryCreate:              decideAdversaryCreate,
	commandTypeAdversaryUpdate:              wrapDaggerheartDecisionWithoutState(decideAdversaryUpdate),
	commandTypeAdversaryDelete:              wrapDaggerheartDecisionWithoutState(decideAdversaryDelete),
	commandTypeMultiTargetDamageApply:       decideMultiTargetDamageApply,
}

// DeciderHandledCommands returns the command types this decider handles.
// Derived from daggerheartCommandDefinitions so the list stays in sync.
func (Decider) DeciderHandledCommands() []command.Type {
	return commandTypesFromDefinitions()
}

// Decide returns the decision for a system command against current state.
func (Decider) Decide(state any, cmd command.Command, now func() time.Time) command.Decision {
	snapshotState, hasSnapshot := snapshotOrDefault(state)
	handler, ok := daggerheartDecisionHandlers[cmd.Type]
	if !ok {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCommandTypeUnsupported,
			Message: "command type is not supported by daggerheart decider",
		})
	}
	return handler(snapshotState, hasSnapshot, cmd, now)
}

func wrapDaggerheartDecisionWithoutState(
	handler func(command.Command, func() time.Time) command.Decision,
) daggerheartDecisionHandler {
	return func(_ SnapshotState, _ bool, cmd command.Command, now func() time.Time) command.Decision {
		return handler(cmd, now)
	}
}

func wrapDaggerheartDecisionWithStateNoSnapshotFlag(
	handler func(SnapshotState, command.Command, func() time.Time) command.Decision,
) daggerheartDecisionHandler {
	return func(snapshotState SnapshotState, _ bool, cmd command.Command, now func() time.Time) command.Decision {
		return handler(snapshotState, cmd, now)
	}
}
