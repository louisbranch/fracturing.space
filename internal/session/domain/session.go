package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/id"
)

// SessionStatus describes the lifecycle state of a session.
type SessionStatus int

const (
	// SessionStatusUnspecified represents an invalid session status value.
	SessionStatusUnspecified SessionStatus = iota
	// SessionStatusActive indicates the session is currently active.
	SessionStatusActive
	// SessionStatusEnded indicates the session has ended.
	SessionStatusEnded
)

var (
	// ErrEmptyCampaignID indicates a missing campaign ID.
	ErrEmptyCampaignID = errors.New("campaign id is required")
)

// Session represents a game session within a campaign.
type Session struct {
	ID         string
	CampaignID string
	Name       string
	Status     SessionStatus
	StartedAt  time.Time
	UpdatedAt  time.Time
	EndedAt    *time.Time // nil when session is not ended
}

// CreateSessionInput describes the metadata needed to create a session.
type CreateSessionInput struct {
	CampaignID string
	Name       string
}

// CreateSession creates a new session with a generated ID and timestamps.
// The session is created with ACTIVE status.
func CreateSession(input CreateSessionInput, now func() time.Time, idGenerator func() (string, error)) (Session, error) {
	if now == nil {
		now = time.Now
	}
	if idGenerator == nil {
		idGenerator = id.NewID
	}

	normalized, err := NormalizeCreateSessionInput(input)
	if err != nil {
		return Session{}, err
	}

	sessionID, err := idGenerator()
	if err != nil {
		return Session{}, fmt.Errorf("generate session id: %w", err)
	}

	createdAt := now().UTC()
	return Session{
		ID:         sessionID,
		CampaignID: normalized.CampaignID,
		Name:       normalized.Name,
		Status:     SessionStatusActive,
		StartedAt:  createdAt,
		UpdatedAt:  createdAt,
		EndedAt:    nil,
	}, nil
}

// NormalizeCreateSessionInput trims and validates session input metadata.
func NormalizeCreateSessionInput(input CreateSessionInput) (CreateSessionInput, error) {
	input.CampaignID = strings.TrimSpace(input.CampaignID)
	if input.CampaignID == "" {
		return CreateSessionInput{}, ErrEmptyCampaignID
	}
	input.Name = strings.TrimSpace(input.Name)
	// Name is optional, so empty string is allowed
	return input, nil
}
