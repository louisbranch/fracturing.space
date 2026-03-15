package daggerheart

import (
	"context"
	"fmt"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/contenttransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DaggerheartAssetService implements the Daggerheart asset-map gRPC API.
type DaggerheartAssetService struct {
	pb.UnimplementedDaggerheartAssetServiceServer
	store contentstore.DaggerheartContentReadStore
}

// NewDaggerheartAssetService creates a configured gRPC handler for asset-map APIs.
func NewDaggerheartAssetService(store contentstore.DaggerheartContentReadStore) (*DaggerheartAssetService, error) {
	if store == nil {
		return nil, fmt.Errorf("content store is required")
	}
	return &DaggerheartAssetService{store: store}, nil
}

// GetAssetMap returns resolved content-image selectors for Daggerheart entities.
func (s *DaggerheartAssetService) GetAssetMap(ctx context.Context, in *pb.GetDaggerheartAssetMapRequest) (*pb.GetDaggerheartAssetMapResponse, error) {
	return contenttransport.NewHandler(s.storeOrNil()).GetAssetMap(ctx, in)
}

func (s *DaggerheartAssetService) assetStore() (contentstore.DaggerheartContentReadStore, error) {
	store := s.storeOrNil()
	if store == nil {
		return nil, status.Error(codes.Internal, "content store is not configured")
	}
	return store, nil
}

func (s *DaggerheartAssetService) storeOrNil() contentstore.DaggerheartContentReadStore {
	if s == nil {
		return nil
	}
	return s.store
}
