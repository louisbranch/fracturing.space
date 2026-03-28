package ai

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// userIDHeader is injected by trusted edge/auth layers and consumed here for
	// ownership enforcement. Direct callers must not be allowed to spoof it.
	userIDHeader = "x-fracturing-space-user-id"

	defaultPageSize = 10
	maxPageSize     = 50
)

// requireUnaryRequest enforces that unary handlers receive a request message so
// each RPC can keep its own caller-facing error text while sharing the same
// transport policy.
func requireUnaryRequest[T any](in *T, message string) error {
	if in != nil {
		return nil
	}
	return status.Error(codes.InvalidArgument, message)
}

// requireCallerUserID enforces the transport contract that user-scoped RPCs run
// only when trusted middleware has attached a caller identity.
func requireCallerUserID(ctx context.Context) (string, error) {
	userID := userIDFromContext(ctx)
	if userID != "" {
		return userID, nil
	}
	return "", status.Error(codes.PermissionDenied, "missing user identity")
}

// requireUserScopedUnaryRequest composes the common request-presence and caller
// identity checks used by user-scoped unary RPCs.
func requireUserScopedUnaryRequest[T any](ctx context.Context, in *T, message string) (string, error) {
	if err := requireUnaryRequest(in, message); err != nil {
		return "", err
	}
	return requireCallerUserID(ctx)
}
