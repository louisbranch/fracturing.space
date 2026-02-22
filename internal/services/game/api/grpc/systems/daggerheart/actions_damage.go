package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func (s *DaggerheartService) ApplyDamage(ctx context.Context, in *pb.DaggerheartApplyDamageRequest) (*pb.DaggerheartApplyDamageResponse, error) {
	return newDamageApplication(s).runApplyDamage(ctx, in)
}

func (s *DaggerheartService) ApplyAdversaryDamage(ctx context.Context, in *pb.DaggerheartApplyAdversaryDamageRequest) (*pb.DaggerheartApplyAdversaryDamageResponse, error) {
	return newDamageApplication(s).runApplyAdversaryDamage(ctx, in)
}
