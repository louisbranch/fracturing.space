package daggerheart

import (
	"testing"

	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func TestWithActiveCompanionExperience(t *testing.T) {
	state := daggerheartstate.CharacterCompanionState{Status: daggerheartstate.CompanionStatusPresent}
	got := daggerheartstate.WithActiveCompanionExperience(state, "exp-1")
	if got.Status != daggerheartstate.CompanionStatusAway {
		t.Fatalf("status = %q, want %q", got.Status, daggerheartstate.CompanionStatusAway)
	}
	if got.ActiveExperienceID != "exp-1" {
		t.Fatalf("active experience = %q, want %q", got.ActiveExperienceID, "exp-1")
	}
}

func TestWithActiveCompanionExperience_EmptyExperienceReturnsPresent(t *testing.T) {
	state := daggerheartstate.CharacterCompanionState{Status: daggerheartstate.CompanionStatusAway, ActiveExperienceID: "exp-1"}
	got := daggerheartstate.WithActiveCompanionExperience(state, "  ")
	// Normalized: away with empty experience ID reverts to present.
	if got.Status != daggerheartstate.CompanionStatusPresent {
		t.Fatalf("status = %q, want %q", got.Status, daggerheartstate.CompanionStatusPresent)
	}
}

func TestWithCompanionPresent(t *testing.T) {
	state := daggerheartstate.CharacterCompanionState{Status: daggerheartstate.CompanionStatusAway, ActiveExperienceID: "exp-1"}
	got := daggerheartstate.WithCompanionPresent(state)
	if got.Status != daggerheartstate.CompanionStatusPresent {
		t.Fatalf("status = %q, want %q", got.Status, daggerheartstate.CompanionStatusPresent)
	}
	if got.ActiveExperienceID != "" {
		t.Fatalf("active experience = %q, want empty", got.ActiveExperienceID)
	}
}

func TestCharacterCompanionState_IsZero(t *testing.T) {
	if !(daggerheartstate.CharacterCompanionState{}).IsZero() {
		t.Fatal("zero-value companion state should be IsZero")
	}
	if (daggerheartstate.CharacterCompanionState{Status: daggerheartstate.CompanionStatusAway, ActiveExperienceID: "exp-1"}).IsZero() {
		t.Fatal("away companion state should not be IsZero")
	}
}

func TestCharacterCompanionState_Normalized(t *testing.T) {
	// Unknown status normalizes to present.
	got := daggerheartstate.CharacterCompanionState{Status: "flying"}.Normalized()
	if got.Status != daggerheartstate.CompanionStatusPresent {
		t.Fatalf("unknown status normalized = %q, want %q", got.Status, daggerheartstate.CompanionStatusPresent)
	}

	// Whitespace trimming.
	got = daggerheartstate.CharacterCompanionState{Status: " AWAY ", ActiveExperienceID: " exp-1 "}.Normalized()
	if got.Status != daggerheartstate.CompanionStatusAway {
		t.Fatalf("trimmed status = %q, want %q", got.Status, daggerheartstate.CompanionStatusAway)
	}
	if got.ActiveExperienceID != "exp-1" {
		t.Fatalf("trimmed experience = %q, want %q", got.ActiveExperienceID, "exp-1")
	}
}
