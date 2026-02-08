// Package invite provides participant invite management.
package invite

import (
	"fmt"
	"strings"
	"time"

	apperrors "github.com/louisbranch/fracturing.space/internal/errors"
	"github.com/louisbranch/fracturing.space/internal/id"
)

var (
	// ErrEmptyCampaignID indicates a missing campaign ID.
	ErrEmptyCampaignID = apperrors.New(apperrors.CodeInviteEmptyCampaignID, "campaign id is required")
	// ErrEmptyParticipantID indicates a missing participant ID.
	ErrEmptyParticipantID = apperrors.New(apperrors.CodeInviteEmptyParticipantID, "participant id is required")
)

// Status represents the lifecycle status of an invite.
type Status int

const (
	// StatusUnspecified represents an invalid invite status.
	StatusUnspecified Status = iota
	// StatusPending indicates an invite is available to claim.
	StatusPending
	// StatusClaimed indicates an invite has been claimed.
	StatusClaimed
	// StatusRevoked indicates an invite has been revoked.
	StatusRevoked
)

// Invite represents a seat-targeted invite.
type Invite struct {
	ID                     string
	CampaignID             string
	ParticipantID          string
	Status                 Status
	CreatedByParticipantID string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// CreateInviteInput describes the metadata needed to create an invite.
type CreateInviteInput struct {
	CampaignID             string
	ParticipantID          string
	CreatedByParticipantID string
}

// CreateInvite creates a new invite with a generated ID and timestamps.
func CreateInvite(input CreateInviteInput, now func() time.Time, idGenerator func() (string, error)) (Invite, error) {
	if now == nil {
		now = time.Now
	}
	if idGenerator == nil {
		idGenerator = id.NewID
	}

	normalized, err := NormalizeCreateInviteInput(input)
	if err != nil {
		return Invite{}, err
	}

	inviteID, err := idGenerator()
	if err != nil {
		return Invite{}, fmt.Errorf("generate invite id: %w", err)
	}

	createdAt := now().UTC()
	return Invite{
		ID:                     inviteID,
		CampaignID:             normalized.CampaignID,
		ParticipantID:          normalized.ParticipantID,
		Status:                 StatusPending,
		CreatedByParticipantID: normalized.CreatedByParticipantID,
		CreatedAt:              createdAt,
		UpdatedAt:              createdAt,
	}, nil
}

// NormalizeCreateInviteInput trims and validates invite input metadata.
func NormalizeCreateInviteInput(input CreateInviteInput) (CreateInviteInput, error) {
	input.CampaignID = strings.TrimSpace(input.CampaignID)
	if input.CampaignID == "" {
		return CreateInviteInput{}, ErrEmptyCampaignID
	}
	input.ParticipantID = strings.TrimSpace(input.ParticipantID)
	if input.ParticipantID == "" {
		return CreateInviteInput{}, ErrEmptyParticipantID
	}
	input.CreatedByParticipantID = strings.TrimSpace(input.CreatedByParticipantID)
	return input, nil
}

// StatusLabel returns the string label for an invite status.
func StatusLabel(status Status) string {
	switch status {
	case StatusPending:
		return "PENDING"
	case StatusClaimed:
		return "CLAIMED"
	case StatusRevoked:
		return "REVOKED"
	default:
		return "UNSPECIFIED"
	}
}

// StatusFromLabel converts a status label to a Status value.
func StatusFromLabel(label string) Status {
	switch strings.ToUpper(strings.TrimSpace(label)) {
	case "PENDING":
		return StatusPending
	case "CLAIMED":
		return StatusClaimed
	case "REVOKED":
		return StatusRevoked
	default:
		return StatusUnspecified
	}
}
