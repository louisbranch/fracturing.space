package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/countdowntransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

func (s *DaggerheartService) countdownHandler() *countdowntransport.Handler {
	return countdowntransport.NewHandler(countdowntransport.Dependencies{
		Campaign:    s.stores.Campaign,
		Session:     s.stores.Session,
		SessionGate: s.stores.SessionGate,
		Daggerheart: s.stores.Daggerheart,
		NewID:       id.NewID,
		ExecuteDomainCommand: func(ctx context.Context, in countdowntransport.DomainCommandInput) error {
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

func (s *DaggerheartService) CreateSceneCountdown(ctx context.Context, in *pb.DaggerheartCreateSceneCountdownRequest) (*pb.DaggerheartCreateSceneCountdownResponse, error) {
	result, err := s.countdownHandler().CreateSceneCountdown(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.DaggerheartCreateSceneCountdownResponse{
		Countdown: countdowntransport.SceneCountdownToProto(result.Countdown),
	}, nil
}

func (s *DaggerheartService) AdvanceSceneCountdown(ctx context.Context, in *pb.DaggerheartAdvanceSceneCountdownRequest) (*pb.DaggerheartAdvanceSceneCountdownResponse, error) {
	result, err := s.countdownHandler().AdvanceSceneCountdown(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.DaggerheartAdvanceSceneCountdownResponse{
		Countdown: countdowntransport.SceneCountdownToProto(result.Countdown),
		Advance:   countdowntransport.AdvanceSummaryToProto(result.Countdown, result.Summary, in.GetReason()),
	}, nil
}

func (s *DaggerheartService) ResolveSceneCountdownTrigger(ctx context.Context, in *pb.DaggerheartResolveSceneCountdownTriggerRequest) (*pb.DaggerheartResolveSceneCountdownTriggerResponse, error) {
	result, err := s.countdownHandler().ResolveSceneCountdownTrigger(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.DaggerheartResolveSceneCountdownTriggerResponse{Countdown: countdowntransport.SceneCountdownToProto(result.Countdown)}, nil
}

func (s *DaggerheartService) DeleteSceneCountdown(ctx context.Context, in *pb.DaggerheartDeleteSceneCountdownRequest) (*pb.DaggerheartDeleteSceneCountdownResponse, error) {
	result, err := s.countdownHandler().DeleteSceneCountdown(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.DaggerheartDeleteSceneCountdownResponse{
		CountdownId: result.CountdownID,
	}, nil
}

func (s *DaggerheartService) GetSceneCountdown(ctx context.Context, in *pb.DaggerheartGetSceneCountdownRequest) (*pb.DaggerheartGetSceneCountdownResponse, error) {
	return s.countdownHandler().GetSceneCountdown(ctx, in)
}

func (s *DaggerheartService) ListSceneCountdowns(ctx context.Context, in *pb.DaggerheartListSceneCountdownsRequest) (*pb.DaggerheartListSceneCountdownsResponse, error) {
	return s.countdownHandler().ListSceneCountdowns(ctx, in)
}

func (s *DaggerheartService) CreateCampaignCountdown(ctx context.Context, in *pb.DaggerheartCreateCampaignCountdownRequest) (*pb.DaggerheartCreateCampaignCountdownResponse, error) {
	result, err := s.countdownHandler().CreateCampaignCountdown(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.DaggerheartCreateCampaignCountdownResponse{
		Countdown: countdowntransport.CampaignCountdownToProto(result.Countdown),
	}, nil
}

func (s *DaggerheartService) AdvanceCampaignCountdown(ctx context.Context, in *pb.DaggerheartAdvanceCampaignCountdownRequest) (*pb.DaggerheartAdvanceCampaignCountdownResponse, error) {
	result, err := s.countdownHandler().AdvanceCampaignCountdown(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.DaggerheartAdvanceCampaignCountdownResponse{
		Countdown: countdowntransport.CampaignCountdownToProto(result.Countdown),
		Advance:   countdowntransport.AdvanceSummaryToProto(result.Countdown, result.Summary, in.GetReason()),
	}, nil
}

func (s *DaggerheartService) ResolveCampaignCountdownTrigger(ctx context.Context, in *pb.DaggerheartResolveCampaignCountdownTriggerRequest) (*pb.DaggerheartResolveCampaignCountdownTriggerResponse, error) {
	result, err := s.countdownHandler().ResolveCampaignCountdownTrigger(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.DaggerheartResolveCampaignCountdownTriggerResponse{Countdown: countdowntransport.CampaignCountdownToProto(result.Countdown)}, nil
}

func (s *DaggerheartService) DeleteCampaignCountdown(ctx context.Context, in *pb.DaggerheartDeleteCampaignCountdownRequest) (*pb.DaggerheartDeleteCampaignCountdownResponse, error) {
	result, err := s.countdownHandler().DeleteCampaignCountdown(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.DaggerheartDeleteCampaignCountdownResponse{
		CountdownId: result.CountdownID,
	}, nil
}

func (s *DaggerheartService) GetCampaignCountdown(ctx context.Context, in *pb.DaggerheartGetCampaignCountdownRequest) (*pb.DaggerheartGetCampaignCountdownResponse, error) {
	return s.countdownHandler().GetCampaignCountdown(ctx, in)
}

func (s *DaggerheartService) ListCampaignCountdowns(ctx context.Context, in *pb.DaggerheartListCampaignCountdownsRequest) (*pb.DaggerheartListCampaignCountdownsResponse, error) {
	return s.countdownHandler().ListCampaignCountdowns(ctx, in)
}
