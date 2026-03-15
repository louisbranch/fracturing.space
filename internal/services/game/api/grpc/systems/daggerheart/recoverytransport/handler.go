package recoverytransport

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler owns Daggerheart recovery and life-state mutation transport.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a recovery transport handler from explicit reads and
// callback seams.
func NewHandler(deps Dependencies) *Handler {
	if deps.ResolveSeed == nil {
		deps.ResolveSeed = random.ResolveSeed
	}
	return &Handler{deps: deps}
}

func (h *Handler) requireDependencies(requireSeed bool) error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.ExecuteSystemCommand == nil:
		return status.Error(codes.Internal, "system command executor is not configured")
	case h.deps.ApplyStressConditionChange == nil:
		return status.Error(codes.Internal, "stress condition callback is not configured")
	case h.deps.AppendCharacterDeletedEvent == nil:
		return status.Error(codes.Internal, "character deleted callback is not configured")
	case requireSeed && h.deps.SeedGenerator == nil:
		return status.Error(codes.Internal, "seed generator is not configured")
	default:
		return nil
	}
}
