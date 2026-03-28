package sessiontransport

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"

	"context"
	"sort"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type activeUserSessionRecord struct {
	campaign storage.CampaignRecord
	session  storage.SessionRecord
}

type activeUserSessionPage struct {
	sessions []activeUserSessionRecord
	hasMore  bool
}

type activeSessionContext struct {
	session   *storage.SessionRecord
	gate      *storage.SessionGate
	spotlight *storage.SessionSpotlight
}

func (a sessionApplication) ListSessions(ctx context.Context, in *campaignv1.ListSessionsRequest) (storage.SessionPage, error) {
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return storage.SessionPage{}, err
	}

	campaignRecord, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.SessionPage{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpRead); err != nil {
		return storage.SessionPage{}, err
	}
	if err := authz.RequireReadPolicy(ctx, a.auth, campaignRecord); err != nil {
		return storage.SessionPage{}, err
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListSessionsPageSize,
		Max:     maxListSessionsPageSize,
	})
	page, err := a.stores.Session.ListSessions(ctx, campaignID, pageSize, in.GetPageToken())
	if err != nil {
		return storage.SessionPage{}, grpcerror.Internal("list sessions", err)
	}
	return page, nil
}

func (a sessionApplication) GetSession(ctx context.Context, in *campaignv1.GetSessionRequest) (storage.SessionRecord, error) {
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return storage.SessionRecord{}, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return storage.SessionRecord{}, err
	}

	campaignRecord, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.SessionRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpRead); err != nil {
		return storage.SessionRecord{}, err
	}
	if err := authz.RequireReadPolicy(ctx, a.auth, campaignRecord); err != nil {
		return storage.SessionRecord{}, err
	}

	sess, err := a.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return storage.SessionRecord{}, err
	}
	return sess, nil
}

func (a sessionApplication) GetSessionSpotlight(ctx context.Context, in *campaignv1.GetSessionSpotlightRequest) (storage.SessionSpotlight, error) {
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return storage.SessionSpotlight{}, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return storage.SessionSpotlight{}, err
	}

	campaignRecord, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.SessionSpotlight{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpRead); err != nil {
		return storage.SessionSpotlight{}, err
	}
	if err := authz.RequireReadPolicy(ctx, a.auth, campaignRecord); err != nil {
		return storage.SessionSpotlight{}, err
	}
	if _, err := a.stores.Session.GetSession(ctx, campaignID, sessionID); err != nil {
		return storage.SessionSpotlight{}, err
	}

	spotlight, err := a.stores.SessionSpotlight.GetSessionSpotlight(ctx, campaignID, sessionID)
	if err != nil {
		return storage.SessionSpotlight{}, err
	}
	return spotlight, nil
}

func (a sessionApplication) GetActiveSessionContext(ctx context.Context, campaignID string) (activeSessionContext, error) {
	if a.stores.Session == nil {
		return activeSessionContext{}, status.Error(codes.Internal, "session store is not configured")
	}

	activeSession, err := a.stores.Session.GetActiveSession(ctx, campaignID)
	if err != nil {
		if grpcerror.OptionalLookupErrorContext(ctx, err, "get active session") == nil {
			return activeSessionContext{}, nil
		}
		return activeSessionContext{}, grpcerror.OptionalLookupErrorContext(ctx, err, "get active session")
	}

	contextState := activeSessionContext{session: &activeSession}
	if a.stores.SessionGate != nil {
		activeGate, err := a.stores.SessionGate.GetOpenSessionGate(ctx, campaignID, activeSession.ID)
		if err != nil {
			if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "get open session gate"); lookupErr != nil {
				return activeSessionContext{}, lookupErr
			}
		} else {
			contextState.gate = &activeGate
		}
	}

	if a.stores.SessionSpotlight != nil {
		spotlight, err := a.stores.SessionSpotlight.GetSessionSpotlight(ctx, campaignID, activeSession.ID)
		if err != nil {
			if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "get session spotlight"); lookupErr != nil {
				return activeSessionContext{}, lookupErr
			}
		} else {
			contextState.spotlight = &spotlight
		}
	}

	return contextState, nil
}

func (a sessionApplication) ListActiveSessionsForUser(ctx context.Context, in *campaignv1.ListActiveSessionsForUserRequest) (activeUserSessionPage, error) {
	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	if userID == "" {
		return activeUserSessionPage{}, status.Error(codes.InvalidArgument, "user id is required")
	}
	if a.stores.Participant == nil {
		return activeUserSessionPage{}, status.Error(codes.Internal, "participant store is not configured")
	}
	if a.stores.Campaign == nil {
		return activeUserSessionPage{}, status.Error(codes.Internal, "campaign store is not configured")
	}
	if a.stores.Session == nil {
		return activeUserSessionPage{}, status.Error(codes.Internal, "session store is not configured")
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListActiveSessionsForUserPageSize,
		Max:     maxListActiveSessionsForUserPageSize,
	})
	campaignIDs, err := a.stores.Participant.ListCampaignIDsByUser(ctx, userID)
	if err != nil {
		return activeUserSessionPage{}, grpcerror.Internal("list campaign IDs by user", err)
	}

	activeSessions := make([]activeUserSessionRecord, 0, len(campaignIDs))
	for _, campaignID := range campaignIDs {
		campaignID = strings.TrimSpace(campaignID)
		if campaignID == "" {
			continue
		}

		campaignRecord, err := a.stores.Campaign.Get(ctx, campaignID)
		if err != nil {
			if grpcerror.OptionalLookupErrorContext(ctx, err, "get campaign") == nil {
				continue
			}
			return activeUserSessionPage{}, grpcerror.OptionalLookupErrorContext(ctx, err, "get campaign")
		}

		sess, err := a.stores.Session.GetActiveSession(ctx, campaignID)
		if err != nil {
			if grpcerror.OptionalLookupErrorContext(ctx, err, "get active session") == nil {
				continue
			}
			return activeUserSessionPage{}, grpcerror.OptionalLookupErrorContext(ctx, err, "get active session")
		}

		activeSessions = append(activeSessions, activeUserSessionRecord{
			campaign: campaignRecord,
			session:  sess,
		})
	}

	sort.SliceStable(activeSessions, func(i, j int) bool {
		left := activeSessions[i]
		right := activeSessions[j]
		if !left.session.StartedAt.Equal(right.session.StartedAt) {
			return left.session.StartedAt.After(right.session.StartedAt)
		}
		return left.campaign.ID < right.campaign.ID
	})

	page := activeUserSessionPage{sessions: activeSessions}
	page.hasMore = len(page.sessions) > pageSize
	if len(page.sessions) > pageSize {
		page.sessions = page.sessions[:pageSize]
	}
	return page, nil
}
