package adversarytransport

import "google.golang.org/grpc/codes"

// requireBaseDependencies guards the shared read-side contracts every handler
// path depends on before it performs campaign or adversary lookups.
func (h *Handler) requireBaseDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return internal("campaign store is not configured")
	case h.deps.Gate == nil:
		return internal("session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return internal("daggerheart store is not configured")
	default:
		return nil
	}
}

func invalidArgument(message string) error {
	return statusError(codes.InvalidArgument, message)
}

func internal(message string) error {
	return statusError(codes.Internal, message)
}
