package recoverytransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/profile"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) ApplyDeathMove(ctx context.Context, in *pb.DaggerheartApplyDeathMoveRequest) (DeathMoveResult, error) {
	if in == nil {
		return DeathMoveResult{}, status.Error(codes.InvalidArgument, "apply death move request is required")
	}
	if err := h.requireDependencies(true); err != nil {
		return DeathMoveResult{}, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return DeathMoveResult{}, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return DeathMoveResult{}, err
	}
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return DeathMoveResult{}, handleDomainError(ctx, err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return DeathMoveResult{}, handleDomainError(ctx, err)
	}
	if err := requireDaggerheartSystem(record, "campaign system does not support daggerheart death moves"); err != nil {
		return DeathMoveResult{}, err
	}
	sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
	if err != nil {
		return DeathMoveResult{}, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	if err := ensureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return DeathMoveResult{}, err
	}
	move, err := deathMoveFromProto(in.Move)
	if err != nil {
		return DeathMoveResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	if move != daggerheart.DeathMoveRiskItAll && (in.HpClear != nil || in.StressClear != nil) {
		return DeathMoveResult{}, status.Error(codes.InvalidArgument, "hp_clear and stress_clear are only valid for risk it all")
	}
	seed, err := resolveSeed(in.GetRng(), h.deps.SeedGenerator, h.deps.ResolveSeed)
	if err != nil {
		return DeathMoveResult{}, grpcerror.Internal("failed to resolve death move seed", err)
	}
	profile, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return DeathMoveResult{}, handleDomainError(ctx, err)
	}
	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return DeathMoveResult{}, handleDomainError(ctx, err)
	}
	if state.Hp > 0 {
		return DeathMoveResult{}, status.Error(codes.FailedPrecondition, "death move requires hp to be zero")
	}
	if state.LifeState == mechanics.LifeStateDead {
		return DeathMoveResult{}, status.Error(codes.FailedPrecondition, "character is already dead")
	}

	hpMax := profile.HpMax
	if hpMax == 0 {
		hpMax = daggerheartprofile.PCHpMax
	}
	stressMax := profile.StressMax
	if stressMax < 0 {
		stressMax = 0
	}
	hopeMax := state.HopeMax
	if hopeMax == 0 {
		hopeMax = mechanics.HopeMax
	}
	level := profile.Level
	if level == 0 {
		level = daggerheartprofile.PCLevelDefault
	}

	var hpClear *int
	var stressClear *int
	if in.HpClear != nil {
		value := int(in.GetHpClear())
		hpClear = &value
	}
	if in.StressClear != nil {
		value := int(in.GetStressClear())
		stressClear = &value
	}

	outcome, err := daggerheart.ResolveDeathMove(daggerheart.DeathMoveInput{
		Move:             move,
		Level:            level,
		HP:               state.Hp,
		HPMax:            hpMax,
		Hope:             state.Hope,
		HopeMax:          hopeMax,
		Stress:           state.Stress,
		StressMax:        stressMax,
		RiskItAllHPClear: hpClear,
		RiskItAllStClear: stressClear,
		Seed:             seed,
	})
	if err != nil {
		return DeathMoveResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	hpBefore, hpAfter := outcome.HPBefore, outcome.HPAfter
	hopeBefore, hopeAfter := outcome.HopeBefore, outcome.HopeAfter
	hopeMaxBefore, hopeMaxAfter := outcome.HopeMaxBefore, outcome.HopeMaxAfter
	stressBefore, stressAfter := outcome.StressBefore, outcome.StressAfter
	lifeStateBefore := state.LifeState
	if lifeStateBefore == "" {
		lifeStateBefore = daggerheartstate.LifeStateAlive
	}
	patchPayload := daggerheartpayload.CharacterStatePatchPayload{CharacterID: ids.CharacterID(characterID)}
	if hpBefore != hpAfter {
		patchPayload.HPBefore = &hpBefore
		patchPayload.HPAfter = &hpAfter
	}
	if hopeBefore != hopeAfter {
		patchPayload.HopeBefore = &hopeBefore
		patchPayload.HopeAfter = &hopeAfter
	}
	if hopeMaxBefore != hopeMaxAfter {
		patchPayload.HopeMaxBefore = &hopeMaxBefore
		patchPayload.HopeMaxAfter = &hopeMaxAfter
	}
	if stressBefore != stressAfter {
		patchPayload.StressBefore = &stressBefore
		patchPayload.StressAfter = &stressAfter
	}
	if lifeStateBefore != outcome.LifeState {
		lifeStateAfter := outcome.LifeState
		patchPayload.LifeStateBefore = &lifeStateBefore
		patchPayload.LifeStateAfter = &lifeStateAfter
	}
	if patchPayload.HPBefore == nil && patchPayload.HopeBefore == nil && patchPayload.HopeMaxBefore == nil && patchPayload.StressBefore == nil && patchPayload.LifeStateBefore == nil {
		return DeathMoveResult{}, status.Error(codes.Internal, "death move did not change character state")
	}
	payloadJSON, err := json.Marshal(patchPayload)
	if err != nil {
		return DeathMoveResult{}, grpcerror.Internal("encode payload", err)
	}
	if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartCharacterStatePatch,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "character",
		EntityID:        characterID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "death move did not emit an event",
		ApplyErrMessage: "apply character state event",
	}); err != nil {
		return DeathMoveResult{}, err
	}
	updatedState := applyDeathOutcomeState(state, outcome)
	if outcome.LifeState == mechanics.LifeStateDead {
		if err := h.deps.AppendCharacterDeletedEvent(ctx, CharacterDeleteInput{
			CampaignID:  campaignID,
			CharacterID: characterID,
			Reason:      outcome.Move,
		}); err != nil {
			return DeathMoveResult{}, err
		}
	}
	if err := h.deps.ApplyStressConditionChange(ctx, StressConditionInput{
		CampaignID:   campaignID,
		SessionID:    sessionID,
		CharacterID:  characterID,
		Conditions:   state.Conditions,
		StressBefore: stressBefore,
		StressAfter:  stressAfter,
		StressMax:    stressMax,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
	}); err != nil {
		return DeathMoveResult{}, err
	}
	if outcome.LifeState != mechanics.LifeStateDead {
		updated, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
		if err != nil {
			return DeathMoveResult{}, grpcerror.Internal("load daggerheart state", err)
		}
		updatedState = updated
	}
	return DeathMoveResult{
		CharacterID: characterID,
		State:       updatedState,
		Outcome: DeathOutcome{
			Move:          outcome.Move,
			LifeState:     outcome.LifeState,
			HopeDie:       outcome.HopeDie,
			FearDie:       outcome.FearDie,
			HPCleared:     outcome.HPCleared,
			StressCleared: outcome.StressCleared,
			ScarGained:    outcome.ScarGained,
		},
	}, nil
}
