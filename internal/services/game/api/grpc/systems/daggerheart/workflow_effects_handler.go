package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workfloweffects"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowruntime"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowwrite"
)

func (s *DaggerheartService) workflowEffectsHandler() *workfloweffects.Handler {
	runtime := workflowwrite.NewRuntime(s.stores.Write, s.stores.Event, s.stores.Daggerheart)
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
			return runtime.ExecuteSystemCommand(ctx, workflowruntime.SystemCommandInput{
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
		CreateCountdown: func(ctx context.Context, in *pb.DaggerheartCreateCountdownRequest) error {
			_, err := s.CreateCountdown(ctx, in)
			return err
		},
		UpdateCountdown: func(ctx context.Context, in *pb.DaggerheartUpdateCountdownRequest) error {
			_, err := s.UpdateCountdown(ctx, in)
			return err
		},
	})
}
