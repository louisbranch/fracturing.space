package validator

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func TestValidatorBranchCoverageGMAndRest(t *testing.T) {
	t.Parallel()

	after := daggerheartstate.GMFearMax + 1
	assertErrorContains(t, ValidateGMFearSetPayload(mustJSONRawMessage(t, payload.GMFearSetPayload{})), "after is required")
	assertErrorContains(t, ValidateGMFearSetPayload(mustJSONRawMessage(t, payload.GMFearSetPayload{After: &after})), "after must be in range")
	assertErrorContains(t, ValidateGMFearChangedPayload(mustJSONRawMessage(t, payload.GMFearChangedPayload{Value: daggerheartstate.GMFearMin - 1})), "value must be in range")

	t.Run("gm move target families", func(t *testing.T) {
		t.Parallel()

		assertErrorContains(t, ValidateGMMoveApplyPayload(mustJSONRawMessage(t, payload.GMMoveApplyPayload{})), "fear_spent must be greater than zero")
		assertErrorContains(t, ValidateGMMoveAppliedPayload(mustJSONRawMessage(t, payload.GMMoveAppliedPayload{})), "fear_spent must be greater than zero")

		assertErrorContains(t, ValidateGMMoveTarget(payload.GMMoveTarget{}), "target type is unsupported")
		assertErrorContains(t, ValidateGMMoveTarget(payload.GMMoveTarget{
			Type:  rules.GMMoveTargetTypeDirectMove,
			Kind:  rules.GMMoveKindInterruptAndMove,
			Shape: rules.GMMoveShapeCustom,
		}), "description is required for custom shape")
		assertErrorContains(t, ValidateGMMoveTarget(payload.GMMoveTarget{
			Type:  rules.GMMoveTargetTypeDirectMove,
			Kind:  rules.GMMoveKindInterruptAndMove,
			Shape: rules.GMMoveShapeSpotlightAdversary,
		}), "adversary_id is required for spotlight_adversary")
		assertErrorContains(t, ValidateGMMoveTarget(payload.GMMoveTarget{
			Type: rules.GMMoveTargetTypeAdversaryFeature,
		}), "adversary_id is required")
		assertErrorContains(t, ValidateGMMoveTarget(payload.GMMoveTarget{
			Type:        rules.GMMoveTargetTypeAdversaryFeature,
			AdversaryID: dhids.AdversaryID("adv-1"),
		}), "feature_id is required")
		assertErrorContains(t, ValidateGMMoveTarget(payload.GMMoveTarget{
			Type: rules.GMMoveTargetTypeEnvironmentFeature,
		}), "environment_entity_id is required")
		assertErrorContains(t, ValidateGMMoveTarget(payload.GMMoveTarget{
			Type:          rules.GMMoveTargetTypeEnvironmentFeature,
			EnvironmentID: "env.fog",
		}), "feature_id is required")
		assertErrorContains(t, ValidateGMMoveTarget(payload.GMMoveTarget{
			Type:        rules.GMMoveTargetTypeAdversaryExperience,
			AdversaryID: dhids.AdversaryID("adv-1"),
		}), "experience_name is required")
		assertErrorContains(t, ValidateGMMoveTarget(payload.GMMoveTarget{
			Type:                rules.GMMoveTargetTypeEnvironmentFeature,
			EnvironmentEntityID: dhids.EnvironmentEntityID("entity-1"),
			FeatureID:           "fog-burst",
		}), "")
	})

	t.Run("rest and loadout guards", func(t *testing.T) {
		t.Parallel()

		assertErrorContains(t, ValidateLoadoutSwapPayload(mustJSONRawMessage(t, payload.LoadoutSwapPayload{})), "character_id is required")
		assertErrorContains(t, ValidateLoadoutSwappedPayload(mustJSONRawMessage(t, payload.LoadoutSwappedPayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "card_id is required")

		assertErrorContains(t, ValidateRestTakePayload(mustJSONRawMessage(t, payload.RestTakePayload{})), "rest_type is required")
		assertErrorContains(t, ValidateRestTakePayload(mustJSONRawMessage(t, payload.RestTakePayload{
			RestType: "short",
		})), "participants are required")
		assertErrorContains(t, ValidateRestTakePayload(mustJSONRawMessage(t, payload.RestTakePayload{
			RestType:     "short",
			Participants: []ids.CharacterID{"char-1"},
			CampaignCountdownAdvances: []payload.CampaignCountdownAdvancePayload{
				{},
			},
		})), "campaign_countdown.countdown_id is required")
		assertErrorContains(t, ValidateRestTakePayload(mustJSONRawMessage(t, payload.RestTakePayload{
			RestType:     "short",
			Participants: []ids.CharacterID{"char-1"},
			DowntimeMoves: []payload.DowntimeMoveAppliedPayload{
				{ActorCharacterID: ids.CharacterID("char-1")},
			},
		})), "move is required")

		assertErrorContains(t, ValidateRestTakenPayload(mustJSONRawMessage(t, payload.RestTakenPayload{
			RestType:     "short",
			GMFear:       daggerheartstate.GMFearMax + 1,
			Participants: []ids.CharacterID{"char-1"},
		})), "gm_fear_after must be in range")
		assertErrorContains(t, ValidateRestTakenPayload(mustJSONRawMessage(t, payload.RestTakenPayload{
			RestType: "short",
			GMFear:   1,
		})), "participants are required")

		assertErrorContains(t, ValidateRestCampaignCountdownPayload(payload.CampaignCountdownAdvancePayload{}), "campaign_countdown.countdown_id is required")
		assertErrorContains(t, ValidateRestCampaignCountdownPayload(payload.CampaignCountdownAdvancePayload{
			CountdownID: dhids.CountdownID("cd-1"),
		}), "campaign_countdown advance must be positive")

		if HasRestTakeMutation(payload.RestTakePayload{}) {
			t.Fatal("HasRestTakeMutation(zero) = true, want false")
		}
		if !HasRestTakeMutation(payload.RestTakePayload{RefreshLongRest: true}) {
			t.Fatal("HasRestTakeMutation(refresh long rest) = false, want true")
		}
	})
}

func TestValidatorBranchCoverageProgressionAndDamage(t *testing.T) {
	t.Parallel()

	t.Run("progression validators", func(t *testing.T) {
		t.Parallel()

		assertErrorContains(t, ValidateAdversaryCreatePayload(mustJSONRawMessage(t, payload.AdversaryCreatePayload{})), "adversary_id is required")
		assertErrorContains(t, ValidateAdversaryCreatePayload(mustJSONRawMessage(t, payload.AdversaryCreatePayload{
			AdversaryID:      dhids.AdversaryID("adv-1"),
			AdversaryEntryID: "entry-1",
			Name:             "Goblin",
			SessionID:        ids.SessionID("sess-1"),
		})), "scene_id is required")
		assertErrorContains(t, ValidateAdversaryUpdatePayload(mustJSONRawMessage(t, payload.AdversaryUpdatePayload{
			AdversaryID: dhids.AdversaryID("adv-1"),
		})), "adversary_entry_id is required")

		assertErrorContains(t, ValidateEnvironmentEntityCreatePayload(mustJSONRawMessage(t, payload.EnvironmentEntityCreatePayload{})), "environment_entity_id is required")
		assertErrorContains(t, ValidateEnvironmentEntityCreatePayload(mustJSONRawMessage(t, payload.EnvironmentEntityCreatePayload{
			EnvironmentEntityID: dhids.EnvironmentEntityID("env-1"),
			EnvironmentID:       "env.fog",
			Name:                "Fog Bank",
			Type:                "hazard",
			SessionID:           ids.SessionID("sess-1"),
			SceneID:             ids.SceneID("scene-1"),
			Tier:                -1,
			Difficulty:          14,
		})), "tier must be non-negative")
		assertErrorContains(t, ValidateEnvironmentEntityCreatePayload(mustJSONRawMessage(t, payload.EnvironmentEntityCreatePayload{
			EnvironmentEntityID: dhids.EnvironmentEntityID("env-1"),
			EnvironmentID:       "env.fog",
			Name:                "Fog Bank",
			Type:                "hazard",
			SessionID:           ids.SessionID("sess-1"),
			SceneID:             ids.SceneID("scene-1"),
			Difficulty:          0,
		})), "difficulty must be positive")

		assertErrorContains(t, ValidateLevelUpApplyPayload(mustJSONRawMessage(t, payload.LevelUpApplyPayload{
			CharacterID: ids.CharacterID("char-1"),
			LevelBefore: 0,
			LevelAfter:  1,
			Advancements: []payload.LevelUpAdvancementPayload{
				{Type: "add_hp_slots"},
			},
		})), "level_before must be in range")
		assertErrorContains(t, ValidateLevelUpApplyPayload(mustJSONRawMessage(t, payload.LevelUpApplyPayload{
			CharacterID: ids.CharacterID("char-1"),
			LevelBefore: 1,
			LevelAfter:  2,
			Advancements: []payload.LevelUpAdvancementPayload{
				{Type: "add_hp_slots"},
			},
			Rewards: []payload.LevelUpRewardPayload{
				{Type: "domain_card"},
			},
		})), "reward domain_card_id is required")
		assertErrorContains(t, ValidateLevelUpApplyPayload(mustJSONRawMessage(t, payload.LevelUpApplyPayload{
			CharacterID: ids.CharacterID("char-1"),
			LevelBefore: 1,
			LevelAfter:  2,
			Advancements: []payload.LevelUpAdvancementPayload{
				{Type: "add_hp_slots"},
			},
			Rewards: []payload.LevelUpRewardPayload{
				{Type: "companion_bonus_choices"},
			},
		})), "reward companion_bonus_choices must be positive")
		assertErrorContains(t, ValidateLevelUpAppliedPayload(mustJSONRawMessage(t, payload.LevelUpAppliedPayload{
			CharacterID: ids.CharacterID("char-1"),
			Level:       0,
			Advancements: []payload.LevelUpAdvancementPayload{
				{Type: "add_hp_slots"},
			},
		})), "level_after must be in range")
		assertErrorContains(t, ValidateLevelUpAppliedPayload(mustJSONRawMessage(t, payload.LevelUpAppliedPayload{
			CharacterID: ids.CharacterID("char-1"),
			Level:       2,
		})), "advancements is required")

		assertErrorContains(t, ValidateGoldUpdatePayload(mustJSONRawMessage(t, payload.GoldUpdatePayload{
			CharacterID:   ids.CharacterID("char-1"),
			HandfulsAfter: -1,
		})), "handfuls_after must be in range")
		assertErrorContains(t, ValidateGoldUpdatePayload(mustJSONRawMessage(t, payload.GoldUpdatePayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "gold update must change at least one denomination")
		assertErrorContains(t, ValidateGoldUpdatedPayload(mustJSONRawMessage(t, payload.GoldUpdatedPayload{
			CharacterID: ids.CharacterID("char-1"),
			Bags:        10,
		})), "bags_after must be in range")

		assertErrorContains(t, ValidateEquipmentSwapPayload(mustJSONRawMessage(t, payload.EquipmentSwapPayload{
			CharacterID: ids.CharacterID("char-1"),
			ItemID:      "item-1",
			ItemType:    "trinket",
			From:        "inventory",
			To:          "active",
		})), "item_type must be weapon or armor")
		assertErrorContains(t, ValidateEquipmentSwapPayload(mustJSONRawMessage(t, payload.EquipmentSwapPayload{
			CharacterID: ids.CharacterID("char-1"),
			ItemID:      "item-1",
			ItemType:    "weapon",
			From:        "stashed",
			To:          "active",
		})), "from and to must be active, inventory, or none")
		assertErrorContains(t, ValidateEquipmentSwapPayload(mustJSONRawMessage(t, payload.EquipmentSwapPayload{
			CharacterID: ids.CharacterID("char-1"),
			ItemID:      "item-1",
			ItemType:    "weapon",
			From:        "active",
			To:          "active",
		})), "from and to must differ")

		assertErrorContains(t, ValidateConsumableUsePayload(mustJSONRawMessage(t, payload.ConsumableUsePayload{
			CharacterID:  ids.CharacterID("char-1"),
			ConsumableID: "potion-1",
		})), "quantity_before must be positive")
		assertErrorContains(t, ValidateConsumableUsePayload(mustJSONRawMessage(t, payload.ConsumableUsePayload{
			CharacterID:    ids.CharacterID("char-1"),
			ConsumableID:   "potion-1",
			QuantityBefore: 2,
			QuantityAfter:  2,
		})), "quantity_after must be quantity_before - 1")
		assertErrorContains(t, ValidateConsumableAcquirePayload(mustJSONRawMessage(t, payload.ConsumableAcquirePayload{
			CharacterID:    ids.CharacterID("char-1"),
			ConsumableID:   "potion-1",
			QuantityBefore: 5,
			QuantityAfter:  6,
		})), "quantity_after must be in range 1..5")
		assertErrorContains(t, ValidateConsumableAcquirePayload(mustJSONRawMessage(t, payload.ConsumableAcquirePayload{
			CharacterID:    ids.CharacterID("char-1"),
			ConsumableID:   "potion-1",
			QuantityBefore: 1,
			QuantityAfter:  3,
		})), "quantity_after must be quantity_before + 1")
		assertErrorContains(t, ValidateConsumableAcquiredPayload(mustJSONRawMessage(t, payload.ConsumableAcquiredPayload{
			CharacterID:  ids.CharacterID("char-1"),
			ConsumableID: "potion-1",
			Quantity:     0,
		})), "quantity_after must be in range 1..5")
	})

	t.Run("damage validators", func(t *testing.T) {
		t.Parallel()

		rollSeqZero := uint64(0)
		hpBefore := 5
		hpAfter := 4
		armorBefore := 1
		armorAfter := 0
		stressAfter := 1

		assertErrorContains(t, ValidateDamageApplyPayload(mustJSONRawMessage(t, payload.DamageApplyPayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "damage apply must change hp, stress, or armor")
		assertErrorContains(t, ValidateDamageApplyPayload(mustJSONRawMessage(t, payload.DamageApplyPayload{
			CharacterID: ids.CharacterID("char-1"),
			HpBefore:    &hpBefore,
			HpAfter:     &hpAfter,
			ArmorSpent:  mechanics.ArmorMaxCap + 1,
		})), "armor_spent must be in range")
		assertErrorContains(t, ValidateDamageAppliedPayload(mustJSONRawMessage(t, payload.DamageAppliedPayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "damage applied must include hp, stress, or armor")
		assertErrorContains(t, ValidateDamageAppliedPayload(mustJSONRawMessage(t, payload.DamageAppliedPayload{
			CharacterID: ids.CharacterID("char-1"),
			Hp:          &hpAfter,
			RollSeq:     &rollSeqZero,
		})), "roll_seq must be positive")
		assertErrorContains(t, ValidateDamageAdapterInvariants(payload.DamageApplyPayload{
			RollSeq: &rollSeqZero,
		}), "roll_seq must be positive")
		assertErrorContains(t, ValidateDamageAdapterInvariants(payload.DamageApplyPayload{
			SourceCharacterIDs: []ids.CharacterID{" "},
		}), "source_character_ids must not contain empty values")
		assertErrorContains(t, ValidateDamageAdapterInvariants(payload.DamageApplyPayload{
			HpBefore:    &hpBefore,
			HpAfter:     &hpAfter,
			ArmorBefore: &armorBefore,
			ArmorAfter:  &armorAfter,
			StressAfter: &stressAfter,
			Severity:    "major",
		}), "")

		assertErrorContains(t, ValidateDowntimeMoveAppliedPayloadFields(payload.DowntimeMoveAppliedPayload{}), "actor_character_id is required")
		assertErrorContains(t, ValidateDowntimeMoveAppliedPayloadFields(payload.DowntimeMoveAppliedPayload{
			ActorCharacterID: ids.CharacterID("char-1"),
		}), "move is required")
		assertErrorContains(t, ValidateDowntimeMoveAppliedPayloadFields(payload.DowntimeMoveAppliedPayload{
			ActorCharacterID: ids.CharacterID("char-1"),
			Move:             "prepare",
		}), "downtime_move applied must target a character or countdown")
		assertErrorContains(t, ValidateDowntimeMoveAppliedPayloadFields(payload.DowntimeMoveAppliedPayload{
			ActorCharacterID:  ids.CharacterID("char-1"),
			TargetCharacterID: ids.CharacterID("char-2"),
			Move:              "prepare",
		}), "downtime_move applied target requires a state change or countdown update")
		assertErrorContains(t, ValidateCharacterTemporaryArmorApplyPayload(mustJSONRawMessage(t, payload.CharacterTemporaryArmorApplyPayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "source is required")
		assertErrorContains(t, ValidateCharacterTemporaryArmorApplyPayload(mustJSONRawMessage(t, payload.CharacterTemporaryArmorApplyPayload{
			CharacterID: ids.CharacterID("char-1"),
			Source:      "ritual",
			Duration:    "forever",
			Amount:      1,
		})), "duration must be short_rest, long_rest, session, or scene")
		assertErrorContains(t, ValidateCharacterTemporaryArmorApplyPayload(mustJSONRawMessage(t, payload.CharacterTemporaryArmorApplyPayload{
			CharacterID: ids.CharacterID("char-1"),
			Source:      "ritual",
			Duration:    "scene",
		})), "amount must be greater than zero")
	})
}

func TestValidatorBranchCoverageConditionsAndCharacterState(t *testing.T) {
	t.Parallel()

	hidden := mustStandardConditionState(t, rules.ConditionHidden)

	t.Run("condition changed payloads", func(t *testing.T) {
		t.Parallel()

		assertErrorContains(t, ValidateConditionChangedPayload(mustJSONRawMessage(t, payload.ConditionChangedPayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "conditions_after is required")
		assertErrorContains(t, ValidateConditionChangedPayload([]byte(`{"character_id":"char-1","conditions_after":[{"id":"broken"}]}`)), "conditions_after:")
		assertErrorContains(t, ValidateAdversaryConditionChangedPayload(mustJSONRawMessage(t, payload.AdversaryConditionChangedPayload{
			AdversaryID: dhids.AdversaryID("adv-1"),
		})), "conditions_after is required")
		assertErrorContains(t, ValidateAdversaryConditionChangedPayload([]byte(`{"adversary_id":"adv-1","conditions_after":[{"id":"broken"}]}`)), "conditions_after:")
		assertErrorContains(t, ValidateConditionSetPayload(nil, []rules.ConditionState{hidden}, nil, []rules.ConditionState{hidden}), "conditions_before is required when removed are provided")
	})

	t.Run("beastform and companion spend guards", func(t *testing.T) {
		t.Parallel()

		assertErrorContains(t, ValidateBeastformDropPayload(mustJSONRawMessage(t, payload.BeastformDropPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			CharacterID:      ids.CharacterID("char-1"),
			BeastformID:      "wolf-form",
		})), "beastform drop must change class state")
		assertErrorContains(t, ValidateCompanionReturnPayload(mustJSONRawMessage(t, payload.CompanionReturnPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			CharacterID:      ids.CharacterID("char-1"),
			Resolution:       "complete",
		})), "companion return must change at least one field")
		assertErrorContains(t, ValidateHopeSpendPayload(mustJSONRawMessage(t, payload.HopeSpendPayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "amount must be greater than zero")
		assertErrorContains(t, ValidateHopeSpendPayload(mustJSONRawMessage(t, payload.HopeSpendPayload{
			CharacterID: ids.CharacterID("char-1"),
			Amount:      1,
			Before:      3,
			After:       3,
		})), "before and after must differ")
		assertErrorContains(t, ValidateHopeSpendPayload(mustJSONRawMessage(t, payload.HopeSpendPayload{
			CharacterID: ids.CharacterID("char-1"),
			Amount:      2,
			Before:      3,
			After:       2,
		})), "amount must match before and after delta")
		assertErrorContains(t, ValidateStressSpendPayload(mustJSONRawMessage(t, payload.StressSpendPayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "amount must be greater than zero")
		assertErrorContains(t, ValidateStressSpendPayload(mustJSONRawMessage(t, payload.StressSpendPayload{
			CharacterID: ids.CharacterID("char-1"),
			Amount:      3,
			Before:      1,
			After:       3,
		})), "amount must match before and after delta")
	})

	t.Run("character patch and subclass guards", func(t *testing.T) {
		t.Parallel()

		assertErrorContains(t, ValidateCharacterStatePatchPayload(mustJSONRawMessage(t, payload.CharacterStatePatchPayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "character_state patch must change at least one field")
		assertErrorContains(t, ValidateSubclassFeatureApplyPayload(mustJSONRawMessage(t, payload.SubclassFeatureApplyPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			Feature:          "bind",
		})), "subclass_feature apply requires at least one consequence")
		assertErrorContains(t, ValidateSubclassFeatureApplyPayload(mustJSONRawMessage(t, payload.SubclassFeatureApplyPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			Feature:          "bind",
			Targets: []payload.SubclassFeatureTargetPatchPayload{
				{},
			},
		})), "subclass_feature apply target character_id is required")
		assertErrorContains(t, ValidateSubclassFeatureApplyPayload(mustJSONRawMessage(t, payload.SubclassFeatureApplyPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			Feature:          "bind",
			Targets: []payload.SubclassFeatureTargetPatchPayload{
				{CharacterID: ids.CharacterID("char-1")},
			},
		})), "subclass_feature apply must change at least one field")
	})
}

func TestValidatorCountdownAndHelperCoverage(t *testing.T) {
	t.Parallel()

	t.Run("countdown validators cover guard branches", func(t *testing.T) {
		t.Parallel()

		assertErrorContains(t, ValidateSceneCountdownCreatePayload(mustJSONRawMessage(t, payload.SceneCountdownCreatePayload{})), "countdown_id is required")
		assertErrorContains(t, ValidateSceneCountdownCreatePayload(mustJSONRawMessage(t, payload.SceneCountdownCreatePayload{
			CountdownID:       dhids.CountdownID("cd-1"),
			Name:              "Clock",
			Tone:              rules.CountdownToneConsequence,
			AdvancementPolicy: "manual",
			StartingValue:     4,
			RemainingValue:    5,
			LoopBehavior:      rules.CountdownLoopBehaviorNone,
			Status:            rules.CountdownStatusActive,
		})), "remaining_value must be in range")
		assertErrorContains(t, ValidateSceneCountdownCreatePayload(mustJSONRawMessage(t, payload.SceneCountdownCreatePayload{
			CountdownID:       dhids.CountdownID("cd-1"),
			Name:              "Clock",
			Tone:              rules.CountdownToneConsequence,
			AdvancementPolicy: "manual",
			StartingValue:     4,
			RemainingValue:    4,
			LoopBehavior:      rules.CountdownLoopBehaviorNone,
			Status:            rules.CountdownStatusActive,
			StartingRoll: &payload.CountdownStartingRollPayload{
				Min: 0, Max: 6, Value: 1,
			},
		})), "starting_roll range is invalid")

		assertErrorContains(t, ValidateSceneCountdownAdvancePayload(mustJSONRawMessage(t, payload.SceneCountdownAdvancePayload{})), "countdown_id is required")
		assertErrorContains(t, ValidateSceneCountdownAdvancePayload(mustJSONRawMessage(t, payload.SceneCountdownAdvancePayload{
			CountdownID:     dhids.CountdownID("cd-1"),
			BeforeRemaining: -1,
			AfterRemaining:  1,
			AdvancedBy:      1,
		})), "countdown remaining values must be non-negative")
		assertErrorContains(t, ValidateSceneCountdownAdvancePayload(mustJSONRawMessage(t, payload.SceneCountdownAdvancePayload{
			CountdownID:     dhids.CountdownID("cd-1"),
			BeforeRemaining: 2,
			AfterRemaining:  1,
		})), "advanced_by must be positive")

		assertErrorContains(t, ValidateSceneCountdownTriggerResolvePayload(mustJSONRawMessage(t, payload.SceneCountdownTriggerResolvePayload{})), "countdown_id is required")
		assertErrorContains(t, ValidateCampaignCountdownTriggerResolvePayload(mustJSONRawMessage(t, payload.CampaignCountdownTriggerResolvePayload{
			CountdownID: dhids.CountdownID("cd-1"),
		})), "")
	})

	t.Run("common and state helper branches", func(t *testing.T) {
		t.Parallel()

		if err := ValidatePayload([]byte(`{`), func(p struct{}) error { return nil }); err == nil {
			t.Fatal("ValidatePayload(invalid JSON) = nil, want error")
		}
		assertErrorContains(t, RequirePositive(0, "count"), "count must be positive")
		assertErrorContains(t, RequirePositive(1, "count"), "")

		if _, ok, err := NormalizeConditionStateListField(nil, "conditions_after", false); err != nil || ok {
			t.Fatalf("NormalizeConditionStateListField(nil,false) = (_, %v, %v), want ok=false err=nil", ok, err)
		}
		if _, _, err := NormalizeConditionStateListField(nil, "conditions_after", true); err == nil {
			t.Fatal("NormalizeConditionStateListField(nil,true) = nil, want error")
		}

		if HasClassStateFieldChange(nil, nil) {
			t.Fatal("HasClassStateFieldChange(nil,nil) = true, want false")
		}
		if !HasClassStateFieldChange(nil, &daggerheartstate.CharacterClassState{}) {
			t.Fatal("HasClassStateFieldChange(nil,value) = false, want true")
		}
		classState := &daggerheartstate.CharacterClassState{}
		if HasClassStateFieldChange(classState, classState) {
			t.Fatal("HasClassStateFieldChange(equal) = true, want false")
		}
		if HasSubclassStateFieldChange(nil, nil) {
			t.Fatal("HasSubclassStateFieldChange(nil,nil) = true, want false")
		}
		if !HasSubclassStateFieldChange(nil, &daggerheartstate.CharacterSubclassState{}) {
			t.Fatal("HasSubclassStateFieldChange(nil,value) = false, want true")
		}
		subclassState := &daggerheartstate.CharacterSubclassState{}
		if HasSubclassStateFieldChange(subclassState, subclassState) {
			t.Fatal("HasSubclassStateFieldChange(equal) = true, want false")
		}
		if !HasConditionListMutation([]string{"???"}, []string{"hidden"}) {
			t.Fatal("HasConditionListMutation(invalid) = false, want true")
		}
		if HasStringFieldChange(strPtr("a"), strPtr("a")) {
			t.Fatal("HasStringFieldChange(equal) = true, want false")
		}
		if !HasStringFieldChange(strPtr("a"), strPtr("b")) {
			t.Fatal("HasStringFieldChange(change) = false, want true")
		}
		if HasIntFieldChange(intPtr(1), nil) {
			t.Fatal("HasIntFieldChange(after nil) = true, want false")
		}
		if !HasIntFieldChange(nil, intPtr(1)) {
			t.Fatal("HasIntFieldChange(before nil) = false, want true")
		}
		if HasIntFieldChange(intPtr(1), intPtr(1)) {
			t.Fatal("HasIntFieldChange(equal) = true, want false")
		}
		if HasBoolFieldChange(boolPtr(true), nil) {
			t.Fatal("HasBoolFieldChange(after nil) = true, want false")
		}
	})
}

func TestValidatorFeatureAndProfileCoverage(t *testing.T) {
	t.Parallel()

	t.Run("profile, feature, and mutation validators", func(t *testing.T) {
		t.Parallel()

		assertErrorContains(t, ValidateCharacterProfileReplacePayload(mustJSONRawMessage(t, daggerheartstate.CharacterProfileReplacePayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "hp_max")
		assertErrorContains(t, ValidateCharacterProfileReplacedPayload(mustJSONRawMessage(t, daggerheartstate.CharacterProfileReplacedPayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "hp_max")
		assertErrorContains(t, ValidateCharacterStatePatchedPayload(mustJSONRawMessage(t, payload.CharacterStatePatchedPayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "character_state_patched must include at least one after field")

		assertErrorContains(t, ValidateClassFeatureApplyPayload(mustJSONRawMessage(t, payload.ClassFeatureApplyPayload{})), "actor_character_id is required")
		assertErrorContains(t, ValidateClassFeatureApplyPayload(mustJSONRawMessage(t, payload.ClassFeatureApplyPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
		})), "feature is required")
		assertErrorContains(t, ValidateClassFeatureApplyPayload(mustJSONRawMessage(t, payload.ClassFeatureApplyPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			Feature:          "frontline",
		})), "class_feature apply requires at least one target")
		assertErrorContains(t, ValidateClassFeatureApplyPayload(mustJSONRawMessage(t, payload.ClassFeatureApplyPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			Feature:          "frontline",
			Targets:          []payload.ClassFeatureTargetPatchPayload{{}},
		})), "class_feature apply target character_id is required")
		assertErrorContains(t, ValidateClassFeatureApplyPayload(mustJSONRawMessage(t, payload.ClassFeatureApplyPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			Feature:          "frontline",
			Targets: []payload.ClassFeatureTargetPatchPayload{
				{CharacterID: ids.CharacterID("char-1")},
			},
		})), "class_feature apply must change at least one field per target")
		hpBefore := 6
		hpAfter := 5
		assertErrorContains(t, ValidateClassFeatureApplyPayload(mustJSONRawMessage(t, payload.ClassFeatureApplyPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			Feature:          "frontline",
			Targets: []payload.ClassFeatureTargetPatchPayload{
				{CharacterID: ids.CharacterID("char-1"), HPBefore: &hpBefore, HPAfter: &hpAfter},
			},
		})), "")

		assertErrorContains(t, ValidateStatModifierChangePayload(mustJSONRawMessage(t, payload.StatModifierChangePayload{})), "character_id is required")
		assertErrorContains(t, ValidateStatModifierChangePayload(mustJSONRawMessage(t, payload.StatModifierChangePayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "modifiers_after is required")
		assertErrorContains(t, ValidateStatModifierChangedPayload(mustJSONRawMessage(t, payload.StatModifierChangedPayload{})), "character_id is required")
		assertErrorContains(t, ValidateStatModifierChangedPayload(mustJSONRawMessage(t, payload.StatModifierChangedPayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "modifiers_after is required")

		assertErrorContains(t, ValidateAdversaryFeatureApplyPayload(mustJSONRawMessage(t, payload.AdversaryFeatureApplyPayload{})), "actor_adversary_id is required")
		assertErrorContains(t, ValidateAdversaryFeatureApplyPayload(mustJSONRawMessage(t, payload.AdversaryFeatureApplyPayload{
			ActorAdversaryID: dhids.AdversaryID("adv-1"),
			AdversaryID:      dhids.AdversaryID("adv-1"),
			FeatureID:        "feature-1",
		})), "adversary_feature apply must change at least one field")

		assertErrorContains(t, ValidateMultiTargetDamageApplyPayload(mustJSONRawMessage(t, payload.MultiTargetDamageApplyPayload{
			Targets: []payload.DamageApplyPayload{{}},
		})), "targets[0]: character_id is required")
		assertErrorContains(t, ValidateAdversaryDamageApplyPayload(mustJSONRawMessage(t, payload.AdversaryDamageApplyPayload{
			AdversaryID: dhids.AdversaryID("adv-1"),
		})), "damage apply must change hp or armor")
		assertErrorContains(t, ValidateAdversaryDamageAppliedPayload(mustJSONRawMessage(t, payload.AdversaryDamageAppliedPayload{
			AdversaryID: dhids.AdversaryID("adv-1"),
		})), "damage applied must include hp or armor")

		assertErrorContains(t, ValidateDomainCardAcquirePayload(mustJSONRawMessage(t, payload.DomainCardAcquirePayload{
			CharacterID: ids.CharacterID("char-1"),
			CardID:      "card-1",
		})), "card_level must be at least 1")
		assertErrorContains(t, ValidateDomainCardAcquirePayload(mustJSONRawMessage(t, payload.DomainCardAcquirePayload{
			CharacterID: ids.CharacterID("char-1"),
			CardID:      "card-1",
			CardLevel:   1,
			Destination: "stash",
		})), "destination must be vault or loadout")

		assertErrorContains(t, ValidateBeastformTransformPayload(mustJSONRawMessage(t, payload.BeastformTransformPayload{
			CharacterID: ids.CharacterID("char-1"),
			BeastformID: "wolf-form",
		})), "actor_character_id is required")
		assertErrorContains(t, ValidateBeastformTransformedPayload(mustJSONRawMessage(t, payload.BeastformTransformedPayload{
			BeastformID: "wolf-form",
		})), "character_id is required")
		assertErrorContains(t, ValidateBeastformDroppedPayload(mustJSONRawMessage(t, payload.BeastformDroppedPayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "beastform_id is required")
		assertErrorContains(t, ValidateCompanionExperienceBeginPayload(mustJSONRawMessage(t, payload.CompanionExperienceBeginPayload{
			CharacterID:  ids.CharacterID("char-1"),
			ExperienceID: "scouting",
		})), "actor_character_id is required")
		assertErrorContains(t, ValidateCompanionExperienceBegunPayload(mustJSONRawMessage(t, payload.CompanionExperienceBegunPayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "experience_id is required")
		assertErrorContains(t, ValidateCompanionReturnedPayload(mustJSONRawMessage(t, payload.CompanionReturnedPayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "resolution is required")
	})
}

func TestValidatorRemainingCoverageSweep(t *testing.T) {
	t.Parallel()

	t.Run("progression guards", func(t *testing.T) {
		t.Parallel()

		assertErrorContains(t, ValidateAdversaryCreatePayload(mustJSONRawMessage(t, payload.AdversaryCreatePayload{
			AdversaryID:      dhids.AdversaryID("adv-1"),
			AdversaryEntryID: "entry-1",
		})), "name is required")
		assertErrorContains(t, ValidateAdversaryCreatePayload(mustJSONRawMessage(t, payload.AdversaryCreatePayload{
			AdversaryID:      dhids.AdversaryID("adv-1"),
			AdversaryEntryID: "entry-1",
			Name:             "Goblin",
		})), "session_id is required")
		assertErrorContains(t, ValidateAdversaryCreatePayload(mustJSONRawMessage(t, payload.AdversaryCreatePayload{
			AdversaryID:      dhids.AdversaryID("adv-1"),
			AdversaryEntryID: "entry-1",
			Name:             "Goblin",
			SessionID:        ids.SessionID("sess-1"),
			SceneID:          ids.SceneID("scene-1"),
		})), "")

		assertErrorContains(t, ValidateAdversaryUpdatePayload(mustJSONRawMessage(t, payload.AdversaryUpdatePayload{
			AdversaryID:      dhids.AdversaryID("adv-1"),
			AdversaryEntryID: "entry-1",
			Name:             "Goblin",
			SessionID:        ids.SessionID("sess-1"),
		})), "scene_id is required")
		assertErrorContains(t, ValidateAdversaryUpdatePayload(mustJSONRawMessage(t, payload.AdversaryUpdatePayload{
			AdversaryID:      dhids.AdversaryID("adv-1"),
			AdversaryEntryID: "entry-1",
			Name:             "Goblin",
			SessionID:        ids.SessionID("sess-1"),
			SceneID:          ids.SceneID("scene-1"),
		})), "")

		assertErrorContains(t, ValidateEnvironmentEntityCreatePayload(mustJSONRawMessage(t, payload.EnvironmentEntityCreatePayload{
			EnvironmentEntityID: dhids.EnvironmentEntityID("env-1"),
		})), "environment_id is required")
		assertErrorContains(t, ValidateEnvironmentEntityCreatePayload(mustJSONRawMessage(t, payload.EnvironmentEntityCreatePayload{
			EnvironmentEntityID: dhids.EnvironmentEntityID("env-1"),
			EnvironmentID:       "env.fog",
		})), "name is required")
		assertErrorContains(t, ValidateEnvironmentEntityCreatePayload(mustJSONRawMessage(t, payload.EnvironmentEntityCreatePayload{
			EnvironmentEntityID: dhids.EnvironmentEntityID("env-1"),
			EnvironmentID:       "env.fog",
			Name:                "Fog Bank",
			Type:                "hazard",
			SessionID:           ids.SessionID("sess-1"),
			SceneID:             ids.SceneID("scene-1"),
			Tier:                1,
			Difficulty:          12,
		})), "")

		assertErrorContains(t, ValidateEnvironmentEntityUpdatePayload(mustJSONRawMessage(t, payload.EnvironmentEntityUpdatePayload{
			EnvironmentEntityID: dhids.EnvironmentEntityID("env-1"),
		})), "environment_id is required")
		assertErrorContains(t, ValidateEnvironmentEntityUpdatePayload(mustJSONRawMessage(t, payload.EnvironmentEntityUpdatePayload{
			EnvironmentEntityID: dhids.EnvironmentEntityID("env-1"),
			EnvironmentID:       "env.fog",
			Name:                "Fog Bank",
			Type:                "hazard",
			SessionID:           ids.SessionID("sess-1"),
			SceneID:             ids.SceneID("scene-1"),
			Tier:                1,
			Difficulty:          12,
		})), "")

		assertErrorContains(t, ValidateGoldUpdatePayload(mustJSONRawMessage(t, payload.GoldUpdatePayload{
			BagsAfter: 10,
		})), "character_id is required")
		assertErrorContains(t, ValidateGoldUpdatePayload(mustJSONRawMessage(t, payload.GoldUpdatePayload{
			CharacterID: ids.CharacterID("char-1"),
			BagsAfter:   10,
		})), "bags_after must be in range")
		assertErrorContains(t, ValidateGoldUpdatePayload(mustJSONRawMessage(t, payload.GoldUpdatePayload{
			CharacterID: ids.CharacterID("char-1"),
			ChestsAfter: 2,
		})), "chests_after must be in range")
		assertErrorContains(t, ValidateGoldUpdatePayload(mustJSONRawMessage(t, payload.GoldUpdatePayload{
			CharacterID:    ids.CharacterID("char-1"),
			HandfulsBefore: 1,
			HandfulsAfter:  2,
		})), "")

		assertErrorContains(t, ValidateGoldUpdatedPayload(mustJSONRawMessage(t, payload.GoldUpdatedPayload{
			Handfuls: 1,
		})), "character_id is required")
		assertErrorContains(t, ValidateGoldUpdatedPayload(mustJSONRawMessage(t, payload.GoldUpdatedPayload{
			CharacterID: ids.CharacterID("char-1"),
			Chests:      2,
		})), "chests_after must be in range")
		assertErrorContains(t, ValidateGoldUpdatedPayload(mustJSONRawMessage(t, payload.GoldUpdatedPayload{
			CharacterID: ids.CharacterID("char-1"),
			Handfuls:    1,
			Bags:        1,
			Chests:      1,
		})), "")
	})

	t.Run("state and consumable guards", func(t *testing.T) {
		t.Parallel()

		assertErrorContains(t, ValidateLoadoutSwappedPayload(mustJSONRawMessage(t, payload.LoadoutSwappedPayload{})), "character_id is required")
		assertErrorContains(t, ValidateLoadoutSwappedPayload(mustJSONRawMessage(t, payload.LoadoutSwappedPayload{
			CharacterID: ids.CharacterID("char-1"),
			CardID:      "card-1",
		})), "")

		assertErrorContains(t, ValidateConditionChangePayload(mustJSONRawMessage(t, payload.ConditionChangePayload{})), "character_id is required")
		hidden := mustStandardConditionState(t, rules.ConditionHidden)
		assertErrorContains(t, ValidateConditionChangePayload(mustJSONRawMessage(t, payload.ConditionChangePayload{
			CharacterID:     ids.CharacterID("char-1"),
			ConditionsAfter: []rules.ConditionState{hidden},
		})), "")
		assertErrorContains(t, ValidateAdversaryConditionChangePayload(mustJSONRawMessage(t, payload.AdversaryConditionChangePayload{})), "adversary_id is required")
		assertErrorContains(t, ValidateAdversaryConditionChangePayload(mustJSONRawMessage(t, payload.AdversaryConditionChangePayload{
			AdversaryID:     dhids.AdversaryID("adv-1"),
			ConditionsAfter: []rules.ConditionState{hidden},
		})), "")
		assertErrorContains(t, ValidateConditionSetPayload(nil, []rules.ConditionState{hidden}, []rules.ConditionState{}, []rules.ConditionState{hidden}), "conditions_before is required when removed are provided")
		assertErrorContains(t, ValidateConditionSetPayload([]rules.ConditionState{hidden}, []rules.ConditionState{hidden}, []rules.ConditionState{hidden}, nil), "added must match conditions_before and conditions_after diff")

		assertErrorContains(t, ValidateConsumableUsePayload(mustJSONRawMessage(t, payload.ConsumableUsePayload{})), "character_id is required")
		assertErrorContains(t, ValidateConsumableUsePayload(mustJSONRawMessage(t, payload.ConsumableUsePayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "consumable_id is required")
		assertErrorContains(t, ValidateConsumableUsePayload(mustJSONRawMessage(t, payload.ConsumableUsePayload{
			CharacterID:    ids.CharacterID("char-1"),
			ConsumableID:   "potion-1",
			QuantityBefore: 2,
			QuantityAfter:  1,
		})), "")
		assertErrorContains(t, ValidateConsumableUsedPayload(mustJSONRawMessage(t, payload.ConsumableUsedPayload{})), "character_id is required")
		assertErrorContains(t, ValidateConsumableUsedPayload(mustJSONRawMessage(t, payload.ConsumableUsedPayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "consumable_id is required")
		assertErrorContains(t, ValidateConsumableUsedPayload(mustJSONRawMessage(t, payload.ConsumableUsedPayload{
			CharacterID:  ids.CharacterID("char-1"),
			ConsumableID: "potion-1",
		})), "")

		assertErrorContains(t, ValidateConsumableAcquirePayload(mustJSONRawMessage(t, payload.ConsumableAcquirePayload{})), "character_id is required")
		assertErrorContains(t, ValidateConsumableAcquirePayload(mustJSONRawMessage(t, payload.ConsumableAcquirePayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "consumable_id is required")
		assertErrorContains(t, ValidateConsumableAcquirePayload(mustJSONRawMessage(t, payload.ConsumableAcquirePayload{
			CharacterID:    ids.CharacterID("char-1"),
			ConsumableID:   "potion-1",
			QuantityBefore: 1,
			QuantityAfter:  2,
		})), "")
		assertErrorContains(t, ValidateConsumableAcquiredPayload(mustJSONRawMessage(t, payload.ConsumableAcquiredPayload{})), "character_id is required")
		assertErrorContains(t, ValidateConsumableAcquiredPayload(mustJSONRawMessage(t, payload.ConsumableAcquiredPayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "consumable_id is required")
		assertErrorContains(t, ValidateConsumableAcquiredPayload(mustJSONRawMessage(t, payload.ConsumableAcquiredPayload{
			CharacterID:  ids.CharacterID("char-1"),
			ConsumableID: "potion-1",
			Quantity:     1,
		})), "")
	})

	t.Run("beastform, companion, and damage guards", func(t *testing.T) {
		t.Parallel()

		assertErrorContains(t, ValidateBeastformTransformPayload(mustJSONRawMessage(t, payload.BeastformTransformPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
		})), "character_id is required")
		assertErrorContains(t, ValidateBeastformTransformPayload(mustJSONRawMessage(t, payload.BeastformTransformPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			CharacterID:      ids.CharacterID("char-1"),
		})), "beastform_id is required")
		stressBefore := 1
		stressAfter := 2
		assertErrorContains(t, ValidateBeastformTransformPayload(mustJSONRawMessage(t, payload.BeastformTransformPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			CharacterID:      ids.CharacterID("char-1"),
			BeastformID:      "wolf-form",
			StressBefore:     &stressBefore,
			StressAfter:      &stressAfter,
		})), "")

		assertErrorContains(t, ValidateBeastformDropPayload(mustJSONRawMessage(t, payload.BeastformDropPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
		})), "character_id is required")
		assertErrorContains(t, ValidateBeastformDroppedPayload(mustJSONRawMessage(t, payload.BeastformDroppedPayload{})), "character_id is required")

		assertErrorContains(t, ValidateCompanionExperienceBeginPayload(mustJSONRawMessage(t, payload.CompanionExperienceBeginPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
		})), "character_id is required")
		assertErrorContains(t, ValidateCompanionExperienceBeginPayload(mustJSONRawMessage(t, payload.CompanionExperienceBeginPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			CharacterID:      ids.CharacterID("char-1"),
		})), "experience_id is required")
		before := &daggerheartstate.CharacterCompanionState{Status: daggerheartstate.CompanionStatusPresent}
		after := &daggerheartstate.CharacterCompanionState{
			Status:             daggerheartstate.CompanionStatusAway,
			ActiveExperienceID: "scouting",
		}
		assertErrorContains(t, ValidateCompanionExperienceBeginPayload(mustJSONRawMessage(t, payload.CompanionExperienceBeginPayload{
			ActorCharacterID:     ids.CharacterID("actor-1"),
			CharacterID:          ids.CharacterID("char-1"),
			ExperienceID:         "scouting",
			CompanionStateBefore: before,
			CompanionStateAfter:  after,
		})), "")

		assertErrorContains(t, ValidateCompanionReturnPayload(mustJSONRawMessage(t, payload.CompanionReturnPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
		})), "character_id is required")
		assertErrorContains(t, ValidateCompanionReturnPayload(mustJSONRawMessage(t, payload.CompanionReturnPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			CharacterID:      ids.CharacterID("char-1"),
		})), "resolution is required")

		assertErrorContains(t, ValidateStressSpendPayload(mustJSONRawMessage(t, payload.StressSpendPayload{})), "character_id is required")
		assertErrorContains(t, ValidateStressSpendPayload(mustJSONRawMessage(t, payload.StressSpendPayload{
			CharacterID: ids.CharacterID("char-1"),
			Amount:      1,
			Before:      2,
			After:       1,
		})), "")

		assertErrorContains(t, ValidateDamageApplyPayload(mustJSONRawMessage(t, payload.DamageApplyPayload{})), "character_id is required")
		hpBefore := 5
		hpAfter := 4
		assertErrorContains(t, ValidateDamageApplyPayload(mustJSONRawMessage(t, payload.DamageApplyPayload{
			CharacterID: ids.CharacterID("char-1"),
			HpBefore:    &hpBefore,
			HpAfter:     &hpAfter,
		})), "")
		assertErrorContains(t, ValidateDamageAppliedPayload(mustJSONRawMessage(t, payload.DamageAppliedPayload{})), "character_id is required")
		assertErrorContains(t, ValidateDamageAppliedPayload(mustJSONRawMessage(t, payload.DamageAppliedPayload{
			CharacterID: ids.CharacterID("char-1"),
			Hp:          &hpAfter,
		})), "")
		assertErrorContains(t, ValidateDamageAdapterInvariants(payload.DamageApplyPayload{
			Marks: rules.MaxDamageMarks + 1,
		}), "marks must be in range")
		assertErrorContains(t, ValidateDamageAdapterInvariants(payload.DamageApplyPayload{
			Severity: "major",
		}), "")
		assertErrorContains(t, ValidateAdversaryDamageApplyPayload(mustJSONRawMessage(t, payload.AdversaryDamageApplyPayload{})), "adversary_id is required")
		assertErrorContains(t, ValidateAdversaryDamageApplyPayload(mustJSONRawMessage(t, payload.AdversaryDamageApplyPayload{
			AdversaryID: dhids.AdversaryID("adv-1"),
			HpBefore:    &hpBefore,
			HpAfter:     &hpAfter,
		})), "")
		assertErrorContains(t, ValidateAdversaryDamageAppliedPayload(mustJSONRawMessage(t, payload.AdversaryDamageAppliedPayload{})), "adversary_id is required")
		assertErrorContains(t, ValidateAdversaryDamageAppliedPayload(mustJSONRawMessage(t, payload.AdversaryDamageAppliedPayload{
			AdversaryID: dhids.AdversaryID("adv-1"),
			Hp:          &hpAfter,
		})), "")
	})

	t.Run("profile and state patch success paths", func(t *testing.T) {
		t.Parallel()

		validProfile := daggerheartstate.CharacterProfile{
			ClassID:         "class.guardian",
			Level:           1,
			HpMax:           6,
			StressMax:       6,
			Evasion:         10,
			MajorThreshold:  1,
			SevereThreshold: 2,
			Proficiency:     1,
			ArmorScore:      0,
			ArmorMax:        0,
		}
		assertErrorContains(t, ValidateCharacterProfileReplacePayload(mustJSONRawMessage(t, daggerheartstate.CharacterProfileReplacePayload{
			CharacterID: ids.CharacterID("char-1"),
			Profile:     validProfile,
		})), "")
		assertErrorContains(t, ValidateCharacterProfileReplacedPayload(mustJSONRawMessage(t, daggerheartstate.CharacterProfileReplacedPayload{
			CharacterID: ids.CharacterID("char-1"),
			Profile:     validProfile,
		})), "")
		assertErrorContains(t, ValidateCharacterStatePatchPayload(mustJSONRawMessage(t, payload.CharacterStatePatchPayload{})), "character_id is required")
		hpBefore := 6
		hpAfter := 5
		assertErrorContains(t, ValidateCharacterStatePatchPayload(mustJSONRawMessage(t, payload.CharacterStatePatchPayload{
			CharacterID: ids.CharacterID("char-1"),
			HPBefore:    &hpBefore,
			HPAfter:     &hpAfter,
		})), "")
		assertErrorContains(t, ValidateCharacterStatePatchedPayload(mustJSONRawMessage(t, payload.CharacterStatePatchedPayload{})), "character_id is required")
		assertErrorContains(t, ValidateCharacterStatePatchedPayload(mustJSONRawMessage(t, payload.CharacterStatePatchedPayload{
			CharacterID: ids.CharacterID("char-1"),
			HP:          &hpAfter,
		})), "")
	})
}

func TestValidatorResidualBranches(t *testing.T) {
	t.Parallel()

	t.Run("profile and helper edges", func(t *testing.T) {
		t.Parallel()

		assertErrorContains(t, ValidateCharacterProfileReplacePayload(mustJSONRawMessage(t, daggerheartstate.CharacterProfileReplacePayload{})), "character_id is required")
		assertErrorContains(t, ValidateCharacterProfileReplacedPayload(mustJSONRawMessage(t, daggerheartstate.CharacterProfileReplacedPayload{})), "character_id is required")

		if !HasStringFieldChange(nil, strPtr("value")) {
			t.Fatal("HasStringFieldChange(nil, value) = false, want true")
		}
		if !HasBoolFieldChange(nil, boolPtr(true)) {
			t.Fatal("HasBoolFieldChange(nil, value) = false, want true")
		}
	})

	t.Run("countdown and condition edge cases", func(t *testing.T) {
		t.Parallel()

		assertErrorContains(t, ValidateSceneCountdownCreatePayload(mustJSONRawMessage(t, payload.SceneCountdownCreatePayload{
			CountdownID:       dhids.CountdownID("cd-1"),
			Name:              "Clock",
			Tone:              "unknown",
			AdvancementPolicy: "manual",
			StartingValue:     4,
			RemainingValue:    4,
			LoopBehavior:      rules.CountdownLoopBehaviorNone,
			Status:            rules.CountdownStatusActive,
		})), "countdown tone")
		assertErrorContains(t, ValidateSceneCountdownCreatePayload(mustJSONRawMessage(t, payload.SceneCountdownCreatePayload{
			CountdownID:       dhids.CountdownID("cd-1"),
			Name:              "Clock",
			Tone:              rules.CountdownToneConsequence,
			AdvancementPolicy: "unknown",
			StartingValue:     4,
			RemainingValue:    4,
			LoopBehavior:      rules.CountdownLoopBehaviorNone,
			Status:            rules.CountdownStatusActive,
		})), "advancement policy")
		assertErrorContains(t, ValidateSceneCountdownCreatePayload(mustJSONRawMessage(t, payload.SceneCountdownCreatePayload{
			CountdownID:       dhids.CountdownID("cd-1"),
			Name:              "Clock",
			Tone:              rules.CountdownToneConsequence,
			AdvancementPolicy: "manual",
			StartingValue:     4,
			RemainingValue:    4,
			LoopBehavior:      "unknown",
			Status:            rules.CountdownStatusActive,
		})), "loop behavior")
		assertErrorContains(t, ValidateSceneCountdownCreatePayload(mustJSONRawMessage(t, payload.SceneCountdownCreatePayload{
			CountdownID:       dhids.CountdownID("cd-1"),
			Name:              "Clock",
			Tone:              rules.CountdownToneConsequence,
			AdvancementPolicy: "manual",
			StartingValue:     4,
			RemainingValue:    4,
			LoopBehavior:      rules.CountdownLoopBehaviorNone,
			Status:            "unknown",
		})), "status")
		assertErrorContains(t, ValidateSceneCountdownCreatePayload(mustJSONRawMessage(t, payload.SceneCountdownCreatePayload{
			CountdownID:       dhids.CountdownID("cd-1"),
			Name:              "Clock",
			Tone:              rules.CountdownToneConsequence,
			AdvancementPolicy: "manual",
			StartingValue:     0,
			RemainingValue:    0,
			LoopBehavior:      rules.CountdownLoopBehaviorNone,
			Status:            rules.CountdownStatusActive,
		})), "starting_value must be positive")

		hidden := mustStandardConditionState(t, rules.ConditionHidden)
		restrained := mustStandardConditionState(t, rules.ConditionRestrained)
		assertErrorContains(t, ValidateConditionSetPayload(nil, nil, nil, nil), "conditions_after is required")
		assertErrorContains(t, ValidateConditionSetPayload([]rules.ConditionState{hidden}, []rules.ConditionState{hidden, restrained}, nil, []rules.ConditionState{hidden}), "removed must match conditions_before and conditions_after diff")
		assertErrorContains(t, ValidateConditionSetPayload(nil, []rules.ConditionState{hidden}, []rules.ConditionState{hidden}, nil), "")
	})

	t.Run("progression and state success and failure coverage", func(t *testing.T) {
		t.Parallel()

		assertErrorContains(t, ValidateAdversaryUpdatePayload(mustJSONRawMessage(t, payload.AdversaryUpdatePayload{
			AdversaryID:      dhids.AdversaryID("adv-1"),
			AdversaryEntryID: "entry-1",
		})), "name is required")
		assertErrorContains(t, ValidateAdversaryUpdatePayload(mustJSONRawMessage(t, payload.AdversaryUpdatePayload{
			AdversaryID:      dhids.AdversaryID("adv-1"),
			AdversaryEntryID: "entry-1",
			Name:             "Goblin",
		})), "session_id is required")
		assertErrorContains(t, ValidateAdversaryUpdatePayload(mustJSONRawMessage(t, payload.AdversaryUpdatePayload{
			AdversaryID:      dhids.AdversaryID("adv-1"),
			AdversaryEntryID: "entry-1",
			Name:             "Goblin",
			SessionID:        ids.SessionID("sess-1"),
			SceneID:          ids.SceneID("scene-1"),
		})), "")

		assertErrorContains(t, ValidateEnvironmentEntityUpdatePayload(mustJSONRawMessage(t, payload.EnvironmentEntityUpdatePayload{
			EnvironmentEntityID: dhids.EnvironmentEntityID("env-1"),
			EnvironmentID:       "env.fog",
		})), "name is required")
		assertErrorContains(t, ValidateEnvironmentEntityUpdatePayload(mustJSONRawMessage(t, payload.EnvironmentEntityUpdatePayload{
			EnvironmentEntityID: dhids.EnvironmentEntityID("env-1"),
			EnvironmentID:       "env.fog",
			Name:                "Fog Bank",
		})), "type is required")
		assertErrorContains(t, ValidateEnvironmentEntityUpdatePayload(mustJSONRawMessage(t, payload.EnvironmentEntityUpdatePayload{
			EnvironmentEntityID: dhids.EnvironmentEntityID("env-1"),
			EnvironmentID:       "env.fog",
			Name:                "Fog Bank",
			Type:                "hazard",
		})), "session_id is required")

		assertErrorContains(t, ValidateLevelUpApplyPayload(mustJSONRawMessage(t, payload.LevelUpApplyPayload{
			CharacterID: ids.CharacterID("char-1"),
			LevelBefore: 1,
			LevelAfter:  2,
			Advancements: []payload.LevelUpAdvancementPayload{
				{Type: "add_hp_slots"},
			},
			Rewards: []payload.LevelUpRewardPayload{
				{Type: ""},
			},
		})), "reward type is required")
		assertErrorContains(t, ValidateLevelUpApplyPayload(mustJSONRawMessage(t, payload.LevelUpApplyPayload{
			CharacterID: ids.CharacterID("char-1"),
			LevelBefore: 1,
			LevelAfter:  2,
			Advancements: []payload.LevelUpAdvancementPayload{
				{Type: "add_hp_slots"},
			},
			Rewards: []payload.LevelUpRewardPayload{
				{Type: "domain_card", DomainCardID: "card-1", DomainCardLevel: 1},
			},
		})), "")
		assertErrorContains(t, ValidateLevelUpAppliedPayload(mustJSONRawMessage(t, payload.LevelUpAppliedPayload{
			CharacterID: ids.CharacterID("char-1"),
			Level:       2,
			Advancements: []payload.LevelUpAdvancementPayload{
				{Type: "add_hp_slots"},
			},
		})), "")

		assertErrorContains(t, ValidateBeastformDropPayload(mustJSONRawMessage(t, payload.BeastformDropPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			CharacterID:      ids.CharacterID("char-1"),
		})), "beastform_id is required")
		assertErrorContains(t, ValidateHopeSpendPayload(mustJSONRawMessage(t, payload.HopeSpendPayload{
			CharacterID: ids.CharacterID("char-1"),
			Amount:      1,
			Before:      2,
			After:       1,
		})), "")
		assertErrorContains(t, ValidateRestCampaignCountdownPayload(payload.CampaignCountdownAdvancePayload{
			CountdownID: dhids.CountdownID("cd-1"),
			AdvancedBy:  1,
		}), "")
	})
}

func strPtr(value string) *string {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}
