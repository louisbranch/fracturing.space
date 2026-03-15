package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/adversarytransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/conditiontransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/statetransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func (s *DaggerheartService) conditionHandler() *conditiontransport.Handler {
	return conditiontransport.NewHandler(conditiontransport.Dependencies{
		Campaign:    s.stores.Campaign,
		SessionGate: s.stores.SessionGate,
		Daggerheart: s.stores.Daggerheart,
		Event:       s.stores.Event,
		ExecuteDomainCommand: func(ctx context.Context, in conditiontransport.DomainCommandInput) error {
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
		LoadAdversaryForSession: func(ctx context.Context, campaignID, sessionID, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
			return adversarytransport.LoadAdversaryForSession(ctx, s.stores.Daggerheart, campaignID, sessionID, adversaryID)
		},
	})
}

func (s *DaggerheartService) ApplyConditions(ctx context.Context, in *pb.DaggerheartApplyConditionsRequest) (*pb.DaggerheartApplyConditionsResponse, error) {
	result, err := s.conditionHandler().ApplyConditions(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.DaggerheartApplyConditionsResponse{
		CharacterId: result.CharacterID,
		State:       statetransport.CharacterStateToProto(result.State),
		Added:       conditiontransport.ConditionsToProto(result.Added),
		Removed:     conditiontransport.ConditionsToProto(result.Removed),
	}, nil
}

func (s *DaggerheartService) ApplyAdversaryConditions(ctx context.Context, in *pb.DaggerheartApplyAdversaryConditionsRequest) (*pb.DaggerheartApplyAdversaryConditionsResponse, error) {
	result, err := s.conditionHandler().ApplyAdversaryConditions(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.DaggerheartApplyAdversaryConditionsResponse{
		AdversaryId: result.AdversaryID,
		Adversary:   adversarytransport.AdversaryToProto(result.Adversary),
		Added:       conditiontransport.ConditionsToProto(result.Added),
		Removed:     conditiontransport.ConditionsToProto(result.Removed),
	}, nil
}
