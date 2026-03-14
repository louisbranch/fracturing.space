package game

import (
	"context"
	"encoding/json"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type campaignUpdateInput struct {
	Name        *string
	ThemePrompt *string
	Locale      *commonv1.Locale
}

func (c campaignApplication) UpdateCampaign(ctx context.Context, campaignID string, input campaignUpdateInput) (storage.CampaignRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := requirePolicy(ctx, c.auth, domainauthz.CapabilityManageCampaign, campaignRecord); err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.CampaignRecord{}, err
	}

	fields := make(map[string]string, 3)
	if input.Name != nil {
		normalizedName := strings.TrimSpace(*input.Name)
		if normalizedName != strings.TrimSpace(campaignRecord.Name) {
			fields["name"] = normalizedName
		}
	}
	if input.ThemePrompt != nil {
		normalizedThemePrompt := strings.TrimSpace(*input.ThemePrompt)
		if normalizedThemePrompt != strings.TrimSpace(campaignRecord.ThemePrompt) {
			fields["theme_prompt"] = normalizedThemePrompt
		}
	}
	if input.Locale != nil && *input.Locale != campaignRecord.Locale {
		fields["locale"] = platformi18n.LocaleString(*input.Locale)
	}
	if len(fields) == 0 {
		return campaignRecord, nil
	}

	actorID, actorType := resolveCommandActor(ctx)
	payloadJSON, err := json.Marshal(campaign.UpdatePayload{Fields: fields})
	if err != nil {
		return storage.CampaignRecord{}, grpcerror.Internal("encode payload", err)
	}

	_, err = executeAndApplyDomainCommand(
		ctx,
		c.write,
		c.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeCampaignUpdate,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "campaign",
			EntityID:     campaignID,
			PayloadJSON:  payloadJSON,
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

func (c campaignApplication) SetCampaignCover(ctx context.Context, campaignID, coverAssetID, coverSetID string) (storage.CampaignRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := requirePolicy(ctx, c.auth, domainauthz.CapabilityManageCampaign, campaignRecord); err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.CampaignRecord{}, err
	}

	actorID, actorType := resolveCommandActor(ctx)

	fields := map[string]string{"cover_asset_id": coverAssetID}
	if strings.TrimSpace(coverSetID) != "" {
		fields["cover_set_id"] = strings.TrimSpace(coverSetID)
	}
	payload := campaign.UpdatePayload{Fields: fields}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.CampaignRecord{}, grpcerror.Internal("encode payload", err)
	}

	_, err = executeAndApplyDomainCommand(
		ctx,
		c.write,
		c.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeCampaignUpdate,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "campaign",
			EntityID:     campaignID,
			PayloadJSON:  payloadJSON,
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
