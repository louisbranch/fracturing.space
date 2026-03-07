package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func (s *DaggerheartService) ApplyConditions(ctx context.Context, in *pb.DaggerheartApplyConditionsRequest) (*pb.DaggerheartApplyConditionsResponse, error) {
	return s.runApplyConditions(ctx, in)
}

func (s *DaggerheartService) ApplyAdversaryConditions(ctx context.Context, in *pb.DaggerheartApplyAdversaryConditionsRequest) (*pb.DaggerheartApplyAdversaryConditionsResponse, error) {
	return s.runApplyAdversaryConditions(ctx, in)
}

func (s *DaggerheartService) ApplyGmMove(ctx context.Context, in *pb.DaggerheartApplyGmMoveRequest) (*pb.DaggerheartApplyGmMoveResponse, error) {
	return s.runApplyGmMove(ctx, in)
}
