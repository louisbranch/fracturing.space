package contenttransport

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) GetContentCatalog(ctx context.Context, in *pb.GetDaggerheartContentCatalogRequest) (*pb.GetDaggerheartContentCatalogResponse, error) {
	return newContentApplication(h).runGetContentCatalog(ctx, in)
}

// GetAssetMap returns resolved content-image selectors for Daggerheart entities.
func (h *Handler) GetAssetMap(ctx context.Context, in *pb.GetDaggerheartAssetMapRequest) (*pb.GetDaggerheartAssetMapResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "asset map request is required")
	}
	store, err := h.contentStore()
	if err != nil {
		return nil, err
	}

	assetMap, err := buildDaggerheartAssetMap(ctx, store, in.GetLocale())
	if err != nil {
		return nil, grpcerror.Internal("asset map pipeline", err)
	}
	return &pb.GetDaggerheartAssetMapResponse{AssetMap: assetMap}, nil
}
