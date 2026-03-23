package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func TestAdapterMetadataAndSnapshot(t *testing.T) {
	t.Parallel()

	store := newProfileStoreStub()
	store.snapshot = projectionstore.DaggerheartSnapshot{
		CampaignID:            "camp-1",
		GMFear:                2,
		ConsecutiveShortRests: 1,
	}
	a := NewAdapter(store, nil)

	if got := a.ID(); got != systemID {
		t.Fatalf("ID() = %q, want %q", got, systemID)
	}
	if got := a.Version(); got != systemVersion {
		t.Fatalf("Version() = %q, want %q", got, systemVersion)
	}
	if len(a.HandledTypes()) == 0 {
		t.Fatal("HandledTypes() returned empty list")
	}

	if _, err := (&Adapter{}).Snapshot(context.Background(), "camp-1"); err == nil || !strings.Contains(err.Error(), "store is not configured") {
		t.Fatalf("nil Snapshot() error = %v, want store-not-configured", err)
	}
	if _, err := a.Snapshot(context.Background(), " "); err == nil || !strings.Contains(err.Error(), "campaign id is required") {
		t.Fatalf("blank campaign Snapshot() error = %v, want campaign id error", err)
	}

	got, err := a.Snapshot(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("Snapshot() returned error: %v", err)
	}
	snapshot, ok := got.(projectionstore.DaggerheartSnapshot)
	if !ok {
		t.Fatalf("Snapshot() type = %T, want projectionstore.DaggerheartSnapshot", got)
	}
	if snapshot.GMFear != 2 || snapshot.ConsecutiveShortRests != 1 {
		t.Fatalf("Snapshot() = %+v, want stored snapshot", snapshot)
	}

	if err := (&Adapter{}).Apply(context.Background(), event.Event{}); err == nil || !strings.Contains(err.Error(), "store is not configured") {
		t.Fatalf("nil Apply() error = %v, want store-not-configured", err)
	}
}

func TestProjectionConvertersRoundTrip(t *testing.T) {
	t.Parallel()

	subclass := &daggerheartstate.CharacterSubclassState{
		BattleRitualUsedThisLongRest:           true,
		GiftedPerformerRelaxingSongUses:        1,
		GiftedPerformerEpicSongUses:            2,
		GiftedPerformerHeartbreakingSongUses:   3,
		ContactsEverywhereUsesThisSession:      1,
		ContactsEverywhereActionDieBonus:       1,
		ContactsEverywhereDamageDiceBonusCount: 2,
		SparingTouchUsesThisLongRest:           1,
		ElementalistActionBonus:                1,
		ElementalistDamageBonus:                2,
		TranscendenceActive:                    true,
		TranscendenceTraitBonusTarget:          "agility",
		TranscendenceTraitBonusValue:           1,
		TranscendenceProficiencyBonus:          1,
		TranscendenceEvasionBonus:              2,
		TranscendenceSevereThresholdBonus:      3,
		ClarityOfNatureUsedThisLongRest:        true,
		ElementalChannel:                       "fire",
		NemesisTargetID:                        "adv-1",
		RousingSpeechUsedThisLongRest:          true,
		WardensProtectionUsedThisLongRest:      true,
	}
	if got := SubclassStateFromProjection(SubclassStateToProjection(subclass)); !reflect.DeepEqual(got, daggerheartstate.NormalizedSubclassStatePtr(subclass)) {
		t.Fatalf("SubclassState round trip = %#v, want %#v", got, daggerheartstate.NormalizedSubclassStatePtr(subclass))
	}
	if got := SubclassStateToProjection(nil); got != nil {
		t.Fatalf("SubclassStateToProjection(nil) = %#v, want nil", got)
	}

	class := daggerheartstate.CharacterClassState{
		AttackBonusUntilRest:       1,
		EvasionBonusUntilHitOrRest: 2,
		DifficultyPenaltyUntilRest: 1,
		FocusTargetID:              "adv-2",
		ActiveBeastform: &daggerheartstate.CharacterActiveBeastformState{
			BeastformID:            "beastform.bear",
			BaseTrait:              "strength",
			AttackTrait:            "agility",
			TraitBonus:             1,
			EvasionBonus:           2,
			AttackRange:            "melee",
			DamageDice:             []daggerheartstate.CharacterDamageDie{{Count: 2, Sides: 8}},
			DamageBonus:            1,
			DamageType:             "physical",
			EvolutionTraitOverride: "instinct",
			DropOnAnyHPMark:        true,
		},
		StrangePatternsNumber: 3,
		RallyDice:             []int{6, 8},
		PrayerDice:            []int{10},
		Unstoppable: daggerheartstate.CharacterUnstoppableState{
			Active:           true,
			CurrentValue:     2,
			DieSides:         12,
			UsedThisLongRest: true,
		},
		ChannelRawPowerUsedThisLongRest: true,
	}
	classRoundTrip := ClassStateFromProjection(*ClassStateToProjection(&class))
	if !reflect.DeepEqual(classRoundTrip, class.Normalized()) {
		t.Fatalf("ClassState round trip = %#v, want %#v", classRoundTrip, class.Normalized())
	}
	if got := ClassStateToProjection(nil); got != nil {
		t.Fatalf("ClassStateToProjection(nil) = %#v, want nil", got)
	}

	companion := &daggerheartstate.CharacterCompanionState{Status: daggerheartstate.CompanionStatusPresent, ActiveExperienceID: "experience.guard"}
	if got := CompanionStateFromProjection(CompanionStateToProjection(companion)); !reflect.DeepEqual(got, daggerheartstate.NormalizedCompanionStatePtr(companion)) {
		t.Fatalf("CompanionState round trip = %#v, want %#v", got, daggerheartstate.NormalizedCompanionStatePtr(companion))
	}
	if got := CompanionStateToProjection(nil); got != nil {
		t.Fatalf("CompanionStateToProjection(nil) = %#v, want nil", got)
	}

	modifiers := []rules.StatModifierState{{
		ID:       "mod-1",
		Target:   rules.StatModifierTargetEvasion,
		Delta:    2,
		Label:    "Guarded",
		Source:   "feature",
		SourceID: "feature.guard",
		ClearTriggers: []rules.ConditionClearTrigger{
			rules.ConditionClearTriggerShortRest,
		},
	}}
	if got := StatModifiersFromProjection(StatModifiersToProjection(modifiers)); !reflect.DeepEqual(got, modifiers) {
		t.Fatalf("StatModifiers round trip = %#v, want %#v", got, modifiers)
	}
	if got := StatModifiersToProjection(nil); got != nil {
		t.Fatalf("StatModifiersToProjection(nil) = %#v, want nil", got)
	}

	conditions := []rules.ConditionState{{
		ID:       "cond-1",
		Class:    rules.ConditionClassStandard,
		Standard: rules.ConditionVulnerable,
		Code:     "vulnerable",
		Label:    "Vulnerable",
		Source:   "gm",
		SourceID: "move.1",
		ClearTriggers: []rules.ConditionClearTrigger{
			rules.ConditionClearTriggerLongRest,
		},
	}}
	wantConditions := []projectionstore.DaggerheartConditionState{{
		ID:       "cond-1",
		Class:    string(rules.ConditionClassStandard),
		Standard: rules.ConditionVulnerable,
		Code:     "vulnerable",
		Label:    "Vulnerable",
		Source:   "gm",
		SourceID: "move.1",
		ClearTriggers: []string{
			string(rules.ConditionClearTriggerLongRest),
		},
	}}
	if got := ConditionStatesToProjection(conditions); !reflect.DeepEqual(got, wantConditions) {
		t.Fatalf("ConditionStatesToProjection() = %#v, want %#v", got, wantConditions)
	}
	if got := ConditionStatesToProjection(nil); got != nil {
		t.Fatalf("ConditionStatesToProjection(nil) = %#v, want nil", got)
	}
}

func TestStateHelpersAndHandlers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newProfileStoreStub()
	a := NewAdapter(store, nil)

	if _, exists, err := a.GetCharacterStateIfExists(ctx, "camp-1", "missing"); err != nil || exists {
		t.Fatalf("GetCharacterStateIfExists() = exists:%v err:%v, want false nil", exists, err)
	}

	store.getCharacterStateErr = errors.New("boom")
	if _, _, err := a.GetCharacterStateIfExists(ctx, "camp-1", "char-1"); err == nil || !strings.Contains(err.Error(), "get daggerheart character state") {
		t.Fatalf("GetCharacterStateIfExists() error = %v, want wrapped error", err)
	}
	store.getCharacterStateErr = nil

	state, err := a.GetCharacterStateOrDefault(ctx, "camp-1", "char-1")
	if err != nil {
		t.Fatalf("GetCharacterStateOrDefault() returned error: %v", err)
	}
	if state.CampaignID != "camp-1" || state.CharacterID != "char-1" {
		t.Fatalf("GetCharacterStateOrDefault() = %+v, want ids seeded", state)
	}

	store.states[profileKey("camp-1", "char-1")] = projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          5,
		Hope:        2,
		HopeMax:     6,
		Stress:      1,
		Armor:       6,
		TemporaryArmor: []projectionstore.DaggerheartTemporaryArmor{
			{Source: "spell", Duration: "short_rest", SourceID: "spell.aegis", Amount: 2},
			{Source: "ritual", Duration: "long_rest", SourceID: "ritual.ward", Amount: 1},
		},
		LifeState: daggerheartstate.LifeStateAlive,
		StatModifiers: []projectionstore.DaggerheartStatModifier{
			{
				ID:            "mod-short",
				Target:        string(rules.StatModifierTargetEvasion),
				Delta:         2,
				ClearTriggers: []string{string(rules.ConditionClearTriggerShortRest)},
			},
			{
				ID:            "mod-long",
				Target:        string(rules.StatModifierTargetArmorScore),
				Delta:         1,
				ClearTriggers: []string{string(rules.ConditionClearTriggerLongRest)},
			},
		},
		ImpenetrableUsedThisShortRest: true,
	}
	store.profiles[profileKey("camp-1", "char-1")] = projectionstore.DaggerheartCharacterProfile{
		CampaignID:      "camp-1",
		CharacterID:     "char-1",
		ArmorMax:        4,
		ArmorScore:      4,
		EquippedArmorID: "armor.chainmail-armor",
		Evasion:         10,
		MajorThreshold:  8,
		SevereThreshold: 12,
	}

	armorMax, err := a.CharacterArmorMax(ctx, store.states[profileKey("camp-1", "char-1")])
	if err != nil {
		t.Fatalf("CharacterArmorMax() returned error: %v", err)
	}
	if armorMax != 4 {
		t.Fatalf("CharacterArmorMax() = %d, want 4", armorMax)
	}

	fallbackArmorMax, err := a.CharacterArmorMax(ctx, projectionstore.DaggerheartCharacterState{Armor: 3, TemporaryArmor: []projectionstore.DaggerheartTemporaryArmor{{Amount: 1}}})
	if err != nil {
		t.Fatalf("CharacterArmorMax() fallback returned error: %v", err)
	}
	if fallbackArmorMax != 2 {
		t.Fatalf("fallback CharacterArmorMax() = %d, want 2", fallbackArmorMax)
	}

	if err := a.ClearRestTemporaryArmor(ctx, "camp-1", "char-1", true, false); err != nil {
		t.Fatalf("ClearRestTemporaryArmor() returned error: %v", err)
	}
	clearedState := store.states[profileKey("camp-1", "char-1")]
	if clearedState.Armor != 5 {
		t.Fatalf("Armor after ClearRestTemporaryArmor() = %d, want 5", clearedState.Armor)
	}
	if len(clearedState.TemporaryArmor) != 1 || clearedState.TemporaryArmor[0].Duration != "long_rest" {
		t.Fatalf("TemporaryArmor after ClearRestTemporaryArmor() = %#v, want only long-rest bucket", clearedState.TemporaryArmor)
	}

	if err := a.ClearRestStatModifiers(ctx, "camp-1", "char-1", true, false); err != nil {
		t.Fatalf("ClearRestStatModifiers() returned error: %v", err)
	}
	if got := store.states[profileKey("camp-1", "char-1")].StatModifiers; len(got) != 1 || got[0].ID != "mod-long" {
		t.Fatalf("StatModifiers after ClearRestStatModifiers() = %#v, want only long-rest modifier", got)
	}

	if err := a.HandleRestTaken(ctx, event.Event{CampaignID: ids.CampaignID("camp-1")}, payload.RestTakenPayload{
		GMFear:          3,
		ShortRests:      2,
		RefreshRest:     true,
		RefreshLongRest: true,
		Participants:    []ids.CharacterID{ids.CharacterID("char-1")},
	}); err != nil {
		t.Fatalf("HandleRestTaken() returned error: %v", err)
	}
	if store.snapshot.GMFear != 3 || store.snapshot.ConsecutiveShortRests != 2 {
		t.Fatalf("snapshot after HandleRestTaken() = %+v, want GMFear=3 shortRests=2", store.snapshot)
	}
	if got := store.states[profileKey("camp-1", "char-1")]; got.ImpenetrableUsedThisShortRest {
		t.Fatalf("ImpenetrableUsedThisShortRest after HandleRestTaken() = true, want false")
	}

	stressAfter := 3
	armorAfter := 5
	evasionAfter := 11
	armorScoreAfter := 5
	armorMaxAfter := 5
	if err := a.HandleEquipmentSwapped(ctx, event.Event{CampaignID: ids.CampaignID("camp-1")}, payload.EquipmentSwappedPayload{
		CharacterID:     ids.CharacterID("char-1"),
		ItemType:        "armor",
		EquippedArmorID: "armor.full-plate",
		StressCost:      2,
		ArmorAfter:      &armorAfter,
		EvasionAfter:    &evasionAfter,
		ArmorScoreAfter: &armorScoreAfter,
		ArmorMaxAfter:   &armorMaxAfter,
	}); err != nil {
		t.Fatalf("HandleEquipmentSwapped() returned error: %v", err)
	}
	updatedProfile := store.profiles[profileKey("camp-1", "char-1")]
	if updatedProfile.EquippedArmorID != "armor.full-plate" || updatedProfile.ArmorMax != 5 || updatedProfile.ArmorScore != 5 {
		t.Fatalf("profile after HandleEquipmentSwapped() = %+v, want updated armor profile", updatedProfile)
	}
	updatedState := store.states[profileKey("camp-1", "char-1")]
	if updatedState.Armor != 5 || updatedState.Stress != stressAfter {
		t.Fatalf("state after HandleEquipmentSwapped() = %+v, want armor=5 stress=%d", updatedState, stressAfter)
	}

	modifierPayload := payload.StatModifierChangedPayload{
		CharacterID: ids.CharacterID("char-1"),
		Modifiers: []rules.StatModifierState{{
			ID:     "mod-new",
			Target: rules.StatModifierTargetEvasion,
			Delta:  1,
		}},
	}
	if err := a.HandleStatModifierChanged(ctx, event.Event{CampaignID: ids.CampaignID("camp-1")}, modifierPayload); err != nil {
		t.Fatalf("HandleStatModifierChanged() returned error: %v", err)
	}
	if got := store.states[profileKey("camp-1", "char-1")].StatModifiers; len(got) != 1 || got[0].ID != "mod-new" {
		t.Fatalf("state.StatModifiers after HandleStatModifierChanged() = %#v, want mod-new", got)
	}
}

func TestCharacterProfileReplaceAndDeleteHandlers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newProfileStoreStub()
	a := NewAdapter(store, nil)

	profile := daggerheartstate.CharacterProfile{
		HpMax:           6,
		StressMax:       6,
		Evasion:         10,
		MajorThreshold:  8,
		SevereThreshold: 12,
		Proficiency:     1,
		ArmorScore:      4,
		StartingArmorID: "armor.chainmail-armor",
	}
	if err := a.HandleCharacterProfileReplaced(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
		EntityID:   "char-1",
	}, daggerheartstate.CharacterProfileReplacedPayload{Profile: profile}); err != nil {
		t.Fatalf("HandleCharacterProfileReplaced() returned error: %v", err)
	}
	if got := store.profiles[profileKey("camp-1", "char-1")]; got.EquippedArmorID != "armor.chainmail-armor" {
		t.Fatalf("stored profile after replace = %+v, want equipped armor seeded", got)
	}

	store.deleteCharacterProfileErr = errors.New("delete failed")
	if err := a.HandleCharacterProfileDeleted(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
		EntityID:   "char-1",
	}, daggerheartstate.CharacterProfileDeletedPayload{}); err == nil || !strings.Contains(err.Error(), "delete daggerheart profile") {
		t.Fatalf("HandleCharacterProfileDeleted() error = %v, want wrapped delete error", err)
	}
	store.deleteCharacterProfileErr = nil

	if err := a.HandleCharacterProfileDeleted(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
		EntityID:   "char-1",
	}, daggerheartstate.CharacterProfileDeletedPayload{}); err != nil {
		t.Fatalf("HandleCharacterProfileDeleted() returned error: %v", err)
	}
	if _, ok := store.profiles[profileKey("camp-1", "char-1")]; ok {
		t.Fatal("profile still present after HandleCharacterProfileDeleted()")
	}
}

func TestAdapterApplyRoutesDeleteEvent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newProfileStoreStub()
	store.profiles[profileKey("camp-1", "char-1")] = projectionstore.DaggerheartCharacterProfile{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
	}
	a := NewAdapter(store, nil)

	payloadJSON, err := json.Marshal(daggerheartstate.CharacterProfileDeletedPayload{
		CharacterID: ids.CharacterID("char-1"),
	})
	if err != nil {
		t.Fatalf("Marshal() returned error: %v", err)
	}

	err = a.Apply(ctx, event.Event{
		CampaignID:  ids.CampaignID("camp-1"),
		Type:        payload.EventTypeCharacterProfileDeleted,
		EntityID:    "char-1",
		PayloadJSON: payloadJSON,
	})
	if err != nil {
		t.Fatalf("Apply() returned error: %v", err)
	}
	if _, ok := store.profiles[profileKey("camp-1", "char-1")]; ok {
		t.Fatal("profile still present after Apply() delete route")
	}
}

func TestCharacterStatePatchAndRuntimeHandlers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newProfileStoreStub()
	store.profiles[profileKey("camp-1", "char-1")] = projectionstore.DaggerheartCharacterProfile{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		ArmorMax:    4,
	}
	store.states[profileKey("camp-1", "char-1")] = projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          6,
		Hope:        2,
		HopeMax:     6,
		Stress:      1,
		Armor:       4,
		LifeState:   daggerheartstate.LifeStateAlive,
	}
	a := NewAdapter(store, nil)

	classState := daggerheartstate.CharacterClassState{FocusTargetID: "adv-1"}
	subclassState := &daggerheartstate.CharacterSubclassState{BattleRitualUsedThisLongRest: true}
	companionState := &daggerheartstate.CharacterCompanionState{Status: daggerheartstate.CompanionStatusAway, ActiveExperienceID: "exp-1"}
	lifeState := "unconscious"
	hpAfter := 5
	hopeAfter := 3
	hopeMaxAfter := 6
	stressAfter := 2
	armorAfter := 3
	impenetrable := true
	if err := a.ApplyStatePatch(ctx, "camp-1", "char-1", StatePatch{
		HP:                            &hpAfter,
		Hope:                          &hopeAfter,
		HopeMax:                       &hopeMaxAfter,
		Stress:                        &stressAfter,
		Armor:                         &armorAfter,
		LifeState:                     &lifeState,
		ClassState:                    &classState,
		SubclassState:                 subclassState,
		CompanionState:                companionState,
		ImpenetrableUsedThisShortRest: &impenetrable,
	}); err != nil {
		t.Fatalf("ApplyStatePatch() returned error: %v", err)
	}
	patched := store.states[profileKey("camp-1", "char-1")]
	if patched.Hp != 5 || patched.Hope != 3 || patched.HopeMax != 6 || patched.Stress != 2 || patched.Armor != 3 {
		t.Fatalf("state after ApplyStatePatch() = %+v, want patched scalar values", patched)
	}
	if patched.ClassState.FocusTargetID != "adv-1" || patched.SubclassState == nil || patched.CompanionState == nil {
		t.Fatalf("state after ApplyStatePatch() = %+v, want class/subclass/companion state", patched)
	}

	conditions := []rules.ConditionState{{ID: "cond-1", Class: rules.ConditionClassStandard, Standard: rules.ConditionVulnerable, Code: rules.ConditionVulnerable, Label: "Vulnerable"}}
	if err := a.ApplyConditionPatch(ctx, "camp-1", "char-1", conditions); err != nil {
		t.Fatalf("ApplyConditionPatch() returned error: %v", err)
	}
	if got := store.states[profileKey("camp-1", "char-1")].Conditions; len(got) != 1 || got[0].Code != rules.ConditionVulnerable {
		t.Fatalf("state.Conditions after ApplyConditionPatch() = %#v, want vulnerable condition", got)
	}

	damageHP := 4
	damageStress := 3
	damageArmor := 2
	if err := a.HandleDamageApplied(ctx, event.Event{CampaignID: ids.CampaignID("camp-1")}, payload.DamageAppliedPayload{
		CharacterID: ids.CharacterID("char-1"),
		Hp:          &damageHP,
		Stress:      &damageStress,
		Armor:       &damageArmor,
	}); err != nil {
		t.Fatalf("HandleDamageApplied() returned error: %v", err)
	}
	damageState := store.states[profileKey("camp-1", "char-1")]
	if damageState.Hp != 4 || damageState.Stress != 3 || damageState.Armor != 2 {
		t.Fatalf("state after HandleDamageApplied() = %+v, want hp=4 stress=3 armor=2", damageState)
	}

	downtimeHope := 4
	if err := a.HandleDowntimeMoveApplied(ctx, event.Event{CampaignID: ids.CampaignID("camp-1")}, payload.DowntimeMoveAppliedPayload{
		ActorCharacterID: ids.CharacterID("char-1"),
		Move:             "carouse",
		Hope:             &downtimeHope,
	}); err != nil {
		t.Fatalf("HandleDowntimeMoveApplied() returned error: %v", err)
	}
	if got := store.states[profileKey("camp-1", "char-1")].Hope; got != 4 {
		t.Fatalf("Hope after HandleDowntimeMoveApplied() = %d, want 4", got)
	}

	if err := a.HandleCharacterTemporaryArmorApplied(ctx, event.Event{CampaignID: ids.CampaignID("camp-1")}, payload.CharacterTemporaryArmorAppliedPayload{
		CharacterID: ids.CharacterID("char-1"),
		Source:      "spell",
		Duration:    "short_rest",
		Amount:      2,
		SourceID:    "spell.aegis",
	}); err != nil {
		t.Fatalf("HandleCharacterTemporaryArmorApplied() returned error: %v", err)
	}
	tempArmorState := store.states[profileKey("camp-1", "char-1")]
	if tempArmorState.Armor != 4 || len(tempArmorState.TemporaryArmor) != 1 {
		t.Fatalf("state after HandleCharacterTemporaryArmorApplied() = %+v, want armor increased with temp bucket", tempArmorState)
	}

	loadoutStress := 5
	if err := a.HandleLoadoutSwapped(ctx, event.Event{CampaignID: ids.CampaignID("camp-1")}, payload.LoadoutSwappedPayload{
		CharacterID: ids.CharacterID("char-1"),
		Stress:      &loadoutStress,
	}); err != nil {
		t.Fatalf("HandleLoadoutSwapped() returned error: %v", err)
	}
	if got := store.states[profileKey("camp-1", "char-1")].Stress; got != 5 {
		t.Fatalf("Stress after HandleLoadoutSwapped() = %d, want 5", got)
	}

	restedClass := &daggerheartstate.CharacterClassState{FocusTargetID: "adv-2"}
	if err := a.HandleCharacterStatePatched(ctx, event.Event{CampaignID: ids.CampaignID("camp-1")}, payload.CharacterStatePatchedPayload{
		CharacterID: ids.CharacterID("char-1"),
		Hope:        intPtr(5),
		HopeMax:     intPtr(6),
		LifeState:   strPtr(daggerheartstate.LifeStateAlive),
		ClassState:  restedClass,
	}); err != nil {
		t.Fatalf("HandleCharacterStatePatched() returned error: %v", err)
	}
	patchedState := store.states[profileKey("camp-1", "char-1")]
	if patchedState.Hope != 5 || patchedState.HopeMax != 6 || patchedState.LifeState != daggerheartstate.LifeStateAlive || patchedState.ClassState.FocusTargetID != "adv-2" {
		t.Fatalf("state after HandleCharacterStatePatched() = %+v, want patched values", patchedState)
	}

	beastform := &daggerheartstate.CharacterActiveBeastformState{BeastformID: "beastform.bear"}
	if err := a.HandleBeastformTransformed(ctx, event.Event{CampaignID: ids.CampaignID("camp-1")}, payload.BeastformTransformedPayload{
		CharacterID:     ids.CharacterID("char-1"),
		Hope:            intPtr(4),
		Stress:          intPtr(4),
		ActiveBeastform: beastform,
	}); err != nil {
		t.Fatalf("HandleBeastformTransformed() returned error: %v", err)
	}
	transformed := store.states[profileKey("camp-1", "char-1")]
	if transformed.ClassState.ActiveBeastform == nil || transformed.ClassState.ActiveBeastform.BeastformID != "beastform.bear" {
		t.Fatalf("state after HandleBeastformTransformed() = %+v, want active beastform", transformed)
	}
	if err := a.HandleBeastformDropped(ctx, event.Event{CampaignID: ids.CampaignID("camp-1")}, payload.BeastformDroppedPayload{
		CharacterID: ids.CharacterID("char-1"),
	}); err != nil {
		t.Fatalf("HandleBeastformDropped() returned error: %v", err)
	}
	if got := store.states[profileKey("camp-1", "char-1")].ClassState.ActiveBeastform; got != nil {
		t.Fatalf("ActiveBeastform after HandleBeastformDropped() = %+v, want nil", got)
	}

	if err := a.HandleCompanionExperienceBegun(ctx, event.Event{CampaignID: ids.CampaignID("camp-1")}, payload.CompanionExperienceBegunPayload{
		CharacterID:    ids.CharacterID("char-1"),
		CompanionState: &daggerheartstate.CharacterCompanionState{Status: daggerheartstate.CompanionStatusAway, ActiveExperienceID: "exp-2"},
	}); err != nil {
		t.Fatalf("HandleCompanionExperienceBegun() returned error: %v", err)
	}
	if got := store.states[profileKey("camp-1", "char-1")].CompanionState; got == nil || got.ActiveExperienceID != "exp-2" {
		t.Fatalf("CompanionState after HandleCompanionExperienceBegun() = %+v, want exp-2", got)
	}
	if err := a.HandleCompanionReturned(ctx, event.Event{CampaignID: ids.CampaignID("camp-1")}, payload.CompanionReturnedPayload{
		CharacterID:    ids.CharacterID("char-1"),
		Stress:         intPtr(6),
		CompanionState: &daggerheartstate.CharacterCompanionState{Status: daggerheartstate.CompanionStatusPresent},
	}); err != nil {
		t.Fatalf("HandleCompanionReturned() returned error: %v", err)
	}
	returned := store.states[profileKey("camp-1", "char-1")]
	if returned.Stress != 6 || returned.CompanionState == nil || returned.CompanionState.Status != daggerheartstate.CompanionStatusPresent {
		t.Fatalf("state after HandleCompanionReturned() = %+v, want present companion and stress 6", returned)
	}
}

func TestProgressionProfileHandlers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newProfileStoreStub()
	store.profiles[profileKey("camp-1", "char-1")] = projectionstore.DaggerheartCharacterProfile{
		CampaignID:      "camp-1",
		CharacterID:     "char-1",
		Level:           1,
		HpMax:           6,
		StressMax:       6,
		Evasion:         10,
		MajorThreshold:  8,
		SevereThreshold: 12,
		Proficiency:     1,
	}
	a := NewAdapter(store, func(profile *daggerheartstate.CharacterProfile, p payload.LevelUpAppliedPayload) {
		profile.Level = p.Level
		profile.HpMax++
	})

	if err := a.HandleLevelUpApplied(ctx, event.Event{CampaignID: ids.CampaignID("camp-1")}, payload.LevelUpAppliedPayload{
		CharacterID: ids.CharacterID("char-1"),
		Level:       2,
	}); err != nil {
		t.Fatalf("HandleLevelUpApplied() returned error: %v", err)
	}
	if got := store.profiles[profileKey("camp-1", "char-1")]; got.Level != 2 || got.HpMax != 7 {
		t.Fatalf("profile after HandleLevelUpApplied() = %+v, want level 2 hp 7", got)
	}

	if err := a.HandleGoldUpdated(ctx, event.Event{CampaignID: ids.CampaignID("camp-1")}, payload.GoldUpdatedPayload{
		CharacterID: ids.CharacterID("char-1"),
		Handfuls:    1,
		Bags:        2,
		Chests:      3,
	}); err != nil {
		t.Fatalf("HandleGoldUpdated() returned error: %v", err)
	}
	if got := store.profiles[profileKey("camp-1", "char-1")]; got.GoldHandfuls != 1 || got.GoldBags != 2 || got.GoldChests != 3 {
		t.Fatalf("profile after HandleGoldUpdated() = %+v, want updated gold", got)
	}

	if err := a.HandleDomainCardAcquired(ctx, event.Event{CampaignID: ids.CampaignID("camp-1")}, payload.DomainCardAcquiredPayload{
		CharacterID: ids.CharacterID("char-1"),
		CardID:      "domain.card-1",
	}); err != nil {
		t.Fatalf("HandleDomainCardAcquired() returned error: %v", err)
	}
	if err := a.HandleDomainCardAcquired(ctx, event.Event{CampaignID: ids.CampaignID("camp-1")}, payload.DomainCardAcquiredPayload{
		CharacterID: ids.CharacterID("char-1"),
		CardID:      "domain.card-1",
	}); err != nil {
		t.Fatalf("HandleDomainCardAcquired() duplicate returned error: %v", err)
	}
	if got := store.profiles[profileKey("camp-1", "char-1")].DomainCardIDs; !reflect.DeepEqual(got, []string{"domain.card-1"}) {
		t.Fatalf("DomainCardIDs after HandleDomainCardAcquired() = %#v, want unique card id", got)
	}

	if err := a.HandleConsumableUsed(ctx, event.Event{}, payload.ConsumableUsedPayload{}); err != nil {
		t.Fatalf("HandleConsumableUsed() returned error: %v", err)
	}
	if err := a.HandleConsumableAcquired(ctx, event.Event{}, payload.ConsumableAcquiredPayload{}); err != nil {
		t.Fatalf("HandleConsumableAcquired() returned error: %v", err)
	}
}

func TestGMAndCountdownHandlers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newProfileStoreStub()
	store.snapshot = projectionstore.DaggerheartSnapshot{
		CampaignID:            "camp-1",
		GMFear:                1,
		ConsecutiveShortRests: 2,
	}
	a := NewAdapter(store, nil)

	if err := a.HandleGMFearChanged(ctx, event.Event{CampaignID: ids.CampaignID("camp-1")}, payload.GMFearChangedPayload{Value: daggerheartstate.GMFearMax + 1}); err == nil {
		t.Fatal("HandleGMFearChanged() error = nil, want bounds error")
	}
	if err := a.HandleGMFearChanged(ctx, event.Event{CampaignID: ids.CampaignID("camp-1")}, payload.GMFearChangedPayload{Value: 3}); err != nil {
		t.Fatalf("HandleGMFearChanged() returned error: %v", err)
	}
	if store.snapshot.GMFear != 3 || store.snapshot.ConsecutiveShortRests != 2 {
		t.Fatalf("snapshot after HandleGMFearChanged() = %+v, want updated fear and preserved short rests", store.snapshot)
	}

	create := payload.SceneCountdownCreatedPayload{
		CountdownID:       dhids.CountdownID("count-1"),
		Name:              "Doom",
		Tone:              "danger",
		AdvancementPolicy: "manual",
		StartingValue:     4,
		RemainingValue:    1,
		LoopBehavior:      "none",
		Status:            "active",
	}
	if err := a.HandleSceneCountdownCreated(ctx, event.Event{CampaignID: ids.CampaignID("camp-1")}, create); err != nil {
		t.Fatalf("HandleSceneCountdownCreated() returned error: %v", err)
	}
	if got := store.countdowns[profileKey("camp-1", "count-1")]; got.Name != "Doom" || got.RemainingValue != 1 {
		t.Fatalf("countdown after HandleSceneCountdownCreated() = %+v, want stored countdown", got)
	}

	if err := a.HandleSceneCountdownAdvanced(ctx, event.Event{CampaignID: ids.CampaignID("camp-1")}, payload.SceneCountdownAdvancedPayload{
		CountdownID:     dhids.CountdownID("count-1"),
		BeforeRemaining: 1,
		AfterRemaining:  3,
		AdvancedBy:      2,
		StatusBefore:    "active",
		StatusAfter:     "active",
	}); err != nil {
		t.Fatalf("HandleSceneCountdownAdvanced() returned error: %v", err)
	}
	if got := store.countdowns[profileKey("camp-1", "count-1")]; got.RemainingValue != 3 {
		t.Fatalf("countdown after HandleSceneCountdownAdvanced() = %+v, want remaining_value=3", got)
	}

	if err := a.HandleSceneCountdownDeleted(ctx, event.Event{CampaignID: ids.CampaignID("camp-1")}, payload.SceneCountdownDeletedPayload{
		CountdownID: dhids.CountdownID("count-1"),
	}); err != nil {
		t.Fatalf("HandleSceneCountdownDeleted() returned error: %v", err)
	}
	if _, ok := store.countdowns[profileKey("camp-1", "count-1")]; ok {
		t.Fatal("countdown still present after HandleSceneCountdownDeleted()")
	}
}

func intPtr(v int) *int { return &v }

func strPtr(v string) *string { return &v }
