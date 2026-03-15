package recoverytransport

import (
	"context"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// characterMutationContext keeps the shared request-derived transport state for
// character mutation handlers so each endpoint can focus on its own payload.
type characterMutationContext struct {
	campaignID  string
	characterID string
	sessionID   string
	sceneID     string
}

// loadCharacterMutationContext centralizes the repeated campaign/session gate
// validation shared by downtime, temporary armor, and loadout handlers.
func (h *Handler) loadCharacterMutationContext(
	ctx context.Context,
	campaignID string,
	characterID string,
	sceneID string,
	unsupportedMsg string,
) (characterMutationContext, error) {
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return characterMutationContext{}, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return characterMutationContext{}, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, unsupportedMsg); err != nil {
		return characterMutationContext{}, err
	}

	sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
	if err != nil {
		return characterMutationContext{}, err
	}
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return characterMutationContext{}, err
	}

	return characterMutationContext{
		campaignID:  campaignID,
		characterID: characterID,
		sessionID:   sessionID,
		sceneID:     strings.TrimSpace(sceneID),
	}, nil
}

// loadMutableCharacterState returns the profile, stored projection state, and
// domain state view needed by mutation handlers that compute local changes
// before emitting commands.
func (h *Handler) loadMutableCharacterState(
	ctx context.Context,
	campaignID string,
	characterID string,
) (projectionstore.DaggerheartCharacterProfile, projectionstore.DaggerheartCharacterState, *daggerheart.CharacterState, error) {
	profile, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, projectionstore.DaggerheartCharacterState{}, nil, grpcerror.HandleDomainError(err)
	}
	current, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, projectionstore.DaggerheartCharacterState{}, nil, grpcerror.HandleDomainError(err)
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

	return profile, current, state, nil
}

// loadUpdatedCharacterState keeps the final projection reload error message
// consistent across mutation entrypoints.
func (h *Handler) loadUpdatedCharacterState(
	ctx context.Context,
	campaignID string,
	characterID string,
) (CharacterStateResult, error) {
	updated, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return CharacterStateResult{}, grpcerror.Internal("load daggerheart state", err)
	}
	return CharacterStateResult{CharacterID: characterID, State: updated}, nil
}

// requireMutationPayload keeps request-level ID validation uniform for the
// character mutation handlers before they load campaign/session state.
func requireMutationPayload(campaignID string, characterID string) error {
	if _, err := validate.RequiredID(campaignID, "campaign id"); err != nil {
		return err
	}
	if _, err := validate.RequiredID(characterID, "character id"); err != nil {
		return err
	}
	return nil
}

// requireNonNegativeRecallCost preserves the transport-level validation
// contract for loadout swaps before any domain state is loaded.
func requireNonNegativeRecallCost(cost int32) error {
	if cost < 0 {
		return status.Error(codes.InvalidArgument, "recall_cost must be non-negative")
	}
	return nil
}
