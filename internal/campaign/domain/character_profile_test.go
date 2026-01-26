package domain

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateCharacterProfileSuccess(t *testing.T) {
	input := CreateCharacterProfileInput{
		CampaignID:      "camp-123",
		CharacterID:     "char-456",
		Traits:          map[string]int{"agility": 1},
		HpMax:           6,
		StressMax:       4,
		Evasion:         3,
		MajorThreshold:  2,
		SevereThreshold: 5,
	}

	profile, err := CreateCharacterProfile(input)
	if err != nil {
		t.Fatalf("create character profile: %v", err)
	}
	if profile.CampaignID != input.CampaignID {
		t.Fatalf("expected campaign id %q, got %q", input.CampaignID, profile.CampaignID)
	}
	if profile.CharacterID != input.CharacterID {
		t.Fatalf("expected character id %q, got %q", input.CharacterID, profile.CharacterID)
	}
	if profile.Traits["agility"] != 1 {
		t.Fatalf("expected agility 1, got %d", profile.Traits["agility"])
	}
	if profile.HpMax != 6 || profile.StressMax != 4 || profile.Evasion != 3 {
		t.Fatalf("expected hp/stress/evasion preserved")
	}
	if profile.MajorThreshold != 2 || profile.SevereThreshold != 5 {
		t.Fatalf("expected thresholds preserved")
	}
}

func TestCreateCharacterProfileValidationErrors(t *testing.T) {
	tests := []struct {
		name  string
		input CreateCharacterProfileInput
		err   error
	}{
		{
			name: "invalid hp max",
			input: CreateCharacterProfileInput{
				Traits:          map[string]int{"agility": 0},
				HpMax:           0,
				StressMax:       0,
				Evasion:         0,
				MajorThreshold:  0,
				SevereThreshold: 0,
			},
			err: ErrInvalidProfileHpMax,
		},
		{
			name: "invalid stress max",
			input: CreateCharacterProfileInput{
				Traits:          map[string]int{"agility": 0},
				HpMax:           1,
				StressMax:       -1,
				Evasion:         0,
				MajorThreshold:  0,
				SevereThreshold: 0,
			},
			err: ErrInvalidProfileStressMax,
		},
		{
			name: "invalid evasion",
			input: CreateCharacterProfileInput{
				Traits:          map[string]int{"agility": 0},
				HpMax:           1,
				StressMax:       0,
				Evasion:         -1,
				MajorThreshold:  0,
				SevereThreshold: 0,
			},
			err: ErrInvalidProfileEvasion,
		},
		{
			name: "invalid thresholds",
			input: CreateCharacterProfileInput{
				Traits:          map[string]int{"agility": 0},
				HpMax:           1,
				StressMax:       0,
				Evasion:         0,
				MajorThreshold:  3,
				SevereThreshold: 2,
			},
			err: ErrInvalidProfileThresholds,
		},
		{
			name: "invalid trait",
			input: CreateCharacterProfileInput{
				Traits:          map[string]int{"agility": 9},
				HpMax:           1,
				StressMax:       0,
				Evasion:         0,
				MajorThreshold:  0,
				SevereThreshold: 0,
			},
			err: ErrInvalidTraitValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CreateCharacterProfile(tt.input)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expected error %v, got %v", tt.err, err)
			}
		})
	}
}

func TestPatchCharacterProfileUpdates(t *testing.T) {
	existing := CharacterProfile{
		CampaignID:      "camp-123",
		CharacterID:     "char-456",
		Traits:          map[string]int{"agility": 0},
		HpMax:           6,
		StressMax:       4,
		Evasion:         3,
		MajorThreshold:  2,
		SevereThreshold: 5,
	}
	newHpMax := 8
	newStressMax := 5
	patch := PatchCharacterProfileInput{
		Traits:    map[string]int{"agility": 2},
		HpMax:     &newHpMax,
		StressMax: &newStressMax,
	}

	updated, err := PatchCharacterProfile(existing, patch)
	if err != nil {
		t.Fatalf("patch character profile: %v", err)
	}
	if updated.Traits["agility"] != 2 {
		t.Fatalf("expected updated agility, got %d", updated.Traits["agility"])
	}
	if updated.HpMax != 8 || updated.StressMax != 5 {
		t.Fatalf("expected hp/stress updated")
	}
	if updated.Evasion != existing.Evasion {
		t.Fatalf("expected evasion to remain %d", existing.Evasion)
	}
}

func TestPatchCharacterProfileInvalid(t *testing.T) {
	existing := CharacterProfile{
		Traits:          map[string]int{"agility": 0},
		HpMax:           6,
		StressMax:       4,
		Evasion:         3,
		MajorThreshold:  2,
		SevereThreshold: 5,
	}
	invalidHpMax := 0
	patch := PatchCharacterProfileInput{HpMax: &invalidHpMax}

	_, err := PatchCharacterProfile(existing, patch)
	if !errors.Is(err, ErrInvalidProfileHpMax) {
		t.Fatalf("expected error %v, got %v", ErrInvalidProfileHpMax, err)
	}
}

func TestLoadCharacterDefaultsFromPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "defaults.json")
	payload := []byte(`{
  "PC": {
    "traits": {"agility": 0},
    "hp_max": 6,
    "stress_max": 4,
    "evasion": 3,
    "major_threshold": 2,
    "severe_threshold": 5
  },
  "NPC": {
    "traits": {"agility": 0},
    "hp_max": 3,
    "stress_max": 2,
    "evasion": 1,
    "major_threshold": 1,
    "severe_threshold": 2
  }
}`)
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatalf("write defaults: %v", err)
	}

	defaults, err := LoadCharacterDefaults(path)
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}
	pc, ok := defaults[CharacterKindPC]
	if !ok {
		t.Fatal("expected pc defaults")
	}
	if pc.HpMax != 6 {
		t.Fatalf("expected pc hp max 6, got %d", pc.HpMax)
	}
}

func TestLoadCharacterDefaultsInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "defaults.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatalf("write defaults: %v", err)
	}

	_, err := LoadCharacterDefaults(path)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetDefaultProfileMissingKind(t *testing.T) {
	defaults := map[CharacterKind]CharacterProfileDefaults{}
	_, err := GetDefaultProfile(CharacterKindPC, defaults)
	if err == nil {
		t.Fatal("expected error")
	}
}
