package sessionrolltransport

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) requireAdversaryRollDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.Session == nil:
		return status.Error(codes.Internal, "session store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	case h.deps.SeedFunc == nil:
		return status.Error(codes.Internal, "seed generator is not configured")
	case h.deps.ExecuteAdversaryRollResolve == nil:
		return status.Error(codes.Internal, "adversary roll executor is not configured")
	case h.deps.LoadAdversaryForSession == nil:
		return status.Error(codes.Internal, "adversary loader is not configured")
	default:
		return nil
	}
}

func (h *Handler) requireAdversaryActionCheckDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.Session == nil:
		return status.Error(codes.Internal, "session store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	case h.deps.SeedFunc == nil:
		return status.Error(codes.Internal, "seed generator is not configured")
	case h.deps.LoadAdversaryForSession == nil:
		return status.Error(codes.Internal, "adversary loader is not configured")
	default:
		return nil
	}
}

func (h *Handler) nextAdversaryRollSeq(ctx context.Context, campaignID string) (uint64, error) {
	latestSeq, err := h.deps.Event.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return 0, err
	}
	return latestSeq + 1, nil
}
