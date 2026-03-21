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

func (s *DaggerheartService) CreateCountdown(ctx context.Context, in *pb.DaggerheartCreateCountdownRequest) (*pb.DaggerheartCreateCountdownResponse, error) {
	result, err := s.countdownHandler().CreateCountdown(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.DaggerheartCreateCountdownResponse{
		Countdown: countdowntransport.CountdownToProto(result.Countdown),
	}, nil
}

func (s *DaggerheartService) UpdateCountdown(ctx context.Context, in *pb.DaggerheartUpdateCountdownRequest) (*pb.DaggerheartUpdateCountdownResponse, error) {
	result, err := s.countdownHandler().UpdateCountdown(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.DaggerheartUpdateCountdownResponse{
		Countdown: countdowntransport.CountdownToProto(result.Countdown),
		Before:    int32(result.Before),
		After:     int32(result.After),
		Delta:     int32(result.Delta),
	}, nil
}

func (s *DaggerheartService) DeleteCountdown(ctx context.Context, in *pb.DaggerheartDeleteCountdownRequest) (*pb.DaggerheartDeleteCountdownResponse, error) {
	result, err := s.countdownHandler().DeleteCountdown(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.DaggerheartDeleteCountdownResponse{
		CountdownId: result.CountdownID,
	}, nil
}
