package daggerheart

import (
	"fmt"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/contenttransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DaggerheartContentService implements the Daggerheart content/catalog gRPC API.
type DaggerheartContentService struct {
	pb.UnimplementedDaggerheartContentServiceServer
	store contentstore.DaggerheartContentReadStore
}

// NewDaggerheartContentService creates a configured gRPC handler for content APIs.
func NewDaggerheartContentService(store contentstore.DaggerheartContentReadStore) (*DaggerheartContentService, error) {
	if store == nil {
		return nil, fmt.Errorf("content store is required")
	}
	return &DaggerheartContentService{store: store}, nil
}

func (s *DaggerheartContentService) handler() *contenttransport.Handler {
	return contenttransport.NewHandler(s.storeOrNil())
}

func (s *DaggerheartContentService) storeOrNil() contentstore.DaggerheartContentReadStore {
	if s == nil {
		return nil
	}
	return s.store
}

func (s *DaggerheartContentService) contentStore() (contentstore.DaggerheartContentReadStore, error) {
	store := s.storeOrNil()
	if store == nil {
		return nil, status.Error(codes.Internal, "content store is not configured")
	}
	return store, nil
}
