package daggerheart

import "testing"

func TestWithActiveCompanionExperience(t *testing.T) {
	state := CharacterCompanionState{Status: CompanionStatusPresent}
	got := WithActiveCompanionExperience(state, "exp-1")
	if got.Status != CompanionStatusAway {
		t.Fatalf("status = %q, want %q", got.Status, CompanionStatusAway)
	}
	if got.ActiveExperienceID != "exp-1" {
		t.Fatalf("active experience = %q, want %q", got.ActiveExperienceID, "exp-1")
	}
}

func TestWithActiveCompanionExperience_EmptyExperienceReturnsPresent(t *testing.T) {
	state := CharacterCompanionState{Status: CompanionStatusAway, ActiveExperienceID: "exp-1"}
	got := WithActiveCompanionExperience(state, "  ")
	// Normalized: away with empty experience ID reverts to present.
	if got.Status != CompanionStatusPresent {
		t.Fatalf("status = %q, want %q", got.Status, CompanionStatusPresent)
	}
}

func TestWithCompanionPresent(t *testing.T) {
	state := CharacterCompanionState{Status: CompanionStatusAway, ActiveExperienceID: "exp-1"}
	got := WithCompanionPresent(state)
	if got.Status != CompanionStatusPresent {
		t.Fatalf("status = %q, want %q", got.Status, CompanionStatusPresent)
	}
	if got.ActiveExperienceID != "" {
		t.Fatalf("active experience = %q, want empty", got.ActiveExperienceID)
	}
}

func TestCharacterCompanionState_IsZero(t *testing.T) {
	if !(CharacterCompanionState{}).IsZero() {
		t.Fatal("zero-value companion state should be IsZero")
	}
	if (CharacterCompanionState{Status: CompanionStatusAway, ActiveExperienceID: "exp-1"}).IsZero() {
		t.Fatal("away companion state should not be IsZero")
	}
}

func TestCharacterCompanionState_Normalized(t *testing.T) {
	// Unknown status normalizes to present.
	got := CharacterCompanionState{Status: "flying"}.Normalized()
	if got.Status != CompanionStatusPresent {
		t.Fatalf("unknown status normalized = %q, want %q", got.Status, CompanionStatusPresent)
	}

	// Whitespace trimming.
	got = CharacterCompanionState{Status: " AWAY ", ActiveExperienceID: " exp-1 "}.Normalized()
	if got.Status != CompanionStatusAway {
		t.Fatalf("trimmed status = %q, want %q", got.Status, CompanionStatusAway)
	}
	if got.ActiveExperienceID != "exp-1" {
		t.Fatalf("trimmed experience = %q, want %q", got.ActiveExperienceID, "exp-1")
	}
}
