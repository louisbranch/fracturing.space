package daggerheart

import (
	"errors"
	"testing"
)

func TestValidateTrait(t *testing.T) {
	tests := []struct {
		name    string
		trait   string
		value   int
		wantErr bool
	}{
		{"valid zero", "agility", 0, false},
		{"valid positive", "strength", 4, false},
		{"valid negative", "finesse", -2, false},
		{"invalid too high", "instinct", 5, true},
		{"invalid too low", "presence", -3, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTrait(tt.trait, tt.value)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for value %d", tt.value)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateTraits(t *testing.T) {
	validTraits := Traits{
		Agility:   2,
		Strength:  -1,
		Finesse:   0,
		Instinct:  4,
		Presence:  -2,
		Knowledge: 3,
	}
	if err := ValidateTraits(validTraits); err != nil {
		t.Errorf("unexpected error for valid traits: %v", err)
	}

	invalidTraits := Traits{
		Agility:   5, // too high
		Strength:  0,
		Finesse:   0,
		Instinct:  0,
		Presence:  0,
		Knowledge: 0,
	}
	err := ValidateTraits(invalidTraits)
	if err == nil {
		t.Error("expected error for invalid traits")
	}
	if !errors.Is(err, ErrInvalidTraitValue) {
		t.Errorf("expected ErrInvalidTraitValue, got %v", err)
	}
}

func TestDefaultTraits(t *testing.T) {
	traits := DefaultTraits()
	if traits.Agility != TraitDefault ||
		traits.Strength != TraitDefault ||
		traits.Finesse != TraitDefault ||
		traits.Instinct != TraitDefault ||
		traits.Presence != TraitDefault ||
		traits.Knowledge != TraitDefault {
		t.Error("expected all traits to be default value")
	}
}

func TestGetProfileDefaults(t *testing.T) {
	pc := GetProfileDefaults("PC")
	if pc.StressMax != PCStressMax {
		t.Errorf("expected PC stress max %d, got %d", PCStressMax, pc.StressMax)
	}
	if pc.Evasion != PCEvasion {
		t.Errorf("expected PC evasion %d, got %d", PCEvasion, pc.Evasion)
	}

	npc := GetProfileDefaults("NPC")
	if npc.StressMax != NPCStressMax {
		t.Errorf("expected NPC stress max %d, got %d", NPCStressMax, npc.StressMax)
	}
	if npc.Evasion != NPCEvasion {
		t.Errorf("expected NPC evasion %d, got %d", NPCEvasion, npc.Evasion)
	}

	// Unknown kind should default to PC
	unknown := GetProfileDefaults("UNKNOWN")
	if unknown.StressMax != PCStressMax {
		t.Errorf("expected unknown kind to default to PC stress max %d, got %d", PCStressMax, unknown.StressMax)
	}
}

func TestValidateProfile(t *testing.T) {
	validTraits := DefaultTraits()
	validTraits.Agility = 2

	tests := []struct {
		name            string
		stressMax       int
		evasion         int
		majorThreshold  int
		severeThreshold int
		traits          Traits
		wantErr         error
	}{
		{"valid", 6, 10, 8, 12, validTraits, nil},
		{"zero stress max is valid", 0, 10, 8, 12, validTraits, nil},
		{"negative stress max", -1, 10, 8, 12, validTraits, ErrInvalidStressMax},
		{"negative evasion", 6, -1, 8, 12, validTraits, ErrInvalidEvasion},
		{"negative major threshold", 6, 10, -1, 12, validTraits, ErrInvalidThresholds},
		{"severe < major", 6, 10, 12, 8, validTraits, ErrInvalidThresholds},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProfile(tt.stressMax, tt.evasion, tt.majorThreshold, tt.severeThreshold, tt.traits)
			if tt.wantErr == nil && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}
