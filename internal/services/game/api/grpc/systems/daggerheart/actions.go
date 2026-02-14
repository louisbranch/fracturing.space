package daggerheart

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/snapshot"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/core/dice"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/core/random"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (s *DaggerheartService) ApplyDamage(ctx context.Context, in *pb.DaggerheartApplyDamageRequest) (*pb.DaggerheartApplyDamageResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply damage request is required")
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
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart damage")
	}

	if in.Damage == nil {
		return nil, status.Error(codes.InvalidArgument, "damage is required")
	}
	if in.Damage.Amount < 0 {
		return nil, status.Error(codes.InvalidArgument, "damage amount must be non-negative")
	}
	if in.Damage.DamageType == pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "damage_type is required")
	}

	profile, err := s.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	state, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	result, mitigated, err := applyDaggerheartDamage(in.Damage, profile, state)
	if err != nil {
		return nil, handleDomainError(err)
	}

	hpBefore := result.HPBefore
	hpAfter := result.HPAfter
	armorBefore := result.ArmorBefore
	armorAfter := result.ArmorAfter
	requireDamageRoll := in.GetRequireDamageRoll()
	var rollSeq *uint64
	if in.RollSeq != nil {
		value := in.GetRollSeq()
		rollSeq = &value
	}
	sourceCharacterIDs := normalizeTargets(in.Damage.GetSourceCharacterIds())
	if requireDamageRoll && rollSeq == nil {
		return nil, status.Error(codes.InvalidArgument, "roll_seq is required when require_damage_roll is true")
	}
	if rollSeq != nil {
		rollEvent, err := s.stores.Event.GetEventBySeq(ctx, campaignID, *rollSeq)
		if err != nil {
			return nil, handleDomainError(err)
		}
		if rollEvent.Type != daggerheart.EventTypeDamageRollResolved {
			return nil, status.Error(codes.InvalidArgument, "roll_seq must reference action.damage_roll_resolved")
		}
		var rollPayload daggerheart.DamageRollResolvedPayload
		if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
			return nil, status.Errorf(codes.Internal, "decode damage roll payload: %v", err)
		}
		if rollPayload.CharacterID != characterID && !containsString(sourceCharacterIDs, rollPayload.CharacterID) {
			return nil, status.Error(codes.InvalidArgument, "roll_seq does not match target or source character")
		}
	}
	payload := daggerheart.DamageAppliedPayload{
		CharacterID:        characterID,
		HpBefore:           &hpBefore,
		HpAfter:            &hpAfter,
		ArmorBefore:        &armorBefore,
		ArmorAfter:         &armorAfter,
		ArmorSpent:         result.ArmorSpent,
		Severity:           daggerheartSeverityToString(result.Result.Severity),
		Marks:              result.Result.Marks,
		DamageType:         daggerheartDamageTypeToString(in.Damage.DamageType),
		RollSeq:            rollSeq,
		ResistPhysical:     in.Damage.ResistPhysical,
		ResistMagic:        in.Damage.ResistMagic,
		ImmunePhysical:     in.Damage.ImmunePhysical,
		ImmuneMagic:        in.Damage.ImmuneMagic,
		Direct:             in.Damage.Direct,
		MassiveDamage:      in.Damage.MassiveDamage,
		Mitigated:          mitigated,
		Source:             in.Damage.Source,
		SourceCharacterIDs: sourceCharacterIDs,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          daggerheart.EventTypeDamageApplied,
		SessionID:     grpcmeta.SessionIDFromContext(ctx),
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "character",
		EntityID:      characterID,
		SystemID:      c.System.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	if err := adapter.ApplyEvent(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	updated, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load daggerheart state: %v", err)
	}
	return &pb.DaggerheartApplyDamageResponse{
		CharacterId: characterID,
		State:       daggerheartStateToProto(updated),
	}, nil
}

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

	payload := daggerheart.RestTakenPayload{
		RestType:         daggerheartRestTypeToString(restType),
		Interrupted:      in.Rest.Interrupted,
		GMFearBefore:     gmFearBefore,
		GMFearAfter:      gmFearAfter,
		ShortRestsBefore: shortBefore,
		ShortRestsAfter:  shortAfter,
		RefreshRest:      outcome.RefreshRest,
		RefreshLongRest:  outcome.RefreshLongRest,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          daggerheart.EventTypeRestTaken,
		SessionID:     grpcmeta.SessionIDFromContext(ctx),
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "session",
		EntityID:      campaignID,
		SystemID:      c.System.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	if err := adapter.ApplyEvent(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	updatedSnap, err := s.stores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load daggerheart snapshot: %v", err)
	}

	entries := make([]*pb.DaggerheartCharacterStateEntry, 0, len(in.CharacterIds))
	for _, id := range in.CharacterIds {
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
	payload := daggerheart.DowntimeMoveAppliedPayload{
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

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          daggerheart.EventTypeDowntimeMoveApplied,
		SessionID:     grpcmeta.SessionIDFromContext(ctx),
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "character",
		EntityID:      characterID,
		SystemID:      c.System.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	if err := adapter.ApplyEvent(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
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

	stressBefore := state.Stress()
	if !in.Swap.InRest && in.Swap.RecallCost > 0 {
		if _, _, err := state.SpendResource(daggerheart.ResourceStress, int(in.Swap.RecallCost)); err != nil {
			return nil, handleDomainError(err)
		}
	}
	stressAfter := state.Stress()

	payload := daggerheart.LoadoutSwappedPayload{
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

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          daggerheart.EventTypeLoadoutSwapped,
		SessionID:     grpcmeta.SessionIDFromContext(ctx),
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "character",
		EntityID:      characterID,
		SystemID:      c.System.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	if err := adapter.ApplyEvent(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}
	if !in.Swap.InRest && in.Swap.RecallCost > 0 {
		spendPayload := daggerheart.StressSpentPayload{
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
		spendEvent, err := s.stores.Event.AppendEvent(ctx, event.Event{
			CampaignID:    campaignID,
			Timestamp:     time.Now().UTC(),
			Type:          daggerheart.EventTypeStressSpent,
			SessionID:     grpcmeta.SessionIDFromContext(ctx),
			RequestID:     grpcmeta.RequestIDFromContext(ctx),
			InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
			ActorType:     event.ActorTypeSystem,
			EntityType:    "character",
			EntityID:      characterID,
			SystemID:      c.System.String(),
			SystemVersion: daggerheart.SystemVersion,
			PayloadJSON:   spendJSON,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "append stress spend event: %v", err)
		}
		if err := adapter.ApplyEvent(ctx, spendEvent); err != nil {
			return nil, status.Errorf(codes.Internal, "apply stress spend event: %v", err)
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
	if result.LifeState == daggerheart.LifeStateDead && s.stores.Character == nil {
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
	payload := daggerheart.DeathMoveResolvedPayload{
		CharacterID:     characterID,
		Move:            move,
		LifeStateBefore: &lifeStateBefore,
		LifeStateAfter:  result.LifeState,
		HpBefore:        &hpBefore,
		HpAfter:         &hpAfter,
		HopeBefore:      &hopeBefore,
		HopeAfter:       &hopeAfter,
		HopeMaxBefore:   &hopeMaxBefore,
		HopeMaxAfter:    &hopeMaxAfter,
		StressBefore:    &stressBefore,
		StressAfter:     &stressAfter,
		HopeDie:         result.HopeDie,
		FearDie:         result.FearDie,
		ScarGained:      result.ScarGained,
		HPCleared:       result.HPCleared,
		StressCleared:   result.StressCleared,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          daggerheart.EventTypeDeathMoveResolved,
		SessionID:     grpcmeta.SessionIDFromContext(ctx),
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "character",
		EntityID:      characterID,
		SystemID:      c.System.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	if err := adapter.ApplyEvent(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}
	if result.LifeState == daggerheart.LifeStateDead {
		if err := s.appendCharacterDeletedEvent(ctx, campaignID, characterID, result.Move); err != nil {
			return nil, err
		}
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

func (s *DaggerheartService) ApplyConditions(ctx context.Context, in *pb.DaggerheartApplyConditionsRequest) (*pb.DaggerheartApplyConditionsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply conditions request is required")
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
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart conditions")
	}

	addConditions, err := daggerheartConditionsFromProto(in.GetAdd())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	removeConditions, err := daggerheartConditionsFromProto(in.GetRemove())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if len(addConditions) == 0 && len(removeConditions) == 0 {
		return nil, status.Error(codes.InvalidArgument, "conditions to add or remove are required")
	}

	normalizedAdd := []string{}
	if len(addConditions) > 0 {
		normalizedAdd, err = daggerheart.NormalizeConditions(addConditions)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	normalizedRemove := []string{}
	if len(removeConditions) > 0 {
		normalizedRemove, err = daggerheart.NormalizeConditions(removeConditions)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	removeSet := make(map[string]struct{}, len(normalizedRemove))
	for _, value := range normalizedRemove {
		removeSet[value] = struct{}{}
	}
	for _, value := range normalizedAdd {
		if _, ok := removeSet[value]; ok {
			return nil, status.Error(codes.InvalidArgument, "conditions cannot be both added and removed")
		}
	}

	state, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	before, err := daggerheart.NormalizeConditions(state.Conditions)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "invalid stored conditions: %v", err)
	}

	afterSet := make(map[string]struct{}, len(before)+len(normalizedAdd))
	for _, value := range before {
		afterSet[value] = struct{}{}
	}
	for _, value := range normalizedRemove {
		delete(afterSet, value)
	}
	for _, value := range normalizedAdd {
		afterSet[value] = struct{}{}
	}

	afterList := make([]string, 0, len(afterSet))
	for value := range afterSet {
		afterList = append(afterList, value)
	}
	after, err := daggerheart.NormalizeConditions(afterList)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "invalid condition set: %v", err)
	}

	added, removed := daggerheart.DiffConditions(before, after)
	if len(added) == 0 && len(removed) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "no condition changes to apply")
	}

	source := strings.TrimSpace(in.GetSource())
	var rollSeq *uint64
	if in.RollSeq != nil {
		value := in.GetRollSeq()
		rollSeq = &value
		rollEvent, err := s.stores.Event.GetEventBySeq(ctx, campaignID, value)
		if err != nil {
			return nil, handleDomainError(err)
		}
		sessionID := strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
		if sessionID != "" && rollEvent.SessionID != sessionID {
			return nil, status.Error(codes.InvalidArgument, "roll seq does not match session")
		}
	}

	payload := daggerheart.ConditionChangedPayload{
		CharacterID:      characterID,
		ConditionsBefore: before,
		ConditionsAfter:  after,
		Added:            added,
		Removed:          removed,
		Source:           source,
		RollSeq:          rollSeq,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode condition payload: %v", err)
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          daggerheart.EventTypeConditionChanged,
		SessionID:     grpcmeta.SessionIDFromContext(ctx),
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "character",
		EntityID:      characterID,
		SystemID:      c.System.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append condition event: %v", err)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	if err := adapter.ApplyEvent(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply condition event: %v", err)
	}

	updated, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load daggerheart state: %v", err)
	}

	return &pb.DaggerheartApplyConditionsResponse{
		CharacterId: characterID,
		State:       daggerheartStateToProto(updated),
		Added:       daggerheartConditionsToProto(added),
		Removed:     daggerheartConditionsToProto(removed),
	}, nil
}

func (s *DaggerheartService) ApplyGmMove(ctx context.Context, in *pb.DaggerheartApplyGmMoveRequest) (*pb.DaggerheartApplyGmMoveResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply gm move request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
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
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	move := strings.TrimSpace(in.GetMove())
	if move == "" {
		return nil, status.Error(codes.InvalidArgument, "move is required")
	}
	if in.GetFearSpent() < 0 {
		return nil, status.Error(codes.InvalidArgument, "fear_spent must be non-negative")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart gm moves")
	}

	sess, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.SessionStatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}

	gmFearBefore := 0
	gmFearAfter := 0
	if snap, err := s.stores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID); err == nil {
		gmFearBefore = snap.GMFear
		gmFearAfter = snap.GMFear
	}

	fearSpent := int(in.GetFearSpent())
	if fearSpent > 0 {
		_, before, after, err := snapshot.ApplyGmFearSpend(snapshot.GmFear{CampaignID: campaignID, Value: gmFearBefore}, fearSpent)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		gmFearBefore = before
		gmFearAfter = after

		gmPayload := daggerheart.GMFearChangedPayload{Before: before, After: after, Reason: "gm_move"}
		gmPayloadJSON, err := json.Marshal(gmPayload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "encode gm fear payload: %v", err)
		}
		storedGM, err := s.stores.Event.AppendEvent(ctx, event.Event{
			CampaignID:    campaignID,
			Timestamp:     time.Now().UTC(),
			Type:          daggerheart.EventTypeGMFearChanged,
			SessionID:     sessionID,
			RequestID:     grpcmeta.RequestIDFromContext(ctx),
			InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
			ActorType:     event.ActorTypeSystem,
			EntityType:    "campaign",
			EntityID:      campaignID,
			SystemID:      c.System.String(),
			SystemVersion: daggerheart.SystemVersion,
			PayloadJSON:   gmPayloadJSON,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "append gm fear event: %v", err)
		}
		adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
		if err := adapter.ApplyEvent(ctx, storedGM); err != nil {
			return nil, status.Errorf(codes.Internal, "apply gm fear event: %v", err)
		}
	}

	payload := daggerheart.GMMoveAppliedPayload{
		Move:        move,
		Description: strings.TrimSpace(in.GetDescription()),
		FearSpent:   fearSpent,
		Source:      strings.TrimSpace(in.GetSource()),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode gm move payload: %v", err)
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          daggerheart.EventTypeGMMoveApplied,
		SessionID:     sessionID,
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "gm_move",
		EntityID:      campaignID,
		SystemID:      c.System.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append gm move event: %v", err)
	}
	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	if err := adapter.ApplyEvent(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply gm move event: %v", err)
	}

	return &pb.DaggerheartApplyGmMoveResponse{
		CampaignId:   campaignID,
		GmFearBefore: int32(gmFearBefore),
		GmFearAfter:  int32(gmFearAfter),
	}, nil
}

func (s *DaggerheartService) CreateCountdown(ctx context.Context, in *pb.DaggerheartCreateCountdownRequest) (*pb.DaggerheartCreateCountdownResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create countdown request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
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
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	name := strings.TrimSpace(in.GetName())
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	kind, err := daggerheartCountdownKindFromProto(in.GetKind())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	direction, err := daggerheartCountdownDirectionFromProto(in.GetDirection())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	max := int(in.GetMax())
	if max <= 0 {
		return nil, status.Error(codes.InvalidArgument, "max must be positive")
	}
	current := int(in.GetCurrent())
	if current < 0 || current > max {
		return nil, status.Errorf(codes.InvalidArgument, "current must be in range 0..%d", max)
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart countdowns")
	}

	sess, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.SessionStatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}

	countdownID := strings.TrimSpace(in.GetCountdownId())
	if countdownID == "" {
		countdownID, err = id.NewID()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "generate countdown id: %v", err)
		}
	}
	if _, err := s.stores.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID); err == nil {
		return nil, status.Error(codes.FailedPrecondition, "countdown already exists")
	} else if !errors.Is(err, storage.ErrNotFound) {
		return nil, handleDomainError(err)
	}

	payload := daggerheart.CountdownCreatedPayload{
		CountdownID: countdownID,
		Name:        name,
		Kind:        kind,
		Current:     current,
		Max:         max,
		Direction:   direction,
		Looping:     in.GetLooping(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode countdown payload: %v", err)
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          daggerheart.EventTypeCountdownCreated,
		SessionID:     sessionID,
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "countdown",
		EntityID:      countdownID,
		SystemID:      c.System.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append countdown created event: %v", err)
	}
	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	if err := adapter.ApplyEvent(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply countdown created event: %v", err)
	}

	countdown, err := s.stores.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load countdown: %v", err)
	}

	return &pb.DaggerheartCreateCountdownResponse{
		Countdown: daggerheartCountdownToProto(countdown),
	}, nil
}

func (s *DaggerheartService) UpdateCountdown(ctx context.Context, in *pb.DaggerheartUpdateCountdownRequest) (*pb.DaggerheartUpdateCountdownResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update countdown request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
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
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	countdownID := strings.TrimSpace(in.GetCountdownId())
	if countdownID == "" {
		return nil, status.Error(codes.InvalidArgument, "countdown id is required")
	}

	if in.Current == nil && in.GetDelta() == 0 {
		return nil, status.Error(codes.InvalidArgument, "delta or current is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart countdowns")
	}

	sess, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.SessionStatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}

	storedCountdown, err := s.stores.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
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
	var override *int
	if in.Current != nil {
		value := int(in.GetCurrent())
		override = &value
	}
	update, err := daggerheart.ApplyCountdownUpdate(countdown, int(in.GetDelta()), override)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	payload := daggerheart.CountdownUpdatedPayload{
		CountdownID: countdownID,
		Before:      update.Before,
		After:       update.After,
		Delta:       update.Delta,
		Looped:      update.Looped,
		Reason:      strings.TrimSpace(in.GetReason()),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode countdown update payload: %v", err)
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          daggerheart.EventTypeCountdownUpdated,
		SessionID:     sessionID,
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "countdown",
		EntityID:      countdownID,
		SystemID:      c.System.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append countdown update event: %v", err)
	}
	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	if err := adapter.ApplyEvent(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply countdown update event: %v", err)
	}

	updatedCountdown, err := s.stores.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load countdown: %v", err)
	}

	return &pb.DaggerheartUpdateCountdownResponse{
		Countdown: daggerheartCountdownToProto(updatedCountdown),
		Before:    int32(update.Before),
		After:     int32(update.After),
		Delta:     int32(update.Delta),
	}, nil
}

func (s *DaggerheartService) DeleteCountdown(ctx context.Context, in *pb.DaggerheartDeleteCountdownRequest) (*pb.DaggerheartDeleteCountdownResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "delete countdown request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
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
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	countdownID := strings.TrimSpace(in.GetCountdownId())
	if countdownID == "" {
		return nil, status.Error(codes.InvalidArgument, "countdown id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart countdowns")
	}

	sess, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.SessionStatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}

	if _, err := s.stores.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID); err != nil {
		return nil, handleDomainError(err)
	}

	payload := daggerheart.CountdownDeletedPayload{
		CountdownID: countdownID,
		Reason:      strings.TrimSpace(in.GetReason()),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode countdown delete payload: %v", err)
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          daggerheart.EventTypeCountdownDeleted,
		SessionID:     sessionID,
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "countdown",
		EntityID:      countdownID,
		SystemID:      c.System.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append countdown delete event: %v", err)
	}
	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	if err := adapter.ApplyEvent(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply countdown delete event: %v", err)
	}

	return &pb.DaggerheartDeleteCountdownResponse{CountdownId: countdownID}, nil
}

func (s *DaggerheartService) ResolveBlazeOfGlory(ctx context.Context, in *pb.DaggerheartResolveBlazeOfGloryRequest) (*pb.DaggerheartResolveBlazeOfGloryResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "resolve blaze of glory request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Character == nil {
		return nil, status.Error(codes.Internal, "character store is not configured")
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
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart blaze of glory")
	}

	state, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if state.LifeState == "" {
		state.LifeState = daggerheart.LifeStateAlive
	}
	if state.LifeState == daggerheart.LifeStateDead {
		return nil, status.Error(codes.FailedPrecondition, "character is already dead")
	}
	if state.LifeState != daggerheart.LifeStateBlazeOfGlory {
		return nil, status.Error(codes.FailedPrecondition, "character is not in blaze of glory")
	}

	lifeStateBefore := state.LifeState
	lifeStateAfter := daggerheart.LifeStateDead
	payload := daggerheart.BlazeOfGloryResolvedPayload{
		CharacterID:     characterID,
		LifeStateBefore: &lifeStateBefore,
		LifeStateAfter:  lifeStateAfter,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          daggerheart.EventTypeBlazeOfGloryResolved,
		SessionID:     grpcmeta.SessionIDFromContext(ctx),
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "character",
		EntityID:      characterID,
		SystemID:      c.System.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	if err := adapter.ApplyEvent(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}
	updated, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load daggerheart state: %v", err)
	}
	if err := s.appendCharacterDeletedEvent(ctx, campaignID, characterID, daggerheart.LifeStateBlazeOfGlory); err != nil {
		return nil, err
	}

	return &pb.DaggerheartResolveBlazeOfGloryResponse{
		CharacterId: characterID,
		State:       daggerheartStateToProto(updated),
		Result: &pb.DaggerheartBlazeOfGloryResult{
			LifeState: daggerheartLifeStateToProto(lifeStateAfter),
		},
	}, nil
}

func (s *DaggerheartService) appendCharacterDeletedEvent(ctx context.Context, campaignID, characterID, reason string) error {
	if s.stores.Campaign == nil {
		return status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Character == nil {
		return status.Error(codes.Internal, "character store is not configured")
	}
	if s.stores.Event == nil {
		return status.Error(codes.Internal, "event store is not configured")
	}
	payload := event.CharacterDeletedPayload{
		CharacterID: characterID,
		Reason:      strings.TrimSpace(reason),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return status.Errorf(codes.Internal, "encode payload: %v", err)
	}
	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    time.Now().UTC(),
		Type:         event.TypeCharacterDeleted,
		SessionID:    grpcmeta.SessionIDFromContext(ctx),
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    event.ActorTypeSystem,
		EntityType:   "character",
		EntityID:     characterID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return status.Errorf(codes.Internal, "append event: %v", err)
	}
	applier := projection.Applier{Campaign: s.stores.Campaign, Character: s.stores.Character}
	if err := applier.Apply(ctx, stored); err != nil {
		return status.Errorf(codes.Internal, "apply event: %v", err)
	}
	return nil
}

func (s *DaggerheartService) SessionActionRoll(ctx context.Context, in *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session action roll request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
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

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}
	trait := strings.TrimSpace(in.GetTrait())
	if trait == "" {
		return nil, status.Error(codes.InvalidArgument, "trait is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart rolls")
	}

	sess, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.SessionStatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}

	state, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	modifierTotal, modifierList := normalizeActionModifiers(in.GetModifiers())
	rollKind := normalizeRollKind(in.GetRollKind())
	hopeSpends := hopeSpendsFromModifiers(in.GetModifiers())
	spendEventCount := 0
	totalSpend := 0
	for _, spend := range hopeSpends {
		if spend.Amount > 0 {
			spendEventCount++
			totalSpend += spend.Amount
		}
	}
	if rollKind == pb.RollKind_ROLL_KIND_REACTION && spendEventCount > 0 {
		return nil, status.Error(codes.InvalidArgument, "reaction rolls cannot spend hope")
	}
	statePatchNeeded := totalSpend > 0

	latestSeq, err := s.stores.Event.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load latest event seq: %v", err)
	}
	preEvents := spendEventCount
	if statePatchNeeded {
		preEvents++
	}
	rollSeq := latestSeq + uint64(preEvents) + 1

	seed, seedSource, rollMode, err := random.ResolveSeed(
		in.GetRng(),
		s.seedFunc,
		func(mode commonv1.RollMode) bool { return mode == commonv1.RollMode_REPLAY },
	)
	if err != nil {
		if errors.Is(err, random.ErrSeedOutOfRange()) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to resolve seed: %v", err)
	}

	if rollKind == pb.RollKind_ROLL_KIND_ACTION && spendEventCount > 0 {
		hopeBefore := state.Hope
		hopeAfter := hopeBefore
		if hopeBefore < totalSpend {
			return nil, status.Error(codes.FailedPrecondition, "insufficient hope")
		}

		for _, spend := range hopeSpends {
			if spend.Amount <= 0 {
				continue
			}
			before := hopeAfter
			after := before - spend.Amount
			payload := daggerheart.HopeSpentPayload{
				CharacterID: characterID,
				Amount:      spend.Amount,
				Before:      before,
				After:       after,
				RollSeq:     &rollSeq,
				Source:      spend.Source,
			}
			payloadJSON, err := json.Marshal(payload)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "encode hope spend payload: %v", err)
			}
			if _, err := s.stores.Event.AppendEvent(ctx, event.Event{
				CampaignID:    campaignID,
				Timestamp:     time.Now().UTC(),
				Type:          daggerheart.EventTypeHopeSpent,
				SessionID:     sessionID,
				RequestID:     grpcmeta.RequestIDFromContext(ctx),
				InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
				ActorType:     event.ActorTypeSystem,
				EntityType:    "character",
				EntityID:      characterID,
				SystemID:      c.System.String(),
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}); err != nil {
				return nil, status.Errorf(codes.Internal, "append hope spend event: %v", err)
			}
			hopeAfter = after
		}

		if hopeAfter != hopeBefore {
			statePatchNeeded = true
			payload := daggerheart.CharacterStatePatchedPayload{
				CharacterID: characterID,
				HopeBefore:  &hopeBefore,
				HopeAfter:   &hopeAfter,
			}
			payloadJSON, err := json.Marshal(payload)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "encode character state payload: %v", err)
			}
			storedState, err := s.stores.Event.AppendEvent(ctx, event.Event{
				CampaignID:    campaignID,
				Timestamp:     time.Now().UTC(),
				Type:          daggerheart.EventTypeCharacterStatePatched,
				SessionID:     sessionID,
				RequestID:     grpcmeta.RequestIDFromContext(ctx),
				InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
				ActorType:     event.ActorTypeSystem,
				EntityType:    "character",
				EntityID:      characterID,
				SystemID:      c.System.String(),
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "append character state event: %v", err)
			}
			adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
			if err := adapter.ApplyEvent(ctx, storedState); err != nil {
				return nil, status.Errorf(codes.Internal, "apply character state event: %v", err)
			}
		}
	}

	difficulty := int(in.GetDifficulty())
	result, generateHopeFear, triggerGMMove, critNegatesEffects, err := resolveRoll(
		rollKind,
		daggerheartdomain.ActionRequest{Modifier: modifierTotal, Difficulty: &difficulty, Seed: seed},
	)
	if err != nil {
		if errors.Is(err, daggerheartdomain.ErrInvalidDifficulty) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to roll action: %v", err)
	}

	rollModeLabel := rollMode.String()
	outcomeCode := outcomeToProto(result.Outcome).String()
	flavor := outcomeFlavorFromCode(outcomeCode)
	if !generateHopeFear {
		flavor = ""
	}

	results := map[string]any{
		"rng": map[string]any{
			"seed_used":   uint64(seed),
			"rng_algo":    random.RngAlgoMathRandV1,
			"seed_source": seedSource,
			"roll_mode":   rollModeLabel,
		},
		"dice": map[string]any{
			"hope_die":      result.Hope,
			"fear_die":      result.Fear,
			"advantage_die": result.AdvantageDie,
		},
		"modifier":           result.Modifier,
		"advantage_modifier": result.AdvantageModifier,
		"total":              result.Total,
		"difficulty":         difficulty,
		"success":            result.MeetsDifficulty,
		"crit":               result.IsCrit,
	}
	if len(modifierList) > 0 {
		results["modifiers"] = modifierList
	}

	systemData := map[string]any{
		"character_id": characterID,
		"trait":        trait,
		"roll_kind":    rollKind.String(),
		"outcome":      outcomeCode,
		"flavor":       flavor,
		"crit":         result.IsCrit,
		"hope_fear":    generateHopeFear,
		"gm_move":      triggerGMMove,
		"crit_negates": critNegatesEffects,
	}
	if len(modifierList) > 0 {
		systemData["modifiers"] = modifierList
	}

	payload := event.RollResolvedPayload{
		RequestID:  grpcmeta.RequestIDFromContext(ctx),
		RollSeq:    rollSeq,
		Results:    results,
		Outcome:    outcomeCode,
		SystemData: systemData,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          event.TypeRollResolved,
		SessionID:     sessionID,
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "roll",
		EntityID:      grpcmeta.RequestIDFromContext(ctx),
		SystemID:      c.System.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	return &pb.SessionActionRollResponse{
		RollSeq:    stored.Seq,
		HopeDie:    int32(result.Hope),
		FearDie:    int32(result.Fear),
		Total:      int32(result.Total),
		Difficulty: int32(difficulty),
		Success:    result.MeetsDifficulty,
		Flavor:     flavor,
		Crit:       result.IsCrit,
		Rng: &commonv1.RngResponse{
			SeedUsed:   uint64(seed),
			RngAlgo:    random.RngAlgoMathRandV1,
			SeedSource: seedSource,
			RollMode:   rollMode,
		},
	}, nil
}

func (s *DaggerheartService) SessionDamageRoll(ctx context.Context, in *pb.SessionDamageRollRequest) (*pb.SessionDamageRollResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session damage roll request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
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

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}
	if len(in.GetDice()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "dice are required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart rolls")
	}

	sess, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.SessionStatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}

	diceSpecs, err := damageDiceFromProto(in.GetDice())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	latestSeq, err := s.stores.Event.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load latest event seq: %v", err)
	}
	rollSeq := latestSeq + 1

	seed, seedSource, rollMode, err := random.ResolveSeed(
		in.GetRng(),
		s.seedFunc,
		func(mode commonv1.RollMode) bool { return mode == commonv1.RollMode_REPLAY },
	)
	if err != nil {
		if errors.Is(err, random.ErrSeedOutOfRange()) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to resolve seed: %v", err)
	}

	result, err := daggerheart.RollDamage(daggerheart.DamageRollRequest{
		Dice:     diceSpecs,
		Modifier: int(in.GetModifier()),
		Seed:     seed,
		Critical: in.GetCritical(),
	})
	if err != nil {
		if errors.Is(err, dice.ErrMissingDice) || errors.Is(err, dice.ErrInvalidDiceSpec) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to roll damage: %v", err)
	}

	rollModeLabel := rollMode.String()
	payload := daggerheart.DamageRollResolvedPayload{
		CharacterID:   characterID,
		RollSeq:       rollSeq,
		Rolls:         result.Rolls,
		BaseTotal:     result.BaseTotal,
		Modifier:      result.Modifier,
		CriticalBonus: result.CriticalBonus,
		Total:         result.Total,
		Critical:      in.GetCritical(),
		Rng: daggerheart.RollRngInfo{
			SeedUsed:   uint64(seed),
			RngAlgo:    random.RngAlgoMathRandV1,
			SeedSource: seedSource,
			RollMode:   rollModeLabel,
		},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          daggerheart.EventTypeDamageRollResolved,
		SessionID:     sessionID,
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "roll",
		EntityID:      grpcmeta.RequestIDFromContext(ctx),
		SystemID:      c.System.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	response := &pb.SessionDamageRollResponse{
		RollSeq:       stored.Seq,
		Rolls:         diceRollsToProto(result.Rolls),
		BaseTotal:     int32(result.BaseTotal),
		Modifier:      int32(result.Modifier),
		CriticalBonus: int32(result.CriticalBonus),
		Total:         int32(result.Total),
		Critical:      in.GetCritical(),
		Rng: &commonv1.RngResponse{
			SeedUsed:   uint64(seed),
			RngAlgo:    random.RngAlgoMathRandV1,
			SeedSource: seedSource,
			RollMode:   rollMode,
		},
	}

	return response, nil
}

func (s *DaggerheartService) SessionAttackFlow(ctx context.Context, in *pb.SessionAttackFlowRequest) (*pb.SessionAttackFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session attack flow request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
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

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	attackerID := strings.TrimSpace(in.GetCharacterId())
	if attackerID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}
	trait := strings.TrimSpace(in.GetTrait())
	if trait == "" {
		return nil, status.Error(codes.InvalidArgument, "trait is required")
	}
	targetID := strings.TrimSpace(in.GetTargetId())
	if targetID == "" {
		return nil, status.Error(codes.InvalidArgument, "target id is required")
	}
	if in.GetDamage() == nil {
		return nil, status.Error(codes.InvalidArgument, "damage is required")
	}
	if in.GetDamage().GetDamageType() == pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "damage_type is required")
	}

	rollResp, err := s.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		CharacterId: attackerID,
		Trait:       trait,
		RollKind:    pb.RollKind_ROLL_KIND_ACTION,
		Difficulty:  in.GetDifficulty(),
		Modifiers:   in.GetModifiers(),
		Rng:         in.GetActionRng(),
	})
	if err != nil {
		return nil, err
	}

	ctxWithMeta := withCampaignSessionMetadata(ctx, campaignID, sessionID)
	rollOutcome, err := s.ApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		return nil, err
	}

	attackOutcome, err := s.ApplyAttackOutcome(ctxWithMeta, &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: sessionID,
		RollSeq:   rollResp.GetRollSeq(),
		Targets:   []string{targetID},
	})
	if err != nil {
		return nil, err
	}

	response := &pb.SessionAttackFlowResponse{
		ActionRoll:    rollResp,
		RollOutcome:   rollOutcome,
		AttackOutcome: attackOutcome,
	}

	if attackOutcome.GetResult() == nil || !attackOutcome.GetResult().GetSuccess() {
		return response, nil
	}

	if len(in.GetDamageDice()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "damage_dice are required")
	}

	critical := attackOutcome.GetResult().GetCrit() || in.GetDamageCritical()
	damageRoll, err := s.SessionDamageRoll(ctx, &pb.SessionDamageRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		CharacterId: attackerID,
		Dice:        in.GetDamageDice(),
		Modifier:    in.GetDamageModifier(),
		Critical:    critical,
		Rng:         in.GetDamageRng(),
	})
	if err != nil {
		return nil, err
	}

	damageReq := &pb.DaggerheartDamageRequest{
		Amount:             damageRoll.GetTotal(),
		DamageType:         in.GetDamage().GetDamageType(),
		ResistPhysical:     in.GetDamage().GetResistPhysical(),
		ResistMagic:        in.GetDamage().GetResistMagic(),
		ImmunePhysical:     in.GetDamage().GetImmunePhysical(),
		ImmuneMagic:        in.GetDamage().GetImmuneMagic(),
		Direct:             in.GetDamage().GetDirect(),
		MassiveDamage:      in.GetDamage().GetMassiveDamage(),
		Source:             in.GetDamage().GetSource(),
		SourceCharacterIds: normalizeTargets(in.GetDamage().GetSourceCharacterIds()),
	}

	applyDamage, err := s.ApplyDamage(ctxWithMeta, &pb.DaggerheartApplyDamageRequest{
		CampaignId:        campaignID,
		CharacterId:       targetID,
		Damage:            damageReq,
		RollSeq:           &damageRoll.RollSeq,
		RequireDamageRoll: in.GetRequireDamageRoll(),
	})
	if err != nil {
		return nil, err
	}

	response.DamageRoll = damageRoll
	response.DamageApplied = applyDamage
	return response, nil
}

func (s *DaggerheartService) SessionReactionFlow(ctx context.Context, in *pb.SessionReactionFlowRequest) (*pb.SessionReactionFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session reaction flow request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
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

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	actorID := strings.TrimSpace(in.GetCharacterId())
	if actorID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}
	trait := strings.TrimSpace(in.GetTrait())
	if trait == "" {
		return nil, status.Error(codes.InvalidArgument, "trait is required")
	}

	rollResp, err := s.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		CharacterId: actorID,
		Trait:       trait,
		RollKind:    pb.RollKind_ROLL_KIND_REACTION,
		Difficulty:  in.GetDifficulty(),
		Modifiers:   in.GetModifiers(),
		Rng:         in.GetReactionRng(),
	})
	if err != nil {
		return nil, err
	}

	ctxWithMeta := withCampaignSessionMetadata(ctx, campaignID, sessionID)
	rollOutcome, err := s.ApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		return nil, err
	}

	reactionOutcome, err := s.ApplyReactionOutcome(ctxWithMeta, &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: sessionID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		return nil, err
	}

	response := &pb.SessionReactionFlowResponse{
		ActionRoll:      rollResp,
		RollOutcome:     rollOutcome,
		ReactionOutcome: reactionOutcome,
	}

	return response, nil
}

func (s *DaggerheartService) SessionAdversaryAttackRoll(ctx context.Context, in *pb.SessionAdversaryAttackRollRequest) (*pb.SessionAdversaryAttackRollResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session adversary attack roll request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
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

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	adversaryID := strings.TrimSpace(in.GetAdversaryId())
	if adversaryID == "" {
		return nil, status.Error(codes.InvalidArgument, "adversary id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart adversary rolls")
	}

	sess, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.SessionStatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}

	if _, err := s.loadAdversaryForSession(ctx, campaignID, sessionID, adversaryID); err != nil {
		return nil, err
	}

	seed, seedSource, rollMode, err := random.ResolveSeed(
		in.GetRng(),
		s.seedFunc,
		func(mode commonv1.RollMode) bool { return mode == commonv1.RollMode_REPLAY },
	)
	if err != nil {
		if errors.Is(err, random.ErrSeedOutOfRange()) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to resolve seed: %v", err)
	}

	advantage := int(in.GetAdvantage())
	disadvantage := int(in.GetDisadvantage())
	if advantage > 0 && disadvantage > 0 {
		advantage = 0
		disadvantage = 0
	}
	rollCount := 1
	if advantage > 0 || disadvantage > 0 {
		rollCount = 2
	}

	result, err := dice.RollDice(dice.Request{
		Dice: []dice.Spec{{Sides: 20, Count: rollCount}},
		Seed: seed,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to roll adversary die: %v", err)
	}
	rolls := result.Rolls[0].Results
	selected := rolls[0]
	if rollCount == 2 {
		if advantage > 0 {
			if rolls[1] > selected {
				selected = rolls[1]
			}
		} else if disadvantage > 0 {
			if rolls[1] < selected {
				selected = rolls[1]
			}
		}
	}
	modifier := int(in.GetAttackModifier())
	total := selected + modifier

	latestSeq, err := s.stores.Event.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load latest event seq: %v", err)
	}
	rollSeq := latestSeq + 1

	payload := daggerheart.AdversaryRollResolvedPayload{
		AdversaryID:  adversaryID,
		RollSeq:      rollSeq,
		Rolls:        rolls,
		Roll:         selected,
		Modifier:     modifier,
		Total:        total,
		Advantage:    advantage,
		Disadvantage: disadvantage,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode adversary roll payload: %v", err)
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          daggerheart.EventTypeAdversaryRollResolved,
		SessionID:     sessionID,
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "adversary",
		EntityID:      adversaryID,
		SystemID:      c.System.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append adversary roll event: %v", err)
	}

	rollValues := make([]int32, 0, len(rolls))
	for _, roll := range rolls {
		rollValues = append(rollValues, int32(roll))
	}

	return &pb.SessionAdversaryAttackRollResponse{
		RollSeq: stored.Seq,
		Roll:    int32(selected),
		Total:   int32(total),
		Rolls:   rollValues,
		Rng: &commonv1.RngResponse{
			SeedUsed:   uint64(seed),
			RngAlgo:    random.RngAlgoMathRandV1,
			SeedSource: seedSource,
			RollMode:   rollMode,
		},
	}, nil
}

func (s *DaggerheartService) SessionAdversaryAttackFlow(ctx context.Context, in *pb.SessionAdversaryAttackFlowRequest) (*pb.SessionAdversaryAttackFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session adversary attack flow request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
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

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	adversaryID := strings.TrimSpace(in.GetAdversaryId())
	if adversaryID == "" {
		return nil, status.Error(codes.InvalidArgument, "adversary id is required")
	}
	targetID := strings.TrimSpace(in.GetTargetId())
	if targetID == "" {
		return nil, status.Error(codes.InvalidArgument, "target id is required")
	}
	if in.GetDifficulty() < 0 {
		return nil, status.Error(codes.InvalidArgument, "difficulty must be non-negative")
	}
	if in.GetDamage() == nil {
		return nil, status.Error(codes.InvalidArgument, "damage is required")
	}
	if in.GetDamage().GetDamageType() == pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "damage_type is required")
	}

	rollResp, err := s.SessionAdversaryAttackRoll(ctx, &pb.SessionAdversaryAttackRollRequest{
		CampaignId:     campaignID,
		SessionId:      sessionID,
		AdversaryId:    adversaryID,
		AttackModifier: in.GetAttackModifier(),
		Advantage:      in.GetAdvantage(),
		Disadvantage:   in.GetDisadvantage(),
		Rng:            in.GetAttackRng(),
	})
	if err != nil {
		return nil, err
	}

	ctxWithMeta := withCampaignSessionMetadata(ctx, campaignID, sessionID)
	attackOutcome, err := s.ApplyAdversaryAttackOutcome(ctxWithMeta, &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId:  sessionID,
		RollSeq:    rollResp.GetRollSeq(),
		Targets:    []string{targetID},
		Difficulty: in.GetDifficulty(),
	})
	if err != nil {
		return nil, err
	}

	response := &pb.SessionAdversaryAttackFlowResponse{
		AttackRoll:    rollResp,
		AttackOutcome: attackOutcome,
	}

	if attackOutcome.GetResult() == nil || !attackOutcome.GetResult().GetSuccess() {
		return response, nil
	}

	if len(in.GetDamageDice()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "damage_dice are required")
	}

	critical := attackOutcome.GetResult().GetCrit() || in.GetDamageCritical()
	damageRoll, err := s.SessionDamageRoll(ctx, &pb.SessionDamageRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		CharacterId: adversaryID,
		Dice:        in.GetDamageDice(),
		Modifier:    in.GetDamageModifier(),
		Critical:    critical,
		Rng:         in.GetDamageRng(),
	})
	if err != nil {
		return nil, err
	}

	sourceCharacterIDs := normalizeTargets(in.GetDamage().GetSourceCharacterIds())
	sourceCharacterIDs = append(sourceCharacterIDs, adversaryID)
	sourceCharacterIDs = normalizeTargets(sourceCharacterIDs)

	damageReq := &pb.DaggerheartDamageRequest{
		Amount:             damageRoll.GetTotal(),
		DamageType:         in.GetDamage().GetDamageType(),
		ResistPhysical:     in.GetDamage().GetResistPhysical(),
		ResistMagic:        in.GetDamage().GetResistMagic(),
		ImmunePhysical:     in.GetDamage().GetImmunePhysical(),
		ImmuneMagic:        in.GetDamage().GetImmuneMagic(),
		Direct:             in.GetDamage().GetDirect(),
		MassiveDamage:      in.GetDamage().GetMassiveDamage(),
		Source:             in.GetDamage().GetSource(),
		SourceCharacterIds: sourceCharacterIDs,
	}

	applyDamage, err := s.ApplyDamage(ctxWithMeta, &pb.DaggerheartApplyDamageRequest{
		CampaignId:        campaignID,
		CharacterId:       targetID,
		Damage:            damageReq,
		RollSeq:           &damageRoll.RollSeq,
		RequireDamageRoll: in.GetRequireDamageRoll(),
	})
	if err != nil {
		return nil, err
	}

	response.DamageRoll = damageRoll
	response.DamageApplied = applyDamage
	return response, nil
}

func (s *DaggerheartService) SessionGroupActionFlow(ctx context.Context, in *pb.SessionGroupActionFlowRequest) (*pb.SessionGroupActionFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session group action flow request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
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

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	leaderID := strings.TrimSpace(in.GetLeaderCharacterId())
	if leaderID == "" {
		return nil, status.Error(codes.InvalidArgument, "leader character id is required")
	}
	leaderTrait := strings.TrimSpace(in.GetLeaderTrait())
	if leaderTrait == "" {
		return nil, status.Error(codes.InvalidArgument, "leader trait is required")
	}
	if in.GetDifficulty() == 0 {
		return nil, status.Error(codes.InvalidArgument, "difficulty is required")
	}
	supporters := in.GetSupporters()
	if len(supporters) == 0 {
		return nil, status.Error(codes.InvalidArgument, "supporters are required")
	}

	supportRolls := make([]*pb.GroupActionSupporterRoll, 0, len(supporters))
	supportSuccesses := 0
	supportFailures := 0
	for _, supporter := range supporters {
		if supporter == nil {
			return nil, status.Error(codes.InvalidArgument, "supporter is required")
		}
		supporterID := strings.TrimSpace(supporter.GetCharacterId())
		if supporterID == "" {
			return nil, status.Error(codes.InvalidArgument, "supporter character id is required")
		}
		supporterTrait := strings.TrimSpace(supporter.GetTrait())
		if supporterTrait == "" {
			return nil, status.Error(codes.InvalidArgument, "supporter trait is required")
		}

		rollResp, err := s.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
			CampaignId:  campaignID,
			SessionId:   sessionID,
			CharacterId: supporterID,
			Trait:       supporterTrait,
			RollKind:    pb.RollKind_ROLL_KIND_REACTION,
			Difficulty:  in.GetDifficulty(),
			Modifiers:   supporter.GetModifiers(),
			Rng:         supporter.GetRng(),
		})
		if err != nil {
			return nil, err
		}
		if rollResp.GetSuccess() {
			supportSuccesses++
		} else {
			supportFailures++
		}

		supportRolls = append(supportRolls, &pb.GroupActionSupporterRoll{
			CharacterId: supporterID,
			ActionRoll:  rollResp,
			Success:     rollResp.GetSuccess(),
		})
	}

	supportModifier := supportSuccesses - supportFailures
	leaderModifiers := append([]*pb.ActionRollModifier{}, in.GetLeaderModifiers()...)
	if supportModifier != 0 {
		leaderModifiers = append(leaderModifiers, &pb.ActionRollModifier{
			Value:  int32(supportModifier),
			Source: "group_action_support",
		})
	}

	leaderRoll, err := s.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		CharacterId: leaderID,
		Trait:       leaderTrait,
		RollKind:    pb.RollKind_ROLL_KIND_ACTION,
		Difficulty:  in.GetDifficulty(),
		Modifiers:   leaderModifiers,
		Rng:         in.GetLeaderRng(),
	})
	if err != nil {
		return nil, err
	}

	ctxWithMeta := withCampaignSessionMetadata(ctx, campaignID, sessionID)
	leaderOutcome, err := s.ApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		RollSeq:   leaderRoll.GetRollSeq(),
	})
	if err != nil {
		return nil, err
	}

	payload := daggerheart.GroupActionResolvedPayload{
		LeaderCharacterID: leaderID,
		LeaderRollSeq:     leaderRoll.GetRollSeq(),
		SupportSuccesses:  supportSuccesses,
		SupportFailures:   supportFailures,
		SupportModifier:   supportModifier,
		Supporters:        make([]daggerheart.GroupActionSupporterRoll, 0, len(supportRolls)),
	}
	for _, roll := range supportRolls {
		if roll == nil || roll.ActionRoll == nil {
			continue
		}
		payload.Supporters = append(payload.Supporters, daggerheart.GroupActionSupporterRoll{
			CharacterID: roll.GetCharacterId(),
			RollSeq:     roll.GetActionRoll().GetRollSeq(),
			Success:     roll.GetSuccess(),
		})
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode group action payload: %v", err)
	}

	if _, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          daggerheart.EventTypeGroupActionResolved,
		SessionID:     sessionID,
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "group_action",
		EntityID:      leaderID,
		SystemID:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "append group action event: %v", err)
	}

	return &pb.SessionGroupActionFlowResponse{
		LeaderRoll:       leaderRoll,
		LeaderOutcome:    leaderOutcome,
		SupporterRolls:   supportRolls,
		SupportModifier:  int32(supportModifier),
		SupportSuccesses: int32(supportSuccesses),
		SupportFailures:  int32(supportFailures),
	}, nil
}

func (s *DaggerheartService) SessionTagTeamFlow(ctx context.Context, in *pb.SessionTagTeamFlowRequest) (*pb.SessionTagTeamFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session tag team flow request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
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

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if in.GetDifficulty() == 0 {
		return nil, status.Error(codes.InvalidArgument, "difficulty is required")
	}
	first := in.GetFirst()
	if first == nil {
		return nil, status.Error(codes.InvalidArgument, "first participant is required")
	}
	second := in.GetSecond()
	if second == nil {
		return nil, status.Error(codes.InvalidArgument, "second participant is required")
	}
	firstID := strings.TrimSpace(first.GetCharacterId())
	if firstID == "" {
		return nil, status.Error(codes.InvalidArgument, "first character id is required")
	}
	secondID := strings.TrimSpace(second.GetCharacterId())
	if secondID == "" {
		return nil, status.Error(codes.InvalidArgument, "second character id is required")
	}
	if firstID == secondID {
		return nil, status.Error(codes.InvalidArgument, "tag team participants must be distinct")
	}
	firstTrait := strings.TrimSpace(first.GetTrait())
	if firstTrait == "" {
		return nil, status.Error(codes.InvalidArgument, "first trait is required")
	}
	secondTrait := strings.TrimSpace(second.GetTrait())
	if secondTrait == "" {
		return nil, status.Error(codes.InvalidArgument, "second trait is required")
	}
	selectedID := strings.TrimSpace(in.GetSelectedCharacterId())
	if selectedID == "" {
		return nil, status.Error(codes.InvalidArgument, "selected character id is required")
	}
	if selectedID != firstID && selectedID != secondID {
		return nil, status.Error(codes.InvalidArgument, "selected character id must match a participant")
	}

	firstRoll, err := s.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		CharacterId: firstID,
		Trait:       firstTrait,
		RollKind:    pb.RollKind_ROLL_KIND_ACTION,
		Difficulty:  in.GetDifficulty(),
		Modifiers:   first.GetModifiers(),
		Rng:         first.GetRng(),
	})
	if err != nil {
		return nil, err
	}

	secondRoll, err := s.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		CharacterId: secondID,
		Trait:       secondTrait,
		RollKind:    pb.RollKind_ROLL_KIND_ACTION,
		Difficulty:  in.GetDifficulty(),
		Modifiers:   second.GetModifiers(),
		Rng:         second.GetRng(),
	})
	if err != nil {
		return nil, err
	}

	selectedRoll := firstRoll
	if selectedID == secondID {
		selectedRoll = secondRoll
	}

	ctxWithMeta := withCampaignSessionMetadata(ctx, campaignID, sessionID)
	selectedOutcome, err := s.ApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		RollSeq:   selectedRoll.GetRollSeq(),
	})
	if err != nil {
		return nil, err
	}

	payload := daggerheart.TagTeamResolvedPayload{
		FirstCharacterID:    firstID,
		FirstRollSeq:        firstRoll.GetRollSeq(),
		SecondCharacterID:   secondID,
		SecondRollSeq:       secondRoll.GetRollSeq(),
		SelectedCharacterID: selectedID,
		SelectedRollSeq:     selectedRoll.GetRollSeq(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode tag team payload: %v", err)
	}

	if _, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          daggerheart.EventTypeTagTeamResolved,
		SessionID:     sessionID,
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "tag_team",
		EntityID:      selectedID,
		SystemID:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "append tag team event: %v", err)
	}

	return &pb.SessionTagTeamFlowResponse{
		FirstRoll:           firstRoll,
		SecondRoll:          secondRoll,
		SelectedOutcome:     selectedOutcome,
		SelectedCharacterId: selectedID,
		SelectedRollSeq:     selectedRoll.GetRollSeq(),
	}, nil
}

func (s *DaggerheartService) ApplyRollOutcome(ctx context.Context, in *pb.ApplyRollOutcomeRequest) (*pb.ApplyRollOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply roll outcome request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(grpcmeta.CampaignIDFromContext(ctx))
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		sessionID = strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	}
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if in.GetRollSeq() == 0 {
		return nil, status.Error(codes.InvalidArgument, "roll seq is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart outcomes")
	}

	sess, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.SessionStatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}

	rollEvent, err := s.stores.Event.GetEventBySeq(ctx, campaignID, in.GetRollSeq())
	if err != nil {
		return nil, handleDomainError(err)
	}
	if rollEvent.Type != event.TypeRollResolved {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not reference action.roll_resolved")
	}
	if rollEvent.SessionID != sessionID {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not match session")
	}

	var rollPayload event.RollResolvedPayload
	if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
		return nil, status.Errorf(codes.Internal, "decode roll payload: %v", err)
	}

	rollRequestID := strings.TrimSpace(rollPayload.RequestID)
	if rollRequestID == "" {
		rollRequestID = strings.TrimSpace(rollEvent.RequestID)
	}
	if rollRequestID == "" {
		return nil, status.Error(codes.InvalidArgument, "roll request id is required")
	}

	existing, err := s.stores.Event.ListEventsPage(ctx, storage.ListEventsPageRequest{
		CampaignID:   campaignID,
		PageSize:     1,
		Descending:   true,
		FilterClause: "session_id = ? AND request_id = ? AND event_type = ?",
		FilterParams: []any{sessionID, rollRequestID, string(event.TypeOutcomeApplied)},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "check outcome applied: %v", err)
	}
	if len(existing.Events) > 0 {
		return nil, status.Error(codes.FailedPrecondition, "outcome already applied")
	}

	rollSystemData := rollPayload.SystemData
	rollKind := rollKindFromSystemData(rollSystemData)
	generateHopeFear := boolFromSystemData(rollSystemData, "hope_fear", rollKind != pb.RollKind_ROLL_KIND_REACTION)
	triggerGMMove := boolFromSystemData(rollSystemData, "gm_move", rollKind != pb.RollKind_ROLL_KIND_REACTION)
	rollOutcome := outcomeFromSystemData(rollSystemData, rollPayload.Outcome)
	if rollOutcome == "" {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is required")
	}
	flavor := outcomeFlavorFromCode(rollOutcome)
	if flavor == "" {
		return nil, status.Error(codes.InvalidArgument, "roll outcome flavor is required")
	}
	if !generateHopeFear {
		flavor = ""
	}
	crit := critFromSystemData(rollSystemData, rollOutcome)

	targets := normalizeTargets(in.GetTargets())
	if len(targets) == 0 {
		rollerID := stringFromSystemData(rollSystemData, "character_id")
		if strings.TrimSpace(rollerID) == "" {
			return nil, status.Error(codes.InvalidArgument, "targets are required")
		}
		targets = []string{rollerID}
	}

	gmFearDelta := 0
	if triggerGMMove && flavor == "FEAR" && !crit {
		gmFearDelta = len(targets)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	changes := make([]event.OutcomeAppliedChange, 0)
	updatedStates := make([]*pb.OutcomeCharacterState, 0, len(targets))

	if gmFearDelta > 0 {
		currentSnap, err := s.stores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return nil, status.Errorf(codes.Internal, "load gm fear: %v", err)
		}
		beforeFear := currentSnap.GMFear
		_, before, after, err := snapshot.ApplyGmFearGain(snapshot.GmFear{CampaignID: campaignID, Value: beforeFear}, gmFearDelta)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "gm fear update invalid: %v", err)
		}

		gmPayload := daggerheart.GMFearChangedPayload{Before: before, After: after}
		gmPayloadJSON, err := json.Marshal(gmPayload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "encode gm fear payload: %v", err)
		}
		storedGM, err := s.stores.Event.AppendEvent(ctx, event.Event{
			CampaignID:    campaignID,
			Timestamp:     time.Now().UTC(),
			Type:          daggerheart.EventTypeGMFearChanged,
			SessionID:     sessionID,
			RequestID:     rollRequestID,
			InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
			ActorType:     event.ActorTypeSystem,
			EntityType:    "campaign",
			EntityID:      campaignID,
			SystemID:      c.System.String(),
			SystemVersion: daggerheart.SystemVersion,
			PayloadJSON:   gmPayloadJSON,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "append gm fear event: %v", err)
		}
		if err := adapter.ApplyEvent(ctx, storedGM); err != nil {
			return nil, status.Errorf(codes.Internal, "apply gm fear event: %v", err)
		}

		changes = append(changes, event.OutcomeAppliedChange{Field: string(session.OutcomeFieldGMFear), Before: before, After: after})
	}

	for _, target := range targets {
		profile, err := s.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, target)
		if err != nil {
			return nil, handleDomainError(err)
		}
		state, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, target)
		if err != nil {
			return nil, handleDomainError(err)
		}

		hopeBefore := state.Hope
		stressBefore := state.Stress
		hopeMax := state.HopeMax
		if hopeMax == 0 {
			hopeMax = daggerheart.HopeMax
		}
		hopeAfter := hopeBefore
		stressAfter := stressBefore
		if generateHopeFear && flavor == "HOPE" {
			hopeAfter = clamp(hopeBefore+1, daggerheart.HopeMin, hopeMax)
		}
		if generateHopeFear && crit {
			stressAfter = clamp(stressBefore-1, daggerheart.StressMin, profile.StressMax)
		}

		if hopeAfter != hopeBefore || stressAfter != stressBefore {
			payload := daggerheart.CharacterStatePatchedPayload{
				CharacterID:  target,
				HopeBefore:   &hopeBefore,
				HopeAfter:    &hopeAfter,
				StressBefore: &stressBefore,
				StressAfter:  &stressAfter,
			}
			payloadJSON, err := json.Marshal(payload)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "encode character state payload: %v", err)
			}
			storedState, err := s.stores.Event.AppendEvent(ctx, event.Event{
				CampaignID:    campaignID,
				Timestamp:     time.Now().UTC(),
				Type:          daggerheart.EventTypeCharacterStatePatched,
				SessionID:     sessionID,
				RequestID:     rollRequestID,
				InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
				ActorType:     event.ActorTypeSystem,
				EntityType:    "character",
				EntityID:      target,
				SystemID:      c.System.String(),
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "append character state event: %v", err)
			}
			if err := adapter.ApplyEvent(ctx, storedState); err != nil {
				return nil, status.Errorf(codes.Internal, "apply character state event: %v", err)
			}
		}

		if hopeAfter != hopeBefore {
			changes = append(changes, event.OutcomeAppliedChange{CharacterID: target, Field: string(session.OutcomeFieldHope), Before: hopeBefore, After: hopeAfter})
		}
		if stressAfter != stressBefore {
			changes = append(changes, event.OutcomeAppliedChange{CharacterID: target, Field: string(session.OutcomeFieldStress), Before: stressBefore, After: stressAfter})
		}
		updatedStates = append(updatedStates, &pb.OutcomeCharacterState{
			CharacterId: target,
			Hope:        int32(hopeAfter),
			Stress:      int32(stressAfter),
			Hp:          int32(state.Hp),
		})
	}

	requiresComplication := flavor == "FEAR" && !crit && triggerGMMove
	payload := event.OutcomeAppliedPayload{
		RequestID:            rollRequestID,
		RollSeq:              in.GetRollSeq(),
		Targets:              targets,
		RequiresComplication: requiresComplication,
		AppliedChanges:       changes,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode outcome payload: %v", err)
	}

	if _, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    time.Now().UTC(),
		Type:         event.TypeOutcomeApplied,
		SessionID:    sessionID,
		RequestID:    rollRequestID,
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    event.ActorTypeSystem,
		EntityType:   "outcome",
		EntityID:     rollRequestID,
		PayloadJSON:  payloadJSON,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "append outcome applied event: %v", err)
	}

	response := &pb.ApplyRollOutcomeResponse{
		RollSeq:              in.GetRollSeq(),
		RequiresComplication: requiresComplication,
		Updated: &pb.OutcomeUpdated{
			CharacterStates: updatedStates,
		},
	}
	if gmFearDelta > 0 {
		currentSnap, err := s.stores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "load gm fear snapshot: %v", err)
		}
		value := int32(currentSnap.GMFear)
		response.Updated.GmFear = &value
	}

	return response, nil
}

func (s *DaggerheartService) ApplyAttackOutcome(ctx context.Context, in *pb.DaggerheartApplyAttackOutcomeRequest) (*pb.DaggerheartApplyAttackOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply attack outcome request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(grpcmeta.CampaignIDFromContext(ctx))
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		sessionID = strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	}
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if in.GetRollSeq() == 0 {
		return nil, status.Error(codes.InvalidArgument, "roll seq is required")
	}
	if len(in.GetTargets()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "targets are required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart attack outcomes")
	}

	sess, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.SessionStatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}

	rollEvent, err := s.stores.Event.GetEventBySeq(ctx, campaignID, in.GetRollSeq())
	if err != nil {
		return nil, handleDomainError(err)
	}
	if rollEvent.Type != event.TypeRollResolved {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not reference action.roll_resolved")
	}
	if rollEvent.SessionID != sessionID {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not match session")
	}

	var rollPayload event.RollResolvedPayload
	if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
		return nil, status.Errorf(codes.Internal, "decode roll payload: %v", err)
	}

	rollRequestID := strings.TrimSpace(rollPayload.RequestID)
	if rollRequestID == "" {
		rollRequestID = strings.TrimSpace(rollEvent.RequestID)
	}
	if rollRequestID == "" {
		return nil, status.Error(codes.InvalidArgument, "roll request id is required")
	}

	rollKind := rollKindFromSystemData(rollPayload.SystemData)
	if rollKind == pb.RollKind_ROLL_KIND_REACTION {
		return nil, status.Error(codes.FailedPrecondition, "roll seq references a reaction roll")
	}
	rollOutcome := outcomeFromSystemData(rollPayload.SystemData, rollPayload.Outcome)
	if rollOutcome == "" {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is required")
	}
	crit := critFromSystemData(rollPayload.SystemData, rollOutcome)
	flavor := outcomeFlavorFromCode(rollOutcome)
	if !boolFromSystemData(rollPayload.SystemData, "hope_fear", true) {
		flavor = ""
	}
	rollSuccess, ok := outcomeSuccessFromCode(rollOutcome)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is invalid")
	}
	attackerID := stringFromSystemData(rollPayload.SystemData, "character_id")
	if attackerID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}
	targets := normalizeTargets(in.GetTargets())
	if len(targets) == 0 {
		return nil, status.Error(codes.InvalidArgument, "targets are required")
	}

	existing, err := s.stores.Event.ListEventsPage(ctx, storage.ListEventsPageRequest{
		CampaignID:   campaignID,
		PageSize:     1,
		Descending:   true,
		FilterClause: "session_id = ? AND request_id = ? AND event_type = ?",
		FilterParams: []any{sessionID, rollRequestID, string(daggerheart.EventTypeAttackResolved)},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "check attack outcome applied: %v", err)
	}
	if len(existing.Events) > 0 {
		return nil, status.Error(codes.FailedPrecondition, "attack outcome already applied")
	}

	payload := daggerheart.AttackResolvedPayload{
		CharacterID: attackerID,
		RollSeq:     in.GetRollSeq(),
		Targets:     targets,
		Outcome:     rollOutcome,
		Success:     rollSuccess,
		Crit:        crit,
		Flavor:      flavor,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode attack payload: %v", err)
	}

	if _, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          daggerheart.EventTypeAttackResolved,
		SessionID:     sessionID,
		RequestID:     rollRequestID,
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "attack",
		EntityID:      rollRequestID,
		SystemID:      c.System.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "append attack event: %v", err)
	}

	return &pb.DaggerheartApplyAttackOutcomeResponse{
		RollSeq:     in.GetRollSeq(),
		CharacterId: attackerID,
		Targets:     targets,
		Result: &pb.DaggerheartAttackOutcomeResult{
			Outcome: outcomeCodeToProto(rollOutcome),
			Success: rollSuccess,
			Crit:    crit,
			Flavor:  flavor,
		},
	}, nil
}

func (s *DaggerheartService) ApplyAdversaryAttackOutcome(ctx context.Context, in *pb.DaggerheartApplyAdversaryAttackOutcomeRequest) (*pb.DaggerheartApplyAdversaryAttackOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply adversary attack outcome request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(grpcmeta.CampaignIDFromContext(ctx))
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		sessionID = strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	}
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if in.GetRollSeq() == 0 {
		return nil, status.Error(codes.InvalidArgument, "roll seq is required")
	}
	if len(in.GetTargets()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "targets are required")
	}
	if in.GetDifficulty() < 0 {
		return nil, status.Error(codes.InvalidArgument, "difficulty must be non-negative")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart adversary attack outcomes")
	}

	sess, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.SessionStatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}

	rollEvent, err := s.stores.Event.GetEventBySeq(ctx, campaignID, in.GetRollSeq())
	if err != nil {
		return nil, handleDomainError(err)
	}
	if rollEvent.Type != daggerheart.EventTypeAdversaryRollResolved {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not reference action.adversary_roll_resolved")
	}
	if rollEvent.SessionID != sessionID {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not match session")
	}

	var rollPayload daggerheart.AdversaryRollResolvedPayload
	if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
		return nil, status.Errorf(codes.Internal, "decode adversary roll payload: %v", err)
	}
	adversaryID := strings.TrimSpace(rollPayload.AdversaryID)
	if adversaryID == "" {
		return nil, status.Error(codes.InvalidArgument, "adversary id is required")
	}

	rollRequestID := strings.TrimSpace(rollEvent.RequestID)
	if rollRequestID == "" {
		return nil, status.Error(codes.InvalidArgument, "roll request id is required")
	}

	targets := normalizeTargets(in.GetTargets())
	if len(targets) == 0 {
		return nil, status.Error(codes.InvalidArgument, "targets are required")
	}

	existing, err := s.stores.Event.ListEventsPage(ctx, storage.ListEventsPageRequest{
		CampaignID:   campaignID,
		PageSize:     1,
		Descending:   true,
		FilterClause: "session_id = ? AND request_id = ? AND event_type = ?",
		FilterParams: []any{sessionID, rollRequestID, string(daggerheart.EventTypeAdversaryAttackResolved)},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "check adversary attack outcome applied: %v", err)
	}
	if len(existing.Events) > 0 {
		return nil, status.Error(codes.FailedPrecondition, "adversary attack outcome already applied")
	}

	roll := rollPayload.Roll
	modifier := rollPayload.Modifier
	total := rollPayload.Total
	difficulty := int(in.GetDifficulty())
	success := total >= difficulty
	crit := roll == 20

	payload := daggerheart.AdversaryAttackResolvedPayload{
		AdversaryID: adversaryID,
		RollSeq:     in.GetRollSeq(),
		Targets:     targets,
		Roll:        roll,
		Modifier:    modifier,
		Total:       total,
		Difficulty:  difficulty,
		Success:     success,
		Crit:        crit,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode adversary attack payload: %v", err)
	}

	if _, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          daggerheart.EventTypeAdversaryAttackResolved,
		SessionID:     sessionID,
		RequestID:     rollRequestID,
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "attack",
		EntityID:      rollRequestID,
		SystemID:      c.System.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "append adversary attack event: %v", err)
	}

	return &pb.DaggerheartApplyAdversaryAttackOutcomeResponse{
		RollSeq:     in.GetRollSeq(),
		AdversaryId: adversaryID,
		Targets:     targets,
		Result: &pb.DaggerheartAdversaryAttackOutcomeResult{
			Success:    success,
			Crit:       crit,
			Roll:       int32(roll),
			Total:      int32(total),
			Difficulty: int32(difficulty),
		},
	}, nil
}

func (s *DaggerheartService) ApplyReactionOutcome(ctx context.Context, in *pb.DaggerheartApplyReactionOutcomeRequest) (*pb.DaggerheartApplyReactionOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply reaction outcome request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(grpcmeta.CampaignIDFromContext(ctx))
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		sessionID = strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	}
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if in.GetRollSeq() == 0 {
		return nil, status.Error(codes.InvalidArgument, "roll seq is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart reaction outcomes")
	}

	sess, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.SessionStatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}

	rollEvent, err := s.stores.Event.GetEventBySeq(ctx, campaignID, in.GetRollSeq())
	if err != nil {
		return nil, handleDomainError(err)
	}
	if rollEvent.Type != event.TypeRollResolved {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not reference action.roll_resolved")
	}
	if rollEvent.SessionID != sessionID {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not match session")
	}

	var rollPayload event.RollResolvedPayload
	if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
		return nil, status.Errorf(codes.Internal, "decode roll payload: %v", err)
	}

	rollRequestID := strings.TrimSpace(rollPayload.RequestID)
	if rollRequestID == "" {
		rollRequestID = strings.TrimSpace(rollEvent.RequestID)
	}
	if rollRequestID == "" {
		return nil, status.Error(codes.InvalidArgument, "roll request id is required")
	}

	rollKind := rollKindFromSystemData(rollPayload.SystemData)
	if rollKind != pb.RollKind_ROLL_KIND_REACTION {
		return nil, status.Error(codes.FailedPrecondition, "roll seq does not reference a reaction roll")
	}
	rollOutcome := outcomeFromSystemData(rollPayload.SystemData, rollPayload.Outcome)
	if rollOutcome == "" {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is required")
	}
	crit := critFromSystemData(rollPayload.SystemData, rollOutcome)
	rollSuccess, ok := outcomeSuccessFromCode(rollOutcome)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is invalid")
	}
	critNegates := boolFromSystemData(rollPayload.SystemData, "crit_negates", false)
	effectsNegated := crit && critNegates
	actorID := stringFromSystemData(rollPayload.SystemData, "character_id")
	if actorID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}

	existing, err := s.stores.Event.ListEventsPage(ctx, storage.ListEventsPageRequest{
		CampaignID:   campaignID,
		PageSize:     1,
		Descending:   true,
		FilterClause: "session_id = ? AND request_id = ? AND event_type = ?",
		FilterParams: []any{sessionID, rollRequestID, string(daggerheart.EventTypeReactionResolved)},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "check reaction outcome applied: %v", err)
	}
	if len(existing.Events) > 0 {
		return nil, status.Error(codes.FailedPrecondition, "reaction outcome already applied")
	}

	payload := daggerheart.ReactionResolvedPayload{
		CharacterID:        actorID,
		RollSeq:            in.GetRollSeq(),
		Outcome:            rollOutcome,
		Success:            rollSuccess,
		Crit:               crit,
		CritNegatesEffects: critNegates,
		EffectsNegated:     effectsNegated,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode reaction payload: %v", err)
	}

	if _, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          daggerheart.EventTypeReactionResolved,
		SessionID:     sessionID,
		RequestID:     rollRequestID,
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "reaction",
		EntityID:      rollRequestID,
		SystemID:      c.System.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "append reaction event: %v", err)
	}

	return &pb.DaggerheartApplyReactionOutcomeResponse{
		RollSeq:     in.GetRollSeq(),
		CharacterId: actorID,
		Result: &pb.DaggerheartReactionOutcomeResult{
			Outcome:            outcomeCodeToProto(rollOutcome),
			Success:            rollSuccess,
			Crit:               crit,
			CritNegatesEffects: critNegates,
			EffectsNegated:     effectsNegated,
		},
	}, nil
}

func normalizeActionModifiers(modifiers []*pb.ActionRollModifier) (int, []map[string]any) {
	if len(modifiers) == 0 {
		return 0, nil
	}

	entries := make([]map[string]any, 0, len(modifiers))
	total := 0
	for _, modifier := range modifiers {
		if modifier == nil {
			continue
		}
		value := int(modifier.GetValue())
		total += value
		entry := map[string]any{"value": value}
		if source := strings.TrimSpace(modifier.GetSource()); source != "" {
			entry["source"] = source
		}
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return total, nil
	}
	return total, entries
}

func normalizeRollKind(kind pb.RollKind) pb.RollKind {
	if kind == pb.RollKind_ROLL_KIND_UNSPECIFIED {
		return pb.RollKind_ROLL_KIND_ACTION
	}
	return kind
}

func withCampaignSessionMetadata(ctx context.Context, campaignID, sessionID string) context.Context {
	md, _ := metadata.FromIncomingContext(ctx)
	md = metadata.Join(md, metadata.Pairs(grpcmeta.CampaignIDHeader, campaignID, grpcmeta.SessionIDHeader, sessionID))
	return metadata.NewIncomingContext(ctx, md)
}

func containsString(values []string, target string) bool {
	if target == "" {
		return false
	}
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func damageDiceFromProto(specs []*pb.DiceSpec) ([]daggerheart.DamageDieSpec, error) {
	if len(specs) == 0 {
		return nil, dice.ErrMissingDice
	}
	converted := make([]daggerheart.DamageDieSpec, 0, len(specs))
	for _, spec := range specs {
		if spec == nil {
			return nil, dice.ErrInvalidDiceSpec
		}
		sides := int(spec.GetSides())
		count := int(spec.GetCount())
		if sides <= 0 || count <= 0 {
			return nil, dice.ErrInvalidDiceSpec
		}
		converted = append(converted, daggerheart.DamageDieSpec{Sides: sides, Count: count})
	}
	return converted, nil
}

func diceRollsToProto(rolls []dice.Roll) []*pb.DiceRoll {
	if len(rolls) == 0 {
		return nil
	}
	converted := make([]*pb.DiceRoll, 0, len(rolls))
	for _, roll := range rolls {
		converted = append(converted, &pb.DiceRoll{
			Sides:   int32(roll.Sides),
			Results: int32Slice(roll.Results),
			Total:   int32(roll.Total),
		})
	}
	return converted
}

func resolveRoll(kind pb.RollKind, request daggerheartdomain.ActionRequest) (daggerheartdomain.ActionResult, bool, bool, bool, error) {
	switch normalizeRollKind(kind) {
	case pb.RollKind_ROLL_KIND_REACTION:
		result, err := daggerheartdomain.RollReaction(daggerheartdomain.ReactionRequest{
			Modifier:   request.Modifier,
			Difficulty: request.Difficulty,
			Seed:       request.Seed,
		})
		if err != nil {
			return daggerheartdomain.ActionResult{}, false, false, false, err
		}
		return result.ActionResult, result.GeneratesHopeFear, result.TriggersGMMove, result.CritNegatesEffects, nil
	default:
		result, err := daggerheartdomain.RollAction(request)
		if err != nil {
			return daggerheartdomain.ActionResult{}, true, true, false, err
		}
		return result, true, true, false, nil
	}
}

type hopeSpend struct {
	Source string
	Amount int
}

func hopeSpendsFromModifiers(modifiers []*pb.ActionRollModifier) []hopeSpend {
	if len(modifiers) == 0 {
		return nil
	}

	spends := make([]hopeSpend, 0)
	for _, modifier := range modifiers {
		if modifier == nil {
			continue
		}
		sourceKey := normalizeHopeSpendSource(modifier.GetSource())
		amount := 0
		switch sourceKey {
		case "experience", "help":
			amount = 1
		case "tag_team", "hope_feature":
			amount = 3
		default:
			continue
		}
		spends = append(spends, hopeSpend{Source: sourceKey, Amount: amount})
	}

	if len(spends) == 0 {
		return nil
	}
	return spends
}

func normalizeHopeSpendSource(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	normalized := strings.ToLower(trimmed)
	replacer := strings.NewReplacer(" ", "_", "-", "_")
	return replacer.Replace(normalized)
}

func outcomeFlavorFromCode(code string) string {
	switch strings.TrimSpace(code) {
	case pb.Outcome_ROLL_WITH_HOPE.String(),
		pb.Outcome_SUCCESS_WITH_HOPE.String(),
		pb.Outcome_FAILURE_WITH_HOPE.String(),
		pb.Outcome_CRITICAL_SUCCESS.String():
		return "HOPE"
	case pb.Outcome_ROLL_WITH_FEAR.String(),
		pb.Outcome_SUCCESS_WITH_FEAR.String(),
		pb.Outcome_FAILURE_WITH_FEAR.String():
		return "FEAR"
	default:
		return ""
	}
}

func outcomeSuccessFromCode(code string) (bool, bool) {
	switch strings.TrimSpace(code) {
	case pb.Outcome_SUCCESS_WITH_HOPE.String(),
		pb.Outcome_SUCCESS_WITH_FEAR.String(),
		pb.Outcome_CRITICAL_SUCCESS.String():
		return true, true
	case pb.Outcome_FAILURE_WITH_HOPE.String(),
		pb.Outcome_FAILURE_WITH_FEAR.String(),
		pb.Outcome_ROLL_WITH_HOPE.String(),
		pb.Outcome_ROLL_WITH_FEAR.String():
		return false, true
	default:
		return false, false
	}
}

func outcomeCodeToProto(code string) pb.Outcome {
	switch strings.TrimSpace(code) {
	case pb.Outcome_ROLL_WITH_HOPE.String():
		return pb.Outcome_ROLL_WITH_HOPE
	case pb.Outcome_ROLL_WITH_FEAR.String():
		return pb.Outcome_ROLL_WITH_FEAR
	case pb.Outcome_SUCCESS_WITH_HOPE.String():
		return pb.Outcome_SUCCESS_WITH_HOPE
	case pb.Outcome_SUCCESS_WITH_FEAR.String():
		return pb.Outcome_SUCCESS_WITH_FEAR
	case pb.Outcome_FAILURE_WITH_HOPE.String():
		return pb.Outcome_FAILURE_WITH_HOPE
	case pb.Outcome_FAILURE_WITH_FEAR.String():
		return pb.Outcome_FAILURE_WITH_FEAR
	case pb.Outcome_CRITICAL_SUCCESS.String():
		return pb.Outcome_CRITICAL_SUCCESS
	default:
		return pb.Outcome_OUTCOME_UNSPECIFIED
	}
}

func outcomeFromSystemData(systemData map[string]any, fallback string) string {
	if systemData == nil {
		return strings.TrimSpace(fallback)
	}
	if value, ok := systemData["outcome"]; ok {
		if outcome, ok := value.(string); ok {
			return strings.TrimSpace(outcome)
		}
	}
	return strings.TrimSpace(fallback)
}

func rollKindFromSystemData(systemData map[string]any) pb.RollKind {
	if systemData == nil {
		return pb.RollKind_ROLL_KIND_ACTION
	}
	value, ok := systemData["roll_kind"]
	if !ok {
		return pb.RollKind_ROLL_KIND_ACTION
	}
	kind, ok := value.(string)
	if !ok {
		return pb.RollKind_ROLL_KIND_ACTION
	}
	switch strings.TrimSpace(kind) {
	case pb.RollKind_ROLL_KIND_REACTION.String():
		return pb.RollKind_ROLL_KIND_REACTION
	case pb.RollKind_ROLL_KIND_ACTION.String():
		return pb.RollKind_ROLL_KIND_ACTION
	default:
		return pb.RollKind_ROLL_KIND_ACTION
	}
}

func boolFromSystemData(systemData map[string]any, key string, fallback bool) bool {
	if systemData == nil {
		return fallback
	}
	value, ok := systemData[key]
	if !ok {
		return fallback
	}
	boolValue, ok := value.(bool)
	if !ok {
		return fallback
	}
	return boolValue
}

func critFromSystemData(systemData map[string]any, outcome string) bool {
	if systemData != nil {
		if value, ok := systemData["crit"]; ok {
			if crit, ok := value.(bool); ok {
				return crit
			}
		}
	}
	return strings.TrimSpace(outcome) == pb.Outcome_CRITICAL_SUCCESS.String()
}

func stringFromSystemData(systemData map[string]any, key string) string {
	if systemData == nil {
		return ""
	}
	value, ok := systemData[key]
	if !ok {
		return ""
	}
	stringValue, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(stringValue)
}

func normalizeTargets(targets []string) []string {
	if len(targets) == 0 {
		return nil
	}

	result := make([]string, 0, len(targets))
	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		trimmed := strings.TrimSpace(target)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func applyDaggerheartDamage(req *pb.DaggerheartDamageRequest, profile storage.DaggerheartCharacterProfile, state storage.DaggerheartCharacterState) (daggerheart.DamageApplication, bool, error) {
	damageTypes := daggerheart.DamageTypes{}
	switch req.DamageType {
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL:
		damageTypes.Physical = true
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC:
		damageTypes.Magic = true
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED:
		damageTypes.Physical = true
		damageTypes.Magic = true
	}

	resistance := daggerheart.ResistanceProfile{
		ResistPhysical: req.ResistPhysical,
		ResistMagic:    req.ResistMagic,
		ImmunePhysical: req.ImmunePhysical,
		ImmuneMagic:    req.ImmuneMagic,
	}
	adjusted := daggerheart.ApplyResistance(int(req.Amount), damageTypes, resistance)
	mitigated := adjusted < int(req.Amount)
	options := daggerheart.DamageOptions{EnableMassiveDamage: req.MassiveDamage}
	result, err := daggerheart.EvaluateDamage(adjusted, profile.MajorThreshold, profile.SevereThreshold, options)
	if err != nil {
		return daggerheart.DamageApplication{}, mitigated, err
	}
	if req.Direct {
		app, err := daggerheart.ApplyDamage(state.Hp, adjusted, profile.MajorThreshold, profile.SevereThreshold, options)
		return app, mitigated, err
	}
	app := daggerheart.ApplyDamageWithArmor(state.Hp, state.Armor, result)
	if app.ArmorSpent > 0 {
		mitigated = true
	}
	return app, mitigated, nil
}

func daggerheartSeverityToString(severity daggerheart.DamageSeverity) string {
	switch severity {
	case daggerheart.DamageMinor:
		return "minor"
	case daggerheart.DamageMajor:
		return "major"
	case daggerheart.DamageSevere:
		return "severe"
	case daggerheart.DamageMassive:
		return "massive"
	default:
		return "none"
	}
}

func daggerheartDamageTypeToString(t pb.DaggerheartDamageType) string {
	switch t {
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL:
		return "physical"
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC:
		return "magic"
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED:
		return "mixed"
	default:
		return "unknown"
	}
}

func daggerheartRestTypeFromProto(t pb.DaggerheartRestType) (daggerheart.RestType, error) {
	switch t {
	case pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_SHORT:
		return daggerheart.RestTypeShort, nil
	case pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG:
		return daggerheart.RestTypeLong, nil
	default:
		return daggerheart.RestTypeShort, errors.New("rest_type is required")
	}
}

func daggerheartRestTypeToString(t daggerheart.RestType) string {
	if t == daggerheart.RestTypeLong {
		return "long"
	}
	return "short"
}

func daggerheartDowntimeMoveFromProto(m pb.DaggerheartDowntimeMove) (daggerheart.DowntimeMove, error) {
	switch m {
	case pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_CLEAR_ALL_STRESS:
		return daggerheart.DowntimeClearAllStress, nil
	case pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_REPAIR_ALL_ARMOR:
		return daggerheart.DowntimeRepairAllArmor, nil
	case pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_PREPARE:
		return daggerheart.DowntimePrepare, nil
	case pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_WORK_ON_PROJECT:
		return daggerheart.DowntimeWorkOnProject, nil
	default:
		return daggerheart.DowntimePrepare, errors.New("downtime move is required")
	}
}

func daggerheartDowntimeMoveToString(m daggerheart.DowntimeMove) string {
	switch m {
	case daggerheart.DowntimeClearAllStress:
		return "clear_all_stress"
	case daggerheart.DowntimeRepairAllArmor:
		return "repair_all_armor"
	case daggerheart.DowntimePrepare:
		return "prepare"
	case daggerheart.DowntimeWorkOnProject:
		return "work_on_project"
	default:
		return "unknown"
	}
}

func daggerheartStateToProto(state storage.DaggerheartCharacterState) *pb.DaggerheartCharacterState {
	return &pb.DaggerheartCharacterState{
		Hp:         int32(state.Hp),
		Hope:       int32(state.Hope),
		HopeMax:    int32(state.HopeMax),
		Stress:     int32(state.Stress),
		Armor:      int32(state.Armor),
		Conditions: daggerheartConditionsToProto(state.Conditions),
		LifeState:  daggerheartLifeStateToProto(state.LifeState),
	}
}

func optionalInt32(value *int) *int32 {
	if value == nil {
		return nil
	}
	v := int32(*value)
	return &v
}

func handleDomainError(err error) error {
	return status.Errorf(codes.Internal, "%v", err)
}
