package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func (s *DaggerheartService) runApplyRollOutcome(ctx context.Context, in *pb.ApplyRollOutcomeRequest) (*pb.ApplyRollOutcomeResponse, error) {
	return newOutcomeApplication(s).runApplyRollOutcome(ctx, in)
}

func (s *DaggerheartService) runApplyAttackOutcome(ctx context.Context, in *pb.DaggerheartApplyAttackOutcomeRequest) (*pb.DaggerheartApplyAttackOutcomeResponse, error) {
	return newOutcomeApplication(s).runApplyAttackOutcome(ctx, in)
}

func (s *DaggerheartService) runApplyAdversaryAttackOutcome(ctx context.Context, in *pb.DaggerheartApplyAdversaryAttackOutcomeRequest) (*pb.DaggerheartApplyAdversaryAttackOutcomeResponse, error) {
	return newOutcomeApplication(s).runApplyAdversaryAttackOutcome(ctx, in)
}

func (s *DaggerheartService) runApplyReactionOutcome(ctx context.Context, in *pb.DaggerheartApplyReactionOutcomeRequest) (*pb.DaggerheartApplyReactionOutcomeResponse, error) {
	return newOutcomeApplication(s).runApplyReactionOutcome(ctx, in)
}
