package projection

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
)

func TestFallbackArmorMaxFromState(t *testing.T) {
	state := projectionstore.DaggerheartCharacterState{
		Armor: 5,
		TemporaryArmor: []projectionstore.DaggerheartTemporaryArmor{
			{Amount: 2},
			{Amount: 1},
		},
	}
	if got := FallbackArmorMaxFromState(state); got != 2 {
		t.Fatalf("FallbackArmorMaxFromState() = %d, want 2", got)
	}
}

func TestCharacterStateFromStorage_RoundTrip(t *testing.T) {
	state := projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          6,
		Hope:        2,
		HopeMax:     6,
		Stress:      1,
		Armor:       2,
		Conditions:  []string{"hidden"},
		TemporaryArmor: []projectionstore.DaggerheartTemporaryArmor{
			{Source: " ritual ", Duration: " short_rest ", SourceID: " src-1 ", Amount: 2},
		},
		LifeState: "",
	}
	domainState := CharacterStateFromStorage(state, 4)
	if domainState.LifeState == "" {
		t.Fatal("expected life state default to be populated")
	}
	stored := StorageCharacterStateFromDomain(&domainState)
	if stored.CampaignID != "camp-1" || stored.CharacterID != "char-1" {
		t.Fatalf("round-trip identity mismatch: %+v", stored)
	}
	if len(stored.TemporaryArmor) != 1 {
		t.Fatalf("temporary armor len = %d, want 1", len(stored.TemporaryArmor))
	}
	if stored.TemporaryArmor[0].Source != "ritual" || stored.TemporaryArmor[0].Duration != "short_rest" || stored.TemporaryArmor[0].SourceID != "src-1" {
		t.Fatalf("temporary armor not normalized: %+v", stored.TemporaryArmor[0])
	}
}

func TestApplyStatePatch_ValidatesRanges(t *testing.T) {
	state := projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          6,
		Hope:        2,
		HopeMax:     6,
		Stress:      1,
		Armor:       0,
	}
	badHope := 7
	_, err := ApplyStatePatch(state, 0, nil, &badHope, nil, nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "hope must be in range") {
		t.Fatalf("expected hope range error, got %v", err)
	}

	goodHope := 3
	next, err := ApplyStatePatch(state, 0, nil, &goodHope, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("ApplyStatePatch: %v", err)
	}
	if next.Hope != 3 {
		t.Fatalf("hope = %d, want 3", next.Hope)
	}
}

func TestApplyConditionPatch_ReplacesConditions(t *testing.T) {
	state := projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Conditions:  []string{"old"},
	}
	next := ApplyConditionPatch(state, 0, []string{"new"})
	if len(next.Conditions) != 1 || next.Conditions[0] != "new" {
		t.Fatalf("conditions = %v, want [new]", next.Conditions)
	}
}

func TestApplyTemporaryArmor_AndClearRestTemporaryArmor(t *testing.T) {
	state := projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          6,
		Hope:        2,
		HopeMax:     6,
		Stress:      0,
		Armor:       0,
	}
	patched, err := ApplyTemporaryArmor(state, 1, "ritual", "short_rest", "id-1", 2)
	if err != nil {
		t.Fatalf("ApplyTemporaryArmor: %v", err)
	}
	if patched.Armor != 2 {
		t.Fatalf("armor = %d, want 2", patched.Armor)
	}
	cleared, changed := ClearRestTemporaryArmor(patched, 1, true, false)
	if !changed {
		t.Fatal("expected clear rest temporary armor to report changed=true")
	}
	if cleared.Armor != 1 {
		t.Fatalf("armor after clear = %d, want 1", cleared.Armor)
	}
}

func TestApplyDowntimeMove_RepairArmorPath(t *testing.T) {
	state := projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          6,
		Hope:        2,
		HopeMax:     6,
		Stress:      2,
		Armor:       1,
		TemporaryArmor: []projectionstore.DaggerheartTemporaryArmor{
			{Source: "ritual", Duration: "short_rest", Amount: 1},
		},
	}
	next, err := ApplyDowntimeMove(state, 3, "repair_all_armor", nil, nil, nil)
	if err != nil {
		t.Fatalf("ApplyDowntimeMove: %v", err)
	}
	if next.Armor != 3 {
		t.Fatalf("armor = %d, want 3", next.Armor)
	}
}

func TestValidateAdversaryStats(t *testing.T) {
	if err := ValidateAdversaryStats(6, 6, 1, 2, 10, 2, 4, 1); err != nil {
		t.Fatalf("unexpected valid stats error: %v", err)
	}
	if err := ValidateAdversaryStats(7, 6, 1, 2, 10, 2, 4, 1); err == nil {
		t.Fatal("expected hp range error")
	}
	if err := ValidateAdversaryStats(6, 6, 1, 2, 10, 3, 2, 1); err == nil {
		t.Fatal("expected severe >= major error")
	}
}

func TestApplyAdversaryDamagePatch(t *testing.T) {
	adversary := projectionstore.DaggerheartAdversary{
		CampaignID:  "camp-1",
		AdversaryID: "adv-1",
		HP:          6,
		HPMax:       6,
		Stress:      1,
		StressMax:   2,
		Evasion:     10,
		Major:       2,
		Severe:      4,
		Armor:       1,
	}
	hp := 4
	next, err := ApplyAdversaryDamagePatch(adversary, &hp, nil)
	if err != nil {
		t.Fatalf("ApplyAdversaryDamagePatch: %v", err)
	}
	if next.HP != 4 || next.Armor != 1 {
		t.Fatalf("patched adversary = %+v", next)
	}
}

func TestApplyCountdownUpdate(t *testing.T) {
	countdown := projectionstore.DaggerheartCountdown{
		CampaignID:  "camp-1",
		CountdownID: "cd-1",
		Current:     1,
		Max:         4,
	}
	if _, err := ApplyCountdownUpdate(countdown, 5); err == nil {
		t.Fatal("expected out-of-range error")
	}
	next, err := ApplyCountdownUpdate(countdown, 2)
	if err != nil {
		t.Fatalf("ApplyCountdownUpdate: %v", err)
	}
	if next.Current != 2 {
		t.Fatalf("current = %d, want 2", next.Current)
	}
}
