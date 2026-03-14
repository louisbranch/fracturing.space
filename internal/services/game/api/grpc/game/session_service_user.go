package game

import (
	"context"
	"errors"
	"sort"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultListActiveSessionsForUserPageSize = pageSmall
	maxListActiveSessionsForUserPageSize     = pageSmall
)

type activeUserSessionRecord struct {
	campaign storage.CampaignRecord
	session  storage.SessionRecord
}

// ListActiveSessionsForUser returns active sessions across campaigns for the current user.
func (s *SessionService) ListActiveSessionsForUser(ctx context.Context, in *campaignv1.ListActiveSessionsForUserRequest) (*campaignv1.ListActiveSessionsForUserResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list active sessions for user request is required")
	}
	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}
	if s.stores.Participant == nil {
		return nil, status.Error(codes.Internal, "participant store is not configured")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListActiveSessionsForUserPageSize,
		Max:     maxListActiveSessionsForUserPageSize,
	})

	campaignIDs, err := s.stores.Participant.ListCampaignIDsByUser(ctx, userID)
	if err != nil {
		return nil, grpcerror.Internal("list campaign IDs by user", err)
	}

	activeSessions := make([]activeUserSessionRecord, 0, len(campaignIDs))
	for _, campaignID := range campaignIDs {
		campaignID = strings.TrimSpace(campaignID)
		if campaignID == "" {
			continue
		}

		campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				continue
			}
			return nil, grpcerror.Internal("get campaign", err)
		}

		sess, err := s.stores.Session.GetActiveSession(ctx, campaignID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				continue
			}
			return nil, grpcerror.Internal("get active session", err)
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

	response := &campaignv1.ListActiveSessionsForUserResponse{}
	if len(activeSessions) == 0 {
		return response, nil
	}

	response.HasMore = len(activeSessions) > pageSize
	if len(activeSessions) > pageSize {
		activeSessions = activeSessions[:pageSize]
	}
	response.Sessions = make([]*campaignv1.ActiveUserSession, 0, len(activeSessions))
	for _, item := range activeSessions {
		response.Sessions = append(response.Sessions, activeUserSessionToProto(item.campaign, item.session))
	}

	return response, nil
}
