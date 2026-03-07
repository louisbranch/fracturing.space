package game

import (
	"context"
	"errors"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ensureNoActiveSession(ctx context.Context, store storage.SessionStore, campaignID string) error {
	if store == nil {
		return status.Error(codes.Internal, "session store is not configured")
	}
	_, err := store.GetActiveSession(ctx, campaignID)
	if err == nil {
		return apperrors.HandleError(storage.ErrActiveSessionExists, apperrors.DefaultLocale)
	}
	if errors.Is(err, storage.ErrNotFound) {
		return nil
	}
	return status.Errorf(codes.Internal, "check active session: %v", err)
}

// handleDomainError converts domain errors to gRPC status using the structured error system.
// For domain errors (*apperrors.Error), it returns a properly formatted gRPC status with
// error details including ErrorInfo and LocalizedMessage.
// For non-domain errors, it falls back to an internal error.
//
// TODO: Extract locale from gRPC metadata (e.g., "accept-language" header) to enable
// proper i18n support. Currently hardcoded to DefaultLocale.
//
// This keeps API responses deterministic today while leaving room for locale-aware
// user experience in follow-up work.
//
// The default locale is intentional so behavior is stable while auth/web and
// gRPC metadata propagation is still being aligned for user-facing localization.
func handleDomainError(err error) error {
	return grpcerror.HandleDomainError(err)
}
