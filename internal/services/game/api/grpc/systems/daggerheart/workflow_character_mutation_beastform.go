package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func (s *DaggerheartService) TransformBeastform(ctx context.Context, in *pb.DaggerheartTransformBeastformRequest) (*pb.DaggerheartTransformBeastformResponse, error) {
	return s.characterMutationHandler().TransformBeastform(ctx, in)
}

func (s *DaggerheartService) DropBeastform(ctx context.Context, in *pb.DaggerheartDropBeastformRequest) (*pb.DaggerheartDropBeastformResponse, error) {
	return s.characterMutationHandler().DropBeastform(ctx, in)
}
