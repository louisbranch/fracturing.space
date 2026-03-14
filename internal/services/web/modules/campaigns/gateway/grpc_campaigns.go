package gateway

import (
	"context"
	"strconv"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// ListCampaigns returns the package view collection for this workflow.
func (g catalogReadGateway) ListCampaigns(ctx context.Context) ([]campaignapp.CampaignSummary, error) {
	resp, err := g.read.Campaign.ListCampaigns(ctx, &statev1.ListCampaignsRequest{PageSize: 10})
	if err != nil {
		return nil, err
	}
	items := make([]campaignapp.CampaignSummary, 0, len(resp.GetCampaigns()))
	for _, campaign := range resp.GetCampaigns() {
		if campaign == nil {
			continue
		}
		campaignID := strings.TrimSpace(campaign.GetId())
		name := strings.TrimSpace(campaign.GetName())
		if name == "" {
			name = campaignID
		}
		items = append(items, campaignapp.CampaignSummary{
			ID:                campaignID,
			Name:              name,
			Theme:             campaignapp.TruncateCampaignTheme(campaign.GetThemePrompt()),
			CoverImageURL:     campaignapp.CampaignCoverImageURL(g.assetBaseURL, campaignID, campaign.GetCoverSetId(), campaign.GetCoverAssetId()),
			ParticipantCount:  strconv.FormatInt(int64(campaign.GetParticipantCount()), 10),
			CharacterCount:    strconv.FormatInt(int64(campaign.GetCharacterCount()), 10),
			CreatedAtUnixNano: campaignCreatedAtUnixNano(campaign),
			UpdatedAtUnixNano: campaignUpdatedAtUnixNano(campaign),
		})
	}
	return items, nil
}

// CampaignName centralizes this web behavior in one helper seam.
func (g workspaceReadGateway) CampaignName(ctx context.Context, campaignID string) (string, error) {
	resp, err := g.read.Campaign.GetCampaign(ctx, &statev1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil {
		return "", err
	}
	if resp.GetCampaign() == nil {
		return "", nil
	}
	return strings.TrimSpace(resp.GetCampaign().GetName()), nil
}

// CampaignWorkspace centralizes this web behavior in one helper seam.
func (g workspaceReadGateway) CampaignWorkspace(ctx context.Context, campaignID string) (campaignapp.CampaignWorkspace, error) {
	resp, err := g.read.Campaign.GetCampaign(ctx, &statev1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil {
		return campaignapp.CampaignWorkspace{}, err
	}
	if resp.GetCampaign() == nil {
		return campaignapp.CampaignWorkspace{}, apperrors.E(apperrors.KindNotFound, "campaign not found")
	}
	campaign := resp.GetCampaign()
	resolvedCampaignID := strings.TrimSpace(campaign.GetId())
	if resolvedCampaignID == "" {
		resolvedCampaignID = strings.TrimSpace(campaignID)
	}
	name := strings.TrimSpace(campaign.GetName())
	if name == "" {
		name = resolvedCampaignID
	}
	return campaignapp.CampaignWorkspace{
		ID:               resolvedCampaignID,
		Name:             name,
		Theme:            strings.TrimSpace(campaign.GetThemePrompt()),
		System:           campaignSystemLabel(campaign.GetSystem()),
		GMMode:           campaignGMModeLabel(campaign.GetGmMode()),
		AIAgentID:        strings.TrimSpace(campaign.GetAiAgentId()),
		Status:           campaignStatusLabel(campaign.GetStatus()),
		Locale:           campaignLocaleLabel(campaign.GetLocale()),
		Intent:           campaignIntentLabel(campaign.GetIntent()),
		AccessPolicy:     campaignAccessPolicyLabel(campaign.GetAccessPolicy()),
		ParticipantCount: strconv.FormatInt(int64(campaign.GetParticipantCount()), 10),
		CharacterCount:   strconv.FormatInt(int64(campaign.GetCharacterCount()), 10),
		CoverPreviewURL:  campaignapp.CampaignCoverPreviewImageURL(g.assetBaseURL, resolvedCampaignID, campaign.GetCoverSetId(), campaign.GetCoverAssetId()),
		CoverImageURL:    campaignapp.CampaignCoverBackgroundImageURL(g.assetBaseURL, resolvedCampaignID, campaign.GetCoverSetId(), campaign.GetCoverAssetId()),
	}, nil
}

// CreateCampaign executes package-scoped creation behavior for this flow.
func (g catalogMutationGateway) CreateCampaign(ctx context.Context, input campaignapp.CreateCampaignInput) (campaignapp.CreateCampaignResult, error) {
	locale := platformi18n.LocaleForTag(input.Locale)
	locale = platformi18n.NormalizeLocale(locale)
	if locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		locale = commonv1.Locale_LOCALE_EN_US
	}
	resp, err := g.mutation.Campaign.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
		Name:        input.Name,
		Locale:      locale,
		System:      mapGameSystemToProto(input.System),
		GmMode:      mapGmModeToProto(input.GMMode),
		ThemePrompt: input.ThemePrompt,
	})
	if err != nil {
		return campaignapp.CreateCampaignResult{}, err
	}
	campaignID := strings.TrimSpace(resp.GetCampaign().GetId())
	if campaignID == "" {
		return campaignapp.CreateCampaignResult{}, apperrors.E(apperrors.KindUnknown, "created campaign id was empty")
	}
	return campaignapp.CreateCampaignResult{CampaignID: campaignID}, nil
}

// UpdateCampaign applies this package workflow transition.
func (g configurationMutationGateway) UpdateCampaign(ctx context.Context, campaignID string, input campaignapp.UpdateCampaignInput) error {
	if g.mutation.Campaign == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.campaign_service_client_is_not_configured", "campaign service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}

	req := &statev1.UpdateCampaignRequest{CampaignId: campaignID}
	if input.Name != nil {
		req.Name = wrapperspb.String(strings.TrimSpace(*input.Name))
	}
	if input.ThemePrompt != nil {
		req.ThemePrompt = wrapperspb.String(strings.TrimSpace(*input.ThemePrompt))
	}
	if input.Locale != nil {
		locale, ok := platformi18n.ParseLocale(strings.TrimSpace(*input.Locale))
		if !ok {
			return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_locale_value_is_invalid", "campaign locale value is invalid")
		}
		req.Locale = locale
	}

	_, err := g.mutation.Campaign.UpdateCampaign(ctx, req)
	if err != nil {
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_update_campaign",
			FallbackMessage: "failed to update campaign",
		})
	}
	return nil
}
