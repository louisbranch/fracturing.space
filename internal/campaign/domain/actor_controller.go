package domain

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrInvalidActorController indicates that the actor controller is invalid (none or both fields set).
	ErrInvalidActorController = errors.New("actor controller must be exactly one of: gm or participant")
	// ErrEmptyParticipantID indicates a missing participant ID when participant controller is specified.
	ErrEmptyParticipantID = errors.New("participant id is required when participant controller is specified")
)

// ActorController represents the default controller for an actor.
// Exactly one of IsGM or ParticipantID must be set.
type ActorController struct {
	// IsGM is true if the controller is the GM.
	IsGM bool
	// ParticipantID is the participant ID if the controller is a participant.
	// Must be empty if IsGM is true.
	ParticipantID string
}

// Validate ensures that exactly one controller type is set.
func (c ActorController) Validate() error {
	hasGM := c.IsGM
	hasParticipant := strings.TrimSpace(c.ParticipantID) != ""

	if hasGM && hasParticipant {
		return ErrInvalidActorController
	}
	if !hasGM && !hasParticipant {
		return ErrInvalidActorController
	}
	return nil
}

// NewGmController creates an ActorController for GM control.
func NewGmController() ActorController {
	return ActorController{
		IsGM: true,
	}
}

// NewParticipantController creates an ActorController for participant control.
func NewParticipantController(participantID string) (ActorController, error) {
	participantID = strings.TrimSpace(participantID)
	if participantID == "" {
		return ActorController{}, ErrEmptyParticipantID
	}
	return ActorController{
		ParticipantID: participantID,
	}, nil
}

// MustNewParticipantController creates an ActorController for participant control, panicking on error.
// Use only when participantID is guaranteed to be non-empty.
func MustNewParticipantController(participantID string) ActorController {
	ctrl, err := NewParticipantController(participantID)
	if err != nil {
		panic(fmt.Sprintf("must new participant controller: %v", err))
	}
	return ctrl
}
