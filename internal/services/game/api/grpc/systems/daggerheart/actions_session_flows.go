package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func (s *DaggerheartService) runSessionActionRoll(ctx context.Context, in *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error) {
	return newSessionFlowApplication(s).runSessionActionRoll(ctx, in)
}

func (s *DaggerheartService) runSessionDamageRoll(ctx context.Context, in *pb.SessionDamageRollRequest) (*pb.SessionDamageRollResponse, error) {
	return newSessionFlowApplication(s).runSessionDamageRoll(ctx, in)
}

func (s *DaggerheartService) runSessionAttackFlow(ctx context.Context, in *pb.SessionAttackFlowRequest) (*pb.SessionAttackFlowResponse, error) {
	return newSessionFlowApplication(s).runSessionAttackFlow(ctx, in)
}

func (s *DaggerheartService) runSessionReactionFlow(ctx context.Context, in *pb.SessionReactionFlowRequest) (*pb.SessionReactionFlowResponse, error) {
	return newSessionFlowApplication(s).runSessionReactionFlow(ctx, in)
}

func (s *DaggerheartService) runSessionAdversaryAttackRoll(ctx context.Context, in *pb.SessionAdversaryAttackRollRequest) (*pb.SessionAdversaryAttackRollResponse, error) {
	return newSessionFlowApplication(s).runSessionAdversaryAttackRoll(ctx, in)
}

func (s *DaggerheartService) runSessionAdversaryActionCheck(ctx context.Context, in *pb.SessionAdversaryActionCheckRequest) (*pb.SessionAdversaryActionCheckResponse, error) {
	return newSessionFlowApplication(s).runSessionAdversaryActionCheck(ctx, in)
}

func (s *DaggerheartService) runSessionAdversaryAttackFlow(ctx context.Context, in *pb.SessionAdversaryAttackFlowRequest) (*pb.SessionAdversaryAttackFlowResponse, error) {
	return newSessionFlowApplication(s).runSessionAdversaryAttackFlow(ctx, in)
}

func (s *DaggerheartService) runSessionGroupActionFlow(ctx context.Context, in *pb.SessionGroupActionFlowRequest) (*pb.SessionGroupActionFlowResponse, error) {
	return newSessionFlowApplication(s).runSessionGroupActionFlow(ctx, in)
}

func (s *DaggerheartService) runSessionTagTeamFlow(ctx context.Context, in *pb.SessionTagTeamFlowRequest) (*pb.SessionTagTeamFlowResponse, error) {
	return newSessionFlowApplication(s).runSessionTagTeamFlow(ctx, in)
}
