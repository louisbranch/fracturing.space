package recoverytransport

import (
	"context"
	"encoding/json"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ApplyDeathMove resolves a death move, applies the resulting state patch, and
// then runs the delete/stress side effects owned by recovery transport.
func (h *Handler) ApplyDeathMove(ctx context.Context, in *pb.DaggerheartApplyDeathMoveRequest) (DeathMoveResult, error) {
	if in == nil {
		return DeathMoveResult{}, status.Error(codes.InvalidArgument, "apply death move request is required")
	}
	if err := h.requireDependencies(true); err != nil {
		return DeathMoveResult{}, err
	}
	if err := requireMutationPayload(in.GetCampaignId(), in.GetCharacterId()); err != nil {
		return DeathMoveResult{}, err
	}

	mutation, err := h.loadCharacterMutationContext(
		ctx,
		in.GetCampaignId(),
		in.GetCharacterId(),
		in.GetSceneId(),
		"campaign system does not support daggerheart death moves",
	)
	if err != nil {
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
	profile, state, outcome, err := h.resolveDeathMoveOutcome(ctx, mutation, move, seed, in)
	if err != nil {
		return DeathMoveResult{}, err
	}
	if err := h.applyDeathMoveState(ctx, mutation, state, outcome); err != nil {
		return DeathMoveResult{}, err
	}
	if outcome.LifeState == daggerheart.LifeStateDead {
		if err := h.deps.AppendCharacterDeletedEvent(ctx, CharacterDeleteInput{
			CampaignID:  mutation.campaignID,
			CharacterID: mutation.characterID,
			Reason:      outcome.Move,
		}); err != nil {
			return DeathMoveResult{}, err
		}
	}
	if err := h.deps.ApplyStressConditionChange(ctx, StressConditionInput{
		CampaignID:   mutation.campaignID,
		SessionID:    mutation.sessionID,
		CharacterID:  mutation.characterID,
		Conditions:   state.Conditions,
		StressBefore: outcome.StressBefore,
		StressAfter:  outcome.StressAfter,
		StressMax:    profile.StressMax,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
	}); err != nil {
		return DeathMoveResult{}, err
	}

	updated, err := h.loadUpdatedCharacterState(ctx, mutation.campaignID, mutation.characterID)
	if err != nil {
		return DeathMoveResult{}, err
	}
	return DeathMoveResult{
		CharacterID: updated.CharacterID,
		State:       updated.State,
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

// resolveDeathMoveOutcome loads the current profile/state and maps the incoming
// request into the domain death-move resolver.
func (h *Handler) resolveDeathMoveOutcome(
	ctx context.Context,
	mutation characterMutationContext,
	move string,
	seed int64,
	in *pb.DaggerheartApplyDeathMoveRequest,
) (projectionstore.DaggerheartCharacterProfile, projectionstore.DaggerheartCharacterState, daggerheart.DeathMoveOutcome, error) {
	profile, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, mutation.campaignID, mutation.characterID)
	if err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, projectionstore.DaggerheartCharacterState{}, daggerheart.DeathMoveOutcome{}, grpcerror.HandleDomainError(err)
	}
	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, mutation.campaignID, mutation.characterID)
	if err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, projectionstore.DaggerheartCharacterState{}, daggerheart.DeathMoveOutcome{}, grpcerror.HandleDomainError(err)
	}
	if state.Hp > 0 {
		return projectionstore.DaggerheartCharacterProfile{}, projectionstore.DaggerheartCharacterState{}, daggerheart.DeathMoveOutcome{}, status.Error(codes.FailedPrecondition, "death move requires hp to be zero")
	}
	if state.LifeState == daggerheart.LifeStateDead {
		return projectionstore.DaggerheartCharacterProfile{}, projectionstore.DaggerheartCharacterState{}, daggerheart.DeathMoveOutcome{}, status.Error(codes.FailedPrecondition, "character is already dead")
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
		hopeMax = daggerheart.HopeMax
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
		return projectionstore.DaggerheartCharacterProfile{}, projectionstore.DaggerheartCharacterState{}, daggerheart.DeathMoveOutcome{}, status.Error(codes.InvalidArgument, err.Error())
	}

	return profile, state, outcome, nil
}

// applyDeathMoveState converts the resolved outcome into a single character
// patch command so recovery transport keeps one authority for the mutation.
func (h *Handler) applyDeathMoveState(
	ctx context.Context,
	mutation characterMutationContext,
	state projectionstore.DaggerheartCharacterState,
	outcome daggerheart.DeathMoveOutcome,
) error {
	lifeStateBefore := state.LifeState
	if lifeStateBefore == "" {
		lifeStateBefore = daggerheart.LifeStateAlive
	}
	patchPayload := daggerheart.CharacterStatePatchPayload{CharacterID: ids.CharacterID(mutation.characterID)}
	if outcome.HPBefore != outcome.HPAfter {
		patchPayload.HPBefore = &outcome.HPBefore
		patchPayload.HPAfter = &outcome.HPAfter
	}
	if outcome.HopeBefore != outcome.HopeAfter {
		patchPayload.HopeBefore = &outcome.HopeBefore
		patchPayload.HopeAfter = &outcome.HopeAfter
	}
	if outcome.HopeMaxBefore != outcome.HopeMaxAfter {
		patchPayload.HopeMaxBefore = &outcome.HopeMaxBefore
		patchPayload.HopeMaxAfter = &outcome.HopeMaxAfter
	}
	if outcome.StressBefore != outcome.StressAfter {
		patchPayload.StressBefore = &outcome.StressBefore
		patchPayload.StressAfter = &outcome.StressAfter
	}
	if lifeStateBefore != outcome.LifeState {
		lifeStateAfter := outcome.LifeState
		patchPayload.LifeStateBefore = &lifeStateBefore
		patchPayload.LifeStateAfter = &lifeStateAfter
	}
	if patchPayload.HPBefore == nil &&
		patchPayload.HopeBefore == nil &&
		patchPayload.HopeMaxBefore == nil &&
		patchPayload.StressBefore == nil &&
		patchPayload.LifeStateBefore == nil {
		return status.Error(codes.Internal, "death move did not change character state")
	}
	payloadJSON, err := json.Marshal(patchPayload)
	if err != nil {
		return grpcerror.Internal("encode payload", err)
	}
	return h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      mutation.campaignID,
		CommandType:     commandids.DaggerheartCharacterStatePatch,
		SessionID:       mutation.sessionID,
		SceneID:         mutation.sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "character",
		EntityID:        mutation.characterID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "death move did not emit an event",
		ApplyErrMessage: "apply character state event",
	})
}
