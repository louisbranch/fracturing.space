package damagetransport

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// requireDamageDependencies guards the character-damage path so it fails fast
// when the root package wires an incomplete read/write dependency set.
func (h *Handler) requireDamageDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	case h.deps.ExecuteSystemCommand == nil:
		return status.Error(codes.Internal, "system command executor is not configured")
	default:
		return nil
	}
}

// requireAdversaryDamageDependencies adds the session-scoped adversary loader
// needed by the adversary-damage path on top of the shared read/write seam.
func (h *Handler) requireAdversaryDamageDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	case h.deps.ExecuteSystemCommand == nil:
		return status.Error(codes.Internal, "system command executor is not configured")
	case h.deps.LoadAdversaryForSession == nil:
		return status.Error(codes.Internal, "adversary loader is not configured")
	default:
		return nil
	}
}
