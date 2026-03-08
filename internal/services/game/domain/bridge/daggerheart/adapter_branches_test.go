package daggerheart

import (
	"context"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestHandleDowntimeMoveApplied_CreatesMissingCharacterState(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)
	err := adapter.handleDowntimeMoveApplied(context.Background(), event.Event{CampaignID: "camp-1"}, DowntimeMoveAppliedPayload{
		CharacterID: "char-1",
		Move:        "prepare",
		Hope:        intPtr(3),
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
		Conditions:  []string{"hidden"},
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
		Conditions:  []string{"hidden"},
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
	if err := adapter.applyConditionPatch(context.Background(), "camp-1", "char-1", []string{"hidden"}); err != nil {
		t.Fatalf("applyConditionPatch: %v", err)
	}
	state, err := store.GetDaggerheartCharacterState(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get character state: %v", err)
	}
	if len(state.Conditions) != 1 || state.Conditions[0] != "hidden" {
		t.Fatalf("conditions = %v, want [hidden]", state.Conditions)
	}
}

func TestApplyProfileTraitIncrease_KnownTraitsAndUnknownNoOp(t *testing.T) {
	profile := storage.DaggerheartCharacterProfile{}
	traits := []string{"agility", "strength", "finesse", "instinct", "presence", "knowledge"}
	for _, trait := range traits {
		applyProfileTraitIncrease(&profile, trait)
	}

	if profile.Agility != 1 || profile.Strength != 1 || profile.Finesse != 1 || profile.Instinct != 1 || profile.Presence != 1 || profile.Knowledge != 1 {
		t.Fatalf("trait values after increment = %+v, want each trait at 1", profile)
	}

	applyProfileTraitIncrease(&profile, "unknown")
	if profile.Agility != 1 || profile.Strength != 1 || profile.Finesse != 1 || profile.Instinct != 1 || profile.Presence != 1 || profile.Knowledge != 1 {
		t.Fatalf("unknown trait changed profile = %+v", profile)
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

func profileForArmorMax(campaignID, characterID string, armorMax int) storage.DaggerheartCharacterProfile {
	return storage.DaggerheartCharacterProfile{
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

func characterStateForArmorMax(campaignID, characterID string, armor int) storage.DaggerheartCharacterState {
	return storage.DaggerheartCharacterState{
		CampaignID:  campaignID,
		CharacterID: characterID,
		Hp:          6,
		Hope:        2,
		HopeMax:     6,
		Stress:      0,
		Armor:       armor,
	}
}
