package daggerheart

import (
	"context"
	"encoding/json"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type damageApplication struct {
	service *DaggerheartService
}

func newDamageApplication(service *DaggerheartService) damageApplication {
	return damageApplication{service: service}
}

func (a damageApplication) runApplyDamage(ctx context.Context, in *pb.DaggerheartApplyDamageRequest) (*pb.DaggerheartApplyDamageResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply damage request is required")
	}
	if a.service.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if a.service.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if a.service.stores.Event == nil {
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

	c, err := a.service.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart damage")
	}

	sessionID := strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if err := a.service.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return nil, err
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

	profile, err := a.service.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	state, err := a.service.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
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
		rollEvent, err := a.service.stores.Event.GetEventBySeq(ctx, campaignID, *rollSeq)
		if err != nil {
			return nil, handleDomainError(err)
		}
		if rollEvent.Type != eventTypeActionRollResolved {
			return nil, status.Error(codes.InvalidArgument, "roll_seq must reference action.roll_resolved")
		}
		var rollPayload action.RollResolvePayload
		if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
			return nil, status.Errorf(codes.Internal, "decode damage roll payload: %v", err)
		}
		rollCharacterID := stringFromSystemData(rollPayload.SystemData, "character_id")
		if stringFromSystemData(rollPayload.SystemData, "roll_kind") != "damage_roll" {
			return nil, status.Error(codes.InvalidArgument, "roll_seq does not reference a damage roll")
		}
		if rollCharacterID != characterID && !containsString(sourceCharacterIDs, rollCharacterID) {
			return nil, status.Error(codes.InvalidArgument, "roll_seq does not match target or source character")
		}
	}
	payload := daggerheart.DamageApplyPayload{
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
	adapter := daggerheart.NewAdapter(a.service.stores.Daggerheart)
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	_, err = a.service.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:    campaignID,
		Type:          commandTypeDaggerheartDamageApply,
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
		missingEventMsg: "damage did not emit an event",
		applyErrMessage: "apply damage event",
		executeErrMsg:   "execute domain command",
	})
	if err != nil {
		return nil, err
	}

	updated, err := a.service.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load daggerheart state: %v", err)
	}
	return &pb.DaggerheartApplyDamageResponse{
		CharacterId: characterID,
		State:       daggerheartStateToProto(updated),
	}, nil
}

func (a damageApplication) runApplyAdversaryDamage(ctx context.Context, in *pb.DaggerheartApplyAdversaryDamageRequest) (*pb.DaggerheartApplyAdversaryDamageResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply adversary damage request is required")
	}
	if a.service.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if a.service.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if a.service.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}
	if a.service.stores.Domain == nil {
		return nil, status.Error(codes.Internal, "domain engine is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	adversaryID := strings.TrimSpace(in.GetAdversaryId())
	if adversaryID == "" {
		return nil, status.Error(codes.InvalidArgument, "adversary id is required")
	}

	c, err := a.service.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart damage")
	}

	sessionID := strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if err := a.service.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return nil, err
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

	adversary, err := a.service.loadAdversaryForSession(ctx, campaignID, sessionID, adversaryID)
	if err != nil {
		return nil, err
	}

	result, mitigated, err := applyDaggerheartAdversaryDamage(in.Damage, adversary)
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
		rollEvent, err := a.service.stores.Event.GetEventBySeq(ctx, campaignID, *rollSeq)
		if err != nil {
			return nil, handleDomainError(err)
		}
		if rollEvent.Type != eventTypeActionRollResolved {
			return nil, status.Error(codes.InvalidArgument, "roll_seq must reference action.roll_resolved")
		}
		var rollPayload action.RollResolvePayload
		if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
			return nil, status.Errorf(codes.Internal, "decode damage roll payload: %v", err)
		}
		if stringFromSystemData(rollPayload.SystemData, "roll_kind") != "damage_roll" {
			return nil, status.Error(codes.InvalidArgument, "roll_seq does not reference a damage roll")
		}
		if len(sourceCharacterIDs) > 0 {
			rollCharacterID := stringFromSystemData(rollPayload.SystemData, "character_id")
			if !containsString(sourceCharacterIDs, rollCharacterID) {
				return nil, status.Error(codes.InvalidArgument, "roll_seq does not match source character")
			}
		}
	}

	payload := daggerheart.AdversaryDamageApplyPayload{
		AdversaryID:        adversaryID,
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
	adapter := daggerheart.NewAdapter(a.service.stores.Daggerheart)
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	_, err = a.service.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:    campaignID,
		Type:          commandTypeDaggerheartAdversaryDamageApply,
		ActorType:     command.ActorTypeSystem,
		SessionID:     sessionID,
		RequestID:     requestID,
		InvocationID:  invocationID,
		EntityType:    "adversary",
		EntityID:      adversaryID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}, adapter, domainCommandApplyOptions{
		requireEvents:   true,
		missingEventMsg: "adversary damage did not emit an event",
		applyErrMessage: "apply adversary damage event",
		executeErrMsg:   "execute domain command",
	})
	if err != nil {
		return nil, err
	}

	updated, err := a.service.stores.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load daggerheart adversary: %v", err)
	}

	return &pb.DaggerheartApplyAdversaryDamageResponse{
		AdversaryId: adversaryID,
		Adversary:   daggerheartAdversaryToProto(updated),
	}, nil
}
