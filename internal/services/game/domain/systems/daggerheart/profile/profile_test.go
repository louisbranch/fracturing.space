package profile

import (
	"errors"
	"testing"
)

func TestGetDefaults(t *testing.T) {
	pc := GetDefaults("PC")
	if pc.Level != PCLevelDefault {
		t.Fatalf("pc level = %d, want %d", pc.Level, PCLevelDefault)
	}
	if pc.HpMax != PCHpMax || pc.StressMax != PCStressMax || pc.Evasion != PCEvasion {
		t.Fatalf("unexpected pc defaults: %+v", pc)
	}
	if pc.MajorThreshold != PCLevelDefault || pc.SevereThreshold != PCLevelDefault*2 {
		t.Fatalf("unexpected pc unarmored thresholds: %d/%d", pc.MajorThreshold, pc.SevereThreshold)
	}

	npc := GetDefaults("NPC")
	if npc.Level != NPCLevelDefault {
		t.Fatalf("npc level = %d, want %d", npc.Level, NPCLevelDefault)
	}
	if npc.HpMax != NPCHpMax || npc.StressMax != NPCStressMax || npc.Evasion != NPCEvasion {
		t.Fatalf("unexpected npc defaults: %+v", npc)
	}
	if npc.MajorThreshold != NPCLevelDefault || npc.SevereThreshold != NPCLevelDefault*2 {
		t.Fatalf("unexpected npc unarmored thresholds: %d/%d", npc.MajorThreshold, npc.SevereThreshold)
	}
}

func TestDeriveThresholds(t *testing.T) {
	major, severe := DeriveThresholds(3, 0, 8, 12)
	if major != 3 || severe != 6 {
		t.Fatalf("DeriveThresholds(unarmored) = %d/%d, want 3/6", major, severe)
	}

	major, severe = DeriveThresholds(3, 2, 8, 12)
	if major != 8 || severe != 12 {
		t.Fatalf("DeriveThresholds(armored) = %d/%d, want 8/12", major, severe)
	}
}

func TestValidateLevel(t *testing.T) {
	if err := ValidateLevel(0); !errors.Is(err, ErrInvalidLevel) {
		t.Fatalf("expected ErrInvalidLevel for 0, got %v", err)
	}
	if err := ValidateLevel(11); !errors.Is(err, ErrInvalidLevel) {
		t.Fatalf("expected ErrInvalidLevel for 11, got %v", err)
	}
	if err := ValidateLevel(1); err != nil {
		t.Fatalf("expected valid level, got %v", err)
	}
}

func TestValidateTraitAndTraits(t *testing.T) {
	if err := ValidateTrait("agility", TraitMin-1); !errors.Is(err, ErrInvalidTraitValue) {
		t.Fatalf("expected ErrInvalidTraitValue, got %v", err)
	}
	if err := ValidateTrait("agility", TraitMax+1); !errors.Is(err, ErrInvalidTraitValue) {
		t.Fatalf("expected ErrInvalidTraitValue, got %v", err)
	}
	if err := ValidateTraits(Traits{
		Agility:   TraitMin,
		Strength:  TraitMax,
		Finesse:   0,
		Instinct:  1,
		Presence:  -1,
		Knowledge: 2,
	}); err != nil {
		t.Fatalf("expected valid traits, got %v", err)
	}
}

func TestValidateProfile(t *testing.T) {
	validTraits := Traits{
		Agility:   1,
		Strength:  1,
		Finesse:   0,
		Instinct:  0,
		Presence:  -1,
		Knowledge: 2,
	}
	validExperiences := []Experience{{Name: "Tactics", Modifier: 2}}

	if err := Validate(
		1,
		PCHpMax,
		PCStressMax,
		PCEvasion,
		PCMajorThreshold,
		PCSevereThreshold,
		PCProficiency,
		PCArmorScore,
		PCArmorMax,
		validTraits,
		validExperiences,
	); err != nil {
		t.Fatalf("expected valid profile, got %v", err)
	}

	tests := []struct {
		name string
		err  error
		call func() error
	}{
		{
			name: "invalid hp",
			err:  ErrInvalidHpMax,
			call: func() error {
				return Validate(1, 0, PCStressMax, PCEvasion, 1, 2, 1, 0, 0, validTraits, validExperiences)
			},
		},
		{
			name: "invalid stress",
			err:  ErrInvalidStressMax,
			call: func() error {
				return Validate(1, PCHpMax, StressMaxCap+1, PCEvasion, 1, 2, 1, 0, 0, validTraits, validExperiences)
			},
		},
		{
			name: "invalid evasion",
			err:  ErrInvalidEvasion,
			call: func() error {
				return Validate(1, PCHpMax, PCStressMax, -1, 1, 2, 1, 0, 0, validTraits, validExperiences)
			},
		},
		{
			name: "invalid thresholds",
			err:  ErrInvalidThresholds,
			call: func() error {
				return Validate(1, PCHpMax, PCStressMax, PCEvasion, 3, 2, 1, 0, 0, validTraits, validExperiences)
			},
		},
		{
			name: "invalid proficiency",
			err:  ErrInvalidProficiency,
			call: func() error {
				return Validate(1, PCHpMax, PCStressMax, PCEvasion, 1, 2, -1, 0, 0, validTraits, validExperiences)
			},
		},
		{
			name: "invalid armor score",
			err:  ErrInvalidArmorScore,
			call: func() error {
				return Validate(1, PCHpMax, PCStressMax, PCEvasion, 1, 2, 1, -1, 0, validTraits, validExperiences)
			},
		},
		{
			name: "invalid armor max",
			err:  ErrInvalidArmorMax,
			call: func() error {
				return Validate(1, PCHpMax, PCStressMax, PCEvasion, 1, 2, 1, 0, ArmorMaxCap+1, validTraits, validExperiences)
			},
		},
		{
			name: "empty experience name",
			err:  ErrInvalidExperience,
			call: func() error {
				return Validate(1, PCHpMax, PCStressMax, PCEvasion, 1, 2, 1, 0, 0, validTraits, []Experience{{Name: "", Modifier: 2}})
			},
		},
		{
			name: "invalid trait in profile",
			err:  ErrInvalidTraitValue,
			call: func() error {
				invalidTraits := validTraits
				invalidTraits.Agility = TraitMax + 1
				return Validate(1, PCHpMax, PCStressMax, PCEvasion, 1, 2, 1, 0, 0, invalidTraits, validExperiences)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); !errors.Is(err, tc.err) {
				t.Fatalf("expected %v, got %v", tc.err, err)
			}
		})
	}
}
