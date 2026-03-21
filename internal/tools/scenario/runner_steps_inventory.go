package scenario

import (
	"context"
	"fmt"
	"strings"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
)

func (r *Runner) runLevelUpStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("level_up target is required")
	}
	levelAfter := optionalInt(step.Args, "level_after", 0)
	if levelAfter <= 0 {
		return r.failf("level_up level_after is required")
	}

	characterID, err := actorID(state, name)
	if err != nil {
		return err
	}

	// Build advancement entries from the "advancements" list.
	var advancements []*daggerheartv1.DaggerheartLevelUpAdvancement
	rawAdvancements, ok := step.Args["advancements"].([]any)
	if !ok || len(rawAdvancements) == 0 {
		return r.failf("level_up advancements list is required")
	}
	for _, rawAdv := range rawAdvancements {
		advMap, ok := rawAdv.(map[string]any)
		if !ok {
			return r.failf("level_up advancement entry must be a table")
		}
		entry := &daggerheartv1.DaggerheartLevelUpAdvancement{
			Type:            optionalString(advMap, "type", ""),
			Trait:           optionalString(advMap, "trait", ""),
			DomainCardId:    optionalString(advMap, "domain_card_id", ""),
			DomainCardLevel: int32(optionalInt(advMap, "domain_card_level", 0)),
		}
		rawMulticlass, _ := advMap["multiclass"].(map[string]any)
		if rawMulticlass != nil {
			entry.Multiclass = &daggerheartv1.DaggerheartLevelUpMulticlass{
				SecondaryClassId:    optionalString(rawMulticlass, "secondary_class_id", ""),
				SecondarySubclassId: optionalString(rawMulticlass, "secondary_subclass_id", ""),
				SpellcastTrait:      optionalString(rawMulticlass, "spellcast_trait", ""),
				DomainId:            optionalString(rawMulticlass, "domain_id", ""),
			}
		}
		advancements = append(advancements, entry)
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err = r.env.daggerheartClient.ApplyLevelUp(ctxWithSession, &daggerheartv1.DaggerheartApplyLevelUpRequest{
		CampaignId:   state.campaignID,
		CharacterId:  characterID,
		LevelAfter:   int32(levelAfter),
		Advancements: advancements,
	})
	if err != nil {
		return fmt.Errorf("level_up: %w", err)
	}
	if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheartpayload.EventTypeLevelUpApplied); err != nil {
		return err
	}

	profile, err := r.getDaggerheartProfile(ctxWithSession, state, characterID)
	if err != nil {
		return err
	}
	if want := optionalInt(step.Args, "expect_level", 0); want > 0 && int(profile.GetLevel()) != want {
		return r.assertf("level_up level = %d, want %d", profile.GetLevel(), want)
	}
	if want := optionalInt(step.Args, "expect_subclass_track_count", -1); want >= 0 && len(profile.GetSubclassTracks()) != want {
		return r.assertf("level_up subclass_track_count = %d, want %d", len(profile.GetSubclassTracks()), want)
	}
	if want := optionalString(step.Args, "expect_primary_subclass_rank", ""); want != "" {
		rank, ok := findSubclassTrackRank(profile.GetSubclassTracks(), daggerheartv1.DaggerheartSubclassTrackOrigin_DAGGERHEART_SUBCLASS_TRACK_ORIGIN_PRIMARY)
		if !ok {
			return r.assertf("level_up missing primary subclass track")
		}
		if got := normalizeSubclassTrackRank(rank); got != normalizeScenarioKey(want) {
			return r.assertf("level_up primary subclass rank = %q, want %q", got, normalizeScenarioKey(want))
		}
	}
	if want := optionalString(step.Args, "expect_multiclass_subclass_id", ""); want != "" {
		track, ok := findSubclassTrack(profile.GetSubclassTracks(), daggerheartv1.DaggerheartSubclassTrackOrigin_DAGGERHEART_SUBCLASS_TRACK_ORIGIN_MULTICLASS)
		if !ok {
			return r.assertf("level_up missing multiclass subclass track")
		}
		if track.GetSubclassId() != want {
			return r.assertf("level_up multiclass subclass_id = %q, want %q", track.GetSubclassId(), want)
		}
	}
	for _, want := range readStringSlice(step.Args, "expect_active_feature_ids") {
		if !profileHasActiveSubclassFeature(profile, want) {
			return r.assertf("level_up missing active subclass feature %q", want)
		}
	}
	return nil
}

func (r *Runner) runClassFeatureStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("class_feature target is required")
	}
	feature := normalizeScenarioKey(requiredString(step.Args, "feature"))
	if feature == "" {
		return r.failf("class_feature feature is required")
	}
	characterID, err := actorID(state, name)
	if err != nil {
		return err
	}
	expectedSpec, expectedBefore, err := r.captureExpectedDeltas(ctx, state, step.Args, name)
	if err != nil {
		return err
	}
	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	req := &daggerheartv1.DaggerheartApplyClassFeatureRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
		SessionId:   state.sessionID,
		SceneId:     state.activeSceneID,
	}
	switch feature {
	case "frontline_tank":
		req.Feature = &daggerheartv1.DaggerheartApplyClassFeatureRequest_FrontlineTank{
			FrontlineTank: &daggerheartv1.DaggerheartFrontlineTankFeature{},
		}
	case "rogues_dodge":
		req.Feature = &daggerheartv1.DaggerheartApplyClassFeatureRequest_RoguesDodge{
			RoguesDodge: &daggerheartv1.DaggerheartRoguesDodgeFeature{},
		}
	case "no_mercy":
		req.Feature = &daggerheartv1.DaggerheartApplyClassFeatureRequest_NoMercy{
			NoMercy: &daggerheartv1.DaggerheartNoMercyFeature{},
		}
	case "strange_patterns_choice":
		number := optionalInt(step.Args, "number", 0)
		if number < 1 || number > 12 {
			return r.failf("class_feature strange_patterns_choice number must be in range 1..12")
		}
		req.Feature = &daggerheartv1.DaggerheartApplyClassFeatureRequest_StrangePatternsChoice{
			StrangePatternsChoice: &daggerheartv1.DaggerheartStrangePatternsChoice{Number: int32(number)},
		}
	default:
		return r.failf("unsupported class_feature %q", feature)
	}

	response, err := r.env.daggerheartClient.ApplyClassFeature(withSessionID(ctx, state.sessionID), req)
	if err != nil {
		return fmt.Errorf("class_feature: %w", err)
	}
	if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheartpayload.EventTypeCharacterStatePatched); err != nil {
		return err
	}

	currentState := response.GetState()
	if currentState == nil {
		var err error
		currentState, err = r.getCharacterState(ctx, state, characterID)
		if err != nil {
			return err
		}
	}
	if want, ok := readInt(step.Args, "expect_hope"); ok && int(currentState.GetHope()) != want {
		return r.assertf("class_feature hope = %d, want %d", currentState.GetHope(), want)
	}
	if want, ok := readInt(step.Args, "expect_armor"); ok && int(currentState.GetArmor()) != want {
		return r.assertf("class_feature armor = %d, want %d", currentState.GetArmor(), want)
	}
	classState := currentState.GetClassState()
	if want, ok := readInt(step.Args, "expect_attack_bonus_until_rest"); ok && int(classState.GetAttackBonusUntilRest()) != want {
		return r.assertf("class_feature attack_bonus_until_rest = %d, want %d", classState.GetAttackBonusUntilRest(), want)
	}
	if want, ok := readInt(step.Args, "expect_evasion_bonus_until_hit_or_rest"); ok && int(classState.GetEvasionBonusUntilHitOrRest()) != want {
		return r.assertf("class_feature evasion_bonus_until_hit_or_rest = %d, want %d", classState.GetEvasionBonusUntilHitOrRest(), want)
	}
	if want, ok := readInt(step.Args, "expect_strange_patterns_number"); ok && int(classState.GetStrangePatternsNumber()) != want {
		return r.assertf("class_feature strange_patterns_number = %d, want %d", classState.GetStrangePatternsNumber(), want)
	}
	return r.assertExpectedDeltasAfterState(expectedSpec, expectedBefore, currentState)
}

func findSubclassTrack(tracks []*daggerheartv1.DaggerheartSubclassTrack, origin daggerheartv1.DaggerheartSubclassTrackOrigin) (*daggerheartv1.DaggerheartSubclassTrack, bool) {
	for _, track := range tracks {
		if track.GetOrigin() == origin {
			return track, true
		}
	}
	return nil, false
}

func findSubclassTrackRank(tracks []*daggerheartv1.DaggerheartSubclassTrack, origin daggerheartv1.DaggerheartSubclassTrackOrigin) (daggerheartv1.DaggerheartSubclassTrackRank, bool) {
	track, ok := findSubclassTrack(tracks, origin)
	if !ok {
		return daggerheartv1.DaggerheartSubclassTrackRank_DAGGERHEART_SUBCLASS_TRACK_RANK_UNSPECIFIED, false
	}
	return track.GetRank(), true
}

func normalizeSubclassTrackRank(rank daggerheartv1.DaggerheartSubclassTrackRank) string {
	switch rank {
	case daggerheartv1.DaggerheartSubclassTrackRank_DAGGERHEART_SUBCLASS_TRACK_RANK_FOUNDATION:
		return "foundation"
	case daggerheartv1.DaggerheartSubclassTrackRank_DAGGERHEART_SUBCLASS_TRACK_RANK_SPECIALIZATION:
		return "specialization"
	case daggerheartv1.DaggerheartSubclassTrackRank_DAGGERHEART_SUBCLASS_TRACK_RANK_MASTERY:
		return "mastery"
	default:
		return ""
	}
}

func profileHasActiveSubclassFeature(profile *daggerheartv1.DaggerheartProfile, featureID string) bool {
	want := normalizeScenarioKey(featureID)
	for _, track := range profile.GetActiveSubclassFeatures() {
		for _, feature := range track.GetFoundationFeatures() {
			if normalizeScenarioKey(feature.GetId()) == want {
				return true
			}
		}
		for _, feature := range track.GetSpecializationFeatures() {
			if normalizeScenarioKey(feature.GetId()) == want {
				return true
			}
		}
		for _, feature := range track.GetMasteryFeatures() {
			if normalizeScenarioKey(feature.GetId()) == want {
				return true
			}
		}
	}
	return false
}

func normalizeScenarioKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (r *Runner) runUpdateGoldStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("update_gold target is required")
	}

	characterID, err := actorID(state, name)
	if err != nil {
		return err
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err = r.env.daggerheartClient.UpdateGold(ctxWithSession, &daggerheartv1.DaggerheartUpdateGoldRequest{
		CampaignId:     state.campaignID,
		CharacterId:    characterID,
		HandfulsBefore: int32(optionalInt(step.Args, "handfuls_before", 0)),
		HandfulsAfter:  int32(optionalInt(step.Args, "handfuls_after", 0)),
		BagsBefore:     int32(optionalInt(step.Args, "bags_before", 0)),
		BagsAfter:      int32(optionalInt(step.Args, "bags_after", 0)),
		ChestsBefore:   int32(optionalInt(step.Args, "chests_before", 0)),
		ChestsAfter:    int32(optionalInt(step.Args, "chests_after", 0)),
		Reason:         optionalString(step.Args, "reason", ""),
	})
	if err != nil {
		return fmt.Errorf("update_gold: %w", err)
	}
	return r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheartpayload.EventTypeGoldUpdated)
}

func (r *Runner) runAcquireDomainCardStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("acquire_domain_card target is required")
	}
	cardID := requiredString(step.Args, "card_id")
	if cardID == "" {
		return r.failf("acquire_domain_card card_id is required")
	}

	characterID, err := actorID(state, name)
	if err != nil {
		return err
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err = r.env.daggerheartClient.AcquireDomainCard(ctxWithSession, &daggerheartv1.DaggerheartAcquireDomainCardRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
		CardId:      cardID,
		CardLevel:   int32(optionalInt(step.Args, "card_level", 1)),
		Destination: optionalString(step.Args, "destination", "vault"),
	})
	if err != nil {
		return fmt.Errorf("acquire_domain_card: %w", err)
	}
	return r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheartpayload.EventTypeDomainCardAcquired)
}

func (r *Runner) runSwapEquipmentStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("swap_equipment target is required")
	}
	itemID := requiredString(step.Args, "item_id")
	if itemID == "" {
		return r.failf("swap_equipment item_id is required")
	}

	characterID, err := actorID(state, name)
	if err != nil {
		return err
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err = r.env.daggerheartClient.SwapEquipment(ctxWithSession, &daggerheartv1.DaggerheartSwapEquipmentRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
		ItemId:      itemID,
		ItemType:    optionalString(step.Args, "item_type", "weapon"),
		From:        optionalString(step.Args, "from", "inventory"),
		To:          optionalString(step.Args, "to", "active"),
		StressCost:  int32(optionalInt(step.Args, "stress_cost", 0)),
	})
	if err != nil {
		return fmt.Errorf("swap_equipment: %w", err)
	}
	if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheartpayload.EventTypeEquipmentSwapped); err != nil {
		return err
	}
	return r.assertSwapEquipmentExpectations(ctx, state, characterID, step.Args)
}

func (r *Runner) runUseConsumableStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("use_consumable target is required")
	}
	consumableID := requiredString(step.Args, "consumable_id")
	if consumableID == "" {
		return r.failf("use_consumable consumable_id is required")
	}

	characterID, err := actorID(state, name)
	if err != nil {
		return err
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err = r.env.daggerheartClient.UseConsumable(ctxWithSession, &daggerheartv1.DaggerheartUseConsumableRequest{
		CampaignId:     state.campaignID,
		CharacterId:    characterID,
		ConsumableId:   consumableID,
		QuantityBefore: int32(optionalInt(step.Args, "quantity_before", 0)),
		QuantityAfter:  int32(optionalInt(step.Args, "quantity_after", 0)),
	})
	if err != nil {
		return fmt.Errorf("use_consumable: %w", err)
	}
	return r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheartpayload.EventTypeConsumableUsed)
}

func (r *Runner) runAcquireConsumableStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("acquire_consumable target is required")
	}
	consumableID := requiredString(step.Args, "consumable_id")
	if consumableID == "" {
		return r.failf("acquire_consumable consumable_id is required")
	}

	characterID, err := actorID(state, name)
	if err != nil {
		return err
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err = r.env.daggerheartClient.AcquireConsumable(ctxWithSession, &daggerheartv1.DaggerheartAcquireConsumableRequest{
		CampaignId:     state.campaignID,
		CharacterId:    characterID,
		ConsumableId:   consumableID,
		QuantityBefore: int32(optionalInt(step.Args, "quantity_before", 0)),
		QuantityAfter:  int32(optionalInt(step.Args, "quantity_after", 0)),
	})
	if err != nil {
		return fmt.Errorf("acquire_consumable: %w", err)
	}
	return r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheartpayload.EventTypeConsumableAcquired)
}

func (r *Runner) assertSwapEquipmentExpectations(ctx context.Context, state *scenarioState, characterID string, args map[string]any) error {
	if !hasSwapEquipmentExpectations(args) {
		return nil
	}
	profile, err := r.getDaggerheartProfile(ctx, state, characterID)
	if err != nil {
		return err
	}
	currentState, err := r.getCharacterState(ctx, state, characterID)
	if err != nil {
		return err
	}

	if want := optionalString(args, "expect_equipped_armor_id", ""); want != "" && profile.GetEquippedArmorId() != want {
		return r.assertf("swap_equipment equipped_armor_id = %q, want %q", profile.GetEquippedArmorId(), want)
	}
	if want, ok := readInt(args, "expect_evasion"); ok && int(profile.GetEvasion().GetValue()) != want {
		return r.assertf("swap_equipment evasion = %d, want %d", profile.GetEvasion().GetValue(), want)
	}
	if want, ok := readInt(args, "expect_major_threshold"); ok && int(profile.GetMajorThreshold().GetValue()) != want {
		return r.assertf("swap_equipment major_threshold = %d, want %d", profile.GetMajorThreshold().GetValue(), want)
	}
	if want, ok := readInt(args, "expect_severe_threshold"); ok && int(profile.GetSevereThreshold().GetValue()) != want {
		return r.assertf("swap_equipment severe_threshold = %d, want %d", profile.GetSevereThreshold().GetValue(), want)
	}
	if want, ok := readInt(args, "expect_armor_max"); ok && int(profile.GetArmorMax().GetValue()) != want {
		return r.assertf("swap_equipment armor_max = %d, want %d", profile.GetArmorMax().GetValue(), want)
	}
	if want, ok := readInt(args, "expect_spellcast_roll_bonus"); ok && int(profile.GetSpellcastRollBonus().GetValue()) != want {
		return r.assertf("swap_equipment spellcast_roll_bonus = %d, want %d", profile.GetSpellcastRollBonus().GetValue(), want)
	}
	if want, ok := readInt(args, "expect_agility"); ok && int(profile.GetAgility().GetValue()) != want {
		return r.assertf("swap_equipment agility = %d, want %d", profile.GetAgility().GetValue(), want)
	}
	if want, ok := readInt(args, "expect_presence"); ok && int(profile.GetPresence().GetValue()) != want {
		return r.assertf("swap_equipment presence = %d, want %d", profile.GetPresence().GetValue(), want)
	}
	if want, ok := readInt(args, "expect_armor"); ok && int(currentState.GetArmor()) != want {
		return r.assertf("swap_equipment armor = %d, want %d", currentState.GetArmor(), want)
	}
	return nil
}

func hasSwapEquipmentExpectations(args map[string]any) bool {
	for _, key := range []string{
		"expect_equipped_armor_id",
		"expect_evasion",
		"expect_major_threshold",
		"expect_severe_threshold",
		"expect_armor_max",
		"expect_spellcast_roll_bonus",
		"expect_agility",
		"expect_presence",
		"expect_armor",
	} {
		if _, ok := args[key]; ok {
			return true
		}
	}
	return false
}
