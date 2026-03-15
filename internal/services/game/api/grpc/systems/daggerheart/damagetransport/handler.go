package damagetransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler owns the Daggerheart damage-application transport endpoints.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a Daggerheart damage-application transport handler from
// explicit read-store and write-callback dependencies.
func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

func (h *Handler) ApplyDamage(ctx context.Context, in *pb.DaggerheartApplyDamageRequest) (CharacterDamageResult, error) {
	if in == nil {
		return CharacterDamageResult{}, status.Error(codes.InvalidArgument, "apply damage request is required")
	}
	if err := h.requireDamageDependencies(); err != nil {
		return CharacterDamageResult{}, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return CharacterDamageResult{}, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return CharacterDamageResult{}, err
	}

	c, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return CharacterDamageResult{}, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return CharacterDamageResult{}, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(c, "campaign system does not support daggerheart damage"); err != nil {
		return CharacterDamageResult{}, err
	}

	sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
	if err != nil {
		return CharacterDamageResult{}, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return CharacterDamageResult{}, err
	}

	if in.Damage == nil {
		return CharacterDamageResult{}, status.Error(codes.InvalidArgument, "damage is required")
	}
	if in.Damage.Amount < 0 {
		return CharacterDamageResult{}, status.Error(codes.InvalidArgument, "damage amount must be non-negative")
	}
	if in.Damage.DamageType == pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED {
		return CharacterDamageResult{}, status.Error(codes.InvalidArgument, "damage_type is required")
	}

	profile, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return CharacterDamageResult{}, grpcerror.HandleDomainError(err)
	}
	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return CharacterDamageResult{}, grpcerror.HandleDomainError(err)
	}

	result, mitigated, err := ResolveCharacterDamage(in.Damage, profile, state)
	if err != nil {
		return CharacterDamageResult{}, grpcerror.HandleDomainError(err)
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
	sourceCharacterIDs := workflowtransport.NormalizeTargets(in.Damage.GetSourceCharacterIds())
	if requireDamageRoll && rollSeq == nil {
		return CharacterDamageResult{}, status.Error(codes.InvalidArgument, "roll_seq is required when require_damage_roll is true")
	}
	if rollSeq != nil {
		rollEvent, err := h.deps.Event.GetEventBySeq(ctx, campaignID, *rollSeq)
		if err != nil {
			return CharacterDamageResult{}, grpcerror.HandleDomainError(err)
		}
		if rollEvent.Type != action.EventTypeRollResolved {
			return CharacterDamageResult{}, status.Error(codes.InvalidArgument, "roll_seq must reference action.roll_resolved")
		}
		var rollPayload action.RollResolvePayload
		if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
			return CharacterDamageResult{}, grpcerror.Internal("decode damage roll payload", err)
		}
		rollMetadata, err := workflowtransport.DecodeRollSystemMetadata(rollPayload.SystemData)
		if err != nil {
			return CharacterDamageResult{}, status.Errorf(codes.InvalidArgument, "invalid roll system_data: %v", err)
		}
		rollCharacterID := strings.TrimSpace(rollMetadata.CharacterID)
		if rollMetadata.RollKindCode() != "damage_roll" {
			return CharacterDamageResult{}, status.Error(codes.InvalidArgument, "roll_seq does not reference a damage roll")
		}
		if rollCharacterID != characterID && !containsString(sourceCharacterIDs, rollCharacterID) {
			return CharacterDamageResult{}, status.Error(codes.InvalidArgument, "roll_seq does not match target or source character")
		}
	}

	payloadJSON, err := json.Marshal(daggerheart.DamageApplyPayload{
		CharacterID:        ids.CharacterID(characterID),
		HpBefore:           &hpBefore,
		HpAfter:            &hpAfter,
		ArmorBefore:        &armorBefore,
		ArmorAfter:         &armorAfter,
		ArmorSpent:         result.ArmorSpent,
		Severity:           DamageSeverityString(result.Result.Severity),
		Marks:              result.Result.Marks,
		DamageType:         DamageTypeString(in.Damage.DamageType),
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
	})
	if err != nil {
		return CharacterDamageResult{}, grpcerror.Internal("encode payload", err)
	}

	if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartDamageApply,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "character",
		EntityID:        characterID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "damage did not emit an event",
		ApplyErrMessage: "apply damage event",
	}); err != nil {
		return CharacterDamageResult{}, err
	}

	updated, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return CharacterDamageResult{}, grpcerror.Internal("load daggerheart state", err)
	}
	return CharacterDamageResult{CharacterID: characterID, State: updated}, nil
}

func (h *Handler) ApplyAdversaryDamage(ctx context.Context, in *pb.DaggerheartApplyAdversaryDamageRequest) (AdversaryDamageResult, error) {
	if in == nil {
		return AdversaryDamageResult{}, status.Error(codes.InvalidArgument, "apply adversary damage request is required")
	}
	if err := h.requireAdversaryDamageDependencies(); err != nil {
		return AdversaryDamageResult{}, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return AdversaryDamageResult{}, err
	}
	adversaryID, err := validate.RequiredID(in.GetAdversaryId(), "adversary id")
	if err != nil {
		return AdversaryDamageResult{}, err
	}

	c, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return AdversaryDamageResult{}, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return AdversaryDamageResult{}, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(c, "campaign system does not support daggerheart damage"); err != nil {
		return AdversaryDamageResult{}, err
	}

	sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
	if err != nil {
		return AdversaryDamageResult{}, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return AdversaryDamageResult{}, err
	}

	if in.Damage == nil {
		return AdversaryDamageResult{}, status.Error(codes.InvalidArgument, "damage is required")
	}
	if in.Damage.Amount < 0 {
		return AdversaryDamageResult{}, status.Error(codes.InvalidArgument, "damage amount must be non-negative")
	}
	if in.Damage.DamageType == pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED {
		return AdversaryDamageResult{}, status.Error(codes.InvalidArgument, "damage_type is required")
	}

	adversary, err := h.deps.LoadAdversaryForSession(ctx, campaignID, sessionID, adversaryID)
	if err != nil {
		return AdversaryDamageResult{}, err
	}

	result, mitigated, err := ResolveAdversaryDamage(in.Damage, adversary)
	if err != nil {
		return AdversaryDamageResult{}, grpcerror.HandleDomainError(err)
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
	sourceCharacterIDs := workflowtransport.NormalizeTargets(in.Damage.GetSourceCharacterIds())
	if requireDamageRoll && rollSeq == nil {
		return AdversaryDamageResult{}, status.Error(codes.InvalidArgument, "roll_seq is required when require_damage_roll is true")
	}
	if rollSeq != nil {
		rollEvent, err := h.deps.Event.GetEventBySeq(ctx, campaignID, *rollSeq)
		if err != nil {
			return AdversaryDamageResult{}, grpcerror.HandleDomainError(err)
		}
		if rollEvent.Type != action.EventTypeRollResolved {
			return AdversaryDamageResult{}, status.Error(codes.InvalidArgument, "roll_seq must reference action.roll_resolved")
		}
		var rollPayload action.RollResolvePayload
		if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
			return AdversaryDamageResult{}, grpcerror.Internal("decode damage roll payload", err)
		}
		rollMetadata, err := workflowtransport.DecodeRollSystemMetadata(rollPayload.SystemData)
		if err != nil {
			return AdversaryDamageResult{}, status.Errorf(codes.InvalidArgument, "invalid roll system_data: %v", err)
		}
		if rollMetadata.RollKindCode() != "damage_roll" {
			return AdversaryDamageResult{}, status.Error(codes.InvalidArgument, "roll_seq does not reference a damage roll")
		}
		if len(sourceCharacterIDs) > 0 {
			rollCharacterID := strings.TrimSpace(rollMetadata.CharacterID)
			if !containsString(sourceCharacterIDs, rollCharacterID) {
				return AdversaryDamageResult{}, status.Error(codes.InvalidArgument, "roll_seq does not match source character")
			}
		}
	}

	payloadJSON, err := json.Marshal(daggerheart.AdversaryDamageApplyPayload{
		AdversaryID:        ids.AdversaryID(adversaryID),
		HpBefore:           &hpBefore,
		HpAfter:            &hpAfter,
		ArmorBefore:        &armorBefore,
		ArmorAfter:         &armorAfter,
		ArmorSpent:         result.ArmorSpent,
		Severity:           DamageSeverityString(result.Result.Severity),
		Marks:              result.Result.Marks,
		DamageType:         DamageTypeString(in.Damage.DamageType),
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
	})
	if err != nil {
		return AdversaryDamageResult{}, grpcerror.Internal("encode payload", err)
	}

	if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartAdversaryDamageApply,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "adversary",
		EntityID:        adversaryID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "adversary damage did not emit an event",
		ApplyErrMessage: "apply adversary damage event",
	}); err != nil {
		return AdversaryDamageResult{}, err
	}

	updated, err := h.deps.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return AdversaryDamageResult{}, grpcerror.Internal("load daggerheart adversary", err)
	}
	return AdversaryDamageResult{AdversaryID: adversaryID, Adversary: updated}, nil
}

func (h *Handler) requireDamageDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	case h.deps.ExecuteSystemCommand == nil:
		return status.Error(codes.Internal, "system command executor is not configured")
	default:
		return nil
	}
}

func (h *Handler) requireAdversaryDamageDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	case h.deps.ExecuteSystemCommand == nil:
		return status.Error(codes.Internal, "system command executor is not configured")
	case h.deps.LoadAdversaryForSession == nil:
		return status.Error(codes.Internal, "adversary loader is not configured")
	default:
		return nil
	}
}
