package conditiontransport

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// validateRollSeq keeps condition writes tied to the session that produced the
// referenced roll when callers attach a roll sequence.
func (h *Handler) validateRollSeq(ctx context.Context, campaignID, sessionID string, rollSeq *uint64) error {
	if rollSeq == nil {
		return nil
	}
	rollEvent, err := h.deps.Event.GetEventBySeq(ctx, campaignID, *rollSeq)
	if err != nil {
		return grpcerror.HandleDomainError(err)
	}
	if sessionID != "" && rollEvent.SessionID.String() != sessionID {
		return status.Error(codes.InvalidArgument, "roll seq does not match session")
	}
	return nil
}

// requireCharacterDependencies guards the character-condition path so handler
// methods fail fast when the root package wires an incomplete dependency set.
func (h *Handler) requireCharacterDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	case h.deps.ExecuteDomainCommand == nil:
		return status.Error(codes.Internal, "domain command executor is not configured")
	default:
		return nil
	}
}

// requireAdversaryDependencies guards the adversary path, including the
// session-scoped adversary loader that that workflow depends on.
func (h *Handler) requireAdversaryDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	case h.deps.ExecuteDomainCommand == nil:
		return status.Error(codes.Internal, "domain command executor is not configured")
	case h.deps.LoadAdversaryForSession == nil:
		return status.Error(codes.Internal, "adversary loader is not configured")
	default:
		return nil
	}
}
