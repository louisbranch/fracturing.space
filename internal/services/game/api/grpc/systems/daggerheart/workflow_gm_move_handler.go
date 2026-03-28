package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/gmmovetransport"
)

func (s *DaggerheartService) gmMoveHandler() *gmmovetransport.Handler {
	return gmmovetransport.NewHandler(gmmovetransport.Dependencies{
		Campaign:             s.stores.Campaign,
		Session:              s.stores.Session,
		SessionGate:          s.stores.SessionGate,
		SessionSpotlight:     s.stores.SessionSpotlight,
		Daggerheart:          s.stores.Daggerheart,
		Content:              s.stores.Content,
		ExecuteDomainCommand: s.executeWorkflowDomainCommand,
		ExecuteCoreCommand:   s.applyWorkflowCoreCommand,
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
