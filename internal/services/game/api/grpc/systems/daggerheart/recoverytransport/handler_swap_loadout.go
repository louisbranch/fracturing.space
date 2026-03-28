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
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) SwapLoadout(ctx context.Context, in *pb.DaggerheartSwapLoadoutRequest) (CharacterStateResult, error) {
	if in == nil {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, "swap loadout request is required")
	}
	if err := h.requireDependencies(false); err != nil {
		return CharacterStateResult{}, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return CharacterStateResult{}, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return CharacterStateResult{}, err
	}
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return CharacterStateResult{}, handleDomainError(ctx, err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return CharacterStateResult{}, handleDomainError(ctx, err)
	}
	if err := requireDaggerheartSystem(record, "campaign system does not support daggerheart loadout"); err != nil {
		return CharacterStateResult{}, err
	}
	sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
	if err != nil {
		return CharacterStateResult{}, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	if err := ensureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return CharacterStateResult{}, err
	}
	if in.Swap == nil {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, "swap is required")
	}
	if _, err := validate.RequiredID(in.Swap.CardId, "card_id"); err != nil {
		return CharacterStateResult{}, err
	}
	if in.Swap.RecallCost < 0 {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, "recall_cost must be non-negative")
	}
	profile, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return CharacterStateResult{}, handleDomainError(ctx, err)
	}
	current, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return CharacterStateResult{}, handleDomainError(ctx, err)
	}
	state := daggerheart.NewCharacterState(daggerheart.CharacterStateConfig{
		CampaignID:  campaignID,
		CharacterID: characterID,
		HP:          current.Hp,
		HPMax:       profile.HpMax,
		Hope:        current.Hope,
		HopeMax:     current.HopeMax,
		Stress:      current.Stress,
		StressMax:   profile.StressMax,
		Armor:       current.Armor,
		ArmorMax:    profile.ArmorMax,
		LifeState:   current.LifeState,
	})
	stressBefore := state.Stress
	if !in.Swap.InRest && in.Swap.RecallCost > 0 {
		if _, _, err := state.SpendResource(daggerheart.ResourceStress, int(in.Swap.RecallCost)); err != nil {
			return CharacterStateResult{}, handleDomainError(ctx, err)
		}
	}
	stressAfter := state.Stress
	loadoutJSON, err := json.Marshal(daggerheartpayload.LoadoutSwapPayload{
		CharacterID:  ids.CharacterID(characterID),
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
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartLoadoutSwap,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "character",
		EntityID:        characterID,
		PayloadJSON:     loadoutJSON,
		MissingEventMsg: "loadout swap did not emit an event",
		ApplyErrMessage: "apply loadout swap event",
	}); err != nil {
		return CharacterStateResult{}, err
	}
	if err := h.deps.ApplyStressConditionChange(ctx, StressConditionInput{
		CampaignID:   campaignID,
		SessionID:    sessionID,
		CharacterID:  characterID,
		Conditions:   current.Conditions,
		StressBefore: stressBefore,
		StressAfter:  stressAfter,
		StressMax:    profile.StressMax,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
	}); err != nil {
		return CharacterStateResult{}, err
	}
	if !in.Swap.InRest && in.Swap.RecallCost > 0 {
		stressSpendJSON, err := json.Marshal(daggerheartpayload.StressSpendPayload{
			CharacterID: ids.CharacterID(characterID),
			Amount:      int(in.Swap.RecallCost),
			Before:      stressBefore,
			After:       stressAfter,
			Source:      "loadout_swap",
		})
		if err != nil {
			return CharacterStateResult{}, grpcerror.Internal("encode stress spend payload", err)
		}
		if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
			CampaignID:      campaignID,
			CommandType:     commandids.DaggerheartStressSpend,
			SessionID:       sessionID,
			SceneID:         sceneID,
			RequestID:       grpcmeta.RequestIDFromContext(ctx),
			InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
			EntityType:      "character",
			EntityID:        characterID,
			PayloadJSON:     stressSpendJSON,
			MissingEventMsg: "stress spend did not emit an event",
			ApplyErrMessage: "apply stress spend event",
		}); err != nil {
			return CharacterStateResult{}, err
		}
	}
	updated, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return CharacterStateResult{}, grpcerror.Internal("load daggerheart state", err)
	}
	return CharacterStateResult{CharacterID: characterID, State: updated}, nil
}
