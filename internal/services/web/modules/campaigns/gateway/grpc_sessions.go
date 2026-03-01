package gateway

import (
	"context"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/grpcpaging"
)

func (g GRPCGateway) CampaignSessions(ctx context.Context, campaignID string) ([]campaignapp.CampaignSession, error) {
	if g.SessionClient == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.session_service_client_is_not_configured", "session service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []campaignapp.CampaignSession{}, nil
	}

	return grpcpaging.CollectPages[campaignapp.CampaignSession, *statev1.Session](
		ctx, 10,
		func(ctx context.Context, pageToken string) ([]*statev1.Session, string, error) {
			resp, err := g.SessionClient.ListSessions(ctx, &statev1.ListSessionsRequest{
				CampaignId: campaignID,
				PageSize:   10,
				PageToken:  pageToken,
			})
			if err != nil {
				return nil, "", err
			}
			if resp == nil {
				return nil, "", nil
			}
			return resp.GetSessions(), resp.GetNextPageToken(), nil
		},
		func(session *statev1.Session) (campaignapp.CampaignSession, bool) {
			if session == nil {
				return campaignapp.CampaignSession{}, false
			}
			return campaignapp.CampaignSession{
				ID:        strings.TrimSpace(session.GetId()),
				Name:      strings.TrimSpace(session.GetName()),
				Status:    sessionStatusLabel(session.GetStatus()),
				StartedAt: timestampString(session.GetStartedAt()),
				UpdatedAt: timestampString(session.GetUpdatedAt()),
				EndedAt:   timestampString(session.GetEndedAt()),
			}, true
		},
	)
}

func (g GRPCGateway) StartSession(ctx context.Context, campaignID string, input campaignapp.StartSessionInput) error {
	if g.SessionClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.session_service_client_is_not_configured", "session service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}

	_, err := g.SessionClient.StartSession(ctx, &statev1.StartSessionRequest{
		CampaignId: campaignID,
		Name:       strings.TrimSpace(input.Name),
	})
	if err != nil {
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_start_session",
			FallbackMessage: "failed to start session",
		})
	}
	return nil
}

func (g GRPCGateway) EndSession(ctx context.Context, campaignID string, input campaignapp.EndSessionInput) error {
	if g.SessionClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.session_service_client_is_not_configured", "session service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.session_id_is_required", "session id is required")
	}

	_, err := g.SessionClient.EndSession(ctx, &statev1.EndSessionRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err != nil {
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_end_session",
			FallbackMessage: "failed to end session",
		})
	}
	return nil
}
