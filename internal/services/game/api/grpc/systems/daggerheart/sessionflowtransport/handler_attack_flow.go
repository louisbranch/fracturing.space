package sessionflowtransport

import (
	"context"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SessionAttackFlow runs the attack workflow by composing a roll, outcome, and
// optional damage application.
func (h *Handler) SessionAttackFlow(ctx context.Context, in *pb.SessionAttackFlowRequest) (*pb.SessionAttackFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session attack flow request is required")
	}
	if err := h.requireAttackFlowDeps(); err != nil {
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
	sceneID := strings.TrimSpace(in.GetSceneId())
	attackerID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
	targetID, err := validate.RequiredID(in.GetTargetId(), "target id")
	if err != nil {
		return nil, err
	}
	if in.GetDamage() == nil {
		return nil, status.Error(codes.InvalidArgument, "damage is required")
	}
	if in.GetDamage().GetDamageType() == pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "damage_type is required")
	}
	attackerState, err := h.deps.LoadCharacterState(ctx, campaignID, attackerID)
	if err != nil {
		return nil, err
	}
	attackerClassState := classStateFromProjection(attackerState.ClassState)
	attackerSubclassState := subclassStateFromProjection(attackerState.SubclassState)
	swapHopeFear := strings.TrimSpace(attackerSubclassState.NemesisTargetID) != "" && strings.TrimSpace(attackerSubclassState.NemesisTargetID) == strings.TrimSpace(targetID)
	actionModifiers := append([]*pb.ActionRollModifier{}, in.GetModifiers()...)
	if attackerClassState.AttackBonusUntilRest > 0 {
		actionModifiers = append(actionModifiers, &pb.ActionRollModifier{
			Source: "class_no_mercy",
			Value:  int32(attackerClassState.AttackBonusUntilRest),
		})
	}
	attackTrait, damageDice, damageModifier, attackRange, damageCritical, err := resolveAttackProfile(in, attackerClassState)
	if err != nil {
		return nil, err
	}
	var (
		primaryTargetAdversary *projectionstore.DaggerheartAdversary
		waterSplashTargets     []projectionstore.DaggerheartAdversary
	)
	if in.GetTargetIsAdversary() {
		if h.deps.LoadAdversary == nil {
			return nil, status.Error(codes.Internal, "adversary loader is not configured")
		}
		adversary, err := h.deps.LoadAdversary(ctx, campaignID, targetID, sessionID)
		if err != nil {
			return nil, err
		}
		primaryTargetAdversary = &adversary
		if strings.EqualFold(strings.TrimSpace(attackerSubclassState.ElementalChannel), daggerheartstate.ElementalChannelWater) && attackRange == pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_MELEE {
			if h.deps.ExecuteAdversaryUpdate == nil {
				return nil, status.Error(codes.Internal, "adversary update executor is not configured")
			}
			for _, nearbyID := range filteredNearbyAdversaryIDs(in.GetNearbyAdversaryIds(), targetID) {
				nearbyAdversary, err := h.deps.LoadAdversary(ctx, campaignID, nearbyID, sessionID)
				if err != nil {
					return nil, err
				}
				waterSplashTargets = append(waterSplashTargets, nearbyAdversary)
			}
		}
	}
	actionSubclassState := attackerSubclassState
	if actionSubclassState.ContactsEverywhereActionDieBonus > 0 {
		actionModifiers = append(actionModifiers, &pb.ActionRollModifier{
			Source: "subclass_contacts_everywhere",
			Value:  int32(actionSubclassState.ContactsEverywhereActionDieBonus),
		})
		actionSubclassState.ContactsEverywhereActionDieBonus = 0
	}
	if actionSubclassState.ElementalistActionBonus > 0 {
		actionModifiers = append(actionModifiers, &pb.ActionRollModifier{
			Source: "subclass_elementalist",
			Value:  int32(actionSubclassState.ElementalistActionBonus),
		})
		actionSubclassState.ElementalistActionBonus = 0
	}
	if actionSubclassState.TranscendenceTraitBonusValue > 0 && strings.EqualFold(strings.TrimSpace(actionSubclassState.TranscendenceTraitBonusTarget), strings.TrimSpace(attackTrait)) {
		actionModifiers = append(actionModifiers, &pb.ActionRollModifier{
			Source: "subclass_transcendence_trait",
			Value:  int32(actionSubclassState.TranscendenceTraitBonusValue),
		})
	}
	attackAdvantage := int32(0)
	if actionSubclassState.ElementalChannel == daggerheartstate.ElementalChannelAir && strings.EqualFold(strings.TrimSpace(attackTrait), "agility") {
		attackAdvantage = 1
	}
	if activeBeastform := attackerClassState.ActiveBeastform; activeBeastform != nil {
		actionModifiers = append(actionModifiers, &pb.ActionRollModifier{
			Source: "beastform_trait_bonus",
			Value:  int32(activeBeastform.TraitBonus),
		})
	}

	rollResp, err := h.deps.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:           campaignID,
		SessionId:            sessionID,
		SceneId:              sceneID,
		CharacterId:          attackerID,
		Trait:                attackTrait,
		RollKind:             pb.RollKind_ROLL_KIND_ACTION,
		Difficulty:           in.GetDifficulty(),
		Modifiers:            actionModifiers,
		Advantage:            attackAdvantage,
		Underwater:           in.GetUnderwater(),
		BreathCountdownId:    in.GetBreathCountdownId(),
		Rng:                  in.GetActionRng(),
		ReplaceHopeWithArmor: in.GetReplaceHopeWithArmor(),
	})
	if err != nil {
		return nil, err
	}
	if err := h.patchSubclassState(ctx, campaignID, sessionID, sceneID, grpcmeta.RequestIDFromContext(ctx), grpcmeta.InvocationIDFromContext(ctx), attackerID, "subclass.attack_roll_consumed", attackerSubclassState, actionSubclassState); err != nil {
		return nil, err
	}
	attackerSubclassState = actionSubclassState

	ctxWithMeta := workflowtransport.WithCampaignSessionMetadata(ctx, campaignID, sessionID)
	rollOutcome, err := h.deps.ApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId:    sessionID,
		SceneId:      sceneID,
		RollSeq:      rollResp.GetRollSeq(),
		SwapHopeFear: swapHopeFear,
	})
	if err != nil {
		return nil, err
	}

	attackOutcome, err := h.deps.ApplyAttackOutcome(ctxWithMeta, &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId:    sessionID,
		SceneId:      sceneID,
		RollSeq:      rollResp.GetRollSeq(),
		Targets:      []string{targetID},
		SwapHopeFear: swapHopeFear,
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
	attackerProfile, err := h.deps.LoadCharacterProfile(ctx, campaignID, attackerID)
	if err != nil {
		return nil, err
	}
	subclassRules, err := h.activeSubclassRuleSummary(ctx, attackerProfile)
	if err != nil {
		return nil, err
	}
	if equippedArmorID := strings.TrimSpace(attackerProfile.EquippedArmorID); equippedArmorID != "" {
		armor, err := h.deps.LoadArmor(ctx, equippedArmorID)
		if err != nil {
			return nil, err
		}
		armorRules := rules.EffectiveArmorRules(&armor)
		if attackRange == pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_MELEE && armorRules.SharpDamageBonusDieSides > 0 {
			damageDice = append(damageDice, &pb.DiceSpec{Sides: int32(armorRules.SharpDamageBonusDieSides), Count: 1})
		}
	}
	if strings.EqualFold(attackOutcome.GetResult().GetFlavor(), "FEAR") && subclassRules.BonusMagicDamageDiceCount > 0 && subclassRules.BonusMagicDamageDieSides > 0 {
		damageDice = append(damageDice, &pb.DiceSpec{
			Sides: int32(subclassRules.BonusMagicDamageDieSides),
			Count: int32(subclassRules.BonusMagicDamageDiceCount),
		})
	}
	if hasCondition(sessionCharacterConditions(attackerState), rules.ConditionVulnerable) {
		switch {
		case subclassRules.BonusDamageWhileVulnerableLevel:
			damageModifier += int32(attackerProfile.Level)
		case subclassRules.BonusDamageWhileVulnerable > 0:
			damageModifier += int32(subclassRules.BonusDamageWhileVulnerable)
		}
	}
	damageSubclassState := attackerSubclassState
	if damageSubclassState.ContactsEverywhereDamageDiceBonusCount > 0 {
		damageDice = append(damageDice, &pb.DiceSpec{
			Sides: 8,
			Count: int32(damageSubclassState.ContactsEverywhereDamageDiceBonusCount),
		})
		damageSubclassState.ContactsEverywhereDamageDiceBonusCount = 0
	}
	if damageSubclassState.ElementalistDamageBonus > 0 {
		damageModifier += int32(damageSubclassState.ElementalistDamageBonus)
		damageSubclassState.ElementalistDamageBonus = 0
	}
	if damageSubclassState.TranscendenceProficiencyBonus > 0 {
		damageModifier += int32(damageSubclassState.TranscendenceProficiencyBonus)
	}

	critical := attackOutcome.GetResult().GetCrit() || damageCritical
	damageRoll, err := h.deps.SessionDamageRoll(ctx, &pb.SessionDamageRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
		CharacterId: attackerID,
		Dice:        damageDice,
		Modifier:    damageModifier,
		Critical:    critical,
		Rng:         in.GetDamageRng(),
	})
	if err != nil {
		return nil, err
	}
	if err := h.patchSubclassState(ctx, campaignID, sessionID, sceneID, grpcmeta.RequestIDFromContext(ctx), grpcmeta.InvocationIDFromContext(ctx), attackerID, "subclass.damage_roll_consumed", attackerSubclassState, damageSubclassState); err != nil {
		return nil, err
	}
	attackerSubclassState = damageSubclassState

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
		SourceCharacterIds: workflowtransport.NormalizeTargets(in.GetDamage().GetSourceCharacterIds()),
	}

	if in.GetTargetIsAdversary() {
		if h.deps.ApplyAdversaryDamage == nil {
			return nil, status.Error(codes.Internal, "apply adversary damage handler is not configured")
		}
		applyAdversaryDamage, err := h.deps.ApplyAdversaryDamage(ctxWithMeta, &pb.DaggerheartApplyAdversaryDamageRequest{
			CampaignId:        campaignID,
			SceneId:           sceneID,
			AdversaryId:       targetID,
			Damage:            damageReq,
			RollSeq:           &damageRoll.RollSeq,
			RequireDamageRoll: in.GetRequireDamageRoll(),
		})
		if err != nil {
			return nil, err
		}
		response.DamageRoll = damageRoll
		response.AdversaryDamageApplied = applyAdversaryDamage
		if attackRange == pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_MELEE &&
			strings.EqualFold(strings.TrimSpace(attackerSubclassState.ElementalChannel), daggerheartstate.ElementalChannelWater) &&
			primaryTargetAdversary != nil &&
			adversaryStateWasDamaged(*primaryTargetAdversary, applyAdversaryDamage.GetAdversary()) {
			requestID := grpcmeta.RequestIDFromContext(ctx)
			invocationID := grpcmeta.InvocationIDFromContext(ctx)
			for _, nearbyAdversary := range waterSplashTargets {
				nextStress := nearbyAdversary.Stress + 1
				if nextStress > nearbyAdversary.StressMax {
					nextStress = nearbyAdversary.StressMax
				}
				if nextStress == nearbyAdversary.Stress {
					continue
				}
				if err := h.deps.ExecuteAdversaryUpdate(ctx, AdversaryUpdateInput{
					CampaignID:    campaignID,
					SessionID:     sessionID,
					SceneID:       sceneID,
					RequestID:     requestID,
					InvocationID:  invocationID,
					Adversary:     nearbyAdversary,
					UpdatedStress: nextStress,
					Source:        "subclass.elemental_incarnation.water",
				}); err != nil {
					return nil, err
				}
			}
		}
		if attackRange == pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_MELEE &&
			primaryTargetAdversary != nil &&
			adversaryStateWasDamaged(*primaryTargetAdversary, applyAdversaryDamage.GetAdversary()) {
			entry, err := h.deps.LoadAdversaryEntry(ctx, primaryTargetAdversary.AdversaryEntryID)
			if err != nil {
				return nil, err
			}
			featureID, rule, ok := firstAdversaryFeatureRuleByKind(entry, rules.AdversaryFeatureRuleKindRetaliatoryDamageOnCloseHit)
			if ok && hasReadyAdversaryFeatureState(*primaryTargetAdversary, featureID) && h.deps.ExecuteAdversaryUpdate != nil {
				retaliationRoll, err := h.deps.SessionDamageRoll(ctx, &pb.SessionDamageRollRequest{
					CampaignId:  campaignID,
					SessionId:   sessionID,
					SceneId:     sceneID,
					CharacterId: targetID,
					Dice:        toProtoDamageDice(rule.DamageDice),
					Modifier:    int32(rule.DamageBonus),
				})
				if err != nil {
					return nil, err
				}
				if _, err := h.deps.ApplyDamage(ctxWithMeta, &pb.DaggerheartApplyDamageRequest{
					CampaignId:  campaignID,
					SceneId:     sceneID,
					CharacterId: attackerID,
					Damage: &pb.DaggerheartDamageRequest{
						Amount:             retaliationRoll.GetTotal(),
						DamageType:         attackDamageTypeOrRequest(rule.DamageType, pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC),
						Source:             "adversary.warding_sphere",
						SourceCharacterIds: []string{targetID},
					},
					RollSeq: &retaliationRoll.RollSeq,
				}); err != nil {
					return nil, err
				}
				if err := h.deps.ExecuteAdversaryUpdate(ctx, AdversaryUpdateInput{
					CampaignID:           campaignID,
					SessionID:            sessionID,
					SceneID:              sceneID,
					RequestID:            grpcmeta.RequestIDFromContext(ctx),
					InvocationID:         grpcmeta.InvocationIDFromContext(ctx),
					Adversary:            *primaryTargetAdversary,
					UpdatedStress:        primaryTargetAdversary.Stress,
					UpdatedFeatureStates: setAdversaryFeatureStateStatus(primaryTargetAdversary.FeatureStates, featureID, "cooldown"),
					Source:               "adversary.warding_sphere",
				}); err != nil {
					return nil, err
				}
			}
		}
		return response, nil
	}
	applyDamage, err := h.deps.ApplyDamage(ctxWithMeta, &pb.DaggerheartApplyDamageRequest{
		CampaignId:        campaignID,
		SceneId:           sceneID,
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

// SessionReactionFlow runs the reaction workflow by composing a reaction roll
