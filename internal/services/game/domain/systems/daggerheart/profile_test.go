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
	if pc.Level != PCLevelDefault {
		t.Errorf("expected PC level %d, got %d", PCLevelDefault, pc.Level)
	}
	if pc.Evasion != PCEvasion {
		t.Errorf("expected PC evasion %d, got %d", PCEvasion, pc.Evasion)
	}

	npc := GetProfileDefaults("NPC")
	if npc.StressMax != NPCStressMax {
		t.Errorf("expected NPC stress max %d, got %d", NPCStressMax, npc.StressMax)
	}
	if npc.Level != NPCLevelDefault {
		t.Errorf("expected NPC level %d, got %d", NPCLevelDefault, npc.Level)
	}
	if npc.Evasion != NPCEvasion {
		t.Errorf("expected NPC evasion %d, got %d", NPCEvasion, npc.Evasion)
	}

	// Unknown kind should default to PC
	unknown := GetProfileDefaults("UNKNOWN")
	if unknown.StressMax != PCStressMax {
		t.Errorf("expected unknown kind to default to PC stress max %d, got %d", PCStressMax, unknown.StressMax)
	}
	if unknown.Level != PCLevelDefault {
		t.Errorf("expected unknown kind to default to PC level %d, got %d", PCLevelDefault, unknown.Level)
	}
}

func TestValidateProfile(t *testing.T) {
	validTraits := DefaultTraits()
	validTraits.Agility = 2

	tests := []struct {
		name            string
		level           int
		hpMax           int
		stressMax       int
		evasion         int
		majorThreshold  int
		severeThreshold int
		proficiency     int
		armorScore      int
		armorMax        int
		experiences     []Experience
		traits          Traits
		wantErr         error
	}{
		{"valid", 1, 6, 6, 10, 8, 12, 1, 0, 0, nil, validTraits, nil},
		{"zero stress max is valid", 1, 6, 0, 10, 8, 12, 1, 0, 0, nil, validTraits, nil},
		{"level too low", 0, 6, 6, 10, 8, 12, 1, 0, 0, nil, validTraits, ErrInvalidLevel},
		{"level too high", 11, 6, 6, 10, 8, 12, 1, 0, 0, nil, validTraits, ErrInvalidLevel},
		{"hp max too low", 1, 0, 6, 10, 8, 12, 1, 0, 0, nil, validTraits, ErrInvalidHpMax},
		{"hp max too high", 1, 13, 6, 10, 8, 12, 1, 0, 0, nil, validTraits, ErrInvalidHpMax},
		{"stress max too high", 1, 6, 13, 10, 8, 12, 1, 0, 0, nil, validTraits, ErrInvalidStressMax},
		{"negative stress max", 1, 6, -1, 10, 8, 12, 1, 0, 0, nil, validTraits, ErrInvalidStressMax},
		{"negative evasion", 1, 6, 6, -1, 8, 12, 1, 0, 0, nil, validTraits, ErrInvalidEvasion},
		{"negative major threshold", 1, 6, 6, 10, -1, 12, 1, 0, 0, nil, validTraits, ErrInvalidThresholds},
		{"severe < major", 1, 6, 6, 10, 12, 8, 1, 0, 0, nil, validTraits, ErrInvalidThresholds},
		{"negative proficiency", 1, 6, 6, 10, 8, 12, -1, 0, 0, nil, validTraits, ErrInvalidProficiency},
		{"negative armor score", 1, 6, 6, 10, 8, 12, 1, -1, 0, nil, validTraits, ErrInvalidArmorScore},
		{"armor max too high", 1, 6, 6, 10, 8, 12, 1, 0, 13, nil, validTraits, ErrInvalidArmorMax},
		{"empty experience", 1, 6, 6, 10, 8, 12, 1, 0, 0, []Experience{{Name: "", Modifier: 1}}, validTraits, ErrInvalidExperience},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProfile(tt.level, tt.hpMax, tt.stressMax, tt.evasion, tt.majorThreshold, tt.severeThreshold, tt.proficiency, tt.armorScore, tt.armorMax, tt.traits, tt.experiences)
			if tt.wantErr == nil && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}
