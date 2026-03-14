package daggerheart

import (
	"context"
	"fmt"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DaggerheartAssetService implements the Daggerheart asset-map gRPC API.
type DaggerheartAssetService struct {
	pb.UnimplementedDaggerheartAssetServiceServer
	stores Stores
}

// NewDaggerheartAssetService creates a configured gRPC handler for asset-map APIs.
func NewDaggerheartAssetService(stores Stores) (*DaggerheartAssetService, error) {
	if err := stores.ValidateContent(); err != nil {
		return nil, fmt.Errorf("validate stores: %w", err)
	}
	return &DaggerheartAssetService{stores: stores}, nil
}

// GetAssetMap returns resolved content-image selectors for Daggerheart entities.
func (s *DaggerheartAssetService) GetAssetMap(ctx context.Context, in *pb.GetDaggerheartAssetMapRequest) (*pb.GetDaggerheartAssetMapResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "asset map request is required")
	}
	store, err := s.assetStore()
	if err != nil {
		return nil, err
	}

	assetMap, err := buildDaggerheartAssetMap(ctx, store, in.GetLocale())
	if err != nil {
		return nil, grpcerror.Internal("asset map pipeline", err)
	}
	return &pb.GetDaggerheartAssetMapResponse{AssetMap: assetMap}, nil
}

func (s *DaggerheartAssetService) assetStore() (storage.DaggerheartContentReadStore, error) {
	if s == nil || s.stores.DaggerheartContent == nil {
		return nil, status.Error(codes.Internal, "content store is not configured")
	}
	return s.stores.DaggerheartContent, nil
}
