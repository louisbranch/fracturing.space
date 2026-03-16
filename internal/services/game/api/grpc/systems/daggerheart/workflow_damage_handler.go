package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/adversarytransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/damagetransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/statetransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowruntime"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
)

func (s *DaggerheartService) damageHandler() *damagetransport.Handler {
	runtime := workflowwrite.NewRuntime(s.stores.Write, s.stores.Event, s.stores.Daggerheart)
	return damagetransport.NewHandler(damagetransport.Dependencies{
		Campaign:    s.stores.Campaign,
		SessionGate: s.stores.SessionGate,
		Daggerheart: s.stores.Daggerheart,
		Content:     s.stores.Content,
		Event:       s.stores.Event,
		SeedFunc:    s.seedFunc,
		ExecuteSystemCommand: func(ctx context.Context, in damagetransport.SystemCommandInput) error {
			return runtime.ExecuteSystemCommand(ctx, workflowruntime.SystemCommandInput{
				CampaignID:      in.CampaignID,
				CommandType:     in.CommandType,
				SessionID:       in.SessionID,
				SceneID:         in.SceneID,
				RequestID:       in.RequestID,
				InvocationID:    in.InvocationID,
				EntityType:      in.EntityType,
				EntityID:        in.EntityID,
				PayloadJSON:     in.PayloadJSON,
				MissingEventMsg: in.MissingEventMsg,
				ApplyErrMessage: in.ApplyErrMessage,
			})
		},
		LoadAdversaryForSession: func(ctx context.Context, campaignID, sessionID, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
			return adversarytransport.LoadAdversaryForSession(ctx, s.stores.Daggerheart, campaignID, sessionID, adversaryID)
		},
	})
}

func (s *DaggerheartService) ApplyDamage(ctx context.Context, in *pb.DaggerheartApplyDamageRequest) (*pb.DaggerheartApplyDamageResponse, error) {
	result, err := s.damageHandler().ApplyDamage(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.DaggerheartApplyDamageResponse{
		CharacterId: result.CharacterID,
		State:       statetransport.CharacterStateToProto(result.State),
	}, nil
}

func (s *DaggerheartService) ApplyAdversaryDamage(ctx context.Context, in *pb.DaggerheartApplyAdversaryDamageRequest) (*pb.DaggerheartApplyAdversaryDamageResponse, error) {
	result, err := s.damageHandler().ApplyAdversaryDamage(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.DaggerheartApplyAdversaryDamageResponse{
		AdversaryId: result.AdversaryID,
		Adversary:   adversarytransport.AdversaryToProto(result.Adversary),
	}, nil
}
