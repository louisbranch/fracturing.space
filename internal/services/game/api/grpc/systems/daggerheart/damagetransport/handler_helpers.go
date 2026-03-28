package damagetransport

import (
	"context"
	"encoding/json"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

// autoDropBeastform emits a beastform-drop command when damage causes a
// character to lose their beastform (HP reaches zero or fragile trigger).
func (h *Handler) autoDropBeastform(ctx context.Context, campaignID, sessionID, sceneID, characterID string, previousState, nextState projectionstore.DaggerheartCharacterState) error {
	classState := classStateFromProjection(previousState.ClassState)
	active := classState.ActiveBeastform
	if active == nil {
		return nil
	}
	source := ""
	switch {
	case nextState.Hp == 0:
		source = "beastform.auto_drop.hp_zero"
	case active.DropOnAnyHPMark && nextState.Hp < previousState.Hp:
		source = "beastform.auto_drop.fragile"
	default:
		return nil
	}
	nextClassState := daggerheartstate.WithActiveBeastform(classState, nil)
	payloadJSON, err := json.Marshal(daggerheartpayload.BeastformDropPayload{
		ActorCharacterID: ids.CharacterID(characterID),
		CharacterID:      ids.CharacterID(characterID),
		BeastformID:      active.BeastformID,
		Source:           source,
		ClassStateBefore: classStatePtr(classState),
		ClassStateAfter:  classStatePtr(nextClassState),
	})
	if err != nil {
		return grpcerror.Internal("encode beastform auto-drop payload", err)
	}
	return h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartBeastformDrop,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "character",
		EntityID:        characterID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "beastform auto-drop did not emit an event",
		ApplyErrMessage: source,
	})
}

// classStateFromProjection converts a projection-store class state into the
// domain state type used by command payloads.
func classStateFromProjection(state projectionstore.DaggerheartClassState) daggerheartstate.CharacterClassState {
	return daggerheartstate.CharacterClassState{
		AttackBonusUntilRest:            state.AttackBonusUntilRest,
		EvasionBonusUntilHitOrRest:      state.EvasionBonusUntilHitOrRest,
		DifficultyPenaltyUntilRest:      state.DifficultyPenaltyUntilRest,
		FocusTargetID:                   state.FocusTargetID,
		ActiveBeastform:                 activeBeastformFromProjection(state.ActiveBeastform),
		StrangePatternsNumber:           state.StrangePatternsNumber,
		RallyDice:                       append([]int(nil), state.RallyDice...),
		PrayerDice:                      append([]int(nil), state.PrayerDice...),
		ChannelRawPowerUsedThisLongRest: state.ChannelRawPowerUsedThisLongRest,
		Unstoppable: daggerheartstate.CharacterUnstoppableState{
			Active:           state.Unstoppable.Active,
			CurrentValue:     state.Unstoppable.CurrentValue,
			DieSides:         state.Unstoppable.DieSides,
			UsedThisLongRest: state.Unstoppable.UsedThisLongRest,
		},
	}.Normalized()
}

func classStatePtr(state daggerheartstate.CharacterClassState) *daggerheartstate.CharacterClassState {
	normalized := state.Normalized()
	return &normalized
}

func activeBeastformFromProjection(state *projectionstore.DaggerheartActiveBeastformState) *daggerheartstate.CharacterActiveBeastformState {
	if state == nil {
		return nil
	}
	damageDice := make([]daggerheartstate.CharacterDamageDie, 0, len(state.DamageDice))
	for _, die := range state.DamageDice {
		damageDice = append(damageDice, daggerheartstate.CharacterDamageDie{Count: die.Count, Sides: die.Sides})
	}
	return &daggerheartstate.CharacterActiveBeastformState{
		BeastformID:            state.BeastformID,
		BaseTrait:              state.BaseTrait,
		AttackTrait:            state.AttackTrait,
		TraitBonus:             state.TraitBonus,
		EvasionBonus:           state.EvasionBonus,
		AttackRange:            state.AttackRange,
		DamageDice:             damageDice,
		DamageBonus:            state.DamageBonus,
		DamageType:             state.DamageType,
		EvolutionTraitOverride: state.EvolutionTraitOverride,
		DropOnAnyHPMark:        state.DropOnAnyHPMark,
	}
}
