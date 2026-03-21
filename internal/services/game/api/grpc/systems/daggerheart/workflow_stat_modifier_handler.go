package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/statmodifiertransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

func (s *DaggerheartService) statModifierHandler() *statmodifiertransport.Handler {
	return statmodifiertransport.NewHandler(statmodifiertransport.Dependencies{
		Campaign:    s.stores.Campaign,
		SessionGate: s.stores.SessionGate,
		Daggerheart: s.stores.Daggerheart,
		ExecuteDomainCommand: func(ctx context.Context, in statmodifiertransport.DomainCommandInput) error {
			adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
			_, err := workflowwrite.ExecuteAndApply(ctx, s.stores.Write, adapter, command.Command{
				CampaignID:    ids.CampaignID(in.CampaignID),
				Type:          in.CommandType,
				ActorType:     command.ActorTypeSystem,
				SessionID:     ids.SessionID(in.SessionID),
				SceneID:       ids.SceneID(in.SceneID),
				RequestID:     in.RequestID,
				InvocationID:  in.InvocationID,
				EntityType:    in.EntityType,
				EntityID:      in.EntityID,
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   in.PayloadJSON,
			}, domainwrite.RequireEventsWithDiagnostics(in.MissingEventMsg, in.ApplyErrMessage))
			return err
		},
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
