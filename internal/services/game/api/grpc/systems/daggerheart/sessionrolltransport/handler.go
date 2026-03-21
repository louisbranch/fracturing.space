package sessionrolltransport

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler owns the low-level Daggerheart session roll endpoints.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a low-level Daggerheart session roll handler from explicit
// read-store and write-callback dependencies.
func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

func (h *Handler) requireActionRollDependencies() error {
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
	case h.deps.ExecuteActionRollResolve == nil:
		return status.Error(codes.Internal, "action roll executor is not configured")
	case h.deps.ExecuteHopeSpend == nil:
		return status.Error(codes.Internal, "hope spend executor is not configured")
	case h.deps.AdvanceBreathCountdown == nil:
		return status.Error(codes.Internal, "breath countdown handler is not configured")
	default:
		return nil
	}
}

func (h *Handler) requireDamageRollDependencies() error {
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
	case h.deps.ExecuteDamageRollResolve == nil:
		return status.Error(codes.Internal, "damage roll executor is not configured")
	default:
		return nil
	}
}

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
