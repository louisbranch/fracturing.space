package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func (s *DaggerheartService) BeginCompanionExperience(ctx context.Context, in *pb.DaggerheartBeginCompanionExperienceRequest) (*pb.DaggerheartBeginCompanionExperienceResponse, error) {
	return s.characterMutationHandler().BeginCompanionExperience(ctx, in)
}

func (s *DaggerheartService) ReturnCompanion(ctx context.Context, in *pb.DaggerheartReturnCompanionRequest) (*pb.DaggerheartReturnCompanionResponse, error) {
	return s.characterMutationHandler().ReturnCompanion(ctx, in)
}
