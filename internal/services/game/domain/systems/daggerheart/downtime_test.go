package daggerheart

import "testing"

func TestApplyDowntimeMove(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          6,
		HPMax:       6,
		Hope:        1,
		HopeMax:     HopeMax,
		Stress:      3,
		StressMax:   6,
		Armor:       0,
		ArmorMax:    2,
	})

	result := ApplyDowntimeMove(state, DowntimeClearAllStress, DowntimeOptions{})
	if result.StressAfter != 0 {
		t.Fatalf("stress after = %d, want 0", result.StressAfter)
	}

	result = ApplyDowntimeMove(state, DowntimeRepairAllArmor, DowntimeOptions{})
	if result.ArmorAfter != 2 {
		t.Fatalf("armor after = %d, want 2", result.ArmorAfter)
	}

	result = ApplyDowntimeMove(state, DowntimePrepare, DowntimeOptions{})
	if result.HopeAfter != 2 {
		t.Fatalf("hope after = %d, want 2", result.HopeAfter)
	}

	result = ApplyDowntimeMove(state, DowntimePrepare, DowntimeOptions{PrepareWithGroup: true})
	if result.HopeAfter != 4 {
		t.Fatalf("hope after = %d, want 4", result.HopeAfter)
	}
}
