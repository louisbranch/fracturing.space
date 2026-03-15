package forktransport

import (
	"context"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/fork"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a forkApplication) resolveForkPoint(ctx context.Context, campaignID string, forkPoint fork.ForkPoint) (uint64, error) {
	if forkPoint.IsSessionBoundary() {
		if a.stores.Session == nil {
			return 0, status.Error(codes.Internal, "session store is not configured")
		}
		sessionID := strings.TrimSpace(forkPoint.SessionID)
		if sessionID == "" {
			return 0, status.Error(codes.InvalidArgument, "session id is required for session-based fork points")
		}
		sess, err := a.stores.Session.GetSession(ctx, campaignID, sessionID)
		if err != nil {
			return 0, grpcerror.EnsureStatus(err)
		}
		if sess.Status != session.StatusEnded {
			return 0, status.Error(codes.FailedPrecondition, "session has not ended")
		}

		lastSeq := uint64(0)
		afterSeq := uint64(0)
		for {
			events, err := a.stores.Event.ListEventsBySession(ctx, campaignID, sessionID, afterSeq, forkEventPageSize)
			if err != nil {
				return 0, grpcerror.Internal("list session events", err)
			}
			if len(events) == 0 {
				if lastSeq == 0 {
					return 0, status.Error(codes.FailedPrecondition, "session has no events to fork at")
				}
				return lastSeq, nil
			}
			for _, evt := range events {
				lastSeq = evt.Seq
				afterSeq = evt.Seq
			}
			if len(events) < forkEventPageSize {
				return lastSeq, nil
			}
		}
	}

	// If event seq is 0, use the latest event.
	if forkPoint.EventSeq == 0 {
		latestSeq, err := a.stores.Event.GetLatestEventSeq(ctx, campaignID)
		if err != nil {
			return 0, grpcerror.Internal("get latest event seq", err)
		}
		// If no events exist, fork at seq 0 (start of campaign).
		return latestSeq, nil
	}

	// Validate that the requested event seq exists.
	latestSeq, err := a.stores.Event.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return 0, grpcerror.Internal("get latest event seq", err)
	}

	if forkPoint.EventSeq > latestSeq {
		return 0, status.Error(codes.FailedPrecondition, "fork point is beyond current campaign state")
	}

	return forkPoint.EventSeq, nil
}
