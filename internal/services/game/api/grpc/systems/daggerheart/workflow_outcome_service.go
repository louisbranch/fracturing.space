package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/outcometransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workfloweffects"
)

func (s *DaggerheartService) ApplyRollOutcome(ctx context.Context, in *pb.ApplyRollOutcomeRequest) (*pb.ApplyRollOutcomeResponse, error) {
	return s.outcomeHandler().ApplyRollOutcome(ctx, in)
}

func (s *DaggerheartService) ApplyAttackOutcome(ctx context.Context, in *pb.DaggerheartApplyAttackOutcomeRequest) (*pb.DaggerheartApplyAttackOutcomeResponse, error) {
	return s.outcomeHandler().ApplyAttackOutcome(ctx, in)
}

func (s *DaggerheartService) ApplyAdversaryAttackOutcome(ctx context.Context, in *pb.DaggerheartApplyAdversaryAttackOutcomeRequest) (*pb.DaggerheartApplyAdversaryAttackOutcomeResponse, error) {
	return s.outcomeHandler().ApplyAdversaryAttackOutcome(ctx, in)
}

func (s *DaggerheartService) ApplyReactionOutcome(ctx context.Context, in *pb.DaggerheartApplyReactionOutcomeRequest) (*pb.DaggerheartApplyReactionOutcomeResponse, error) {
	return s.outcomeHandler().ApplyReactionOutcome(ctx, in)
}

func (s *DaggerheartService) outcomeHandler() *outcometransport.Handler {
	return outcometransport.NewHandler(outcometransport.Dependencies{
		Campaign:             s.stores.Campaign,
		Session:              s.stores.Session,
		SessionGate:          s.stores.SessionGate,
		SessionSpotlight:     s.stores.SessionSpotlight,
		Daggerheart:          s.stores.Daggerheart,
		Content:              s.stores.Content,
		Event:                s.stores.Event,
		ExecuteSystemCommand: s.executeWorkflowSystemCommand,
		ExecuteCoreCommand:   s.applyWorkflowCoreCommand,
		ApplyStressVulnerableCondition: func(ctx context.Context, in outcometransport.ApplyStressVulnerableConditionInput) error {
			return s.workflowEffectsHandler().ApplyStressVulnerableCondition(ctx, workfloweffects.ApplyStressVulnerableConditionInput{
				CampaignID:    in.CampaignID,
				SessionID:     in.SessionID,
				CharacterID:   in.CharacterID,
				Conditions:    in.Conditions,
				StressBefore:  in.StressBefore,
				StressAfter:   in.StressAfter,
				StressMax:     in.StressMax,
				RollSeq:       in.RollSeq,
				RequestID:     in.RequestID,
				CorrelationID: in.CorrelationID,
			})
		},
	})
}
