// Package participant provides participant (player/GM) management.
package participant

import (
	"fmt"
	"strings"
	"time"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

// ParticipantRole describes the role of a participant in a campaign.
type ParticipantRole int

const (
	// ParticipantRoleUnspecified represents an invalid participant role value.
	ParticipantRoleUnspecified ParticipantRole = iota
	// ParticipantRoleGM indicates a game master.
	ParticipantRoleGM
	// ParticipantRolePlayer indicates a player.
	ParticipantRolePlayer
)

// Controller describes how a participant is controlled.
type Controller int

const (
	// ControllerUnspecified represents an invalid controller value.
	ControllerUnspecified Controller = iota
	// ControllerHuman indicates a human controller.
	ControllerHuman
	// ControllerAI indicates an AI controller.
	ControllerAI
)

// CampaignAccess describes campaign-level permissions for a participant.
type CampaignAccess int

const (
	// CampaignAccessUnspecified represents an invalid access value.
	CampaignAccessUnspecified CampaignAccess = iota
	// CampaignAccessMember indicates baseline campaign access.
	CampaignAccessMember
	// CampaignAccessManager indicates permissions to manage participants and invites.
	CampaignAccessManager
	// CampaignAccessOwner indicates full campaign ownership permissions.
	CampaignAccessOwner
)

var (
	// ErrEmptyDisplayName indicates a missing participant display name.
	ErrEmptyDisplayName = apperrors.New(apperrors.CodeParticipantEmptyDisplayName, "display name is required")
	// ErrInvalidParticipantRole indicates a missing or invalid participant role.
	ErrInvalidParticipantRole = apperrors.New(apperrors.CodeParticipantInvalidRole, "participant role is required")
	// ErrEmptyCampaignID indicates a missing campaign ID.
	ErrEmptyCampaignID = apperrors.New(apperrors.CodeParticipantEmptyCampaignID, "campaign id is required")
)

// Participant represents a participant (GM or player) in a campaign.
type Participant struct {
	ID          string
	CampaignID  string
	UserID      string
	DisplayName string
	Role        ParticipantRole
	Controller  Controller
	// CampaignAccess indicates campaign-level permissions.
	CampaignAccess CampaignAccess
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CreateParticipantInput describes the metadata needed to create a participant.
type CreateParticipantInput struct {
	CampaignID  string
	UserID      string
	DisplayName string
	Role        ParticipantRole
	Controller  Controller
	// CampaignAccess indicates campaign-level permissions.
	CampaignAccess CampaignAccess
}

// CreateParticipant creates a new participant with a generated ID and timestamps.
func CreateParticipant(input CreateParticipantInput, now func() time.Time, idGenerator func() (string, error)) (Participant, error) {
	if now == nil {
		now = time.Now
	}
	if idGenerator == nil {
		idGenerator = id.NewID
	}

	normalized, err := NormalizeCreateParticipantInput(input)
	if err != nil {
		return Participant{}, err
	}

	participantID, err := idGenerator()
	if err != nil {
		return Participant{}, fmt.Errorf("generate participant id: %w", err)
	}

	createdAt := now().UTC()
	return Participant{
		ID:             participantID,
		CampaignID:     normalized.CampaignID,
		UserID:         normalized.UserID,
		DisplayName:    normalized.DisplayName,
		Role:           normalized.Role,
		Controller:     normalized.Controller,
		CampaignAccess: normalized.CampaignAccess,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}, nil
}

// NormalizeCreateParticipantInput trims and validates participant input metadata.
func NormalizeCreateParticipantInput(input CreateParticipantInput) (CreateParticipantInput, error) {
	input.CampaignID = strings.TrimSpace(input.CampaignID)
	if input.CampaignID == "" {
		return CreateParticipantInput{}, ErrEmptyCampaignID
	}
	input.UserID = strings.TrimSpace(input.UserID)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.DisplayName == "" {
		return CreateParticipantInput{}, ErrEmptyDisplayName
	}
	if input.Role == ParticipantRoleUnspecified {
		return CreateParticipantInput{}, ErrInvalidParticipantRole
	}
	if input.Controller == ControllerUnspecified {
		input.Controller = ControllerHuman
	}
	if input.CampaignAccess == CampaignAccessUnspecified {
		input.CampaignAccess = CampaignAccessMember
	}
	return input, nil
}
