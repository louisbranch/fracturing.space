package character

import (
	"fmt"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/errors"
)

var (
	// ErrInvalidCharacterController indicates that the character controller is invalid (none or both fields set).
	ErrInvalidCharacterController = apperrors.New(apperrors.CodeCharacterInvalidController, "character controller must be exactly one of: gm or participant")
	// ErrEmptyParticipantID indicates a missing participant ID when participant controller is specified.
	ErrEmptyParticipantID = apperrors.New(apperrors.CodeCharacterEmptyParticipantID, "participant id is required when participant controller is specified")
)

// CharacterController represents the default controller for a character.
// Exactly one of IsGM or ParticipantID must be set.
type CharacterController struct {
	// IsGM is true if the controller is the GM.
	IsGM bool
	// ParticipantID is the participant ID if the controller is a participant.
	// Must be empty if IsGM is true.
	ParticipantID string
}

// Validate ensures that exactly one controller type is set.
func (c CharacterController) Validate() error {
	hasGM := c.IsGM
	hasParticipant := strings.TrimSpace(c.ParticipantID) != ""

	if hasGM && hasParticipant {
		return ErrInvalidCharacterController
	}
	if !hasGM && !hasParticipant {
		return ErrInvalidCharacterController
	}
	return nil
}

// NewGmController creates a CharacterController for GM control.
func NewGmController() CharacterController {
	return CharacterController{
		IsGM: true,
	}
}

// NewParticipantController creates a CharacterController for participant control.
func NewParticipantController(participantID string) (CharacterController, error) {
	participantID = strings.TrimSpace(participantID)
	if participantID == "" {
		return CharacterController{}, ErrEmptyParticipantID
	}
	return CharacterController{
		ParticipantID: participantID,
	}, nil
}

// MustNewParticipantController creates a CharacterController for participant control, panicking on error.
// Use only when participantID is guaranteed to be non-empty.
func MustNewParticipantController(participantID string) CharacterController {
	ctrl, err := NewParticipantController(participantID)
	if err != nil {
		panic(fmt.Sprintf("must new participant controller: %v", err))
	}
	return ctrl
}
