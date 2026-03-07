package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func (s *DaggerheartService) ApplyLevelUp(ctx context.Context, in *pb.DaggerheartApplyLevelUpRequest) (*pb.DaggerheartApplyLevelUpResponse, error) {
	return newProgressionApplication(s).runApplyLevelUp(ctx, in)
}
