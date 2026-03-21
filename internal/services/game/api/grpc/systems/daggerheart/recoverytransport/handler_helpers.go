package recoverytransport

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
