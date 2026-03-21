package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/gmconsequence"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/gmmovetransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

func (s *DaggerheartService) gmMoveHandler() *gmmovetransport.Handler {
	return gmmovetransport.NewHandler(gmmovetransport.Dependencies{
		Campaign:         s.stores.Campaign,
		Session:          s.stores.Session,
		SessionGate:      s.stores.SessionGate,
		SessionSpotlight: s.stores.SessionSpotlight,
		Daggerheart:      s.stores.Daggerheart,
		Content:          s.stores.Content,
		ExecuteDomainCommand: func(ctx context.Context, in gmmovetransport.DomainCommandInput) error {
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
		ExecuteCoreCommand: func(ctx context.Context, in gmconsequence.CoreCommandInput) error {
			cmd := commandbuild.CoreSystem(commandbuild.CoreSystemInput{
				CampaignID:   in.CampaignID,
				Type:         in.CommandType,
				SessionID:    in.SessionID,
				SceneID:      in.SceneID,
				RequestID:    in.RequestID,
				InvocationID: in.InvocationID,
				EntityType:   in.EntityType,
				EntityID:     in.EntityID,
				PayloadJSON:  in.PayloadJSON,
			})
			applier, err := s.resolvedApplier()
			if err != nil {
				return err
			}
			_, err = workflowwrite.ExecuteAndApply(ctx, s.stores.Write, applier, cmd, domainwrite.RequireEventsWithDiagnostics(in.MissingEventMsg, in.ApplyErrMessage))
			return err
		},
	})
}

func (s *DaggerheartService) ApplyGmMove(ctx context.Context, in *pb.DaggerheartApplyGmMoveRequest) (*pb.DaggerheartApplyGmMoveResponse, error) {
	result, err := s.gmMoveHandler().ApplyGmMove(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.DaggerheartApplyGmMoveResponse{
		CampaignId:   result.CampaignID,
		GmFearBefore: int32(result.GMFearBefore),
		GmFearAfter:  int32(result.GMFearAfter),
	}, nil
}
