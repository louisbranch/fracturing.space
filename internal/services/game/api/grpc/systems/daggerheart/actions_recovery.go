package daggerheart

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *DaggerheartService) ApplyRest(ctx context.Context, in *pb.DaggerheartApplyRestRequest) (*pb.DaggerheartApplyRestResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply rest request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}
	if s.seedFunc == nil {
		return nil, status.Error(codes.Internal, "seed generator is not configured")
	}
	if s.stores.Domain == nil {
		return nil, status.Error(codes.Internal, "domain engine is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart rest")
	}

	sessionID := strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if err := s.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return nil, err
	}

	if in.Rest == nil {
		return nil, status.Error(codes.InvalidArgument, "rest is required")
	}
	if in.Rest.RestType == pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "rest_type is required")
	}

	seed, _, _, err := random.ResolveSeed(
		in.Rest.GetRng(),
		s.seedFunc,
		func(mode commonv1.RollMode) bool { return mode == commonv1.RollMode_REPLAY },
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resolve rest seed: %v", err)
	}

	currentSnap, err := s.stores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, status.Errorf(codes.Internal, "get daggerheart snapshot: %v", err)
	}
	state := daggerheart.RestState{ConsecutiveShortRests: currentSnap.ConsecutiveShortRests}
	restType, err := daggerheartRestTypeFromProto(in.Rest.RestType)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	outcome, err := daggerheart.ResolveRestOutcome(state, restType, in.Rest.Interrupted, seed, int(in.Rest.PartySize))
	if err != nil {
		return nil, handleDomainError(err)
	}

	gmFearBefore := currentSnap.GMFear
	gmFearAfter := gmFearBefore + outcome.GMFearGain
	if gmFearAfter > daggerheart.GMFearMax {
		gmFearAfter = daggerheart.GMFearMax
	}
	shortBefore := currentSnap.ConsecutiveShortRests
	shortAfter := outcome.State.ConsecutiveShortRests
	longTermCountdownID := strings.TrimSpace(in.Rest.GetLongTermCountdownId())
	var longTermCountdown *daggerheart.CountdownUpdatePayload
	if outcome.AdvanceCountdown && longTermCountdownID != "" {
		storedCountdown, err := s.stores.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, longTermCountdownID)
		if err != nil {
			return nil, handleDomainError(err)
		}
		countdown := daggerheart.Countdown{
			CampaignID: storedCountdown.CampaignID,
			ID:         storedCountdown.CountdownID,
			Name:       storedCountdown.Name,
			Kind:       storedCountdown.Kind,
			Current:    storedCountdown.Current,
			Max:        storedCountdown.Max,
			Direction:  storedCountdown.Direction,
			Looping:    storedCountdown.Looping,
		}
		update, err := daggerheart.ApplyCountdownUpdate(countdown, 1, nil)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		longTermCountdown = &daggerheart.CountdownUpdatePayload{
			CountdownID: longTermCountdownID,
			Before:      update.Before,
			After:       update.After,
			Delta:       update.Delta,
			Looped:      update.Looped,
			Reason:      "long_rest",
		}
	}

	payload := daggerheart.RestTakePayload{
		RestType:          daggerheartRestTypeToString(restType),
		Interrupted:       in.Rest.Interrupted,
		GMFearBefore:      gmFearBefore,
		GMFearAfter:       gmFearAfter,
		ShortRestsBefore:  shortBefore,
		ShortRestsAfter:   shortAfter,
		RefreshRest:       outcome.RefreshRest,
		RefreshLongRest:   outcome.RefreshLongRest,
		LongTermCountdown: longTermCountdown,
	}
	characterIDs := make([]string, len(in.GetCharacterIds()))
	copy(characterIDs, in.GetCharacterIds())
	payload.CharacterStates = make([]daggerheart.RestCharacterStatePatch, 0, len(characterIDs))
	for _, characterID := range characterIDs {
		characterID = strings.TrimSpace(characterID)
		if characterID == "" {
			continue
		}
		payload.CharacterStates = append(payload.CharacterStates, daggerheart.RestCharacterStatePatch{
			CharacterID: characterID,
		})
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:    campaignID,
		Type:          commandTypeDaggerheartRestTake,
		ActorType:     command.ActorTypeSystem,
		SessionID:     sessionID,
		RequestID:     requestID,
		InvocationID:  invocationID,
		EntityType:    "session",
		EntityID:      campaignID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}, adapter, domainCommandApplyOptions{
		requireEvents:   true,
		missingEventMsg: "rest did not emit an event",
		applyErrMessage: "apply rest event",
		executeErrMsg:   "execute domain command",
	})
	if err != nil {
		return nil, err
	}

	updatedSnap, err := s.stores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load daggerheart snapshot: %v", err)
	}

	entries := make([]*pb.DaggerheartCharacterStateEntry, 0, len(characterIDs))
	for _, id := range characterIDs {
		if strings.TrimSpace(id) == "" {
			continue
		}
		state, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				continue
			}
			return nil, status.Errorf(codes.Internal, "get daggerheart character state: %v", err)
		}
		entries = append(entries, &pb.DaggerheartCharacterStateEntry{
			CharacterId: id,
			State:       daggerheartStateToProto(state),
		})
	}

	return &pb.DaggerheartApplyRestResponse{
		Snapshot: &pb.DaggerheartSnapshot{
			GmFear:                int32(updatedSnap.GMFear),
			ConsecutiveShortRests: int32(updatedSnap.ConsecutiveShortRests),
		},
		CharacterStates: entries,
	}, nil
}

func (s *DaggerheartService) ApplyDowntimeMove(ctx context.Context, in *pb.DaggerheartApplyDowntimeMoveRequest) (*pb.DaggerheartApplyDowntimeMoveResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply downtime request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart downtime")
	}

	sessionID := strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if err := s.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return nil, err
	}

	if in.Move == nil {
		return nil, status.Error(codes.InvalidArgument, "move is required")
	}
	if in.Move.Move == pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "downtime move is required")
	}

	profile, err := s.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	current, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	state := daggerheart.NewCharacterState(daggerheart.CharacterStateConfig{
		CampaignID:  campaignID,
		CharacterID: characterID,
		HP:          current.Hp,
		HPMax:       profile.HpMax,
		Hope:        current.Hope,
		HopeMax:     current.HopeMax,
		Stress:      current.Stress,
		StressMax:   profile.StressMax,
		Armor:       current.Armor,
		ArmorMax:    profile.ArmorMax,
		LifeState:   current.LifeState,
	})

	move, err := daggerheartDowntimeMoveFromProto(in.Move.Move)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	result := daggerheart.ApplyDowntimeMove(state, move, daggerheart.DowntimeOptions{PrepareWithGroup: in.Move.PrepareWithGroup})

	hopeBefore := result.HopeBefore
	hopeAfter := result.HopeAfter
	stressBefore := result.StressBefore
	stressAfter := result.StressAfter
	armorBefore := result.ArmorBefore
	armorAfter := result.ArmorAfter
	payload := daggerheart.DowntimeMoveApplyPayload{
		CharacterID:  characterID,
		Move:         daggerheartDowntimeMoveToString(move),
		HopeBefore:   &hopeBefore,
		HopeAfter:    &hopeAfter,
		StressBefore: &stressBefore,
		StressAfter:  &stressAfter,
		ArmorBefore:  &armorBefore,
		ArmorAfter:   &armorAfter,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:    campaignID,
		Type:          commandTypeDaggerheartDowntimeMoveApply,
		ActorType:     command.ActorTypeSystem,
		SessionID:     sessionID,
		RequestID:     requestID,
		InvocationID:  invocationID,
		EntityType:    "character",
		EntityID:      characterID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}, adapter, domainCommandApplyOptions{
		requireEvents:   true,
		missingEventMsg: "downtime move did not emit an event",
		applyErrMessage: "apply downtime move event",
		executeErrMsg:   "execute domain command",
	})
	if err != nil {
		return nil, err
	}
	if err := s.applyStressVulnerableCondition(ctx, campaignID, grpcmeta.SessionIDFromContext(ctx), characterID, current.Conditions, stressBefore, stressAfter, profile.StressMax, nil, grpcmeta.RequestIDFromContext(ctx)); err != nil {
		return nil, err
	}

	updated, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load daggerheart state: %v", err)
	}
	return &pb.DaggerheartApplyDowntimeMoveResponse{
		CharacterId: characterID,
		State:       daggerheartStateToProto(updated),
	}, nil
}

func (s *DaggerheartService) ApplyTemporaryArmor(ctx context.Context, in *pb.DaggerheartApplyTemporaryArmorRequest) (*pb.DaggerheartApplyTemporaryArmorResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply temporary armor request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart temporary armor")
	}

	sessionID := strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if err := s.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return nil, err
	}

	if in.Armor == nil {
		return nil, status.Error(codes.InvalidArgument, "armor is required")
	}
	if _, err := s.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID); err != nil {
		return nil, handleDomainError(err)
	}
	if _, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID); err != nil {
		return nil, handleDomainError(err)
	}

	payload := daggerheart.CharacterTemporaryArmorApplyPayload{
		CharacterID: characterID,
		Source:      strings.TrimSpace(in.Armor.GetSource()),
		Duration:    strings.TrimSpace(in.Armor.GetDuration()),
		Amount:      int(in.Armor.GetAmount()),
		SourceID:    strings.TrimSpace(in.Armor.GetSourceId()),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:    campaignID,
		Type:          commandTypeDaggerheartTemporaryArmorApply,
		ActorType:     command.ActorTypeSystem,
		SessionID:     sessionID,
		RequestID:     requestID,
		InvocationID:  invocationID,
		EntityType:    "character",
		EntityID:      characterID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}, adapter, domainCommandApplyOptions{
		requireEvents:   true,
		missingEventMsg: "temporary armor apply did not emit an event",
		applyErrMessage: "apply temporary armor event",
		executeErrMsg:   "execute domain command",
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load daggerheart state: %v", err)
	}
	return &pb.DaggerheartApplyTemporaryArmorResponse{
		CharacterId: characterID,
		State:       daggerheartStateToProto(updated),
	}, nil
}

func (s *DaggerheartService) SwapLoadout(ctx context.Context, in *pb.DaggerheartSwapLoadoutRequest) (*pb.DaggerheartSwapLoadoutResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "swap loadout request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart loadout")
	}

	sessionID := strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if err := s.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return nil, err
	}

	if in.Swap == nil {
		return nil, status.Error(codes.InvalidArgument, "swap is required")
	}
	if strings.TrimSpace(in.Swap.CardId) == "" {
		return nil, status.Error(codes.InvalidArgument, "card_id is required")
	}
	if in.Swap.RecallCost < 0 {
		return nil, status.Error(codes.InvalidArgument, "recall_cost must be non-negative")
	}

	profile, err := s.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	current, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	state := daggerheart.NewCharacterState(daggerheart.CharacterStateConfig{
		CampaignID:  campaignID,
		CharacterID: characterID,
		HP:          current.Hp,
		HPMax:       profile.HpMax,
		Hope:        current.Hope,
		HopeMax:     current.HopeMax,
		Stress:      current.Stress,
		StressMax:   profile.StressMax,
		Armor:       current.Armor,
		ArmorMax:    profile.ArmorMax,
		LifeState:   current.LifeState,
	})

	stressBefore := state.Stress
	if !in.Swap.InRest && in.Swap.RecallCost > 0 {
		if _, _, err := state.SpendResource(daggerheart.ResourceStress, int(in.Swap.RecallCost)); err != nil {
			return nil, handleDomainError(err)
		}
	}
	stressAfter := state.Stress

	payload := daggerheart.LoadoutSwapPayload{
		CharacterID:  characterID,
		CardID:       in.Swap.CardId,
		From:         "vault",
		To:           "active",
		RecallCost:   int(in.Swap.RecallCost),
		StressBefore: &stressBefore,
		StressAfter:  &stressAfter,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:    campaignID,
		Type:          commandTypeDaggerheartLoadoutSwap,
		ActorType:     command.ActorTypeSystem,
		SessionID:     sessionID,
		RequestID:     requestID,
		InvocationID:  invocationID,
		EntityType:    "character",
		EntityID:      characterID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}, adapter, domainCommandApplyOptions{
		requireEvents:   true,
		missingEventMsg: "loadout swap did not emit an event",
		applyErrMessage: "apply loadout swap event",
		executeErrMsg:   "execute domain command",
	})
	if err != nil {
		return nil, err
	}
	if err := s.applyStressVulnerableCondition(ctx, campaignID, grpcmeta.SessionIDFromContext(ctx), characterID, current.Conditions, stressBefore, stressAfter, profile.StressMax, nil, grpcmeta.RequestIDFromContext(ctx)); err != nil {
		return nil, err
	}
	if !in.Swap.InRest && in.Swap.RecallCost > 0 {
		spendPayload := daggerheart.StressSpendPayload{
			CharacterID: characterID,
			Amount:      int(in.Swap.RecallCost),
			Before:      stressBefore,
			After:       stressAfter,
			Source:      "loadout_swap",
		}
		spendJSON, err := json.Marshal(spendPayload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "encode stress spend payload: %v", err)
		}
		_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
			CampaignID:    campaignID,
			Type:          commandTypeDaggerheartStressSpend,
			ActorType:     command.ActorTypeSystem,
			SessionID:     sessionID,
			RequestID:     requestID,
			InvocationID:  invocationID,
			EntityType:    "character",
			EntityID:      characterID,
			SystemID:      daggerheart.SystemID,
			SystemVersion: daggerheart.SystemVersion,
			PayloadJSON:   spendJSON,
		}, adapter, domainCommandApplyOptions{
			requireEvents:   true,
			missingEventMsg: "stress spend did not emit an event",
			applyErrMessage: "apply stress spend event",
			executeErrMsg:   "execute domain command",
		})
		if err != nil {
			return nil, err
		}
	}

	updated, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load daggerheart state: %v", err)
	}
	return &pb.DaggerheartSwapLoadoutResponse{
		CharacterId: characterID,
		State:       daggerheartStateToProto(updated),
	}, nil
}

func (s *DaggerheartService) ApplyDeathMove(ctx context.Context, in *pb.DaggerheartApplyDeathMoveRequest) (*pb.DaggerheartApplyDeathMoveResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply death move request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}
	if s.seedFunc == nil {
		return nil, status.Error(codes.Internal, "seed generator is not configured")
	}
	if s.stores.Domain == nil {
		return nil, status.Error(codes.Internal, "domain engine is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart death moves")
	}

	sessionID := strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if err := s.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return nil, err
	}

	move, err := daggerheartDeathMoveFromProto(in.Move)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if move != daggerheart.DeathMoveRiskItAll {
		if in.HpClear != nil || in.StressClear != nil {
			return nil, status.Error(codes.InvalidArgument, "hp_clear and stress_clear are only valid for risk it all")
		}
	}

	seed, _, _, err := random.ResolveSeed(
		in.GetRng(),
		s.seedFunc,
		func(mode commonv1.RollMode) bool { return mode == commonv1.RollMode_REPLAY },
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resolve death move seed: %v", err)
	}

	profile, err := s.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	state, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	if state.Hp > 0 {
		return nil, status.Error(codes.FailedPrecondition, "death move requires hp to be zero")
	}
	if state.LifeState == daggerheart.LifeStateDead {
		return nil, status.Error(codes.FailedPrecondition, "character is already dead")
	}

	hpMax := profile.HpMax
	if hpMax == 0 {
		hpMax = daggerheart.PCHpMax
	}
	stressMax := profile.StressMax
	if stressMax < 0 {
		stressMax = 0
	}
	hopeMax := state.HopeMax
	if hopeMax == 0 {
		hopeMax = daggerheart.HopeMax
	}
	level := profile.Level
	if level == 0 {
		level = daggerheart.PCLevelDefault
	}

	var hpClear *int
	var stressClear *int
	if in.HpClear != nil {
		value := int(in.GetHpClear())
		hpClear = &value
	}
	if in.StressClear != nil {
		value := int(in.GetStressClear())
		stressClear = &value
	}

	result, err := daggerheart.ResolveDeathMove(daggerheart.DeathMoveInput{
		Move:             move,
		Level:            level,
		HP:               state.Hp,
		HPMax:            hpMax,
		Hope:             state.Hope,
		HopeMax:          hopeMax,
		Stress:           state.Stress,
		StressMax:        stressMax,
		RiskItAllHPClear: hpClear,
		RiskItAllStClear: stressClear,
		Seed:             seed,
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if result.LifeState == daggerheart.LifeStateDead && s.stores.Domain == nil {
		return nil, status.Error(codes.Internal, "character store is not configured")
	}

	hpBefore := result.HPBefore
	hpAfter := result.HPAfter
	hopeBefore := result.HopeBefore
	hopeAfter := result.HopeAfter
	hopeMaxBefore := result.HopeMaxBefore
	hopeMaxAfter := result.HopeMaxAfter
	stressBefore := result.StressBefore
	stressAfter := result.StressAfter
	lifeStateBefore := state.LifeState
	if lifeStateBefore == "" {
		lifeStateBefore = daggerheart.LifeStateAlive
	}
	patchPayload := daggerheart.CharacterStatePatchedPayload{
		CharacterID: characterID,
	}
	if hpBefore != hpAfter {
		patchPayload.HPBefore = &hpBefore
		patchPayload.HPAfter = &hpAfter
	}
	if hopeBefore != hopeAfter {
		patchPayload.HopeBefore = &hopeBefore
		patchPayload.HopeAfter = &hopeAfter
	}
	if hopeMaxBefore != hopeMaxAfter {
		patchPayload.HopeMaxBefore = &hopeMaxBefore
		patchPayload.HopeMaxAfter = &hopeMaxAfter
	}
	if stressBefore != stressAfter {
		patchPayload.StressBefore = &stressBefore
		patchPayload.StressAfter = &stressAfter
	}
	if lifeStateBefore != result.LifeState {
		lifeStateAfter := result.LifeState
		patchPayload.LifeStateBefore = &lifeStateBefore
		patchPayload.LifeStateAfter = &lifeStateAfter
	}
	if patchPayload.HPBefore == nil &&
		patchPayload.HopeBefore == nil &&
		patchPayload.HopeMaxBefore == nil &&
		patchPayload.StressBefore == nil &&
		patchPayload.LifeStateBefore == nil {
		return nil, status.Error(codes.Internal, "death move did not change character state")
	}
	payloadJSON, err := json.Marshal(patchPayload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:    campaignID,
		Type:          commandTypeDaggerheartCharacterStatePatch,
		ActorType:     command.ActorTypeSystem,
		SessionID:     sessionID,
		RequestID:     requestID,
		InvocationID:  invocationID,
		EntityType:    "character",
		EntityID:      characterID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}, adapter, domainCommandApplyOptions{
		requireEvents:   true,
		missingEventMsg: "death move did not emit an event",
		applyErrMessage: "apply character state event",
		executeErrMsg:   "execute domain command",
	})
	if err != nil {
		return nil, err
	}
	if result.LifeState == daggerheart.LifeStateDead {
		if err := s.appendCharacterDeletedEvent(ctx, campaignID, characterID, result.Move); err != nil {
			return nil, err
		}
	}
	if err := s.applyStressVulnerableCondition(ctx, campaignID, grpcmeta.SessionIDFromContext(ctx), characterID, state.Conditions, stressBefore, stressAfter, stressMax, nil, grpcmeta.RequestIDFromContext(ctx)); err != nil {
		return nil, err
	}

	updated, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load daggerheart state: %v", err)
	}

	return &pb.DaggerheartApplyDeathMoveResponse{
		CharacterId: characterID,
		State:       daggerheartStateToProto(updated),
		Result: &pb.DaggerheartDeathMoveResult{
			Move:          daggerheartDeathMoveToProto(result.Move),
			LifeState:     daggerheartLifeStateToProto(result.LifeState),
			HopeDie:       optionalInt32(result.HopeDie),
			FearDie:       optionalInt32(result.FearDie),
			HpCleared:     int32(result.HPCleared),
			StressCleared: int32(result.StressCleared),
			ScarGained:    result.ScarGained,
		},
	}, nil
}
