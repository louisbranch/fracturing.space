package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func (s *DaggerheartContentService) GetContentCatalog(ctx context.Context, in *pb.GetDaggerheartContentCatalogRequest) (*pb.GetDaggerheartContentCatalogResponse, error) {
	return s.handler().GetContentCatalog(ctx, in)
}
