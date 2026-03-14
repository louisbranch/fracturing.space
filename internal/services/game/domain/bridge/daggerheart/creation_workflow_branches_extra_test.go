package daggerheart

import (
	"testing"

	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/profile"
)

func TestEvaluateCreationReadiness_ErrorBranches(t *testing.T) {
	ready, reason := EvaluateCreationReadiness(CharacterProfile{
		Level:           0,
		HpMax:           -1,
		StressMax:       0,
		Evasion:         0,
		MajorThreshold:  0,
		SevereThreshold: 0,
		Proficiency:     0,
		ArmorScore:      0,
		ArmorMax:        0,
	})
	if ready {
		t.Fatal("ready = true, want false")
	}
	if reason != "class and subclass selection is required" {
		t.Fatalf("reason = %q, want %q", reason, "class and subclass selection is required")
	}
}

func TestHasStartingEquipment_Branches(t *testing.T) {
	tests := []struct {
		name    string
		profile CreationProfile
		want    bool
	}{
		{
			name: "missing weapons",
			profile: CreationProfile{
				StartingArmorID:      "armor-1",
				StartingPotionItemID: StartingPotionMinorHealthID,
			},
			want: false,
		},
		{
			name: "blank weapon id",
			profile: CreationProfile{
				StartingWeaponIDs:    []string{"weapon-1", " "},
				StartingArmorID:      "armor-1",
				StartingPotionItemID: StartingPotionMinorHealthID,
			},
			want: false,
		},
		{
			name: "blank armor",
			profile: CreationProfile{
				StartingWeaponIDs:    []string{"weapon-1"},
				StartingArmorID:      " ",
				StartingPotionItemID: StartingPotionMinorHealthID,
			},
			want: false,
		},
		{
			name: "invalid potion",
			profile: CreationProfile{
				StartingWeaponIDs:    []string{"weapon-1"},
				StartingArmorID:      "armor-1",
				StartingPotionItemID: "not-starting-potion",
			},
			want: false,
		},
		{
			name: "valid",
			profile: CreationProfile{
				StartingWeaponIDs:    []string{"weapon-1"},
				StartingArmorID:      "armor-1",
				StartingPotionItemID: StartingPotionMinorStaminaID,
			},
			want: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasStartingEquipment(tc.profile); got != tc.want {
				t.Fatalf("hasStartingEquipment() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestHasExperiences_AndHasDomainCardIDs_Branches(t *testing.T) {
	if hasExperiences(nil) {
		t.Fatal("expected hasExperiences(nil) = false")
	}
	if hasExperiences([]daggerheartprofile.Experience{{Name: " "}}) {
		t.Fatal("expected hasExperiences(blank name) = false")
	}
	if hasExperiences([]daggerheartprofile.Experience{{Name: "Scout"}}) {
		t.Fatal("expected hasExperiences(1 experience) = false (need exactly 2)")
	}
	if !hasExperiences([]daggerheartprofile.Experience{{Name: "Scout"}, {Name: "Patrol"}}) {
		t.Fatal("expected hasExperiences(2 valid) = true")
	}

	if hasDomainCardIDs(nil) {
		t.Fatal("expected hasDomainCardIDs(nil) = false")
	}
	if hasDomainCardIDs([]string{"card-1", " "}) {
		t.Fatal("expected hasDomainCardIDs(blank id) = false")
	}
	if hasDomainCardIDs([]string{"card-1"}) {
		t.Fatal("expected hasDomainCardIDs(1 card) = false (need exactly 2)")
	}
	if !hasDomainCardIDs([]string{"card-1", "card-2"}) {
		t.Fatal("expected hasDomainCardIDs(2 valid) = true")
	}
}
