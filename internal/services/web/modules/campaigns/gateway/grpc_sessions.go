package gateway

import (
	"context"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/grpcpaging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CampaignSessions centralizes this web behavior in one helper seam.
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

// StartSession applies this package workflow transition.
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
		return mapSessionMutationError(err, "error.web.message.failed_to_start_session", "failed to start session")
	}
	return nil
}

// EndSession applies this package workflow transition.
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
		return mapSessionMutationError(err, "error.web.message.failed_to_end_session", "failed to end session")
	}
	return nil
}

// mapSessionMutationError centralizes session mutation transport status mapping.
func mapSessionMutationError(err error, fallbackKey, fallbackMessage string) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if ok {
		switch st.Code() {
		case codes.InvalidArgument, codes.OutOfRange:
			return apperrors.EK(apperrors.KindInvalidInput, fallbackKey, fallbackMessage)
		case codes.FailedPrecondition, codes.AlreadyExists, codes.Aborted:
			return apperrors.EK(apperrors.KindConflict, fallbackKey, fallbackMessage)
		case codes.Unauthenticated:
			return apperrors.E(apperrors.KindUnauthorized, "authentication required")
		case codes.PermissionDenied:
			return apperrors.E(apperrors.KindForbidden, "access denied")
		case codes.NotFound:
			return apperrors.E(apperrors.KindNotFound, "resource not found")
		case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Canceled:
			return apperrors.E(apperrors.KindUnavailable, "dependency is temporarily unavailable")
		}
	}

	return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
		FallbackKind:    apperrors.KindUnknown,
		FallbackKey:     fallbackKey,
		FallbackMessage: fallbackMessage,
	})
}
