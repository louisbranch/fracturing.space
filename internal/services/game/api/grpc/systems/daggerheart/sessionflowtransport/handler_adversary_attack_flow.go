package sessionflowtransport

import (
	"context"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) SessionAdversaryAttackFlow(ctx context.Context, in *pb.SessionAdversaryAttackFlowRequest) (*pb.SessionAdversaryAttackFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session adversary attack flow request is required")
	}
	if err := h.requireAdversaryAttackFlowDeps(); err != nil {
		return nil, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}
	adversaryID, err := validate.RequiredID(in.GetAdversaryId(), "adversary id")
	if err != nil {
		return nil, err
	}
	targetIDs := workflowtransport.NormalizeTargets(append([]string{in.GetTargetId()}, in.GetTargetIds()...))
	if len(targetIDs) == 0 {
		return nil, status.Error(codes.InvalidArgument, "target id is required")
	}
	targetID := targetIDs[0]
	if in.GetRequireDefenseChoice() && len(targetIDs) > 1 {
		return nil, status.Error(codes.FailedPrecondition, "multi-target defense choice is not supported")
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

	adversary, err := h.deps.LoadAdversary(ctx, campaignID, adversaryID, sessionID)
	if err != nil {
		return nil, grpcerror.LookupErrorContext(ctx, err, "load adversary failed", "adversary not found")
	}
	entry, err := h.deps.LoadAdversaryEntry(ctx, adversary.AdversaryEntryID)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "adversary entry %q not found", adversary.AdversaryEntryID)
	}
	var featureRule *rules.AdversaryFeatureRule
	featureID := strings.TrimSpace(in.GetFeatureId())
	if featureID != "" {
		feature, ok := findAdversaryEntryFeature(entry, featureID)
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "adversary feature %q was not found on adversary entry %q", featureID, adversary.AdversaryEntryID)
		}
		automationStatus, rule := rules.ResolveAdversaryFeatureRuntime(feature)
		if automationStatus != rules.AdversaryFeatureAutomationStatusSupported || rule == nil {
			return nil, status.Errorf(codes.FailedPrecondition, "adversary feature %q is not runtime-supported", featureID)
		}
		featureRule = rule
	}
	attack := rules.AdversaryStandardAttack(entry, adversary.HP, adversary.HPMax)
	targetProfile, err := h.deps.LoadCharacterProfile(ctx, campaignID, targetID)
	if err != nil {
		return nil, err
	}
	targetState, err := h.deps.LoadCharacterState(ctx, campaignID, targetID)
	if err != nil {
		return nil, err
	}
	targetClassState := classStateFromProjection(targetState.ClassState)
	targetSubclassState := subclassStateFromProjection(targetState.SubclassState)
	effectiveDifficulty := int(in.GetDifficulty())
	effectiveAdvantage := int(in.GetAdvantage())
	effectiveDisadvantage := int(in.GetDisadvantage())
	subclassRules, err := h.activeSubclassRuleSummary(ctx, targetProfile)
	if err != nil {
		return nil, err
	}
	effectiveDifficulty += targetClassState.EvasionBonusUntilHitOrRest
	if targetClassState.ActiveBeastform != nil {
		effectiveDifficulty += targetClassState.ActiveBeastform.EvasionBonus
	}
	if subclassRules.EvasionBonusWhileHopeAtLeast > 0 && targetState.Hope >= subclassRules.EvasionBonusRequiredHopeMin {
		effectiveDifficulty += subclassRules.EvasionBonusWhileHopeAtLeast
	}
	effectiveDifficulty += targetSubclassState.TranscendenceEvasionBonus
	for _, mod := range targetState.StatModifiers {
		if mod.Target == "evasion" {
			effectiveDifficulty += mod.Delta
		}
	}
	if featureRule != nil && featureRule.Kind == rules.AdversaryFeatureRuleKindDifficultyBonusWhileActive && hasActiveAdversaryFeatureState(adversary, featureID) {
		effectiveDifficulty += featureRule.DifficultyBonus
	}
	if featureRule != nil && featureRule.Kind == rules.AdversaryFeatureRuleKindHiddenUntilNextAttack && hasActiveAdversaryFeatureState(adversary, featureID) {
		effectiveAdvantage++
	}
	if featureRule != nil && featureRule.Kind == rules.AdversaryFeatureRuleKindFocusTargetDisadvantage {
		if focusedID := focusedTargetIDForFeature(adversary, featureID); focusedID != "" && focusedID == targetID {
			effectiveDisadvantage++
		}
	}
	var targetArmor *contentstore.DaggerheartArmor
	if equippedArmorID := strings.TrimSpace(targetProfile.EquippedArmorID); equippedArmorID != "" {
		if h.deps.LoadArmor == nil {
			return nil, status.Error(codes.Internal, "armor loader is not configured")
		}
		armor, err := h.deps.LoadArmor(ctx, equippedArmorID)
		if err != nil {
			return nil, err
		}
		targetArmor = &armor
	}
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	selectedArmorReaction := in.GetTargetArmorReaction()
	explicitDefenseDecision := selectedArmorReaction != nil
	if decision := in.GetTargetDefenseDecision(); decision != nil {
		explicitDefenseDecision = decision.GetDeclineArmorReaction() || decision.GetArmorReaction() != nil
		if reaction := decision.GetArmorReaction(); reaction != nil {
			selectedArmorReaction = reaction
		} else if decision.GetDeclineArmorReaction() {
			selectedArmorReaction = nil
		}
	}
	if targetArmor != nil {
		armorRules := rules.EffectiveArmorRules(targetArmor)
		if in.GetRequireDefenseChoice() && !explicitDefenseDecision {
			optionCodes := make([]string, 0, 2)
			if armorRules.ShiftingAttackDisadvantage > 0 {
				optionCodes = append(optionCodes, "armor.shifting")
			}
			if armorRules.TimeslowingEvasionBonusDieSides > 0 {
				optionCodes = append(optionCodes, "armor.timeslowing")
			}
			if len(optionCodes) > 0 {
				return &pb.SessionAdversaryAttackFlowResponse{
					ChoiceRequired: &pb.DaggerheartCombatChoiceRequired{
						Stage:       pb.DaggerheartCombatChoiceStage_DAGGERHEART_COMBAT_CHOICE_STAGE_INCOMING_ATTACK_DEFENSE,
						CharacterId: targetID,
						OptionCodes: append(optionCodes, "armor.decline"),
						Reason:      "incoming attack defense choice is required before rolling the adversary attack",
					},
				}, nil
			}
		}
		if armorRules.BurningAttackerStress > 0 && isMeleeAttackRange(attack.Range) {
			if h.deps.ExecuteAdversaryUpdate == nil {
				return nil, status.Error(codes.Internal, "adversary update executor is not configured")
			}
			nextStress := adversary.Stress + armorRules.BurningAttackerStress
			if nextStress > adversary.StressMax {
				nextStress = adversary.StressMax
			}
			if nextStress != adversary.Stress {
				if err := h.deps.ExecuteAdversaryUpdate(ctx, AdversaryUpdateInput{
					CampaignID:    campaignID,
					SessionID:     sessionID,
					SceneID:       strings.TrimSpace(in.GetSceneId()),
					RequestID:     requestID,
					InvocationID:  invocationID,
					Adversary:     adversary,
					UpdatedStress: nextStress,
					Source:        "armor.burning",
				}); err != nil {
					return nil, err
				}
				adversary.Stress = nextStress
			}
		}
		if armorReaction := selectedArmorReaction; armorReaction != nil {
			switch reaction := armorReaction.GetReaction().(type) {
			case *pb.DaggerheartIncomingAttackArmorReaction_Shifting:
				_ = reaction
				if h.deps.ExecuteCharacterStatePatch == nil {
					return nil, status.Error(codes.Internal, "character state patch executor is not configured")
				}
				if armorRules.ShiftingAttackDisadvantage <= 0 {
					return nil, status.Error(codes.FailedPrecondition, "equipped armor does not support shifting")
				}
				armorBefore, armorAfter, ok := rules.ArmorTotalAfterBaseSpend(targetState, targetProfile.ArmorMax)
				if !ok {
					return nil, status.Error(codes.FailedPrecondition, "insufficient equipped armor")
				}
				if err := h.deps.ExecuteCharacterStatePatch(ctx, CharacterStatePatchInput{
					CampaignID:   campaignID,
					SessionID:    sessionID,
					SceneID:      strings.TrimSpace(in.GetSceneId()),
					RequestID:    requestID,
					InvocationID: invocationID,
					CharacterID:  targetID,
					Source:       "armor.shifting",
					ArmorBefore:  &armorBefore,
					ArmorAfter:   &armorAfter,
				}); err != nil {
					return nil, err
				}
				targetState.Armor = armorAfter
				effectiveDisadvantage += armorRules.ShiftingAttackDisadvantage
			case *pb.DaggerheartIncomingAttackArmorReaction_Timeslowing:
				if h.deps.ExecuteCharacterStatePatch == nil {
					return nil, status.Error(codes.Internal, "character state patch executor is not configured")
				}
				if h.deps.SeedFunc == nil {
					return nil, status.Error(codes.Internal, "seed generator is not configured")
				}
				if armorRules.TimeslowingEvasionBonusDieSides <= 0 {
					return nil, status.Error(codes.FailedPrecondition, "equipped armor does not support timeslowing")
				}
				armorBefore, armorAfter, ok := rules.ArmorTotalAfterBaseSpend(targetState, targetProfile.ArmorMax)
				if !ok {
					return nil, status.Error(codes.FailedPrecondition, "insufficient equipped armor")
				}
				bonus, err := h.rollArmorFeatureDie(reaction.Timeslowing.GetRng(), armorRules.TimeslowingEvasionBonusDieSides)
				if err != nil {
					return nil, err
				}
				if err := h.deps.ExecuteCharacterStatePatch(ctx, CharacterStatePatchInput{
					CampaignID:   campaignID,
					SessionID:    sessionID,
					SceneID:      strings.TrimSpace(in.GetSceneId()),
					RequestID:    requestID,
					InvocationID: invocationID,
					CharacterID:  targetID,
					Source:       "armor.timeslowing",
					ArmorBefore:  &armorBefore,
					ArmorAfter:   &armorAfter,
				}); err != nil {
					return nil, err
				}
				targetState.Armor = armorAfter
				effectiveDifficulty += bonus
			}
		}
	}

	rollModifiers := []*pb.ActionRollModifier{{
		Source: "attack_modifier",
		Value:  int32(entry.AttackModifier),
	}}
	if adversary.PendingExperience != nil {
		rollModifiers = append(rollModifiers, &pb.ActionRollModifier{
			Source: "adversary_experience",
			Value:  int32(adversary.PendingExperience.Modifier),
		})
	}
	rollResp, err := h.deps.SessionAdversaryAttackRoll(ctx, &pb.SessionAdversaryAttackRollRequest{
		CampaignId:   campaignID,
		SessionId:    sessionID,
		AdversaryId:  adversaryID,
		Modifiers:    rollModifiers,
		Advantage:    int32(effectiveAdvantage),
		Disadvantage: int32(effectiveDisadvantage),
		Rng:          in.GetAttackRng(),
	})
	if err != nil {
		return nil, err
	}

	ctxWithMeta := workflowtransport.WithCampaignSessionMetadata(ctx, campaignID, sessionID)
	attackOutcome, err := h.deps.ApplyAdversaryAttackOutcome(ctxWithMeta, &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId:  sessionID,
		RollSeq:    rollResp.GetRollSeq(),
		Targets:    targetIDs,
		Difficulty: int32(effectiveDifficulty),
	})
	if err != nil {
		return nil, err
	}

	response := &pb.SessionAdversaryAttackFlowResponse{
		AttackRoll:    rollResp,
		AttackOutcome: attackOutcome,
	}
	if (adversary.PendingExperience != nil || (featureRule != nil && featureRule.Kind == rules.AdversaryFeatureRuleKindHiddenUntilNextAttack && hasActiveAdversaryFeatureState(adversary, featureID))) && h.deps.ExecuteAdversaryUpdate != nil {
		nextFeatureStates := adversary.FeatureStates
		if featureRule != nil && featureRule.Kind == rules.AdversaryFeatureRuleKindHiddenUntilNextAttack {
			nextFeatureStates = clearAdversaryFeatureState(nextFeatureStates, featureID)
		}
		if err := h.deps.ExecuteAdversaryUpdate(ctx, AdversaryUpdateInput{
			CampaignID:             campaignID,
			SessionID:              sessionID,
			SceneID:                strings.TrimSpace(in.GetSceneId()),
			RequestID:              requestID,
			InvocationID:           invocationID,
			Adversary:              adversary,
			UpdatedStress:          adversary.Stress,
			UpdatedFeatureStates:   nextFeatureStates,
			ClearPendingExperience: adversary.PendingExperience != nil,
			Source:                 "adversary.attack.consume_state",
		}); err != nil {
			return nil, err
		}
		adversary.FeatureStates = nextFeatureStates
		adversary.PendingExperience = nil
	}
	if attackOutcome.GetResult() == nil || !attackOutcome.GetResult().GetSuccess() {
		return response, nil
	}
	if targetClassState.EvasionBonusUntilHitOrRest > 0 {
		if h.deps.ExecuteCharacterStatePatch == nil {
			return nil, status.Error(codes.Internal, "character state patch executor is not configured")
		}
		clearedClassState := targetClassState
		clearedClassState.EvasionBonusUntilHitOrRest = 0
		if err := h.deps.ExecuteCharacterStatePatch(ctx, CharacterStatePatchInput{
			CampaignID:       campaignID,
			SessionID:        sessionID,
			SceneID:          strings.TrimSpace(in.GetSceneId()),
			RequestID:        requestID,
			InvocationID:     invocationID,
			CharacterID:      targetID,
			Source:           "class.rogues_dodge.hit",
			ClassStateBefore: classStatePtr(targetClassState),
			ClassStateAfter:  classStatePtr(clearedClassState),
		}); err != nil {
			return nil, err
		}
	}
	gmFearDelta := 0
	if featureRule != nil {
		switch featureRule.Kind {
		case rules.AdversaryFeatureRuleKindMomentumGainFearOnSuccessfulAttack:
			gmFearDelta += featureRule.FearGain
		case rules.AdversaryFeatureRuleKindTerrifyingHopeLossOnSuccessfulAttack:
			gmFearDelta += featureRule.FearGain
			if h.deps.ExecuteCharacterStatePatch == nil {
				return nil, status.Error(codes.Internal, "character state patch executor is not configured")
			}
			for _, affectedTargetID := range targetIDs {
				affectedState, err := h.deps.LoadCharacterState(ctx, campaignID, affectedTargetID)
				if err != nil {
					return nil, err
				}
				if affectedState.Hope <= 0 {
					continue
				}
				nextHope := affectedState.Hope - featureRule.HopeLoss
				if nextHope < 0 {
					nextHope = 0
				}
				if nextHope == affectedState.Hope {
					continue
				}
				if err := h.deps.ExecuteCharacterStatePatch(ctx, CharacterStatePatchInput{
					CampaignID:   campaignID,
					SessionID:    sessionID,
					SceneID:      strings.TrimSpace(in.GetSceneId()),
					RequestID:    requestID,
					InvocationID: invocationID,
					CharacterID:  affectedTargetID,
					Source:       "adversary.terrifying",
					HopeBefore:   intPtr(affectedState.Hope),
					HopeAfter:    intPtr(nextHope),
				}); err != nil {
					return nil, err
				}
			}
		}
	}
	if gmFearDelta > 0 && h.deps.AdjustGMFear != nil {
		if err := h.deps.AdjustGMFear(ctx, GMFearAdjustInput{
			CampaignID:   campaignID,
			SessionID:    sessionID,
			SceneID:      strings.TrimSpace(in.GetSceneId()),
			RequestID:    requestID,
			InvocationID: invocationID,
			Delta:        gmFearDelta,
			Reason:       "adversary_feature",
		}); err != nil {
			return nil, err
		}
	}
	if len(attack.DamageDice) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "adversary attack damage dice are not configured")
	}
	if featureRule != nil {
		switch featureRule.Kind {
		case rules.AdversaryFeatureRuleKindDamageReplacementOnAdvantagedAttack:
			if effectiveAdvantage > 0 && len(featureRule.DamageDice) > 0 {
				attack.DamageDice = featureRule.DamageDice
				attack.DamageBonus = featureRule.DamageBonus
				attack.DamageType = featureRule.DamageType
			}
		case rules.AdversaryFeatureRuleKindConditionalDamageReplacementWithContributor:
			if len(in.GetContributorAdversaryIds()) > 0 && len(featureRule.DamageDice) > 0 {
				attack.DamageDice = featureRule.DamageDice
				attack.DamageBonus = featureRule.DamageBonus
				attack.DamageType = featureRule.DamageType
			}
		}
	}

	critical := attackOutcome.GetResult().GetCrit() || in.GetDamageCritical()
	damageRoll, err := h.deps.SessionDamageRoll(ctx, &pb.SessionDamageRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		CharacterId: adversaryID,
		Dice:        toProtoDamageDice(attack.DamageDice),
		Modifier:    int32(attack.DamageBonus),
		Critical:    critical,
		Rng:         in.GetDamageRng(),
	})
	if err != nil {
		return nil, err
	}
	if featureRule != nil && featureRule.Kind == rules.AdversaryFeatureRuleKindGroupAttack {
		damageRoll.Total = damageRoll.GetTotal() * int32(1+len(workflowtransport.NormalizeTargets(in.GetContributorAdversaryIds())))
	}

	sourceCharacterIDs := workflowtransport.NormalizeTargets(in.GetDamage().GetSourceCharacterIds())
	sourceCharacterIDs = append(sourceCharacterIDs, adversaryID)
	sourceCharacterIDs = workflowtransport.NormalizeTargets(sourceCharacterIDs)

	damageReq := &pb.DaggerheartDamageRequest{
		Amount:             damageRoll.GetTotal(),
		DamageType:         attackDamageTypeOrRequest(attack.DamageType, in.GetDamage().GetDamageType()),
		ResistPhysical:     in.GetDamage().GetResistPhysical(),
		ResistMagic:        in.GetDamage().GetResistMagic(),
		ImmunePhysical:     in.GetDamage().GetImmunePhysical(),
		ImmuneMagic:        in.GetDamage().GetImmuneMagic(),
		Direct:             in.GetDamage().GetDirect(),
		MassiveDamage:      in.GetDamage().GetMassiveDamage(),
		Source:             in.GetDamage().GetSource(),
		SourceCharacterIds: sourceCharacterIDs,
	}

	damageApplications := make([]*pb.DaggerheartApplyDamageResponse, 0, len(targetIDs))
	for _, damageTargetID := range targetIDs {
		applyDamage, err := h.deps.ApplyDamage(ctxWithMeta, &pb.DaggerheartApplyDamageRequest{
			CampaignId:              campaignID,
			CharacterId:             damageTargetID,
			Damage:                  damageReq,
			RollSeq:                 &damageRoll.RollSeq,
			RequireDamageRoll:       in.GetRequireDamageRoll(),
			MitigationDecision:      in.GetTargetMitigationDecision(),
			RequireMitigationChoice: in.GetRequireDefenseChoice(),
		})
		if err != nil {
			return nil, err
		}
		if applyDamage.GetChoiceRequired() != nil {
			response.DamageRoll = damageRoll
			response.ChoiceRequired = applyDamage.GetChoiceRequired()
			return response, nil
		}
		damageApplications = append(damageApplications, applyDamage)
	}
	applyDamage := damageApplications[0]
	if featureRule != nil && featureRule.Kind == rules.AdversaryFeatureRuleKindArmorShredOnSuccessfulAttack && targetState.Armor > 0 && h.deps.ExecuteCharacterStatePatch != nil {
		nextArmor := targetState.Armor - 1
		if nextArmor < 0 {
			nextArmor = 0
		}
		if nextArmor != targetState.Armor {
			if err := h.deps.ExecuteCharacterStatePatch(ctx, CharacterStatePatchInput{
				CampaignID:   campaignID,
				SessionID:    sessionID,
				SceneID:      strings.TrimSpace(in.GetSceneId()),
				RequestID:    requestID,
				InvocationID: invocationID,
				CharacterID:  targetID,
				Source:       "adversary.armor_shred",
				ArmorBefore:  intPtr(targetState.Armor),
				ArmorAfter:   intPtr(nextArmor),
			}); err != nil {
				return nil, err
			}
		}
	}

	if isMeleeAttackRange(attack.Range) &&
		strings.EqualFold(strings.TrimSpace(targetSubclassState.ElementalChannel), daggerheartstate.ElementalChannelFire) &&
		characterStateWasDamaged(targetState, applyDamage.GetState()) {
		if h.deps.ApplyAdversaryDamage == nil {
			return nil, status.Error(codes.Internal, "apply adversary damage handler is not configured")
		}
		retaliationRoll, err := h.deps.SessionDamageRoll(ctx, &pb.SessionDamageRollRequest{
			CampaignId:  campaignID,
			SessionId:   sessionID,
			SceneId:     strings.TrimSpace(in.GetSceneId()),
			CharacterId: targetID,
			Dice: []*pb.DiceSpec{{
				Sides: 10,
				Count: 1,
			}},
		})
		if err != nil {
			return nil, err
		}
		if _, err := h.deps.ApplyAdversaryDamage(ctxWithMeta, &pb.DaggerheartApplyAdversaryDamageRequest{
			CampaignId:  campaignID,
			SceneId:     strings.TrimSpace(in.GetSceneId()),
			AdversaryId: adversaryID,
			Damage: &pb.DaggerheartDamageRequest{
				Amount:             retaliationRoll.GetTotal(),
				DamageType:         pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC,
				Source:             "subclass.elemental_incarnation.fire",
				SourceCharacterIds: []string{targetID},
			},
			RollSeq: &retaliationRoll.RollSeq,
		}); err != nil {
			return nil, err
		}
	}

	response.DamageRoll = damageRoll
	response.DamageApplied = applyDamage
	response.DamageApplications = damageApplications
	return response, nil
}
