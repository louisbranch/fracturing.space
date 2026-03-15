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

// ApplyDamage validates one character damage application, optionally ties it to
// a damage roll, emits the system command, and reloads the resulting state.
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
