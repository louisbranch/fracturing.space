package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/sessiontransport"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultListActiveSessionsForUserPageSize = handler.PageSmall
	maxListActiveSessionsForUserPageSize     = handler.PageSmall
)

// ListActiveSessionsForUser returns active sessions across campaigns for the current user.
func (s *SessionService) ListActiveSessionsForUser(ctx context.Context, in *campaignv1.ListActiveSessionsForUserRequest) (*campaignv1.ListActiveSessionsForUserResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list active sessions for user request is required")
	}
	page, err := newSessionApplication(s).ListActiveSessionsForUser(ctx, in)
	if err != nil {
		return nil, err
	}

	response := &campaignv1.ListActiveSessionsForUserResponse{}
	if len(page.sessions) == 0 {
		return response, nil
	}

	response.HasMore = page.hasMore
	response.Sessions = make([]*campaignv1.ActiveUserSession, 0, len(page.sessions))
	for _, item := range page.sessions {
		response.Sessions = append(response.Sessions, sessiontransport.ActiveUserSessionToProto(item.campaign, item.session))
	}

	return response, nil
}
