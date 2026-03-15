package campaigntransport

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
	return grpcerror.Internal("check active session", err)
}
