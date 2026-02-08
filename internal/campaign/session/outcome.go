package session

import apperrors "github.com/louisbranch/fracturing.space/internal/errors"

var (
	// ErrOutcomeAlreadyApplied indicates the outcome was already applied for a roll.
	ErrOutcomeAlreadyApplied = apperrors.New(apperrors.CodeOutcomeAlreadyApplied, "outcome already applied")
	// ErrOutcomeCharacterNotFound indicates a character state is missing.
	ErrOutcomeCharacterNotFound = apperrors.New(apperrors.CodeOutcomeCharacterNotFound, "character state not found")
	// ErrOutcomeGMFearInvalid indicates a GM fear mutation is invalid.
	ErrOutcomeGMFearInvalid = apperrors.New(apperrors.CodeOutcomeGMFearInvalid, "gm fear update invalid")
)

// OutcomeField identifies a field mutated by an applied outcome.
type OutcomeField string

const (
	// OutcomeFieldHope indicates a hope mutation.
	OutcomeFieldHope OutcomeField = "hope"
	// OutcomeFieldStress indicates a stress mutation.
	OutcomeFieldStress OutcomeField = "stress"
	// OutcomeFieldGMFear indicates a GM fear mutation.
	OutcomeFieldGMFear OutcomeField = "gm_fear"
)

// OutcomeAppliedChange captures a single applied field change.
type OutcomeAppliedChange struct {
	CharacterID string       `json:"character_id,omitempty"`
	Field       OutcomeField `json:"field"`
	Before      int          `json:"before"`
	After       int          `json:"after"`
}

// OutcomeAppliedPayload captures the event payload for applied outcomes.
type OutcomeAppliedPayload struct {
	RollSeq              uint64                 `json:"roll_seq"`
	Targets              []string               `json:"targets"`
	RequiresComplication bool                   `json:"requires_complication"`
	AppliedChanges       []OutcomeAppliedChange `json:"applied_changes,omitempty"`
}

// OutcomeRejectedPayload captures the event payload for rejected outcomes.
type OutcomeRejectedPayload struct {
	RollSeq    uint64 `json:"roll_seq"`
	ReasonCode string `json:"reason_code"`
	Message    string `json:"message,omitempty"`
}
