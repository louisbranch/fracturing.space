package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workfloweffects"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowruntime"
)

func (s *DaggerheartService) workflowEffectsHandler() *workfloweffects.Handler {
	runtime := s.workflowRuntime()
	return workfloweffects.NewHandler(workfloweffects.Dependencies{
		Daggerheart: s.stores.Daggerheart,
		ConditionChangeAlreadyApplied: func(ctx context.Context, in workfloweffects.ConditionChangeReplayCheckInput) (bool, error) {
			return runtime.SessionRequestEventExists(ctx, workflowruntime.ReplayCheckInput{
				CampaignID: in.CampaignID,
				SessionID:  in.SessionID,
				RollSeq:    in.RollSeq,
				RequestID:  in.RequestID,
				EventType:  eventTypeDaggerheartConditionChanged,
				EntityID:   in.CharacterID,
			})
		},
		ExecuteConditionChange: func(ctx context.Context, in workfloweffects.ConditionChangeCommandInput) error {
			return s.executeWorkflowSystemCommand(ctx, workflowruntime.SystemCommandInput{
				CampaignID:      in.CampaignID,
				CommandType:     commandTypeDaggerheartConditionChange,
				SessionID:       in.SessionID,
				RequestID:       in.RequestID,
				InvocationID:    in.InvocationID,
				CorrelationID:   in.CorrelationID,
				EntityType:      "character",
				EntityID:        in.CharacterID,
				PayloadJSON:     in.PayloadJSON,
				MissingEventMsg: "condition change did not emit an event",
				ApplyErrMessage: "apply condition event",
			})
		},
		CreateSceneCountdown: func(ctx context.Context, in *pb.DaggerheartCreateSceneCountdownRequest) error {
			_, err := s.CreateSceneCountdown(ctx, in)
			return err
		},
		AdvanceSceneCountdown: func(ctx context.Context, in *pb.DaggerheartAdvanceSceneCountdownRequest) error {
			_, err := s.AdvanceSceneCountdown(ctx, in)
			return err
		},
	})
}
