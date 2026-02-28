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

// TODO(mutation-activation): session/participant/invite mutations are scaffolded
// but intentionally return KindUnavailable. Activation criteria:
//  1. Backend gRPC service implements the corresponding RPC.
//  2. Gateway method is implemented with real client call + error mapping.
//  3. Route is registered in registerExperimentalRoutesForCampaigns (or promoted to stable).
//  4. Integration tests cover the mutation end-to-end.
func (g GRPCGateway) StartSession(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaign start session is not implemented")
}

func (g GRPCGateway) EndSession(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaign end session is not implemented")
}

func (g GRPCGateway) UpdateParticipants(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaign participant updates are not implemented")
}
