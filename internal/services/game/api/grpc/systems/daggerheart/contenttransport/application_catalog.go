package contenttransport

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
)

// GetContentCatalog returns the entire Daggerheart content catalog.
func (a contentApplication) runGetContentCatalog(ctx context.Context, in *pb.GetDaggerheartContentCatalogRequest) (*pb.GetDaggerheartContentCatalogResponse, error) {
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}
	catalog := newContentCatalog(store, in.GetLocale())
	if err := catalog.run(ctx); err != nil {
		return nil, grpcerror.Internal("content catalog pipeline", err)
	}
	return &pb.GetDaggerheartContentCatalogResponse{Catalog: catalog.proto()}, nil
}
