package campaigntransport

import (
	"context"

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
		return grpcerror.HandleDomainErrorContext(ctx, storage.ErrActiveSessionExists)
	}
	lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "check active session")
	if lookupErr == nil {
		return nil
	}
	return lookupErr
}
