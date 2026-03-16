package daggerheart

import "strings"

const (
	CompanionStatusPresent = "present"
	CompanionStatusAway    = "away"
)

// CharacterCompanionState stores mutable companion-owned runtime state outside
// the static companion sheet profile.
type CharacterCompanionState struct {
	Status             string `json:"status,omitempty"`
	ActiveExperienceID string `json:"active_experience_id,omitempty"`
}

// Normalized keeps companion runtime state replay-safe and trims invalid data.
func (s CharacterCompanionState) Normalized() CharacterCompanionState {
	normalized := s
	normalized.Status = strings.ToLower(strings.TrimSpace(normalized.Status))
	normalized.ActiveExperienceID = strings.TrimSpace(normalized.ActiveExperienceID)
	switch normalized.Status {
	case "", CompanionStatusPresent:
		normalized.Status = CompanionStatusPresent
		normalized.ActiveExperienceID = ""
	case CompanionStatusAway:
		if normalized.ActiveExperienceID == "" {
			normalized.Status = CompanionStatusPresent
		}
	default:
		normalized.Status = CompanionStatusPresent
		normalized.ActiveExperienceID = ""
	}
	return normalized
}

// IsZero reports whether the companion carries no runtime mutation beyond the
// default "present and idle" state.
func (s CharacterCompanionState) IsZero() bool {
	normalized := s.Normalized()
	return normalized.Status == CompanionStatusPresent && normalized.ActiveExperienceID == ""
}

// WithActiveCompanionExperience returns a normalized copy of the companion
// state that is marked away on the selected experience.
func WithActiveCompanionExperience(state CharacterCompanionState, experienceID string) CharacterCompanionState {
	next := state
	next.Status = CompanionStatusAway
	next.ActiveExperienceID = strings.TrimSpace(experienceID)
	return next.Normalized()
}

// WithCompanionPresent returns a normalized copy of the companion state marked
// present and no longer assigned to an active experience.
func WithCompanionPresent(state CharacterCompanionState) CharacterCompanionState {
	next := state
	next.Status = CompanionStatusPresent
	next.ActiveExperienceID = ""
	return next.Normalized()
}

func normalizedCompanionStatePtr(value *CharacterCompanionState) *CharacterCompanionState {
	if value == nil {
		return nil
	}
	normalized := value.Normalized()
	return &normalized
}
