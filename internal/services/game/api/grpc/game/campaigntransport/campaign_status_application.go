package campaigntransport

import (
	"context"
	"fmt"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (c campaignApplication) EndCampaign(ctx context.Context, campaignID string) (storage.CampaignRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := authz.RequirePolicy(ctx, c.auth, domainauthz.CapabilityManageCampaign(), campaignRecord); err != nil {
		return storage.CampaignRecord{}, err
	}

	if err := ensureNoActiveSession(ctx, c.stores.Session, campaignID); err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := validateCampaignStatusTransition(campaignRecord, campaign.StatusCompleted); err != nil {
		return storage.CampaignRecord{}, err
	}
	actorID, actorType := handler.ResolveCommandActor(ctx)
	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		c.write,
		c.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         handler.CommandTypeCampaignEnd,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "campaign",
			EntityID:     campaignID,
		}),
		domainwrite.Options{},
	)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	updated, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, grpcerror.Internal("load campaign", err)
	}

	return updated, nil
}

func (c campaignApplication) ArchiveCampaign(ctx context.Context, campaignID string) (storage.CampaignRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := authz.RequirePolicy(ctx, c.auth, domainauthz.CapabilityManageCampaign(), campaignRecord); err != nil {
		return storage.CampaignRecord{}, err
	}

	if err := ensureNoActiveSession(ctx, c.stores.Session, campaignID); err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := validateCampaignStatusTransition(campaignRecord, campaign.StatusArchived); err != nil {
		return storage.CampaignRecord{}, err
	}
	actorID, actorType := handler.ResolveCommandActor(ctx)
	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		c.write,
		c.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         handler.CommandTypeCampaignArchive,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "campaign",
			EntityID:     campaignID,
		}),
		domainwrite.Options{},
	)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	updated, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, grpcerror.Internal("load campaign", err)
	}

	return updated, nil
}

func (c campaignApplication) RestoreCampaign(ctx context.Context, campaignID string) (storage.CampaignRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := authz.RequirePolicy(ctx, c.auth, domainauthz.CapabilityManageCampaign(), campaignRecord); err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := validateCampaignStatusTransition(campaignRecord, campaign.StatusDraft); err != nil {
		return storage.CampaignRecord{}, err
	}
	actorID, actorType := handler.ResolveCommandActor(ctx)
	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		c.write,
		c.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         handler.CommandTypeCampaignRestore,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "campaign",
			EntityID:     campaignID,
		}),
		domainwrite.Options{},
	)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	updated, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, grpcerror.Internal("load campaign", err)
	}

	return updated, nil
}

// validateCampaignStatusTransition ensures the target status is allowed from the current state.
func validateCampaignStatusTransition(record storage.CampaignRecord, target campaign.Status) error {
	if campaign.IsStatusTransitionAllowed(record.Status, target) {
		return nil
	}
	fromStatus := campaignStatusLabel(record.Status)
	toStatus := campaignStatusLabel(target)
	return apperrors.WithMetadata(
		apperrors.CodeCampaignInvalidStatusTransition,
		fmt.Sprintf("campaign status transition not allowed: %s -> %s", fromStatus, toStatus),
		map[string]string{"FromStatus": fromStatus, "ToStatus": toStatus},
	)
}

// campaignStatusLabel returns a stable label for campaign status errors.
func campaignStatusLabel(status campaign.Status) string {
	switch status {
	case campaign.StatusDraft:
		return "DRAFT"
	case campaign.StatusActive:
		return "ACTIVE"
	case campaign.StatusCompleted:
		return "COMPLETED"
	case campaign.StatusArchived:
		return "ARCHIVED"
	default:
		return "UNSPECIFIED"
	}
}
