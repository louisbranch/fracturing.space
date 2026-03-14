package gateway

import (
	"context"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
)

// CampaignSessionReadiness returns session-start blockers for the campaign.
func (g GRPCGateway) CampaignSessionReadiness(ctx context.Context, campaignID string, locale language.Tag) (campaignapp.CampaignSessionReadiness, error) {
	if g.Read.Campaign == nil {
		return campaignapp.CampaignSessionReadiness{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.campaign_service_client_is_not_configured", "campaign service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return campaignapp.CampaignSessionReadiness{
			Ready:    true,
			Blockers: []campaignapp.CampaignSessionReadinessBlocker{},
		}, nil
	}

	resp, err := g.Read.Campaign.GetCampaignSessionReadiness(ctx, &statev1.GetCampaignSessionReadinessRequest{
		CampaignId: campaignID,
		Locale:     readinessLocaleForTag(locale),
	})
	if err != nil {
		return campaignapp.CampaignSessionReadiness{}, apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_load_session_readiness",
			FallbackMessage: "failed to load session readiness",
		})
	}
	if resp == nil || resp.GetReadiness() == nil {
		return campaignapp.CampaignSessionReadiness{}, apperrors.EK(apperrors.KindUnknown, "error.web.message.failed_to_load_session_readiness", "failed to load session readiness")
	}

	readiness := campaignapp.CampaignSessionReadiness{
		Ready:    resp.GetReadiness().GetReady(),
		Blockers: make([]campaignapp.CampaignSessionReadinessBlocker, 0, len(resp.GetReadiness().GetBlockers())),
	}
	for _, blocker := range resp.GetReadiness().GetBlockers() {
		metadata := make(map[string]string, len(blocker.GetMetadata()))
		for key, value := range blocker.GetMetadata() {
			metadata[key] = value
		}
		readiness.Blockers = append(readiness.Blockers, campaignapp.CampaignSessionReadinessBlocker{
			Code:     strings.TrimSpace(blocker.GetCode()),
			Message:  strings.TrimSpace(blocker.GetMessage()),
			Metadata: metadata,
		})
	}
	if readiness.Ready {
		readiness.Blockers = []campaignapp.CampaignSessionReadinessBlocker{}
	}
	return readiness, nil
}

// readinessLocaleForTag normalizes a web language tag into the game locale enum.
func readinessLocaleForTag(tag language.Tag) commonv1.Locale {
	locale := platformi18n.LocaleForTag(tag)
	return platformi18n.NormalizeLocale(locale)
}
