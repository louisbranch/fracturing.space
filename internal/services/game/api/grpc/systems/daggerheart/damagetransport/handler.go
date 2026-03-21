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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
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
		return CharacterDamageResult{}, grpcerror.HandleDomainError(err)
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
	entry, err := h.deps.Content.GetDaggerheartAdversaryEntry(ctx, adversary.AdversaryEntryID)
	if err != nil {
		return AdversaryDamageResult{}, status.Errorf(codes.FailedPrecondition, "adversary entry %q not found", adversary.AdversaryEntryID)
	}

	result, mitigated, err := ResolveAdversaryDamage(in.Damage, adversary)
	if err != nil {
		return AdversaryDamageResult{}, grpcerror.HandleDomainError(err)
	}
	if rules.AdversaryIsMinion(entry) && in.Damage.Amount > 0 {
		result.HPAfter = 0
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

	payloadJSON, err := json.Marshal(daggerheartpayload.AdversaryDamageApplyPayload{
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
	if rules.AdversaryIsMinion(entry) && in.Damage.Amount > 0 {
		if err := h.deleteAdversary(ctx, campaignID, sessionID, sceneID, adversary.AdversaryID, "minion_defeated"); err != nil {
			return AdversaryDamageResult{}, err
		}
		if err := h.applyMinionSpillover(ctx, campaignID, sessionID, sceneID, adversary, int(in.Damage.Amount)); err != nil {
			return AdversaryDamageResult{}, err
		}
	}
	return AdversaryDamageResult{
		AdversaryID: adversaryID,
		Adversary:   applyAdversaryDamageResult(adversary, result),
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

func (h *Handler) requireAdversaryDamageDependencies() error {
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
	case h.deps.LoadAdversaryForSession == nil:
		return status.Error(codes.Internal, "adversary loader is not configured")
	default:
		return nil
	}
}

func (h *Handler) autoDropBeastform(ctx context.Context, campaignID, sessionID, sceneID, characterID string, previousState, nextState projectionstore.DaggerheartCharacterState) error {
	classState := classStateFromProjection(previousState.ClassState)
	active := classState.ActiveBeastform
	if active == nil {
		return nil
	}
	source := ""
	switch {
	case nextState.Hp == 0:
		source = "beastform.auto_drop.hp_zero"
	case active.DropOnAnyHPMark && nextState.Hp < previousState.Hp:
		source = "beastform.auto_drop.fragile"
	default:
		return nil
	}
	nextClassState := daggerheartstate.WithActiveBeastform(classState, nil)
	payloadJSON, err := json.Marshal(daggerheartpayload.BeastformDropPayload{
		ActorCharacterID: ids.CharacterID(characterID),
		CharacterID:      ids.CharacterID(characterID),
		BeastformID:      active.BeastformID,
		Source:           source,
		ClassStateBefore: classStatePtr(classState),
		ClassStateAfter:  classStatePtr(nextClassState),
	})
	if err != nil {
		return grpcerror.Internal("encode beastform auto-drop payload", err)
	}
	return h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartBeastformDrop,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "character",
		EntityID:        characterID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "beastform auto-drop did not emit an event",
		ApplyErrMessage: source,
	})
}

func classStateFromProjection(state projectionstore.DaggerheartClassState) daggerheartstate.CharacterClassState {
	return daggerheartstate.CharacterClassState{
		AttackBonusUntilRest:            state.AttackBonusUntilRest,
		EvasionBonusUntilHitOrRest:      state.EvasionBonusUntilHitOrRest,
		DifficultyPenaltyUntilRest:      state.DifficultyPenaltyUntilRest,
		FocusTargetID:                   state.FocusTargetID,
		ActiveBeastform:                 activeBeastformFromProjection(state.ActiveBeastform),
		StrangePatternsNumber:           state.StrangePatternsNumber,
		RallyDice:                       append([]int(nil), state.RallyDice...),
		PrayerDice:                      append([]int(nil), state.PrayerDice...),
		ChannelRawPowerUsedThisLongRest: state.ChannelRawPowerUsedThisLongRest,
		Unstoppable: daggerheartstate.CharacterUnstoppableState{
			Active:           state.Unstoppable.Active,
			CurrentValue:     state.Unstoppable.CurrentValue,
			DieSides:         state.Unstoppable.DieSides,
			UsedThisLongRest: state.Unstoppable.UsedThisLongRest,
		},
	}.Normalized()
}

func classStatePtr(state daggerheartstate.CharacterClassState) *daggerheartstate.CharacterClassState {
	normalized := state.Normalized()
	return &normalized
}

func activeBeastformFromProjection(state *projectionstore.DaggerheartActiveBeastformState) *daggerheartstate.CharacterActiveBeastformState {
	if state == nil {
		return nil
	}
	damageDice := make([]daggerheartstate.CharacterDamageDie, 0, len(state.DamageDice))
	for _, die := range state.DamageDice {
		damageDice = append(damageDice, daggerheartstate.CharacterDamageDie{Count: die.Count, Sides: die.Sides})
	}
	return &daggerheartstate.CharacterActiveBeastformState{
		BeastformID:            state.BeastformID,
		BaseTrait:              state.BaseTrait,
		AttackTrait:            state.AttackTrait,
		TraitBonus:             state.TraitBonus,
		EvasionBonus:           state.EvasionBonus,
		AttackRange:            state.AttackRange,
		DamageDice:             damageDice,
		DamageBonus:            state.DamageBonus,
		DamageType:             state.DamageType,
		EvolutionTraitOverride: state.EvolutionTraitOverride,
		DropOnAnyHPMark:        state.DropOnAnyHPMark,
	}
}

func (h *Handler) applyMinionSpillover(ctx context.Context, campaignID, sessionID, sceneID string, primary projectionstore.DaggerheartAdversary, damageAmount int) error {
	primaryEntry, err := h.deps.Content.GetDaggerheartAdversaryEntry(ctx, primary.AdversaryEntryID)
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, "adversary entry %q not found", primary.AdversaryEntryID)
	}
	defeats := rules.AdversaryMinionSpilloverDefeats(primaryEntry, damageAmount)
	if defeats <= 0 {
		return nil
	}
	adversaries, err := h.deps.Daggerheart.ListDaggerheartAdversaries(ctx, campaignID, sessionID)
	if err != nil {
		return grpcerror.Internal("list adversaries", err)
	}
	remaining := defeats
	for _, candidate := range adversaries {
		if remaining == 0 {
			break
		}
		if candidate.AdversaryID == primary.AdversaryID || candidate.SceneID != primary.SceneID {
			continue
		}
		entry, err := h.deps.Content.GetDaggerheartAdversaryEntry(ctx, candidate.AdversaryEntryID)
		if err != nil {
			return status.Errorf(codes.FailedPrecondition, "adversary entry %q not found", candidate.AdversaryEntryID)
		}
		if !rules.AdversaryIsMinion(entry) {
			continue
		}
		if err := h.deleteAdversary(ctx, campaignID, sessionID, sceneID, candidate.AdversaryID, "minion_defeated"); err != nil {
			return err
		}
		remaining--
	}
	return nil
}

func (h *Handler) deleteAdversary(ctx context.Context, campaignID, sessionID, sceneID, adversaryID, reason string) error {
	payloadJSON, err := json.Marshal(daggerheartpayload.AdversaryDeletePayload{
		AdversaryID: ids.AdversaryID(adversaryID),
		Reason:      reason,
	})
	if err != nil {
		return grpcerror.Internal("encode adversary delete payload", err)
	}
	return h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartAdversaryDelete,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "adversary",
		EntityID:        adversaryID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "adversary delete did not emit an event",
		ApplyErrMessage: "apply adversary delete event",
	})
}
