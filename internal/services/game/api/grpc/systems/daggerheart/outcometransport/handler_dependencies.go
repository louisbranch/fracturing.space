package outcometransport

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// requireSessionOutcomeDependencies checks the read-side dependencies shared by
// the session-level outcome handlers.
func (h *Handler) requireSessionOutcomeDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.Session == nil:
		return status.Error(codes.Internal, "session store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	default:
		return nil
	}
}

// requireRollOutcomeDependencies checks the broader read/write dependencies
// needed by ApplyRollOutcome.
func (h *Handler) requireRollOutcomeDependencies() error {
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
	case h.deps.ExecuteSystemCommand == nil || h.deps.ExecuteCoreCommand == nil || h.deps.ApplyStressVulnerableCondition == nil:
		return status.Error(codes.Internal, "domain engine is not configured")
	default:
		return nil
	}
}

// clamp keeps integer state transitions inside their legal domain bounds.
func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
