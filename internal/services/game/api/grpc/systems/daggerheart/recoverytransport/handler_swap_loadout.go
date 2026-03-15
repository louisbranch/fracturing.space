package recoverytransport

import (
	"context"
	"encoding/json"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SwapLoadout activates a card from the vault, applying any recall-cost stress
// and related stress-condition side effects before returning updated state.
func (h *Handler) SwapLoadout(ctx context.Context, in *pb.DaggerheartSwapLoadoutRequest) (CharacterStateResult, error) {
	if in == nil {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, "swap loadout request is required")
	}
	if err := h.requireDependencies(false); err != nil {
		return CharacterStateResult{}, err
	}
	if err := requireMutationPayload(in.GetCampaignId(), in.GetCharacterId()); err != nil {
		return CharacterStateResult{}, err
	}

	mutation, err := h.loadCharacterMutationContext(
		ctx,
		in.GetCampaignId(),
		in.GetCharacterId(),
		in.GetSceneId(),
		"campaign system does not support daggerheart loadout",
	)
	if err != nil {
		return CharacterStateResult{}, err
	}
	if in.Swap == nil {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, "swap is required")
	}
	if _, err := validate.RequiredID(in.Swap.CardId, "card_id"); err != nil {
		return CharacterStateResult{}, err
	}
	if err := requireNonNegativeRecallCost(in.Swap.RecallCost); err != nil {
		return CharacterStateResult{}, err
	}
	profile, current, state, err := h.loadMutableCharacterState(ctx, mutation.campaignID, mutation.characterID)
	if err != nil {
		return CharacterStateResult{}, err
	}

	stressBefore := state.Stress
	if !in.Swap.InRest && in.Swap.RecallCost > 0 {
		if _, _, err := state.SpendResource(daggerheart.ResourceStress, int(in.Swap.RecallCost)); err != nil {
			return CharacterStateResult{}, grpcerror.HandleDomainError(err)
		}
	}
	stressAfter := state.Stress
	loadoutJSON, err := json.Marshal(daggerheart.LoadoutSwapPayload{
		CharacterID:  ids.CharacterID(mutation.characterID),
		CardID:       in.Swap.CardId,
		From:         "vault",
		To:           "active",
		RecallCost:   int(in.Swap.RecallCost),
		StressBefore: &stressBefore,
		StressAfter:  &stressAfter,
	})
	if err != nil {
		return CharacterStateResult{}, grpcerror.Internal("encode payload", err)
	}
	if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      mutation.campaignID,
		CommandType:     commandids.DaggerheartLoadoutSwap,
		SessionID:       mutation.sessionID,
		SceneID:         mutation.sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "character",
		EntityID:        mutation.characterID,
		PayloadJSON:     loadoutJSON,
		MissingEventMsg: "loadout swap did not emit an event",
		ApplyErrMessage: "apply loadout swap event",
	}); err != nil {
		return CharacterStateResult{}, err
	}
	if err := h.deps.ApplyStressConditionChange(ctx, StressConditionInput{
		CampaignID:   mutation.campaignID,
		SessionID:    mutation.sessionID,
		CharacterID:  mutation.characterID,
		Conditions:   current.Conditions,
		StressBefore: stressBefore,
		StressAfter:  stressAfter,
		StressMax:    profile.StressMax,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
	}); err != nil {
		return CharacterStateResult{}, err
	}
	if !in.Swap.InRest && in.Swap.RecallCost > 0 {
		stressSpendJSON, err := json.Marshal(daggerheart.StressSpendPayload{
			CharacterID: ids.CharacterID(mutation.characterID),
			Amount:      int(in.Swap.RecallCost),
			Before:      stressBefore,
			After:       stressAfter,
			Source:      "loadout_swap",
		})
		if err != nil {
			return CharacterStateResult{}, grpcerror.Internal("encode stress spend payload", err)
		}
		if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
			CampaignID:      mutation.campaignID,
			CommandType:     commandids.DaggerheartStressSpend,
			SessionID:       mutation.sessionID,
			SceneID:         mutation.sceneID,
			RequestID:       grpcmeta.RequestIDFromContext(ctx),
			InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
			EntityType:      "character",
			EntityID:        mutation.characterID,
			PayloadJSON:     stressSpendJSON,
			MissingEventMsg: "stress spend did not emit an event",
			ApplyErrMessage: "apply stress spend event",
		}); err != nil {
			return CharacterStateResult{}, err
		}
	}

	return h.loadUpdatedCharacterState(ctx, mutation.campaignID, mutation.characterID)
}
