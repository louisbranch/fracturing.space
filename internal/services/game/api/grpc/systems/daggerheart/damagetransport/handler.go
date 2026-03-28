package damagetransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
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
		return CharacterDamageResult{}, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return CharacterDamageResult{}, grpcerror.HandleDomainErrorContext(ctx, err)
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
		return CharacterDamageResult{}, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return CharacterDamageResult{}, grpcerror.HandleDomainErrorContext(ctx, err)
	}

	var armor *contentstore.DaggerheartArmor
	if equippedArmorID := strings.TrimSpace(profile.EquippedArmorID); equippedArmorID != "" {
		entry, loadErr := h.deps.Content.GetDaggerheartArmor(ctx, equippedArmorID)
		if loadErr != nil {
			return CharacterDamageResult{}, status.Errorf(codes.FailedPrecondition, "equipped armor %q not found", equippedArmorID)
		}
		armor = &entry
	}

	result, mitigated, err := ResolveCharacterDamage(in.Damage, profile, state, armor)
	if err != nil {
		return CharacterDamageResult{}, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	impenetrableBefore := state.ImpenetrableUsedThisShortRest
	impenetrableAfter := impenetrableBefore
	if armor != nil && in.GetArmorReaction() != nil {
		armorRules := rules.EffectiveArmorRules(armor)
		switch reaction := in.GetArmorReaction().GetReaction().(type) {
		case *pb.DaggerheartDamageArmorReaction_Resilient:
			_ = reaction
			if h.deps.SeedFunc == nil {
				return CharacterDamageResult{}, status.Error(codes.Internal, "seed generator is not configured")
			}
			if armorRules.ResilientDieSides <= 0 {
				return CharacterDamageResult{}, status.Error(codes.FailedPrecondition, "equipped armor does not support resilient")
			}
			if in.GetDamage().GetDirect() {
				return CharacterDamageResult{}, status.Error(codes.FailedPrecondition, "resilient cannot apply to direct damage")
			}
			if !rules.IsLastBaseArmorSlot(state, profile.ArmorMax) || result.ArmorSpent == 0 {
				return CharacterDamageResult{}, status.Error(codes.FailedPrecondition, "resilient requires spending the last base armor slot")
			}
			roll, err := rollArmorFeatureDie(h.deps.SeedFunc, reaction.Resilient.GetRng(), armorRules.ResilientDieSides)
			if err != nil {
				return CharacterDamageResult{}, err
			}
			if roll >= armorRules.ResilientSuccessOnOrAbove {
				result.ArmorAfter = result.ArmorBefore
				result.ArmorSpent = 0
			}
		case *pb.DaggerheartDamageArmorReaction_Impenetrable:
			if armorRules.ImpenetrableUsesPerShortRest <= 0 || armorRules.ImpenetrableStressCost <= 0 {
				return CharacterDamageResult{}, status.Error(codes.FailedPrecondition, "equipped armor does not support impenetrable")
			}
			if impenetrableBefore {
				return CharacterDamageResult{}, status.Error(codes.FailedPrecondition, "impenetrable has already been used this short rest")
			}
			if result.HPBefore != 1 || result.HPAfter != 0 {
				return CharacterDamageResult{}, status.Error(codes.FailedPrecondition, "impenetrable only applies when damage would mark the last hit point")
			}
			stressAfter := state.Stress + armorRules.ImpenetrableStressCost
			if stressAfter > profile.StressMax {
				return CharacterDamageResult{}, status.Error(codes.FailedPrecondition, "impenetrable requires available stress")
			}
			result.HPAfter = 1
			result.StressAfter = stressAfter
			impenetrableAfter = true
		}
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
			return CharacterDamageResult{}, grpcerror.HandleDomainErrorContext(ctx, err)
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

	payloadJSON, err := json.Marshal(daggerheartpayload.DamageApplyPayload{
		CharacterID:        ids.CharacterID(characterID),
		HpBefore:           &hpBefore,
		HpAfter:            &hpAfter,
		StressAfter:        &result.StressAfter,
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
	if impenetrableAfter != impenetrableBefore {
		payloadJSON, err := json.Marshal(daggerheartpayload.CharacterStatePatchPayload{
			CharacterID:                         ids.CharacterID(characterID),
			Source:                              "armor.impenetrable",
			ImpenetrableUsedThisShortRestBefore: &impenetrableBefore,
			ImpenetrableUsedThisShortRestAfter:  &impenetrableAfter,
		})
		if err != nil {
			return CharacterDamageResult{}, grpcerror.Internal("encode impenetrable payload", err)
		}
		if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
			CampaignID:      campaignID,
			CommandType:     commandids.DaggerheartCharacterStatePatch,
			SessionID:       sessionID,
			SceneID:         sceneID,
			RequestID:       grpcmeta.RequestIDFromContext(ctx),
			InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
			EntityType:      "character",
			EntityID:        characterID,
			PayloadJSON:     payloadJSON,
			MissingEventMsg: "impenetrable state patch did not emit an event",
			ApplyErrMessage: "apply impenetrable state patch",
		}); err != nil {
			return CharacterDamageResult{}, err
		}
	}
	nextState := applyCharacterDamageResult(state, result)
	nextState.ImpenetrableUsedThisShortRest = impenetrableAfter
	if err := h.autoDropBeastform(ctx, campaignID, sessionID, sceneID, characterID, state, nextState); err != nil {
		return CharacterDamageResult{}, err
	}
	return CharacterDamageResult{
		CharacterID: characterID,
		State:       nextState,
	}, nil
}

func (h *Handler) requireDamageDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.Content == nil:
		return status.Error(codes.Internal, "daggerheart content store is not configured")
	case h.deps.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	case h.deps.ExecuteSystemCommand == nil:
		return status.Error(codes.Internal, "system command executor is not configured")
	default:
		return nil
	}
}
