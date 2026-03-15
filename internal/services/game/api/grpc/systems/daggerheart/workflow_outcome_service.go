package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/outcometransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workfloweffects"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowruntime"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowwrite"
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
	runtime := workflowwrite.NewRuntime(s.stores.Write, s.stores.Event, s.stores.Daggerheart)
	return outcometransport.NewHandler(outcometransport.Dependencies{
		Campaign:         s.stores.Campaign,
		Session:          s.stores.Session,
		SessionGate:      s.stores.SessionGate,
		SessionSpotlight: s.stores.SessionSpotlight,
		Daggerheart:      s.stores.Daggerheart,
		Event:            s.stores.Event,
		ExecuteSystemCommand: func(ctx context.Context, in outcometransport.SystemCommandInput) error {
			return runtime.ExecuteSystemCommand(ctx, workflowruntime.SystemCommandInput{
				CampaignID:      in.CampaignID,
				CommandType:     in.CommandType,
				SessionID:       in.SessionID,
				SceneID:         in.SceneID,
				RequestID:       in.RequestID,
				InvocationID:    in.InvocationID,
				CorrelationID:   in.CorrelationID,
				EntityType:      in.EntityType,
				EntityID:        in.EntityID,
				PayloadJSON:     in.PayloadJSON,
				MissingEventMsg: in.MissingEventMsg,
				ApplyErrMessage: in.ApplyErrMessage,
			})
		},
		ExecuteCoreCommand: func(ctx context.Context, in outcometransport.CoreCommandInput) error {
			cmd := commandbuild.CoreSystem(commandbuild.CoreSystemInput{
				CampaignID:    in.CampaignID,
				Type:          in.CommandType,
				SessionID:     in.SessionID,
				SceneID:       in.SceneID,
				RequestID:     in.RequestID,
				InvocationID:  in.InvocationID,
				CorrelationID: in.CorrelationID,
				EntityType:    in.EntityType,
				EntityID:      in.EntityID,
				PayloadJSON:   in.PayloadJSON,
			})
			_, err := workflowwrite.ExecuteAndApply(ctx, s.stores.Write, s.stores.Applier(), cmd, domainwrite.RequireEventsWithDiagnostics(in.MissingEventMsg, in.ApplyErrMessage))
			return err
		},
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
