package adversarytransport

import (
	"context"
	"errors"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func loadAdversaryForSession(ctx context.Context, store DaggerheartStore, campaignID, sessionID, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
	if store == nil {
		return projectionstore.DaggerheartAdversary{}, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	adversary, err := store.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return projectionstore.DaggerheartAdversary{}, status.Error(codes.NotFound, "adversary not found")
		}
		return projectionstore.DaggerheartAdversary{}, grpcerror.Internal("load adversary", err)
	}
	if adversary.SessionID != "" && adversary.SessionID != sessionID {
		return projectionstore.DaggerheartAdversary{}, status.Error(codes.FailedPrecondition, "adversary is not in session")
	}
	return adversary, nil
}
