package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/statmodifiertransport"
)

func (s *DaggerheartService) statModifierHandler() *statmodifiertransport.Handler {
	return statmodifiertransport.NewHandler(statmodifiertransport.Dependencies{
		Campaign:             s.stores.Campaign,
		SessionGate:          s.stores.SessionGate,
		Daggerheart:          s.stores.Daggerheart,
		ExecuteDomainCommand: s.executeWorkflowDomainCommand,
	})
}

func (s *DaggerheartService) ApplyStatModifiers(ctx context.Context, in *pb.DaggerheartApplyStatModifiersRequest) (*pb.DaggerheartApplyStatModifiersResponse, error) {
	result, err := s.statModifierHandler().ApplyStatModifiers(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.DaggerheartApplyStatModifiersResponse{
		CharacterId:     result.CharacterID,
		ActiveModifiers: statmodifiertransport.StatModifierViewsToProto(result.ActiveModifiers),
		Added:           statmodifiertransport.StatModifierViewsToProto(result.Added),
		Removed:         statmodifiertransport.StatModifierViewsToProto(result.Removed),
	}, nil
}
