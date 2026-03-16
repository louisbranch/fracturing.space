package daggerheart

import (
	"context"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestHandleDowntimeMoveApplied_CreatesMissingCharacterState(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)
	err := adapter.handleDowntimeMoveApplied(context.Background(), event.Event{CampaignID: "camp-1"}, DowntimeMoveAppliedPayload{
		ActorCharacterID:  "char-1",
		TargetCharacterID: "char-1",
		Move:              "prepare",
		Hope:              intPtr(3),
	})
	if err != nil {
		t.Fatalf("handleDowntimeMoveApplied: %v", err)
	}
	state, err := store.GetDaggerheartCharacterState(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get character state: %v", err)
	}
	if state.Hope != 3 {
		t.Fatalf("hope = %d, want 3", state.Hope)
	}
}

func TestHandleCharacterTemporaryArmorApplied_CreatesMissingCharacterState(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)
	err := adapter.handleCharacterTemporaryArmorApplied(context.Background(), event.Event{CampaignID: "camp-1"}, CharacterTemporaryArmorAppliedPayload{
		CharacterID: "char-1",
		Source:      "ritual",
		Duration:    "short_rest",
		Amount:      2,
	})
	if err != nil {
		t.Fatalf("handleCharacterTemporaryArmorApplied: %v", err)
	}
	state, err := store.GetDaggerheartCharacterState(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get character state: %v", err)
	}
	if state.Armor != 2 {
		t.Fatalf("armor = %d, want 2", state.Armor)
	}
}

func TestHandleConditionChanged_RejectsZeroRollSeq(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)
	rollSeq := uint64(0)
	err := adapter.handleConditionChanged(context.Background(), event.Event{CampaignID: "camp-1"}, ConditionChangedPayload{
		CharacterID: "char-1",
		Conditions:  []ConditionState{mustTestConditionState(t, "hidden")},
		RollSeq:     &rollSeq,
	})
	if err == nil || !strings.Contains(err.Error(), "roll_seq must be positive") {
		t.Fatalf("expected roll_seq validation error, got %v", err)
	}
}

func TestHandleAdversaryConditionChanged_RejectsZeroRollSeq(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)
	rollSeq := uint64(0)
	err := adapter.handleAdversaryConditionChanged(context.Background(), event.Event{CampaignID: "camp-1"}, AdversaryConditionChangedPayload{
		AdversaryID: "adv-1",
		Conditions:  []ConditionState{mustTestConditionState(t, "hidden")},
		RollSeq:     &rollSeq,
	})
	if err == nil || !strings.Contains(err.Error(), "roll_seq must be positive") {
		t.Fatalf("expected roll_seq validation error, got %v", err)
	}
}

func TestCharacterArmorMax_UsesProfileWhenPresent(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)
	if err := store.PutDaggerheartCharacterProfile(context.Background(), profileForArmorMax("camp-1", "char-1", 5)); err != nil {
		t.Fatalf("put profile: %v", err)
	}
	armorMax, err := adapter.characterArmorMax(context.Background(), characterStateForArmorMax("camp-1", "char-1", 2))
	if err != nil {
		t.Fatalf("characterArmorMax: %v", err)
	}
	if armorMax != 5 {
		t.Fatalf("armor max = %d, want 5", armorMax)
	}
}

func TestClearRestTemporaryArmor_NoStateNoError(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)
	if err := adapter.clearRestTemporaryArmor(context.Background(), "camp-1", "char-1", true, false); err != nil {
		t.Fatalf("clearRestTemporaryArmor: %v", err)
	}
}

func TestApplyConditionPatch_CreatesStateWhenMissing(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)
	if err := adapter.applyConditionPatch(context.Background(), "camp-1", "char-1", []ConditionState{mustTestConditionState(t, "hidden")}); err != nil {
		t.Fatalf("applyConditionPatch: %v", err)
	}
	state, err := store.GetDaggerheartCharacterState(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get character state: %v", err)
	}
	if len(state.Conditions) != 1 || state.Conditions[0].Code != "hidden" {
		t.Fatalf("conditions = %v, want [hidden]", state.Conditions)
	}
}

func mustTestConditionState(t *testing.T, code string) ConditionState {
	t.Helper()
	return mustConditionState(code)
}

func mustConditionState(code string) ConditionState {
	state, err := StandardConditionState(code)
	if err != nil {
		panic(err)
	}
	return state
}

func TestApplyProfileTraitIncrease_KnownTraitsAndUnknownNoOp(t *testing.T) {
	profile := CharacterProfile{}
	traits := []string{"agility", "strength", "finesse", "instinct", "presence", "knowledge"}
	for _, trait := range traits {
		applyCharacterProfileTraitIncrease(&profile, trait)
	}

	if profile.Agility != 1 || profile.Strength != 1 || profile.Finesse != 1 || profile.Instinct != 1 || profile.Presence != 1 || profile.Knowledge != 1 {
		t.Fatalf("trait values after increment = %+v, want each trait at 1", profile)
	}

	applyCharacterProfileTraitIncrease(&profile, "unknown")
	if profile.Agility != 1 || profile.Strength != 1 || profile.Finesse != 1 || profile.Instinct != 1 || profile.Presence != 1 || profile.Knowledge != 1 {
		t.Fatalf("unknown trait changed profile = %+v", profile)
	}
}

func TestHandleGoldUpdated_UpdatesExistingProfileAndSkipsMissing(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)

	if err := adapter.handleGoldUpdated(context.Background(), event.Event{CampaignID: "camp-1"}, GoldUpdatedPayload{
		CharacterID: "missing",
		Handfuls:    1,
		Bags:        2,
		Chests:      3,
	}); err != nil {
		t.Fatalf("handleGoldUpdated missing profile: %v", err)
	}

	profile := profileForArmorMax("camp-1", "char-1", 2)
	if err := store.PutDaggerheartCharacterProfile(context.Background(), profile); err != nil {
		t.Fatalf("put profile: %v", err)
	}

	if err := adapter.handleGoldUpdated(context.Background(), event.Event{CampaignID: "camp-1"}, GoldUpdatedPayload{
		CharacterID: "char-1",
		Handfuls:    4,
		Bags:        5,
		Chests:      6,
	}); err != nil {
		t.Fatalf("handleGoldUpdated existing profile: %v", err)
	}

	updated, err := store.GetDaggerheartCharacterProfile(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if updated.GoldHandfuls != 4 || updated.GoldBags != 5 || updated.GoldChests != 6 {
		t.Fatalf("gold = (%d, %d, %d), want (4, 5, 6)", updated.GoldHandfuls, updated.GoldBags, updated.GoldChests)
	}
}

func TestHandleDomainCardAcquired_AppendsUniqueAndSkipsMissing(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)

	if err := adapter.handleDomainCardAcquired(context.Background(), event.Event{CampaignID: "camp-1"}, DomainCardAcquiredPayload{
		CharacterID: "missing",
		CardID:      "card-1",
	}); err != nil {
		t.Fatalf("handleDomainCardAcquired missing profile: %v", err)
	}

	profile := profileForArmorMax("camp-1", "char-1", 2)
	profile.DomainCardIDs = []string{"card-1"}
	if err := store.PutDaggerheartCharacterProfile(context.Background(), profile); err != nil {
		t.Fatalf("put profile: %v", err)
	}

	if err := adapter.handleDomainCardAcquired(context.Background(), event.Event{CampaignID: "camp-1"}, DomainCardAcquiredPayload{
		CharacterID: "char-1",
		CardID:      "card-2",
	}); err != nil {
		t.Fatalf("handleDomainCardAcquired append: %v", err)
	}
	if err := adapter.handleDomainCardAcquired(context.Background(), event.Event{CampaignID: "camp-1"}, DomainCardAcquiredPayload{
		CharacterID: "char-1",
		CardID:      "card-1",
	}); err != nil {
		t.Fatalf("handleDomainCardAcquired duplicate: %v", err)
	}

	updated, err := store.GetDaggerheartCharacterProfile(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if len(updated.DomainCardIDs) != 2 {
		t.Fatalf("domain card count = %d, want 2", len(updated.DomainCardIDs))
	}
	if updated.DomainCardIDs[0] != "card-1" || updated.DomainCardIDs[1] != "card-2" {
		t.Fatalf("domain cards = %v, want [card-1 card-2]", updated.DomainCardIDs)
	}
}

func TestAppendUnique_AppendsMissingOnly(t *testing.T) {
	initial := []string{"card-1", "card-2"}
	withNew := appendUnique(initial, "card-3")
	if len(withNew) != 3 {
		t.Fatalf("len(withNew) = %d, want 3", len(withNew))
	}
	if withNew[2] != "card-3" {
		t.Fatalf("last value = %q, want %q", withNew[2], "card-3")
	}

	withDuplicate := appendUnique(withNew, "card-2")
	if len(withDuplicate) != 3 {
		t.Fatalf("len(withDuplicate) = %d, want 3", len(withDuplicate))
	}
	if withDuplicate[0] != "card-1" || withDuplicate[1] != "card-2" || withDuplicate[2] != "card-3" {
		t.Fatalf("unexpected order/content after duplicate append: %v", withDuplicate)
	}
}

func TestHandleEquipmentSwapped_ArmorUpdatesProfileAndState(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)

	profile := profileForArmorMax("camp-1", "char-1", 2)
	profile.Agility = 1
	profile.Strength = 1
	profile.Finesse = 1
	profile.Instinct = 1
	profile.Presence = 1
	profile.Knowledge = 1
	if err := store.PutDaggerheartCharacterProfile(context.Background(), profile); err != nil {
		t.Fatalf("put profile: %v", err)
	}
	if err := store.PutDaggerheartCharacterState(context.Background(), projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          6,
		Hope:        2,
		HopeMax:     6,
		Stress:      1,
		Armor:       2,
	}); err != nil {
		t.Fatalf("put state: %v", err)
	}

	err := adapter.handleEquipmentSwapped(context.Background(), event.Event{CampaignID: "camp-1"}, EquipmentSwappedPayload{
		CharacterID:             "char-1",
		ItemType:                "armor",
		EquippedArmorID:         "armor.chainmail-armor",
		EvasionAfter:            intPtr(8),
		MajorThresholdAfter:     intPtr(7),
		SevereThresholdAfter:    intPtr(15),
		ArmorScoreAfter:         intPtr(4),
		ArmorMaxAfter:           intPtr(4),
		SpellcastRollBonusAfter: intPtr(1),
		AgilityAfter:            intPtr(0),
		StrengthAfter:           intPtr(0),
		FinesseAfter:            intPtr(0),
		InstinctAfter:           intPtr(0),
		PresenceAfter:           intPtr(0),
		KnowledgeAfter:          intPtr(0),
		ArmorAfter:              intPtr(4),
		StressCost:              2,
	})
	if err != nil {
		t.Fatalf("handleEquipmentSwapped: %v", err)
	}

	updatedProfile, err := store.GetDaggerheartCharacterProfile(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if updatedProfile.EquippedArmorID != "armor.chainmail-armor" ||
		updatedProfile.Evasion != 8 ||
		updatedProfile.MajorThreshold != 7 ||
		updatedProfile.SevereThreshold != 15 ||
		updatedProfile.ArmorScore != 4 ||
		updatedProfile.ArmorMax != 4 ||
		updatedProfile.SpellcastRollBonus != 1 {
		t.Fatalf("updated profile core armor fields = %+v", updatedProfile)
	}
	if updatedProfile.Agility != 0 || updatedProfile.Strength != 0 || updatedProfile.Finesse != 0 ||
		updatedProfile.Instinct != 0 || updatedProfile.Presence != 0 || updatedProfile.Knowledge != 0 {
		t.Fatalf("updated profile traits = %+v", updatedProfile)
	}

	updatedState, err := store.GetDaggerheartCharacterState(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if updatedState.Armor != 4 || updatedState.Stress != 3 {
		t.Fatalf("updated state = %+v, want armor=4 stress=3", updatedState)
	}
}

func profileForArmorMax(campaignID, characterID string, armorMax int) projectionstore.DaggerheartCharacterProfile {
	return projectionstore.DaggerheartCharacterProfile{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		Level:           1,
		HpMax:           6,
		StressMax:       6,
		Evasion:         10,
		MajorThreshold:  3,
		SevereThreshold: 6,
		Proficiency:     1,
		ArmorScore:      0,
		ArmorMax:        armorMax,
	}
}

func characterStateForArmorMax(campaignID, characterID string, armor int) projectionstore.DaggerheartCharacterState {
	return projectionstore.DaggerheartCharacterState{
		CampaignID:  campaignID,
		CharacterID: characterID,
		Hp:          6,
		Hope:        2,
		HopeMax:     6,
		Stress:      0,
		Armor:       armor,
	}
}
