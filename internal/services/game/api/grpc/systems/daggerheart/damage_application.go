package daggerheart

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *DaggerheartService) runApplyDamage(ctx context.Context, in *pb.DaggerheartApplyDamageRequest) (*pb.DaggerheartApplyDamageResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply damage request is required")
	}
	if err := s.requireDependencies(dependencyCampaignStore, dependencyDaggerheartStore, dependencyEventStore); err != nil {
		return nil, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}
	if err := requireDaggerheartSystem(c, "campaign system does not support daggerheart damage"); err != nil {
		return nil, err
	}

	sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
	if err != nil {
		return nil, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	if err := s.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
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
		if rollEvent.Type != eventTypeActionRollResolved {
			return nil, status.Error(codes.InvalidArgument, "roll_seq must reference action.roll_resolved")
		}
		var rollPayload action.RollResolvePayload
		if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
			return nil, grpcerror.Internal("decode damage roll payload", err)
		}
		rollMetadata, err := decodeRollSystemMetadata(rollPayload.SystemData)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid roll system_data: %v", err)
		}
		rollCharacterID := strings.TrimSpace(rollMetadata.CharacterID)
		if rollMetadata.rollKindCode() != "damage_roll" {
			return nil, status.Error(codes.InvalidArgument, "roll_seq does not reference a damage roll")
		}
		if rollCharacterID != characterID && !containsString(sourceCharacterIDs, rollCharacterID) {
			return nil, status.Error(codes.InvalidArgument, "roll_seq does not match target or source character")
		}
	}
	payload := daggerheart.DamageApplyPayload{
		CharacterID:        ids.CharacterID(characterID),
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
		SourceCharacterIDs: stringsToCharacterIDs(sourceCharacterIDs),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, grpcerror.Internal("encode payload", err)
	}
	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:    ids.CampaignID(campaignID),
		Type:          commandTypeDaggerheartDamageApply,
		ActorType:     command.ActorTypeSystem,
		SessionID:     ids.SessionID(sessionID),
		SceneID:       ids.SceneID(sceneID),
		RequestID:     requestID,
		InvocationID:  invocationID,
		EntityType:    "character",
		EntityID:      characterID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}, adapter, domainwrite.RequireEventsWithDiagnostics("damage did not emit an event", "apply damage event"))
	if err != nil {
		return nil, err
	}

	updated, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.Internal("load daggerheart state", err)
	}
	return &pb.DaggerheartApplyDamageResponse{
		CharacterId: characterID,
		State:       daggerheartStateToProto(updated),
	}, nil
}

func (s *DaggerheartService) runApplyAdversaryDamage(ctx context.Context, in *pb.DaggerheartApplyAdversaryDamageRequest) (*pb.DaggerheartApplyAdversaryDamageResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply adversary damage request is required")
	}
	if err := s.requireDependencies(dependencyCampaignStore, dependencyDaggerheartStore, dependencyEventStore); err != nil {
		return nil, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	adversaryID, err := validate.RequiredID(in.GetAdversaryId(), "adversary id")
	if err != nil {
		return nil, err
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}
	if err := requireDaggerheartSystem(c, "campaign system does not support daggerheart damage"); err != nil {
		return nil, err
	}

	sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
	if err != nil {
		return nil, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	if err := s.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
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

	adversary, err := s.loadAdversaryForSession(ctx, campaignID, sessionID, adversaryID)
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
		rollEvent, err := s.stores.Event.GetEventBySeq(ctx, campaignID, *rollSeq)
		if err != nil {
			return nil, handleDomainError(err)
		}
		if rollEvent.Type != eventTypeActionRollResolved {
			return nil, status.Error(codes.InvalidArgument, "roll_seq must reference action.roll_resolved")
		}
		var rollPayload action.RollResolvePayload
		if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
			return nil, grpcerror.Internal("decode damage roll payload", err)
		}
		rollMetadata, err := decodeRollSystemMetadata(rollPayload.SystemData)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid roll system_data: %v", err)
		}
		if rollMetadata.rollKindCode() != "damage_roll" {
			return nil, status.Error(codes.InvalidArgument, "roll_seq does not reference a damage roll")
		}
		if len(sourceCharacterIDs) > 0 {
			rollCharacterID := strings.TrimSpace(rollMetadata.CharacterID)
			if !containsString(sourceCharacterIDs, rollCharacterID) {
				return nil, status.Error(codes.InvalidArgument, "roll_seq does not match source character")
			}
		}
	}

	payload := daggerheart.AdversaryDamageApplyPayload{
		AdversaryID:        ids.AdversaryID(adversaryID),
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
		SourceCharacterIDs: stringsToCharacterIDs(sourceCharacterIDs),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, grpcerror.Internal("encode payload", err)
	}
	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:    ids.CampaignID(campaignID),
		Type:          commandTypeDaggerheartAdversaryDamageApply,
		ActorType:     command.ActorTypeSystem,
		SessionID:     ids.SessionID(sessionID),
		SceneID:       ids.SceneID(sceneID),
		RequestID:     requestID,
		InvocationID:  invocationID,
		EntityType:    "adversary",
		EntityID:      adversaryID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}, adapter, domainwrite.RequireEventsWithDiagnostics("adversary damage did not emit an event", "apply adversary damage event"))
	if err != nil {
		return nil, err
	}

	updated, err := s.stores.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return nil, grpcerror.Internal("load daggerheart adversary", err)
	}

	return &pb.DaggerheartApplyAdversaryDamageResponse{
		AdversaryId: adversaryID,
		Adversary:   daggerheartAdversaryToProto(updated),
	}, nil
}
