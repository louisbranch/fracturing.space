package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func (s *DaggerheartService) ApplyRest(ctx context.Context, in *pb.DaggerheartApplyRestRequest) (*pb.DaggerheartApplyRestResponse, error) {
	return s.runApplyRest(ctx, in)
}

func (s *DaggerheartService) ApplyDowntimeMove(ctx context.Context, in *pb.DaggerheartApplyDowntimeMoveRequest) (*pb.DaggerheartApplyDowntimeMoveResponse, error) {
	return s.runApplyDowntimeMove(ctx, in)
}

func (s *DaggerheartService) ApplyTemporaryArmor(ctx context.Context, in *pb.DaggerheartApplyTemporaryArmorRequest) (*pb.DaggerheartApplyTemporaryArmorResponse, error) {
	return s.runApplyTemporaryArmor(ctx, in)
}

func (s *DaggerheartService) SwapLoadout(ctx context.Context, in *pb.DaggerheartSwapLoadoutRequest) (*pb.DaggerheartSwapLoadoutResponse, error) {
	return s.runSwapLoadout(ctx, in)
}

func (s *DaggerheartService) ApplyDeathMove(ctx context.Context, in *pb.DaggerheartApplyDeathMoveRequest) (*pb.DaggerheartApplyDeathMoveResponse, error) {
	return s.runApplyDeathMove(ctx, in)
}
