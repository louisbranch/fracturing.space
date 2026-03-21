package sessionflowtransport

import (
	"context"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
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

func firstAdversaryFeatureRuleByKind(entry contentstore.DaggerheartAdversaryEntry, kind rules.AdversaryFeatureRuleKind) (string, *rules.AdversaryFeatureRule, bool) {
	for _, feature := range entry.Features {
		automationStatus, rule := rules.ResolveAdversaryFeatureRuntime(feature)
		if automationStatus != rules.AdversaryFeatureAutomationStatusSupported || rule == nil {
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

func (h *Handler) activeSubclassRuleSummary(ctx context.Context, profile projectionstore.DaggerheartCharacterProfile) (daggerheartstate.ActiveSubclassRuleSummary, error) {
	if len(profile.SubclassTracks) == 0 {
		return daggerheartstate.ActiveSubclassRuleSummary{}, nil
	}
	typed := daggerheartstate.CharacterProfileFromStorage(profile)
	features := make([]contentstore.DaggerheartFeature, 0)
	for _, track := range typed.SubclassTracks {
		subclass, err := h.deps.LoadSubclass(ctx, track.SubclassID)
		if err != nil {
			return daggerheartstate.ActiveSubclassRuleSummary{}, err
		}
		features = append(features, subclass.FoundationFeatures...)
		switch strings.TrimSpace(track.Rank) {
		case daggerheartstate.SubclassTrackRankSpecialization:
			features = append(features, subclass.SpecializationFeatures...)
		case daggerheartstate.SubclassTrackRankMastery:
			features = append(features, subclass.SpecializationFeatures...)
			features = append(features, subclass.MasteryFeatures...)
		}
	}
	return daggerheartstate.SummarizeActiveSubclassRules(features), nil
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

func subclassStateFromProjection(state *projectionstore.DaggerheartSubclassState) daggerheartstate.CharacterSubclassState {
	if state == nil {
		return daggerheartstate.CharacterSubclassState{}
	}
	return daggerheartstate.CharacterSubclassState{
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

func subclassStatePtr(state daggerheartstate.CharacterSubclassState) *daggerheartstate.CharacterSubclassState {
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
	before daggerheartstate.CharacterSubclassState,
	after daggerheartstate.CharacterSubclassState,
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

func resolveAttackProfile(in *pb.SessionAttackFlowRequest, classState daggerheartstate.CharacterClassState) (string, []*pb.DiceSpec, int32, pb.DaggerheartAttackRange, bool, error) {
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
