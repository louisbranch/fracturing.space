package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/sessionflowtransport"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *DaggerheartService) requireSessionFlowStores() error {
	switch {
	case s.stores.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case s.stores.Session == nil:
		return status.Error(codes.Internal, "session store is not configured")
	case s.stores.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case s.stores.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	case s.seedFunc == nil:
		return status.Error(codes.Internal, "seed generator is not configured")
	case s.stores.Write.Executor == nil:
		return status.Error(codes.Internal, "domain engine is not configured")
	default:
		return nil
	}
}

func (s *DaggerheartService) requireSessionAdversaryFlowStores() error {
	switch {
	case s.stores.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case s.stores.Session == nil:
		return status.Error(codes.Internal, "session store is not configured")
	case s.stores.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case s.stores.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	case s.seedFunc == nil:
		return status.Error(codes.Internal, "seed generator is not configured")
	default:
		return nil
	}
}

func (s *DaggerheartService) sessionFlowHandler() *sessionflowtransport.Handler {
	rolls := s.sessionRollHandler()
	return sessionflowtransport.NewHandler(sessionflowtransport.Dependencies{
		SessionActionRoll:           rolls.SessionActionRoll,
		SessionDamageRoll:           rolls.SessionDamageRoll,
		SessionAdversaryAttackRoll:  rolls.SessionAdversaryAttackRoll,
		ApplyRollOutcome:            s.outcomeHandler().ApplyRollOutcome,
		ApplyAttackOutcome:          s.outcomeHandler().ApplyAttackOutcome,
		ApplyReactionOutcome:        s.outcomeHandler().ApplyReactionOutcome,
		ApplyAdversaryAttackOutcome: s.outcomeHandler().ApplyAdversaryAttackOutcome,
		ApplyDamage:                 s.ApplyDamage,
	})
}

func (s *DaggerheartService) SessionActionRoll(ctx context.Context, in *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error) {
	return s.sessionRollHandler().SessionActionRoll(ctx, in)
}

func (s *DaggerheartService) SessionDamageRoll(ctx context.Context, in *pb.SessionDamageRollRequest) (*pb.SessionDamageRollResponse, error) {
	return s.sessionRollHandler().SessionDamageRoll(ctx, in)
}

func (s *DaggerheartService) SessionAttackFlow(ctx context.Context, in *pb.SessionAttackFlowRequest) (*pb.SessionAttackFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session attack flow request is required")
	}
	if err := s.requireSessionFlowStores(); err != nil {
		return nil, err
	}
	return s.sessionFlowHandler().SessionAttackFlow(ctx, in)
}

func (s *DaggerheartService) SessionReactionFlow(ctx context.Context, in *pb.SessionReactionFlowRequest) (*pb.SessionReactionFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session reaction flow request is required")
	}
	if err := s.requireSessionFlowStores(); err != nil {
		return nil, err
	}
	return s.sessionFlowHandler().SessionReactionFlow(ctx, in)
}

func (s *DaggerheartService) SessionAdversaryAttackRoll(ctx context.Context, in *pb.SessionAdversaryAttackRollRequest) (*pb.SessionAdversaryAttackRollResponse, error) {
	return s.sessionRollHandler().SessionAdversaryAttackRoll(ctx, in)
}

func (s *DaggerheartService) SessionAdversaryActionCheck(ctx context.Context, in *pb.SessionAdversaryActionCheckRequest) (*pb.SessionAdversaryActionCheckResponse, error) {
	return s.sessionRollHandler().SessionAdversaryActionCheck(ctx, in)
}

func (s *DaggerheartService) SessionAdversaryAttackFlow(ctx context.Context, in *pb.SessionAdversaryAttackFlowRequest) (*pb.SessionAdversaryAttackFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session adversary attack flow request is required")
	}
	if err := s.requireSessionAdversaryFlowStores(); err != nil {
		return nil, err
	}
	return s.sessionFlowHandler().SessionAdversaryAttackFlow(ctx, in)
}

func (s *DaggerheartService) SessionGroupActionFlow(ctx context.Context, in *pb.SessionGroupActionFlowRequest) (*pb.SessionGroupActionFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session group action flow request is required")
	}
	if err := s.requireSessionFlowStores(); err != nil {
		return nil, err
	}
	return s.sessionFlowHandler().SessionGroupActionFlow(ctx, in)
}

func (s *DaggerheartService) SessionTagTeamFlow(ctx context.Context, in *pb.SessionTagTeamFlowRequest) (*pb.SessionTagTeamFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session tag team flow request is required")
	}
	if err := s.requireSessionFlowStores(); err != nil {
		return nil, err
	}
	return s.sessionFlowHandler().SessionTagTeamFlow(ctx, in)
}
