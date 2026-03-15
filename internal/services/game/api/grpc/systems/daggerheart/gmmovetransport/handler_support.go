package gmmovetransport

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// requireDependencies keeps the GM-move entrypoint on one explicit read/write
// dependency contract before request-specific validation begins.
func (h *Handler) requireDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.Session == nil:
		return status.Error(codes.Internal, "session store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.ExecuteDomainCommand == nil:
		return status.Error(codes.Internal, "domain command executor is not configured")
	default:
		return nil
	}
}
