package daggerheart

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/conditiontransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/countdowntransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/recoverytransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/statetransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workfloweffects"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *DaggerheartService) recoveryHandler() *recoverytransport.Handler {
	return recoverytransport.NewHandler(recoverytransport.Dependencies{
		Campaign:      s.stores.Campaign,
		SessionGate:   s.stores.SessionGate,
		Daggerheart:   s.stores.Daggerheart,
		SeedGenerator: s.seedFunc,
		ExecuteSystemCommand: func(ctx context.Context, in recoverytransport.SystemCommandInput) error {
			adapter := bridge.NewAdapter(s.stores.Daggerheart)
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
				SystemID:      bridge.SystemID,
				SystemVersion: bridge.SystemVersion,
				PayloadJSON:   in.PayloadJSON,
			}, domainwrite.RequireEventsWithDiagnostics(in.MissingEventMsg, in.ApplyErrMessage))
			return err
		},
		ApplyStressConditionChange: func(ctx context.Context, in recoverytransport.StressConditionInput) error {
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
		AppendCharacterDeletedEvent: func(ctx context.Context, in recoverytransport.CharacterDeleteInput) error {
			if s.stores.Campaign == nil {
				return status.Error(codes.Internal, "campaign store is not configured")
			}
			payloadJSON, err := json.Marshal(character.DeletePayload{
				CharacterID: ids.CharacterID(in.CharacterID),
				Reason:      strings.TrimSpace(in.Reason),
			})
			if err != nil {
				return grpcerror.Internal("encode payload", err)
			}
			_, err = workflowwrite.ExecuteAndApply(ctx, s.stores.Write, s.stores.Applier(), command.Command{
				CampaignID:   ids.CampaignID(in.CampaignID),
				Type:         commandTypeCharacterDelete,
				ActorType:    command.ActorTypeSystem,
				SessionID:    ids.SessionID(grpcmeta.SessionIDFromContext(ctx)),
				RequestID:    grpcmeta.RequestIDFromContext(ctx),
				InvocationID: grpcmeta.InvocationIDFromContext(ctx),
				EntityType:   "character",
				EntityID:     in.CharacterID,
				PayloadJSON:  payloadJSON,
			}, domainwrite.RequireEventsWithDiagnostics("character delete did not emit an event", "apply event"))
			return err
		},
	})
}

func (s *DaggerheartService) ApplyRest(ctx context.Context, in *pb.DaggerheartApplyRestRequest) (*pb.DaggerheartApplyRestResponse, error) {
	result, err := s.recoveryHandler().ApplyRest(ctx, in)
	if err != nil {
		return nil, err
	}
	entries := make([]*pb.DaggerheartCharacterStateEntry, 0, len(result.CharacterStates))
	for _, entry := range result.CharacterStates {
		entries = append(entries, &pb.DaggerheartCharacterStateEntry{
			CharacterId: entry.CharacterID,
			State:       statetransport.CharacterStateToProto(entry.State),
		})
	}
	return &pb.DaggerheartApplyRestResponse{
		Snapshot: &pb.DaggerheartSnapshot{
			GmFear:                int32(result.Snapshot.GMFear),
			ConsecutiveShortRests: int32(result.Snapshot.ConsecutiveShortRests),
		},
		CharacterStates: entries,
		Countdowns: func() []*pb.DaggerheartCountdown {
			countdowns := make([]*pb.DaggerheartCountdown, 0, len(result.Countdowns))
			for _, countdown := range result.Countdowns {
				countdowns = append(countdowns, countdowntransport.CountdownToProto(countdown))
			}
			return countdowns
		}(),
	}, nil
}

func (s *DaggerheartService) ApplyTemporaryArmor(ctx context.Context, in *pb.DaggerheartApplyTemporaryArmorRequest) (*pb.DaggerheartApplyTemporaryArmorResponse, error) {
	result, err := s.recoveryHandler().ApplyTemporaryArmor(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.DaggerheartApplyTemporaryArmorResponse{
		CharacterId: result.CharacterID,
		State:       statetransport.CharacterStateToProto(result.State),
	}, nil
}

func (s *DaggerheartService) SwapLoadout(ctx context.Context, in *pb.DaggerheartSwapLoadoutRequest) (*pb.DaggerheartSwapLoadoutResponse, error) {
	result, err := s.recoveryHandler().SwapLoadout(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.DaggerheartSwapLoadoutResponse{
		CharacterId: result.CharacterID,
		State:       statetransport.CharacterStateToProto(result.State),
	}, nil
}

func (s *DaggerheartService) ApplyDeathMove(ctx context.Context, in *pb.DaggerheartApplyDeathMoveRequest) (*pb.DaggerheartApplyDeathMoveResponse, error) {
	result, err := s.recoveryHandler().ApplyDeathMove(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.DaggerheartApplyDeathMoveResponse{
		CharacterId: result.CharacterID,
		State:       statetransport.CharacterStateToProto(result.State),
		Result: &pb.DaggerheartDeathMoveResult{
			Move:          recoverytransport.DeathMoveToProto(result.Outcome.Move),
			LifeState:     conditiontransport.LifeStateToProto(result.Outcome.LifeState),
			HopeDie:       statetransport.OptionalInt32(result.Outcome.HopeDie),
			FearDie:       statetransport.OptionalInt32(result.Outcome.FearDie),
			HpCleared:     int32(result.Outcome.HPCleared),
			StressCleared: int32(result.Outcome.StressCleared),
			ScarGained:    result.Outcome.ScarGained,
		},
	}, nil
}

func (s *DaggerheartService) ResolveBlazeOfGlory(ctx context.Context, in *pb.DaggerheartResolveBlazeOfGloryRequest) (*pb.DaggerheartResolveBlazeOfGloryResponse, error) {
	result, err := s.recoveryHandler().ResolveBlazeOfGlory(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.DaggerheartResolveBlazeOfGloryResponse{
		CharacterId: result.CharacterID,
		State:       statetransport.CharacterStateToProto(result.State),
		Result: &pb.DaggerheartBlazeOfGloryResult{
			LifeState: conditiontransport.LifeStateToProto(result.LifeState),
		},
	}, nil
}
