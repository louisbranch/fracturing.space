package charactermutationtransport

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// requireDependencies keeps all character mutation entrypoints on the same
// minimal dependency contract before they perform campaign or profile reads.
func (h *Handler) requireDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.ExecuteCharacterCommand == nil:
		return status.Error(codes.Internal, "character command executor is not configured")
	default:
		return nil
	}
}

// validateCharacterPreconditions centralizes the campaign mutation and profile
// existence checks shared by inventory-style character mutations.
func (h *Handler) validateCharacterPreconditions(ctx context.Context, campaignID, characterID, operationName string) (projectionstore.DaggerheartCharacterProfile, error) {
	if err := h.requireDependencies(); err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, err
	}
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystemf(record, "campaign system does not support daggerheart %s", operationName); err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, err
	}
	profile, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, grpcerror.HandleDomainError(err)
	}
	return profile, nil
}

// validateLevelUpPreconditions keeps level-up specific wording separate while
// still reusing the same campaign/profile boundary checks.
func (h *Handler) validateLevelUpPreconditions(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterProfile, error) {
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart level up"); err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, err
	}
	profile, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, grpcerror.HandleDomainError(err)
	}
	return profile, nil
}

// executeCharacterCommand keeps command emission on the same dependency guard
// regardless of which mutation path produced the payload.
func (h *Handler) executeCharacterCommand(ctx context.Context, in CharacterCommandInput) error {
	if err := h.requireDependencies(); err != nil {
		return err
	}
	return h.deps.ExecuteCharacterCommand(ctx, in)
}
