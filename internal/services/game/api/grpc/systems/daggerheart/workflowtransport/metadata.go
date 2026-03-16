package workflowtransport

import (
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

// Canonical keys for Daggerheart roll `system_data`.
const (
	KeyCharacterID = "character_id"
	KeyAdversaryID = "adversary_id"
	KeyRollKind    = "roll_kind"
	KeyOutcome     = "outcome"
	KeyHopeFear    = "hope_fear"
	KeyCrit        = "crit"
	KeyCritNegates = "crit_negates"
	KeyRoll        = "roll"
	KeyModifier    = "modifier"
	KeyTotal       = "total"
)

// RollModifierMetadata captures one normalized modifier entry in roll metadata.
type RollModifierMetadata struct {
	Value  int    `json:"value"`
	Source string `json:"source,omitempty"`
}

// RollSystemMetadata captures the typed `system_data` contract for
// roll-resolved payloads used by Daggerheart transport workflows.
type RollSystemMetadata struct {
	CharacterID       string                 `json:"character_id,omitempty"`
	AdversaryID       string                 `json:"adversary_id,omitempty"`
	Trait             string                 `json:"trait,omitempty"`
	RollKind          string                 `json:"roll_kind,omitempty"`
	RollContext       string                 `json:"roll_context,omitempty"`
	Outcome           string                 `json:"outcome,omitempty"`
	Flavor            string                 `json:"flavor,omitempty"`
	BreathCountdownID string                 `json:"breath_countdown_id,omitempty"`
	HopeFear          *bool                  `json:"hope_fear,omitempty"`
	Crit              *bool                  `json:"crit,omitempty"`
	CritNegates       *bool                  `json:"crit_negates,omitempty"`
	GMMove            *bool                  `json:"gm_move,omitempty"`
	Underwater        *bool                  `json:"underwater,omitempty"`
	Roll              *int                   `json:"roll,omitempty"`
	Modifier          *int                   `json:"modifier,omitempty"`
	Total             *int                   `json:"total,omitempty"`
	BaseTotal         *int                   `json:"base_total,omitempty"`
	Critical          *bool                  `json:"critical,omitempty"`
	CriticalBonus     *int                   `json:"critical_bonus,omitempty"`
	Advantage         *int                   `json:"advantage,omitempty"`
	Disadvantage      *int                   `json:"disadvantage,omitempty"`
	Modifiers         []RollModifierMetadata `json:"modifiers,omitempty"`
}

// OutcomeOrFallback returns the stored outcome, or the provided fallback if the
// stored value is blank.
func (m RollSystemMetadata) OutcomeOrFallback(fallback string) string {
	if trimmed := strings.TrimSpace(m.Outcome); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(fallback)
}

// RollKindCode returns the normalized raw roll kind code.
func (m RollSystemMetadata) RollKindCode() string {
	return strings.TrimSpace(m.RollKind)
}

// RollKindOrDefault maps the raw roll kind code to protobuf, defaulting to
// action rolls when the code is blank or unknown.
func (m RollSystemMetadata) RollKindOrDefault() pb.RollKind {
	switch m.RollKindCode() {
	case pb.RollKind_ROLL_KIND_REACTION.String():
		return pb.RollKind_ROLL_KIND_REACTION
	case pb.RollKind_ROLL_KIND_ACTION.String():
		return pb.RollKind_ROLL_KIND_ACTION
	default:
		return pb.RollKind_ROLL_KIND_ACTION
	}
}
