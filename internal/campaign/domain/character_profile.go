package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var (
	// ErrInvalidProfileHpMax indicates hp_max is less than 1.
	ErrInvalidProfileHpMax = errors.New("hp_max must be at least 1")
	// ErrInvalidProfileStressMax indicates stress_max is negative.
	ErrInvalidProfileStressMax = errors.New("stress_max must be non-negative")
	// ErrInvalidProfileThresholds indicates threshold ordering is invalid.
	ErrInvalidProfileThresholds = errors.New("severe_threshold must be >= major_threshold >= 0")
	// ErrInvalidTraitValue indicates a trait value is outside the valid range.
	ErrInvalidTraitValue = errors.New("trait values must be in range -2..+4")
)

// CharacterProfile represents the static profile values for a character.
type CharacterProfile struct {
	CampaignID      string
	CharacterID     string
	Traits          map[string]int
	HpMax           int
	StressMax       int
	Evasion         int
	MajorThreshold  int
	SevereThreshold int
}

// CharacterProfileDefaults represents the default profile values loaded from JSON.
type CharacterProfileDefaults struct {
	Traits          map[string]int `json:"traits"`
	HpMax           int             `json:"hp_max"`
	StressMax       int             `json:"stress_max"`
	Evasion         int             `json:"evasion"`
	MajorThreshold  int             `json:"major_threshold"`
	SevereThreshold int             `json:"severe_threshold"`
}

// CharacterDefaultsFile represents the structure of the defaults JSON file.
type CharacterDefaultsFile struct {
	PC  CharacterProfileDefaults `json:"PC"`
	NPC CharacterProfileDefaults `json:"NPC"`
}

// CreateCharacterProfileInput describes the input for creating a character profile.
type CreateCharacterProfileInput struct {
	CampaignID      string
	CharacterID     string
	Traits          map[string]int
	HpMax           int
	StressMax       int
	Evasion         int
	MajorThreshold  int
	SevereThreshold int
}

// CreateCharacterProfile creates a new character profile with validation.
func CreateCharacterProfile(input CreateCharacterProfileInput) (CharacterProfile, error) {
	if err := ValidateCharacterProfile(input.Traits, input.HpMax, input.StressMax, input.MajorThreshold, input.SevereThreshold); err != nil {
		return CharacterProfile{}, err
	}

	return CharacterProfile{
		CampaignID:      input.CampaignID,
		CharacterID:     input.CharacterID,
		Traits:          input.Traits,
		HpMax:           input.HpMax,
		StressMax:       input.StressMax,
		Evasion:         input.Evasion,
		MajorThreshold:  input.MajorThreshold,
		SevereThreshold: input.SevereThreshold,
	}, nil
}

// ValidateCharacterProfile validates profile invariants.
func ValidateCharacterProfile(traits map[string]int, hpMax, stressMax, majorThreshold, severeThreshold int) error {
	if hpMax < 1 {
		return ErrInvalidProfileHpMax
	}
	if stressMax < 0 {
		return ErrInvalidProfileStressMax
	}
	if majorThreshold < 0 || severeThreshold < majorThreshold {
		return ErrInvalidProfileThresholds
	}
	for traitName, traitValue := range traits {
		if traitValue < -2 || traitValue > 4 {
			return fmt.Errorf("%w: trait %q has value %d", ErrInvalidTraitValue, traitName, traitValue)
		}
	}
	return nil
}

// PatchCharacterProfileInput describes optional fields for patching a profile.
type PatchCharacterProfileInput struct {
	Traits          map[string]int
	HpMax           *int
	StressMax       *int
	Evasion         *int
	MajorThreshold  *int
	SevereThreshold *int
}

// PatchCharacterProfile applies a patch to an existing profile, returning a new profile.
func PatchCharacterProfile(existing CharacterProfile, patch PatchCharacterProfileInput) (CharacterProfile, error) {
	result := existing

	if patch.Traits != nil {
		result.Traits = patch.Traits
	}
	if patch.HpMax != nil {
		result.HpMax = *patch.HpMax
	}
	if patch.StressMax != nil {
		result.StressMax = *patch.StressMax
	}
	if patch.Evasion != nil {
		result.Evasion = *patch.Evasion
	}
	if patch.MajorThreshold != nil {
		result.MajorThreshold = *patch.MajorThreshold
	}
	if patch.SevereThreshold != nil {
		result.SevereThreshold = *patch.SevereThreshold
	}

	if err := ValidateCharacterProfile(result.Traits, result.HpMax, result.StressMax, result.MajorThreshold, result.SevereThreshold); err != nil {
		return CharacterProfile{}, err
	}

	return result, nil
}

// findRepoRoot finds the repository root by walking up to go.mod.
func findRepoRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		return "", fmt.Errorf("failed to resolve runtime caller")
	}

	dir := filepath.Dir(filename)
	for {
		candidate := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("go.mod not found from %s", filename)
}

// LoadCharacterDefaults loads default profile values from the JSON file.
func LoadCharacterDefaults(path string) (map[CharacterKind]CharacterProfileDefaults, error) {
	if path == "" {
		repoRoot, err := findRepoRoot()
		if err != nil {
			return nil, fmt.Errorf("find repo root: %w", err)
		}
		path = filepath.Join(repoRoot, "data", "character_defaults.json")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read defaults file: %w", err)
	}

	var defaultsFile CharacterDefaultsFile
	if err := json.Unmarshal(data, &defaultsFile); err != nil {
		return nil, fmt.Errorf("unmarshal defaults: %w", err)
	}

	result := make(map[CharacterKind]CharacterProfileDefaults)
	result[CharacterKindPC] = defaultsFile.PC
	result[CharacterKindNPC] = defaultsFile.NPC

	return result, nil
}

// GetDefaultProfile returns the default profile for a character kind.
func GetDefaultProfile(kind CharacterKind, defaults map[CharacterKind]CharacterProfileDefaults) (CharacterProfileDefaults, error) {
	defaultProfile, ok := defaults[kind]
	if !ok {
		return CharacterProfileDefaults{}, fmt.Errorf("no defaults found for character kind %v", kind)
	}
	return defaultProfile, nil
}
