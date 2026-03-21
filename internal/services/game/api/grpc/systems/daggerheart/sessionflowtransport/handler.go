package sessionflowtransport

import (
	"context"
	"errors"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler owns the Daggerheart session gameplay flow transport surface behind a
// narrow callback-based seam.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a session flow transport handler.
func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

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
		if strings.EqualFold(strings.TrimSpace(attackerSubclassState.ElementalChannel), daggerheart.ElementalChannelWater) && attackRange == pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_MELEE {
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
	if actionSubclassState.ElementalChannel == daggerheart.ElementalChannelAir && strings.EqualFold(strings.TrimSpace(attackTrait), "agility") {
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
		rules := daggerheart.EffectiveArmorRules(&armor)
		if attackRange == pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_MELEE && rules.SharpDamageBonusDieSides > 0 {
			damageDice = append(damageDice, &pb.DiceSpec{Sides: int32(rules.SharpDamageBonusDieSides), Count: 1})
		}
	}
	if strings.EqualFold(attackOutcome.GetResult().GetFlavor(), "FEAR") && subclassRules.BonusMagicDamageDiceCount > 0 && subclassRules.BonusMagicDamageDieSides > 0 {
		damageDice = append(damageDice, &pb.DiceSpec{
			Sides: int32(subclassRules.BonusMagicDamageDieSides),
			Count: int32(subclassRules.BonusMagicDamageDiceCount),
		})
	}
	if hasCondition(sessionCharacterConditions(attackerState), daggerheart.ConditionVulnerable) {
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
			strings.EqualFold(strings.TrimSpace(attackerSubclassState.ElementalChannel), daggerheart.ElementalChannelWater) &&
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
			featureID, rule, ok := firstAdversaryFeatureRuleByKind(entry, daggerheart.AdversaryFeatureRuleKindRetaliatoryDamageOnCloseHit)
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
// and its outcome handlers.
func (h *Handler) SessionReactionFlow(ctx context.Context, in *pb.SessionReactionFlowRequest) (*pb.SessionReactionFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session reaction flow request is required")
	}
	if h.deps.SessionActionRoll == nil || h.deps.ApplyRollOutcome == nil || h.deps.ApplyReactionOutcome == nil {
		return nil, status.Error(codes.Internal, "session workflow dependencies are not configured")
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
	actorID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
	trait, err := validate.RequiredID(in.GetTrait(), "trait")
	if err != nil {
		return nil, err
	}
	modifiers := append([]*pb.ActionRollModifier{}, in.GetModifiers()...)
	advantage := in.GetAdvantage()
	if h.deps.LoadCharacterState != nil {
		state, err := h.deps.LoadCharacterState(ctx, campaignID, actorID)
		if err != nil {
			return nil, err
		}
		subclassState := subclassStateFromProjection(state.SubclassState)
		if subclassState.TranscendenceTraitBonusValue > 0 && strings.EqualFold(strings.TrimSpace(subclassState.TranscendenceTraitBonusTarget), strings.TrimSpace(trait)) {
			modifiers = append(modifiers, &pb.ActionRollModifier{
				Source: "subclass_transcendence_trait",
				Value:  int32(subclassState.TranscendenceTraitBonusValue),
			})
		}
		if subclassState.ElementalChannel == daggerheart.ElementalChannelAir && strings.EqualFold(strings.TrimSpace(trait), "agility") {
			advantage++
		}
	}

	rollResp, err := h.deps.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:           campaignID,
		SessionId:            sessionID,
		SceneId:              sceneID,
		CharacterId:          actorID,
		Trait:                trait,
		RollKind:             pb.RollKind_ROLL_KIND_REACTION,
		Difficulty:           in.GetDifficulty(),
		Modifiers:            modifiers,
		Advantage:            advantage,
		Disadvantage:         in.GetDisadvantage(),
		Rng:                  in.GetReactionRng(),
		ReplaceHopeWithArmor: in.GetReplaceHopeWithArmor(),
	})
	if err != nil {
		return nil, err
	}

	ctxWithMeta := workflowtransport.WithCampaignSessionMetadata(ctx, campaignID, sessionID)
	rollOutcome, err := h.deps.ApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		SceneId:   sceneID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		return nil, err
	}
	reactionOutcome, err := h.deps.ApplyReactionOutcome(ctxWithMeta, &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: sessionID,
		SceneId:   sceneID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		return nil, err
	}

	return &pb.SessionReactionFlowResponse{
		ActionRoll:      rollResp,
		RollOutcome:     rollOutcome,
		ReactionOutcome: reactionOutcome,
	}, nil
}

// SessionGroupActionFlow runs the group action orchestration by resolving each
// supporter roll, then the leader roll and its resulting outcome.
func (h *Handler) SessionGroupActionFlow(ctx context.Context, in *pb.SessionGroupActionFlowRequest) (*pb.SessionGroupActionFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session group action flow request is required")
	}
	if h.deps.SessionActionRoll == nil || h.deps.ApplyRollOutcome == nil {
		return nil, status.Error(codes.Internal, "session workflow dependencies are not configured")
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
	leaderID, err := validate.RequiredID(in.GetLeaderCharacterId(), "leader character id")
	if err != nil {
		return nil, err
	}
	leaderTrait, err := validate.RequiredID(in.GetLeaderTrait(), "leader trait")
	if err != nil {
		return nil, err
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
		supporterID, err := validate.RequiredID(supporter.GetCharacterId(), "supporter character id")
		if err != nil {
			return nil, err
		}
		supporterTrait, err := validate.RequiredID(supporter.GetTrait(), "supporter trait")
		if err != nil {
			return nil, err
		}

		rollResp, err := h.deps.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
			CampaignId:  campaignID,
			SessionId:   sessionID,
			SceneId:     sceneID,
			CharacterId: supporterID,
			Trait:       supporterTrait,
			RollKind:    pb.RollKind_ROLL_KIND_REACTION,
			Difficulty:  in.GetDifficulty(),
			Modifiers:   supporter.GetModifiers(),
			Context:     supporter.GetContext(),
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

	leaderRoll, err := h.deps.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
		CharacterId: leaderID,
		Trait:       leaderTrait,
		RollKind:    pb.RollKind_ROLL_KIND_ACTION,
		Difficulty:  in.GetDifficulty(),
		Modifiers:   leaderModifiers,
		Context:     in.GetLeaderContext(),
		Rng:         in.GetLeaderRng(),
	})
	if err != nil {
		return nil, err
	}

	ctxWithMeta := workflowtransport.WithCampaignSessionMetadata(ctx, campaignID, sessionID)
	leaderOutcome, err := h.deps.ApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		SceneId:   sceneID,
		RollSeq:   leaderRoll.GetRollSeq(),
	})
	if err != nil {
		return nil, err
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

// SessionTagTeamFlow runs the tag-team orchestration by resolving both action
// rolls and then applying the chosen result to the combined targets.
func (h *Handler) SessionTagTeamFlow(ctx context.Context, in *pb.SessionTagTeamFlowRequest) (*pb.SessionTagTeamFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session tag team flow request is required")
	}
	if h.deps.SessionActionRoll == nil || h.deps.ApplyRollOutcome == nil {
		return nil, status.Error(codes.Internal, "session workflow dependencies are not configured")
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
	firstID, err := validate.RequiredID(first.GetCharacterId(), "first character id")
	if err != nil {
		return nil, err
	}
	secondID, err := validate.RequiredID(second.GetCharacterId(), "second character id")
	if err != nil {
		return nil, err
	}
	if firstID == secondID {
		return nil, status.Error(codes.InvalidArgument, "tag team participants must be distinct")
	}
	firstTrait, err := validate.RequiredID(first.GetTrait(), "first trait")
	if err != nil {
		return nil, err
	}
	secondTrait, err := validate.RequiredID(second.GetTrait(), "second trait")
	if err != nil {
		return nil, err
	}
	selectedID, err := validate.RequiredID(in.GetSelectedCharacterId(), "selected character id")
	if err != nil {
		return nil, err
	}
	if selectedID != firstID && selectedID != secondID {
		return nil, status.Error(codes.InvalidArgument, "selected character id must match a participant")
	}

	firstRoll, err := h.deps.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
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
	secondRoll, err := h.deps.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
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

	ctxWithMeta := workflowtransport.WithCampaignSessionMetadata(ctx, campaignID, sessionID)
	selectedOutcome, err := h.deps.ApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		SceneId:   sceneID,
		RollSeq:   selectedRoll.GetRollSeq(),
		Targets:   []string{firstID, secondID},
	})
	if err != nil {
		return nil, err
	}

	return &pb.SessionTagTeamFlowResponse{
		FirstRoll:           firstRoll,
		SecondRoll:          secondRoll,
		SelectedOutcome:     selectedOutcome,
		SelectedCharacterId: selectedID,
		SelectedRollSeq:     selectedRoll.GetRollSeq(),
	}, nil
}

// SessionAdversaryAttackFlow runs the adversary attack orchestration by
// composing the adversary roll, outcome, and optional damage application.
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
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "adversary not found")
		}
		return nil, status.Error(codes.Internal, "load adversary failed")
	}
	entry, err := h.deps.LoadAdversaryEntry(ctx, adversary.AdversaryEntryID)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "adversary entry %q not found", adversary.AdversaryEntryID)
	}
	var featureRule *daggerheart.AdversaryFeatureRule
	featureID := strings.TrimSpace(in.GetFeatureId())
	if featureID != "" {
		feature, ok := findAdversaryEntryFeature(entry, featureID)
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "adversary feature %q was not found on adversary entry %q", featureID, adversary.AdversaryEntryID)
		}
		automationStatus, rule := daggerheart.ResolveAdversaryFeatureRuntime(feature)
		if automationStatus != daggerheart.AdversaryFeatureAutomationStatusSupported || rule == nil {
			return nil, status.Errorf(codes.FailedPrecondition, "adversary feature %q is not runtime-supported", featureID)
		}
		featureRule = rule
	}
	attack := daggerheart.AdversaryStandardAttack(entry, adversary.HP, adversary.HPMax)
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
	if featureRule != nil && featureRule.Kind == daggerheart.AdversaryFeatureRuleKindDifficultyBonusWhileActive && hasActiveAdversaryFeatureState(adversary, featureID) {
		effectiveDifficulty += featureRule.DifficultyBonus
	}
	if featureRule != nil && featureRule.Kind == daggerheart.AdversaryFeatureRuleKindHiddenUntilNextAttack && hasActiveAdversaryFeatureState(adversary, featureID) {
		effectiveAdvantage++
	}
	if featureRule != nil && featureRule.Kind == daggerheart.AdversaryFeatureRuleKindFocusTargetDisadvantage {
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
	if targetArmor != nil {
		rules := daggerheart.EffectiveArmorRules(targetArmor)
		if rules.BurningAttackerStress > 0 && isMeleeAttackRange(attack.Range) {
			if h.deps.ExecuteAdversaryUpdate == nil {
				return nil, status.Error(codes.Internal, "adversary update executor is not configured")
			}
			nextStress := adversary.Stress + rules.BurningAttackerStress
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
		if armorReaction := in.GetTargetArmorReaction(); armorReaction != nil {
			switch reaction := armorReaction.GetReaction().(type) {
			case *pb.DaggerheartIncomingAttackArmorReaction_Shifting:
				_ = reaction
				if h.deps.ExecuteCharacterStatePatch == nil {
					return nil, status.Error(codes.Internal, "character state patch executor is not configured")
				}
				if rules.ShiftingAttackDisadvantage <= 0 {
					return nil, status.Error(codes.FailedPrecondition, "equipped armor does not support shifting")
				}
				armorBefore, armorAfter, ok := daggerheart.ArmorTotalAfterBaseSpend(targetState, targetProfile.ArmorMax)
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
				effectiveDisadvantage += rules.ShiftingAttackDisadvantage
			case *pb.DaggerheartIncomingAttackArmorReaction_Timeslowing:
				if h.deps.ExecuteCharacterStatePatch == nil {
					return nil, status.Error(codes.Internal, "character state patch executor is not configured")
				}
				if h.deps.SeedFunc == nil {
					return nil, status.Error(codes.Internal, "seed generator is not configured")
				}
				if rules.TimeslowingEvasionBonusDieSides <= 0 {
					return nil, status.Error(codes.FailedPrecondition, "equipped armor does not support timeslowing")
				}
				armorBefore, armorAfter, ok := daggerheart.ArmorTotalAfterBaseSpend(targetState, targetProfile.ArmorMax)
				if !ok {
					return nil, status.Error(codes.FailedPrecondition, "insufficient equipped armor")
				}
				bonus, err := h.rollArmorFeatureDie(reaction.Timeslowing.GetRng(), rules.TimeslowingEvasionBonusDieSides)
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
	if (adversary.PendingExperience != nil || (featureRule != nil && featureRule.Kind == daggerheart.AdversaryFeatureRuleKindHiddenUntilNextAttack && hasActiveAdversaryFeatureState(adversary, featureID))) && h.deps.ExecuteAdversaryUpdate != nil {
		nextFeatureStates := adversary.FeatureStates
		if featureRule != nil && featureRule.Kind == daggerheart.AdversaryFeatureRuleKindHiddenUntilNextAttack {
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
		case daggerheart.AdversaryFeatureRuleKindMomentumGainFearOnSuccessfulAttack:
			gmFearDelta += featureRule.FearGain
		case daggerheart.AdversaryFeatureRuleKindTerrifyingHopeLossOnSuccessfulAttack:
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
		case daggerheart.AdversaryFeatureRuleKindDamageReplacementOnAdvantagedAttack:
			if effectiveAdvantage > 0 && len(featureRule.DamageDice) > 0 {
				attack.DamageDice = featureRule.DamageDice
				attack.DamageBonus = featureRule.DamageBonus
				attack.DamageType = featureRule.DamageType
			}
		case daggerheart.AdversaryFeatureRuleKindConditionalDamageReplacementWithContributor:
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
	if featureRule != nil && featureRule.Kind == daggerheart.AdversaryFeatureRuleKindGroupAttack {
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
			CampaignId:        campaignID,
			CharacterId:       damageTargetID,
			Damage:            damageReq,
			RollSeq:           &damageRoll.RollSeq,
			RequireDamageRoll: in.GetRequireDamageRoll(),
		})
		if err != nil {
			return nil, err
		}
		damageApplications = append(damageApplications, applyDamage)
	}
	applyDamage := damageApplications[0]
	if featureRule != nil && featureRule.Kind == daggerheart.AdversaryFeatureRuleKindArmorShredOnSuccessfulAttack && targetState.Armor > 0 && h.deps.ExecuteCharacterStatePatch != nil {
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
		strings.EqualFold(strings.TrimSpace(targetSubclassState.ElementalChannel), daggerheart.ElementalChannelFire) &&
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

func (h *Handler) requireAttackFlowDeps() error {
	switch {
	case h.deps.SessionActionRoll == nil:
		return status.Error(codes.Internal, "session action roll handler is not configured")
	case h.deps.SessionDamageRoll == nil:
		return status.Error(codes.Internal, "session damage roll handler is not configured")
	case h.deps.ApplyRollOutcome == nil:
		return status.Error(codes.Internal, "roll outcome handler is not configured")
	case h.deps.ApplyAttackOutcome == nil:
		return status.Error(codes.Internal, "attack outcome handler is not configured")
	case h.deps.ApplyDamage == nil:
		return status.Error(codes.Internal, "apply damage handler is not configured")
	case h.deps.LoadCharacterProfile == nil:
		return status.Error(codes.Internal, "character profile loader is not configured")
	case h.deps.LoadCharacterState == nil:
		return status.Error(codes.Internal, "character state loader is not configured")
	case h.deps.LoadSubclass == nil:
		return status.Error(codes.Internal, "subclass loader is not configured")
	case h.deps.LoadArmor == nil:
		return status.Error(codes.Internal, "armor loader is not configured")
	default:
		return nil
	}
}

func findAdversaryEntryFeature(entry contentstore.DaggerheartAdversaryEntry, featureID string) (contentstore.DaggerheartAdversaryFeature, bool) {
	for _, feature := range entry.Features {
		if strings.TrimSpace(feature.ID) == strings.TrimSpace(featureID) {
			return feature, true
		}
	}
	return contentstore.DaggerheartAdversaryFeature{}, false
}

func hasActiveAdversaryFeatureState(adversary projectionstore.DaggerheartAdversary, featureID string) bool {
	for _, featureState := range adversary.FeatureStates {
		if strings.TrimSpace(featureState.FeatureID) == strings.TrimSpace(featureID) &&
			strings.EqualFold(strings.TrimSpace(featureState.Status), "active") {
			return true
		}
	}
	return false
}

// focusedTargetIDForFeature returns the focused target character ID from an
// active adversary feature state, or empty if the feature is not active or
// has no focused target.
func focusedTargetIDForFeature(adversary projectionstore.DaggerheartAdversary, featureID string) string {
	for _, featureState := range adversary.FeatureStates {
		if strings.TrimSpace(featureState.FeatureID) == strings.TrimSpace(featureID) &&
			strings.EqualFold(strings.TrimSpace(featureState.Status), "active") {
			return strings.TrimSpace(featureState.FocusedTargetID)
		}
	}
	return ""
}

func clearAdversaryFeatureState(current []projectionstore.DaggerheartAdversaryFeatureState, featureID string) []projectionstore.DaggerheartAdversaryFeatureState {
	updated := make([]projectionstore.DaggerheartAdversaryFeatureState, 0, len(current))
	for _, featureState := range current {
		if strings.TrimSpace(featureState.FeatureID) == strings.TrimSpace(featureID) {
			continue
		}
		updated = append(updated, featureState)
	}
	return updated
}

func hasReadyAdversaryFeatureState(adversary projectionstore.DaggerheartAdversary, featureID string) bool {
	for _, featureState := range adversary.FeatureStates {
		if strings.TrimSpace(featureState.FeatureID) == strings.TrimSpace(featureID) &&
			strings.EqualFold(strings.TrimSpace(featureState.Status), "ready") {
			return true
		}
	}
	return false
}

func setAdversaryFeatureStateStatus(current []projectionstore.DaggerheartAdversaryFeatureState, featureID, status string) []projectionstore.DaggerheartAdversaryFeatureState {
	updated := make([]projectionstore.DaggerheartAdversaryFeatureState, 0, len(current))
	for _, featureState := range current {
		if strings.TrimSpace(featureState.FeatureID) == strings.TrimSpace(featureID) {
			featureState.Status = strings.TrimSpace(status)
		}
		updated = append(updated, featureState)
	}
	return updated
}

func firstAdversaryFeatureRuleByKind(entry contentstore.DaggerheartAdversaryEntry, kind daggerheart.AdversaryFeatureRuleKind) (string, *daggerheart.AdversaryFeatureRule, bool) {
	for _, feature := range entry.Features {
		automationStatus, rule := daggerheart.ResolveAdversaryFeatureRuntime(feature)
		if automationStatus != daggerheart.AdversaryFeatureAutomationStatusSupported || rule == nil {
			continue
		}
		if rule.Kind == kind {
			return strings.TrimSpace(feature.ID), rule, true
		}
	}
	return "", nil, false
}

func (h *Handler) requireAdversaryAttackFlowDeps() error {
	switch {
	case h.deps.SessionAdversaryAttackRoll == nil:
		return status.Error(codes.Internal, "session adversary attack roll handler is not configured")
	case h.deps.SessionDamageRoll == nil:
		return status.Error(codes.Internal, "session damage roll handler is not configured")
	case h.deps.ApplyAdversaryAttackOutcome == nil:
		return status.Error(codes.Internal, "adversary attack outcome handler is not configured")
	case h.deps.ApplyDamage == nil:
		return status.Error(codes.Internal, "apply damage handler is not configured")
	case h.deps.LoadAdversary == nil:
		return status.Error(codes.Internal, "adversary loader is not configured")
	case h.deps.LoadAdversaryEntry == nil:
		return status.Error(codes.Internal, "adversary entry loader is not configured")
	case h.deps.LoadCharacterProfile == nil:
		return status.Error(codes.Internal, "character profile loader is not configured")
	case h.deps.LoadCharacterState == nil:
		return status.Error(codes.Internal, "character state loader is not configured")
	case h.deps.LoadSubclass == nil:
		return status.Error(codes.Internal, "subclass loader is not configured")
	default:
		return nil
	}
}

func (h *Handler) activeSubclassRuleSummary(ctx context.Context, profile projectionstore.DaggerheartCharacterProfile) (daggerheart.ActiveSubclassRuleSummary, error) {
	if len(profile.SubclassTracks) == 0 {
		return daggerheart.ActiveSubclassRuleSummary{}, nil
	}
	typed := daggerheart.CharacterProfileFromStorage(profile)
	features := make([]contentstore.DaggerheartFeature, 0)
	for _, track := range typed.SubclassTracks {
		subclass, err := h.deps.LoadSubclass(ctx, track.SubclassID)
		if err != nil {
			return daggerheart.ActiveSubclassRuleSummary{}, err
		}
		features = append(features, subclass.FoundationFeatures...)
		switch strings.TrimSpace(track.Rank) {
		case daggerheart.SubclassTrackRankSpecialization:
			features = append(features, subclass.SpecializationFeatures...)
		case daggerheart.SubclassTrackRankMastery:
			features = append(features, subclass.SpecializationFeatures...)
			features = append(features, subclass.MasteryFeatures...)
		}
	}
	return daggerheart.SummarizeActiveSubclassRules(features), nil
}

func sessionCharacterConditions(state projectionstore.DaggerheartCharacterState) []string {
	if len(state.Conditions) == 0 {
		return nil
	}
	conditions := make([]string, 0, len(state.Conditions))
	for _, condition := range state.Conditions {
		code := strings.TrimSpace(condition.Code)
		if code == "" {
			code = strings.TrimSpace(condition.Standard)
		}
		if code == "" {
			continue
		}
		conditions = append(conditions, code)
	}
	return conditions
}

func hasCondition(conditions []string, want string) bool {
	for _, condition := range conditions {
		if strings.TrimSpace(condition) == strings.TrimSpace(want) {
			return true
		}
	}
	return false
}

func toProtoDamageDice(dice []contentstore.DaggerheartDamageDie) []*pb.DiceSpec {
	items := make([]*pb.DiceSpec, 0, len(dice))
	for _, die := range dice {
		items = append(items, &pb.DiceSpec{Sides: int32(die.Sides), Count: int32(die.Count)})
	}
	return items
}

func attackDamageTypeOrRequest(value string, fallback pb.DaggerheartDamageType) pb.DaggerheartDamageType {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "physical":
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL
	case "magic":
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC
	case "mixed":
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED
	case "":
		return fallback
	default:
		return fallback
	}
}

func filteredNearbyAdversaryIDs(values []string, primaryTargetID string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range workflowtransport.NormalizeTargets(values) {
		if strings.TrimSpace(value) == strings.TrimSpace(primaryTargetID) {
			continue
		}
		filtered = append(filtered, value)
	}
	return filtered
}

func characterStateWasDamaged(before projectionstore.DaggerheartCharacterState, after *pb.DaggerheartCharacterState) bool {
	if after == nil {
		return false
	}
	return before.Hp != int(after.GetHp()) ||
		before.Stress != int(after.GetStress()) ||
		before.Armor != int(after.GetArmor())
}

func adversaryStateWasDamaged(before projectionstore.DaggerheartAdversary, after *pb.DaggerheartAdversary) bool {
	if after == nil {
		return false
	}
	return before.HP != int(after.GetHp()) ||
		before.Armor != int(after.GetArmor())
}

func intPtr(value int) *int {
	return &value
}

func classStateFromProjection(state projectionstore.DaggerheartClassState) daggerheart.CharacterClassState {
	return daggerheart.CharacterClassState{
		AttackBonusUntilRest:            state.AttackBonusUntilRest,
		EvasionBonusUntilHitOrRest:      state.EvasionBonusUntilHitOrRest,
		DifficultyPenaltyUntilRest:      state.DifficultyPenaltyUntilRest,
		FocusTargetID:                   state.FocusTargetID,
		ActiveBeastform:                 activeBeastformFromProjection(state.ActiveBeastform),
		StrangePatternsNumber:           state.StrangePatternsNumber,
		RallyDice:                       append([]int(nil), state.RallyDice...),
		PrayerDice:                      append([]int(nil), state.PrayerDice...),
		ChannelRawPowerUsedThisLongRest: state.ChannelRawPowerUsedThisLongRest,
		Unstoppable: daggerheart.CharacterUnstoppableState{
			Active:           state.Unstoppable.Active,
			CurrentValue:     state.Unstoppable.CurrentValue,
			DieSides:         state.Unstoppable.DieSides,
			UsedThisLongRest: state.Unstoppable.UsedThisLongRest,
		},
	}.Normalized()
}

func classStatePtr(state daggerheart.CharacterClassState) *daggerheart.CharacterClassState {
	normalized := state.Normalized()
	return &normalized
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

func (h *Handler) patchSubclassState(
	ctx context.Context,
	campaignID string,
	sessionID string,
	sceneID string,
	requestID string,
	invocationID string,
	characterID string,
	source string,
	before daggerheart.CharacterSubclassState,
	after daggerheart.CharacterSubclassState,
) error {
	normalizedBefore := before.Normalized()
	normalizedAfter := after.Normalized()
	if normalizedBefore == normalizedAfter {
		return nil
	}
	if h.deps.ExecuteCharacterStatePatch == nil {
		return status.Error(codes.Internal, "character state patch executor is not configured")
	}
	return h.deps.ExecuteCharacterStatePatch(ctx, CharacterStatePatchInput{
		CampaignID:          campaignID,
		SessionID:           sessionID,
		SceneID:             sceneID,
		RequestID:           requestID,
		InvocationID:        invocationID,
		CharacterID:         characterID,
		Source:              source,
		SubclassStateBefore: subclassStatePtr(normalizedBefore),
		SubclassStateAfter:  subclassStatePtr(normalizedAfter),
	})
}

func activeBeastformFromProjection(state *projectionstore.DaggerheartActiveBeastformState) *daggerheart.CharacterActiveBeastformState {
	if state == nil {
		return nil
	}
	damageDice := make([]daggerheart.CharacterDamageDie, 0, len(state.DamageDice))
	for _, die := range state.DamageDice {
		damageDice = append(damageDice, daggerheart.CharacterDamageDie{Count: die.Count, Sides: die.Sides})
	}
	return &daggerheart.CharacterActiveBeastformState{
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

func resolveAttackProfile(in *pb.SessionAttackFlowRequest, classState daggerheart.CharacterClassState) (string, []*pb.DiceSpec, int32, pb.DaggerheartAttackRange, bool, error) {
	if in.GetBeastformAttack() != nil {
		active := classState.ActiveBeastform
		if active == nil {
			return "", nil, 0, pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_UNSPECIFIED, false, status.Error(codes.FailedPrecondition, "character is not in beastform")
		}
		damageDice := make([]*pb.DiceSpec, 0, len(active.DamageDice))
		for _, die := range active.DamageDice {
			damageDice = append(damageDice, &pb.DiceSpec{Count: int32(die.Count), Sides: int32(die.Sides)})
		}
		attackRange, err := beastformAttackRangeToProto(active.AttackRange)
		if err != nil {
			return "", nil, 0, pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_UNSPECIFIED, false, err
		}
		return active.AttackTrait, damageDice, int32(active.DamageBonus), attackRange, false, nil
	}
	if classState.ActiveBeastform != nil {
		return "", nil, 0, pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_UNSPECIFIED, false, status.Error(codes.FailedPrecondition, "standard attacks are unavailable while transformed")
	}
	standard := in.GetStandardAttack()
	if standard == nil {
		return "", nil, 0, pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_UNSPECIFIED, false, status.Error(codes.InvalidArgument, "attack_profile is required")
	}
	trait, err := validate.RequiredID(standard.GetTrait(), "trait")
	if err != nil {
		return "", nil, 0, pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_UNSPECIFIED, false, err
	}
	if standard.GetAttackRange() == pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_UNSPECIFIED {
		return "", nil, 0, pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_UNSPECIFIED, false, status.Error(codes.InvalidArgument, "attack_range is required")
	}
	if len(standard.GetDamageDice()) == 0 {
		return "", nil, 0, pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_UNSPECIFIED, false, status.Error(codes.InvalidArgument, "damage_dice are required")
	}
	return trait, append([]*pb.DiceSpec{}, standard.GetDamageDice()...), standard.GetDamageModifier(), standard.GetAttackRange(), standard.GetDamageCritical(), nil
}

func beastformAttackRangeToProto(value string) (pb.DaggerheartAttackRange, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "melee":
		return pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_MELEE, nil
	case "very_close", "very close", "close", "far", "very_far", "very far", "ranged":
		return pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_RANGED, nil
	default:
		return pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_UNSPECIFIED, status.Errorf(codes.FailedPrecondition, "unsupported beastform attack range %q", value)
	}
}

func isMeleeAttackRange(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "melee")
}
