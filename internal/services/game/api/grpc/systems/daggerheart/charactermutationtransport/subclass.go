package charactermutationtransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/statetransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) ApplySubclassFeature(ctx context.Context, in *pb.DaggerheartApplySubclassFeatureRequest) (*pb.DaggerheartApplySubclassFeatureResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply subclass feature request is required")
	}
	if err := h.requireDependencies(); err != nil {
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

	profile, err := h.validateCharacterPreconditions(ctx, campaignID, characterID, "subclass feature")
	if err != nil {
		return nil, err
	}
	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}

	classState := classStateFromProjection(state.ClassState)
	subclassState := subclassStateFromProjection(state.SubclassState)
	payload, err := h.resolveSubclassFeaturePayload(ctx, campaignID, profile, state, classState, subclassState, in)
	if err != nil {
		return nil, err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, grpcerror.Internal("encode subclass feature payload", err)
	}
	if err := h.executeCharacterCommand(ctx, CharacterCommandInput{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		CommandType:     commandids.DaggerheartSubclassFeatureApply,
		SessionID:       strings.TrimSpace(in.GetSessionId()),
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "subclass feature did not emit an event",
		ApplyErrMessage: "apply subclass feature event",
	}); err != nil {
		return nil, err
	}

	updatedState, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.Internal("load daggerheart state", err)
	}
	return &pb.DaggerheartApplySubclassFeatureResponse{
		CharacterId: characterID,
		State:       statetransport.CharacterStateToProto(updatedState),
	}, nil
}

func (h *Handler) resolveSubclassFeaturePayload(
	ctx context.Context,
	campaignID string,
	profile projectionstore.DaggerheartCharacterProfile,
	state projectionstore.DaggerheartCharacterState,
	classState daggerheart.CharacterClassState,
	subclassState daggerheart.CharacterSubclassState,
	in *pb.DaggerheartApplySubclassFeatureRequest,
) (daggerheart.SubclassFeatureApplyPayload, error) {
	_ = classState
	payload := daggerheart.SubclassFeatureApplyPayload{
		ActorCharacterID: ids.CharacterID(strings.TrimSpace(in.GetCharacterId())),
	}

	switch feature := in.GetFeature().(type) {
	case *pb.DaggerheartApplySubclassFeatureRequest_BattleRitual:
		if !hasUnlockedSubclassRank(profile, "subclass.call_of_the_brave", "foundation") {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "battle_ritual requires Call of the Brave")
		}
		if subclassState.BattleRitualUsedThisLongRest {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "battle_ritual already used this long rest")
		}
		next := subclassState
		next.BattleRitualUsedThisLongRest = true
		payload.Feature = "battle_ritual"
		payload.Targets = []daggerheart.SubclassFeatureTargetPatchPayload{{
			CharacterID:         ids.CharacterID(in.GetCharacterId()),
			HopeBefore:          intPtr(state.Hope),
			HopeAfter:           intPtr(min(state.Hope+2, state.HopeMax)),
			StressBefore:        intPtr(state.Stress),
			StressAfter:         intPtr(max(state.Stress-2, 0)),
			SubclassStateBefore: subclassStatePtr(subclassState),
			SubclassStateAfter:  subclassStatePtr(next),
		}}
	case *pb.DaggerheartApplySubclassFeatureRequest_GiftedPerformer:
		if !hasUnlockedSubclassRank(profile, "subclass.troubadour", "foundation") {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "gifted_performer requires Troubadour")
		}
		maxUses := 1
		if hasUnlockedSubclassRank(profile, "subclass.troubadour", "mastery") {
			maxUses = 2
		}
		switch strings.TrimSpace(feature.GiftedPerformer.GetSong()) {
		case "relaxing_song":
			if subclassState.GiftedPerformerRelaxingSongUses >= maxUses {
				return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "relaxing_song already used the maximum times this long rest")
			}
			next := subclassState
			next.GiftedPerformerRelaxingSongUses++
			targetIDs := append(uniqueTrimmedIDs(feature.GiftedPerformer.GetTargetCharacterIds()), strings.TrimSpace(in.GetCharacterId()))
			payload.Feature = "gifted_performer_relaxing_song"
			payload.Targets = append(payload.Targets, daggerheart.SubclassFeatureTargetPatchPayload{
				CharacterID:         ids.CharacterID(in.GetCharacterId()),
				SubclassStateBefore: subclassStatePtr(subclassState),
				SubclassStateAfter:  subclassStatePtr(next),
			})
			for _, targetID := range uniqueTrimmedIDs(targetIDs) {
				targetState, loadErr := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, targetID)
				if loadErr != nil {
					return daggerheart.SubclassFeatureApplyPayload{}, grpcerror.HandleDomainError(loadErr)
				}
				targetProfile, loadErr := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, targetID)
				if loadErr != nil {
					return daggerheart.SubclassFeatureApplyPayload{}, grpcerror.HandleDomainError(loadErr)
				}
				payload.Targets = append(payload.Targets, daggerheart.SubclassFeatureTargetPatchPayload{
					CharacterID: ids.CharacterID(targetID),
					HPBefore:    intPtr(targetState.Hp),
					HPAfter:     intPtr(min(targetState.Hp+1, targetProfile.HpMax)),
				})
			}
		case "heartbreaking_song":
			if subclassState.GiftedPerformerHeartbreakingSongUses >= maxUses {
				return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "heartbreaking_song already used the maximum times this long rest")
			}
			next := subclassState
			next.GiftedPerformerHeartbreakingSongUses++
			targetIDs := append(uniqueTrimmedIDs(feature.GiftedPerformer.GetTargetCharacterIds()), strings.TrimSpace(in.GetCharacterId()))
			payload.Feature = "gifted_performer_heartbreaking_song"
			payload.Targets = append(payload.Targets, daggerheart.SubclassFeatureTargetPatchPayload{
				CharacterID:         ids.CharacterID(in.GetCharacterId()),
				SubclassStateBefore: subclassStatePtr(subclassState),
				SubclassStateAfter:  subclassStatePtr(next),
			})
			for _, targetID := range uniqueTrimmedIDs(targetIDs) {
				targetState, loadErr := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, targetID)
				if loadErr != nil {
					return daggerheart.SubclassFeatureApplyPayload{}, grpcerror.HandleDomainError(loadErr)
				}
				payload.Targets = append(payload.Targets, daggerheart.SubclassFeatureTargetPatchPayload{
					CharacterID: ids.CharacterID(targetID),
					HopeBefore:  intPtr(targetState.Hope),
					HopeAfter:   intPtr(min(targetState.Hope+1, targetState.HopeMax)),
				})
			}
		case "epic_song":
			if subclassState.GiftedPerformerEpicSongUses >= maxUses {
				return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "epic_song already used the maximum times this long rest")
			}
			targetID := strings.TrimSpace(feature.GiftedPerformer.GetTargetId())
			if targetID == "" {
				return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "gifted_performer.target_id is required for epic_song")
			}
			next := subclassState
			next.GiftedPerformerEpicSongUses++
			payload.Feature = "gifted_performer_epic_song"
			payload.Targets = []daggerheart.SubclassFeatureTargetPatchPayload{{
				CharacterID:         ids.CharacterID(in.GetCharacterId()),
				SubclassStateBefore: subclassStatePtr(subclassState),
				SubclassStateAfter:  subclassStatePtr(next),
			}}
			if feature.GiftedPerformer.GetTargetIsAdversary() {
				target, loadErr := h.deps.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, targetID)
				if loadErr != nil {
					return daggerheart.SubclassFeatureApplyPayload{}, grpcerror.HandleDomainError(loadErr)
				}
				before, normalizeErr := daggerheart.NormalizeConditionStates(projectionConditionStatesToDomain(target.Conditions))
				if normalizeErr != nil {
					return daggerheart.SubclassFeatureApplyPayload{}, grpcerror.Internal("normalize adversary conditions", normalizeErr)
				}
				after, added, diffErr := addStandardConditionState(before, daggerheart.ConditionVulnerable)
				if diffErr != nil {
					return daggerheart.SubclassFeatureApplyPayload{}, grpcerror.Internal("add vulnerable condition", diffErr)
				}
				payload.AdversaryConditionTargets = []daggerheart.AdversaryConditionChangePayload{{
					AdversaryID:      ids.AdversaryID(targetID),
					ConditionsBefore: before,
					ConditionsAfter:  after,
					Added:            added,
				}}
			} else {
				target, loadErr := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, targetID)
				if loadErr != nil {
					return daggerheart.SubclassFeatureApplyPayload{}, grpcerror.HandleDomainError(loadErr)
				}
				before, normalizeErr := daggerheart.NormalizeConditionStates(projectionConditionStatesToDomain(target.Conditions))
				if normalizeErr != nil {
					return daggerheart.SubclassFeatureApplyPayload{}, grpcerror.Internal("normalize character conditions", normalizeErr)
				}
				after, added, diffErr := addStandardConditionState(before, daggerheart.ConditionVulnerable)
				if diffErr != nil {
					return daggerheart.SubclassFeatureApplyPayload{}, grpcerror.Internal("add vulnerable condition", diffErr)
				}
				payload.CharacterConditionTargets = []daggerheart.ConditionChangePayload{{
					CharacterID:      ids.CharacterID(targetID),
					ConditionsBefore: before,
					ConditionsAfter:  after,
					Added:            added,
				}}
			}
		default:
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "gifted_performer.song is required")
		}
	case *pb.DaggerheartApplySubclassFeatureRequest_ContactsEverywhere:
		if !hasUnlockedSubclassRank(profile, "subclass.syndicate", "specialization") {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "contacts_everywhere requires Syndicate specialization")
		}
		maxUses := 1
		if hasUnlockedSubclassRank(profile, "subclass.syndicate", "mastery") {
			maxUses = 3
		}
		if subclassState.ContactsEverywhereUsesThisSession >= maxUses {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "contacts_everywhere already used the maximum times this session")
		}
		next := subclassState
		next.ContactsEverywhereUsesThisSession++
		next.ContactsEverywhereActionDieBonus = 0
		next.ContactsEverywhereDamageDiceBonusCount = 0
		switch strings.TrimSpace(feature.ContactsEverywhere.GetOption()) {
		case "next_action_bonus":
			next.ContactsEverywhereActionDieBonus = 3
		case "next_damage_bonus":
			next.ContactsEverywhereDamageDiceBonusCount = 2
		default:
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "contacts_everywhere.option is unsupported")
		}
		payload.Feature = "contacts_everywhere"
		payload.Targets = []daggerheart.SubclassFeatureTargetPatchPayload{{
			CharacterID:         ids.CharacterID(in.GetCharacterId()),
			SubclassStateBefore: subclassStatePtr(subclassState),
			SubclassStateAfter:  subclassStatePtr(next),
		}}
	case *pb.DaggerheartApplySubclassFeatureRequest_SparingTouch:
		if !hasUnlockedSubclassRank(profile, "subclass.divine_wielder", "foundation") {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "sparing_touch requires Divine Wielder")
		}
		maxUses := 1
		if hasUnlockedSubclassRank(profile, "subclass.divine_wielder", "mastery") {
			maxUses = 2
		}
		if subclassState.SparingTouchUsesThisLongRest >= maxUses {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "sparing_touch already used the maximum times this long rest")
		}
		targetID := strings.TrimSpace(feature.SparingTouch.GetTargetCharacterId())
		if targetID == "" {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "sparing_touch.target_character_id is required")
		}
		targetState, loadErr := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, targetID)
		if loadErr != nil {
			return daggerheart.SubclassFeatureApplyPayload{}, grpcerror.HandleDomainError(loadErr)
		}
		targetProfile, loadErr := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, targetID)
		if loadErr != nil {
			return daggerheart.SubclassFeatureApplyPayload{}, grpcerror.HandleDomainError(loadErr)
		}
		next := subclassState
		next.SparingTouchUsesThisLongRest++
		payload.Feature = "sparing_touch"
		payload.Targets = []daggerheart.SubclassFeatureTargetPatchPayload{{
			CharacterID:         ids.CharacterID(in.GetCharacterId()),
			SubclassStateBefore: subclassStatePtr(subclassState),
			SubclassStateAfter:  subclassStatePtr(next),
		}}
		switch strings.TrimSpace(feature.SparingTouch.GetClear()) {
		case "hp":
			payload.Targets = append(payload.Targets, daggerheart.SubclassFeatureTargetPatchPayload{
				CharacterID: ids.CharacterID(targetID),
				HPBefore:    intPtr(targetState.Hp),
				HPAfter:     intPtr(min(targetState.Hp+2, targetProfile.HpMax)),
			})
		case "stress":
			payload.Targets = append(payload.Targets, daggerheart.SubclassFeatureTargetPatchPayload{
				CharacterID:  ids.CharacterID(targetID),
				StressBefore: intPtr(targetState.Stress),
				StressAfter:  intPtr(max(targetState.Stress-2, 0)),
			})
		default:
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "sparing_touch.clear must be hp or stress")
		}
	case *pb.DaggerheartApplySubclassFeatureRequest_Elementalist:
		if !hasUnlockedSubclassRank(profile, "subclass.elemental_origin", "foundation") {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "elementalist requires Elemental Origin")
		}
		if state.Hope < 1 {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "insufficient hope")
		}
		next := subclassState
		next.ElementalistActionBonus = 0
		next.ElementalistDamageBonus = 0
		switch strings.TrimSpace(feature.Elementalist.GetBonus()) {
		case "action":
			next.ElementalistActionBonus = 2
		case "damage":
			next.ElementalistDamageBonus = 3
		default:
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "elementalist.bonus must be action or damage")
		}
		payload.Feature = "elementalist"
		payload.Targets = []daggerheart.SubclassFeatureTargetPatchPayload{{
			CharacterID:         ids.CharacterID(in.GetCharacterId()),
			HopeBefore:          intPtr(state.Hope),
			HopeAfter:           intPtr(state.Hope - 1),
			SubclassStateBefore: subclassStatePtr(subclassState),
			SubclassStateAfter:  subclassStatePtr(next),
		}}
	case *pb.DaggerheartApplySubclassFeatureRequest_Transcendence:
		if !hasUnlockedSubclassRank(profile, "subclass.elemental_origin", "specialization") {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "transcendence requires Elemental Origin specialization")
		}
		if subclassState.TranscendenceActive {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "transcendence is already active")
		}
		bonuses := uniqueTrimmedIDs(feature.Transcendence.GetBonuses())
		if len(bonuses) != 2 {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "transcendence requires exactly two distinct bonuses")
		}
		next := subclassState
		next.TranscendenceActive = true
		for _, bonus := range bonuses {
			switch bonus {
			case "severe_threshold":
				next.TranscendenceSevereThresholdBonus = 4
			case "trait":
				trait := strings.TrimSpace(feature.Transcendence.GetTrait())
				if trait == "" {
					return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "transcendence.trait is required when choosing the trait bonus")
				}
				next.TranscendenceTraitBonusTarget = trait
				next.TranscendenceTraitBonusValue = 1
			case "proficiency":
				next.TranscendenceProficiencyBonus = 1
			case "evasion":
				next.TranscendenceEvasionBonus = 2
			default:
				return daggerheart.SubclassFeatureApplyPayload{}, status.Errorf(codes.InvalidArgument, "transcendence bonus %q is unsupported", bonus)
			}
		}
		payload.Feature = "transcendence"
		payload.Targets = []daggerheart.SubclassFeatureTargetPatchPayload{{
			CharacterID:         ids.CharacterID(in.GetCharacterId()),
			SubclassStateBefore: subclassStatePtr(subclassState),
			SubclassStateAfter:  subclassStatePtr(next),
		}}
	case *pb.DaggerheartApplySubclassFeatureRequest_VanishingAct:
		if !hasUnlockedSubclassRank(profile, "subclass.nightwalker", "specialization") {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "vanishing_act requires Nightwalker specialization")
		}
		beforeConditions, normalizeErr := daggerheart.NormalizeConditionStates(projectionConditionStatesToDomain(state.Conditions))
		if normalizeErr != nil {
			return daggerheart.SubclassFeatureApplyPayload{}, grpcerror.Internal("normalize character conditions", normalizeErr)
		}
		afterConditions, _, addErr := addStandardConditionStateWithOptions(
			beforeConditions,
			daggerheart.ConditionCloaked,
			daggerheart.WithConditionSource("subclass_feature:vanishing_act", ""),
			daggerheart.WithConditionClearTriggers(
				daggerheart.ConditionClearTriggerShortRest,
				daggerheart.ConditionClearTriggerLongRest,
			),
		)
		if addErr != nil {
			return daggerheart.SubclassFeatureApplyPayload{}, grpcerror.Internal("add cloaked condition", addErr)
		}
		afterConditions = daggerheart.RemoveConditionCode(afterConditions, daggerheart.ConditionRestrained)
		afterConditions, normalizeErr = daggerheart.NormalizeConditionStates(afterConditions)
		if normalizeErr != nil {
			return daggerheart.SubclassFeatureApplyPayload{}, grpcerror.Internal("normalize vanishing act conditions", normalizeErr)
		}
		addedConditions, removedConditions := daggerheart.DiffConditionStates(beforeConditions, afterConditions)
		payload.Feature = "vanishing_act"
		payload.Targets = []daggerheart.SubclassFeatureTargetPatchPayload{{
			CharacterID:         ids.CharacterID(in.GetCharacterId()),
			StressBefore:        intPtr(state.Stress),
			StressAfter:         intPtr(state.Stress + 1),
			SubclassStateBefore: subclassStatePtr(subclassState),
			SubclassStateAfter:  subclassStatePtr(subclassState),
		}}
		if len(addedConditions) > 0 || len(removedConditions) > 0 {
			payload.CharacterConditionTargets = []daggerheart.ConditionChangePayload{{
				CharacterID:      ids.CharacterID(in.GetCharacterId()),
				ConditionsBefore: beforeConditions,
				ConditionsAfter:  afterConditions,
				Added:            addedConditions,
				Removed:          removedConditions,
			}}
		}
	case *pb.DaggerheartApplySubclassFeatureRequest_ClarityOfNature:
		if !hasUnlockedSubclassRank(profile, "subclass.warden_of_renewal", "foundation") {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "clarity_of_nature requires Warden of Renewal")
		}
		if subclassState.ClarityOfNatureUsedThisLongRest {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "clarity_of_nature already used this long rest")
		}
		totalClear := 0
		for _, target := range feature.ClarityOfNature.GetTargets() {
			totalClear += int(target.GetStressClear())
		}
		if totalClear <= 0 || totalClear > profile.Instinct {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "clarity_of_nature stress clear must be positive and no greater than instinct")
		}
		next := subclassState
		next.ClarityOfNatureUsedThisLongRest = true
		payload.Feature = "clarity_of_nature"
		payload.Targets = append(payload.Targets, daggerheart.SubclassFeatureTargetPatchPayload{
			CharacterID:         ids.CharacterID(in.GetCharacterId()),
			SubclassStateBefore: subclassStatePtr(subclassState),
			SubclassStateAfter:  subclassStatePtr(next),
		})
		for _, target := range feature.ClarityOfNature.GetTargets() {
			targetID := strings.TrimSpace(target.GetCharacterId())
			if targetID == "" || target.GetStressClear() <= 0 {
				return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "clarity_of_nature targets require character_id and positive stress_clear")
			}
			targetState, loadErr := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, targetID)
			if loadErr != nil {
				return daggerheart.SubclassFeatureApplyPayload{}, grpcerror.HandleDomainError(loadErr)
			}
			payload.Targets = append(payload.Targets, daggerheart.SubclassFeatureTargetPatchPayload{
				CharacterID:  ids.CharacterID(targetID),
				StressBefore: intPtr(targetState.Stress),
				StressAfter:  intPtr(max(targetState.Stress-int(target.GetStressClear()), 0)),
			})
		}
	case *pb.DaggerheartApplySubclassFeatureRequest_Regeneration:
		if !hasUnlockedSubclassRank(profile, "subclass.warden_of_renewal", "specialization") {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "regeneration requires Warden of Renewal specialization")
		}
		if state.Hope < 3 {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "insufficient hope")
		}
		targetID := strings.TrimSpace(feature.Regeneration.GetTargetCharacterId())
		if targetID == "" {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "regeneration.target_character_id is required")
		}
		clearHP := int(feature.Regeneration.GetClearHp())
		if clearHP < 1 || clearHP > 4 {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "regeneration.clear_hp must be in range 1..4")
		}
		targetState, loadErr := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, targetID)
		if loadErr != nil {
			return daggerheart.SubclassFeatureApplyPayload{}, grpcerror.HandleDomainError(loadErr)
		}
		targetProfile, loadErr := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, targetID)
		if loadErr != nil {
			return daggerheart.SubclassFeatureApplyPayload{}, grpcerror.HandleDomainError(loadErr)
		}
		payload.Feature = "regeneration"
		payload.Targets = []daggerheart.SubclassFeatureTargetPatchPayload{
			{
				CharacterID: ids.CharacterID(in.GetCharacterId()),
				HopeBefore:  intPtr(state.Hope),
				HopeAfter:   intPtr(state.Hope - 3),
			},
			{
				CharacterID: ids.CharacterID(targetID),
				HPBefore:    intPtr(targetState.Hp),
				HPAfter:     intPtr(min(targetState.Hp+clearHP, targetProfile.HpMax)),
			},
		}
	case *pb.DaggerheartApplySubclassFeatureRequest_WardensProtection:
		if !hasUnlockedSubclassRank(profile, "subclass.warden_of_renewal", "mastery") {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "wardens_protection requires Warden of Renewal mastery")
		}
		if subclassState.WardensProtectionUsedThisLongRest {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "wardens_protection already used this long rest")
		}
		if state.Hope < 2 {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "insufficient hope")
		}
		targetIDs := uniqueTrimmedIDs(feature.WardensProtection.GetTargetCharacterIds())
		if len(targetIDs) == 0 || len(targetIDs) > 4 {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "wardens_protection requires 1 to 4 target_character_ids")
		}
		next := subclassState
		next.WardensProtectionUsedThisLongRest = true
		payload.Feature = "wardens_protection"
		payload.Targets = append(payload.Targets, daggerheart.SubclassFeatureTargetPatchPayload{
			CharacterID:         ids.CharacterID(in.GetCharacterId()),
			HopeBefore:          intPtr(state.Hope),
			HopeAfter:           intPtr(state.Hope - 2),
			SubclassStateBefore: subclassStatePtr(subclassState),
			SubclassStateAfter:  subclassStatePtr(next),
		})
		for _, targetID := range targetIDs {
			targetState, loadErr := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, targetID)
			if loadErr != nil {
				return daggerheart.SubclassFeatureApplyPayload{}, grpcerror.HandleDomainError(loadErr)
			}
			targetProfile, loadErr := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, targetID)
			if loadErr != nil {
				return daggerheart.SubclassFeatureApplyPayload{}, grpcerror.HandleDomainError(loadErr)
			}
			payload.Targets = append(payload.Targets, daggerheart.SubclassFeatureTargetPatchPayload{
				CharacterID: ids.CharacterID(targetID),
				HPBefore:    intPtr(targetState.Hp),
				HPAfter:     intPtr(min(targetState.Hp+2, targetProfile.HpMax)),
			})
		}
	case *pb.DaggerheartApplySubclassFeatureRequest_ElementalIncarnation:
		if !hasUnlockedSubclassRank(profile, "subclass.warden_of_the_elements", "foundation") {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "elemental_incarnation requires Warden of the Elements")
		}
		channel := strings.TrimSpace(feature.ElementalIncarnation.GetChannel())
		switch channel {
		case daggerheart.ElementalChannelAir, daggerheart.ElementalChannelEarth, daggerheart.ElementalChannelFire, daggerheart.ElementalChannelWater:
		default:
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "elemental_incarnation.channel is invalid")
		}
		next := subclassState
		next.ElementalChannel = channel
		payload.Feature = "elemental_incarnation"
		payload.Targets = []daggerheart.SubclassFeatureTargetPatchPayload{{
			CharacterID:         ids.CharacterID(in.GetCharacterId()),
			StressBefore:        intPtr(state.Stress),
			StressAfter:         intPtr(state.Stress + 1),
			SubclassStateBefore: subclassStatePtr(subclassState),
			SubclassStateAfter:  subclassStatePtr(next),
		}}
	case *pb.DaggerheartApplySubclassFeatureRequest_RousingSpeech:
		if !hasUnlockedSubclassRank(profile, "subclass.wordsmith", "foundation") {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "rousing_speech requires Wordsmith")
		}
		if subclassState.RousingSpeechUsedThisLongRest {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "rousing_speech already used this long rest")
		}
		targetIDs := uniqueTrimmedIDs(feature.RousingSpeech.GetTargetCharacterIds())
		if len(targetIDs) == 0 {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "rousing_speech requires at least one target_character_id")
		}
		next := subclassState
		next.RousingSpeechUsedThisLongRest = true
		payload.Feature = "rousing_speech"
		payload.Targets = append(payload.Targets, daggerheart.SubclassFeatureTargetPatchPayload{
			CharacterID:         ids.CharacterID(in.GetCharacterId()),
			SubclassStateBefore: subclassStatePtr(subclassState),
			SubclassStateAfter:  subclassStatePtr(next),
		})
		for _, targetID := range targetIDs {
			targetState, loadErr := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, targetID)
			if loadErr != nil {
				return daggerheart.SubclassFeatureApplyPayload{}, grpcerror.HandleDomainError(loadErr)
			}
			payload.Targets = append(payload.Targets, daggerheart.SubclassFeatureTargetPatchPayload{
				CharacterID:  ids.CharacterID(targetID),
				StressBefore: intPtr(targetState.Stress),
				StressAfter:  intPtr(max(targetState.Stress-2, 0)),
			})
		}
	case *pb.DaggerheartApplySubclassFeatureRequest_Nemesis:
		if !hasUnlockedSubclassRank(profile, "subclass.vengeance", "mastery") {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "nemesis requires Vengeance mastery")
		}
		if state.Hope < 2 {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "insufficient hope")
		}
		targetID := strings.TrimSpace(feature.Nemesis.GetAdversaryId())
		if targetID == "" {
			return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "nemesis.adversary_id is required")
		}
		next := subclassState
		next.NemesisTargetID = targetID
		payload.Feature = "nemesis"
		payload.Targets = []daggerheart.SubclassFeatureTargetPatchPayload{{
			CharacterID:         ids.CharacterID(in.GetCharacterId()),
			HopeBefore:          intPtr(state.Hope),
			HopeAfter:           intPtr(state.Hope - 2),
			SubclassStateBefore: subclassStatePtr(subclassState),
			SubclassStateAfter:  subclassStatePtr(next),
		}}
	default:
		return daggerheart.SubclassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "subclass feature is required")
	}

	return payload, nil
}

func subclassStateFromProjection(state *projectionstore.DaggerheartSubclassState) daggerheart.CharacterSubclassState {
	if state == nil {
		return daggerheart.CharacterSubclassState{}
	}
	return daggerheart.CharacterSubclassState{
		BattleRitualUsedThisLongRest:           state.BattleRitualUsedThisLongRest,
		GiftedPerformerRelaxingSongUses:        state.GiftedPerformerRelaxingSongUses,
		GiftedPerformerEpicSongUses:            state.GiftedPerformerEpicSongUses,
		GiftedPerformerHeartbreakingSongUses:   state.GiftedPerformerHeartbreakingSongUses,
		ContactsEverywhereUsesThisSession:      state.ContactsEverywhereUsesThisSession,
		ContactsEverywhereActionDieBonus:       state.ContactsEverywhereActionDieBonus,
		ContactsEverywhereDamageDiceBonusCount: state.ContactsEverywhereDamageDiceBonusCount,
		SparingTouchUsesThisLongRest:           state.SparingTouchUsesThisLongRest,
		ElementalistActionBonus:                state.ElementalistActionBonus,
		ElementalistDamageBonus:                state.ElementalistDamageBonus,
		TranscendenceActive:                    state.TranscendenceActive,
		TranscendenceTraitBonusTarget:          state.TranscendenceTraitBonusTarget,
		TranscendenceTraitBonusValue:           state.TranscendenceTraitBonusValue,
		TranscendenceProficiencyBonus:          state.TranscendenceProficiencyBonus,
		TranscendenceEvasionBonus:              state.TranscendenceEvasionBonus,
		TranscendenceSevereThresholdBonus:      state.TranscendenceSevereThresholdBonus,
		ClarityOfNatureUsedThisLongRest:        state.ClarityOfNatureUsedThisLongRest,
		ElementalChannel:                       state.ElementalChannel,
		NemesisTargetID:                        state.NemesisTargetID,
		RousingSpeechUsedThisLongRest:          state.RousingSpeechUsedThisLongRest,
		WardensProtectionUsedThisLongRest:      state.WardensProtectionUsedThisLongRest,
	}.Normalized()
}

func subclassStatePtr(state daggerheart.CharacterSubclassState) *daggerheart.CharacterSubclassState {
	normalized := state.Normalized()
	return &normalized
}

func hasUnlockedSubclassRank(profile projectionstore.DaggerheartCharacterProfile, subclassID, minimum string) bool {
	order := map[string]int{"foundation": 1, "specialization": 2, "mastery": 3}
	want := order[strings.TrimSpace(minimum)]
	if want == 0 {
		return false
	}
	for _, track := range profile.SubclassTracks {
		if strings.TrimSpace(track.SubclassID) != strings.TrimSpace(subclassID) {
			continue
		}
		if order[strings.TrimSpace(string(track.Rank))] >= want {
			return true
		}
	}
	if strings.TrimSpace(profile.SubclassID) == strings.TrimSpace(subclassID) && want <= 1 {
		return true
	}
	return false
}

func uniqueTrimmedIDs(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func projectionConditionStatesToDomain(current []projectionstore.DaggerheartConditionState) []daggerheart.ConditionState {
	if len(current) == 0 {
		return nil
	}
	next := make([]daggerheart.ConditionState, 0, len(current))
	for _, condition := range current {
		entry := daggerheart.ConditionState{
			ID:       condition.ID,
			Class:    daggerheart.ConditionClass(condition.Class),
			Standard: condition.Standard,
			Code:     condition.Code,
			Label:    condition.Label,
			Source:   condition.Source,
			SourceID: condition.SourceID,
		}
		for _, trigger := range condition.ClearTriggers {
			entry.ClearTriggers = append(entry.ClearTriggers, daggerheart.ConditionClearTrigger(trigger))
		}
		next = append(next, entry)
	}
	return next
}

func addStandardConditionState(current []daggerheart.ConditionState, condition string) ([]daggerheart.ConditionState, []daggerheart.ConditionState, error) {
	return addStandardConditionStateWithOptions(current, condition)
}

func addStandardConditionStateWithOptions(
	current []daggerheart.ConditionState,
	condition string,
	options ...func(*daggerheart.ConditionState),
) ([]daggerheart.ConditionState, []daggerheart.ConditionState, error) {
	next := append([]daggerheart.ConditionState(nil), current...)
	entry, err := daggerheart.StandardConditionState(condition, options...)
	if err != nil {
		return nil, nil, err
	}
	next = append(next, entry)
	normalized, err := daggerheart.NormalizeConditionStates(next)
	if err != nil {
		return nil, nil, err
	}
	added, _ := daggerheart.DiffConditionStates(current, normalized)
	return normalized, added, nil
}
