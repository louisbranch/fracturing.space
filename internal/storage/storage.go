package storage

import (
	"context"
	"errors"
	"time"

	"github.com/louisbranch/fracturing.space/internal/campaign/domain"
	sessiondomain "github.com/louisbranch/fracturing.space/internal/session/domain"
)

// ErrNotFound indicates a requested record is missing.
var ErrNotFound = errors.New("record not found")

// ErrActiveSessionExists indicates an active session already exists for a campaign.
var ErrActiveSessionExists = errors.New("active session already exists for campaign")

// CampaignStore persists campaign metadata records.
type CampaignStore interface {
	Put(ctx context.Context, campaign domain.Campaign) error
	Get(ctx context.Context, id string) (domain.Campaign, error)
	// List returns a page of campaign records starting after the page token.
	List(ctx context.Context, pageSize int, pageToken string) (CampaignPage, error)
}

// CampaignPage describes a page of campaign records.
type CampaignPage struct {
	Campaigns     []domain.Campaign
	NextPageToken string
}

// ParticipantStore persists participant records.
type ParticipantStore interface {
	PutParticipant(ctx context.Context, participant domain.Participant) error
	GetParticipant(ctx context.Context, campaignID, participantID string) (domain.Participant, error)
	// ListParticipantsByCampaign returns all participants for a campaign.
	ListParticipantsByCampaign(ctx context.Context, campaignID string) ([]domain.Participant, error)
	// ListParticipants returns a page of participant records for a campaign starting after the page token.
	ListParticipants(ctx context.Context, campaignID string, pageSize int, pageToken string) (ParticipantPage, error)
}

// ParticipantPage describes a page of participant records.
type ParticipantPage struct {
	Participants  []domain.Participant
	NextPageToken string
}

// CharacterStore persists character records.
type CharacterStore interface {
	PutCharacter(ctx context.Context, character domain.Character) error
	GetCharacter(ctx context.Context, campaignID, characterID string) (domain.Character, error)
	// ListCharacters returns a page of character records for a campaign starting after the page token.
	ListCharacters(ctx context.Context, campaignID string, pageSize int, pageToken string) (CharacterPage, error)
}

// CharacterPage describes a page of character records.
type CharacterPage struct {
	Characters    []domain.Character
	NextPageToken string
}

// CharacterProfileStore persists character profile records.
type CharacterProfileStore interface {
	PutCharacterProfile(ctx context.Context, profile domain.CharacterProfile) error
	GetCharacterProfile(ctx context.Context, campaignID, characterID string) (domain.CharacterProfile, error)
}

// CharacterStateStore persists character state records.
type CharacterStateStore interface {
	PutCharacterState(ctx context.Context, state domain.CharacterState) error
	GetCharacterState(ctx context.Context, campaignID, characterID string) (domain.CharacterState, error)
}

// ControlDefaultStore persists default controller assignments for characters.
type ControlDefaultStore interface {
	// PutControlDefault sets the default controller for a character in a campaign.
	// Overwrites any existing controller for the same (campaign_id, character_id) pair.
	PutControlDefault(ctx context.Context, campaignID, characterID string, controller domain.CharacterController) error
	// GetControlDefault retrieves the default controller for a character in a campaign.
	GetControlDefault(ctx context.Context, campaignID, characterID string) (domain.CharacterController, error)
}

// SessionStore persists session records.
type SessionStore interface {
	// PutSession atomically stores a session and sets it as the active session for the campaign.
	// Returns ErrActiveSessionExists if an active session already exists for the campaign.
	PutSession(ctx context.Context, session sessiondomain.Session) error
	// EndSession marks a session as ended and clears it as active for the campaign.
	// The boolean return value reports whether the session transitioned to ENDED.
	EndSession(ctx context.Context, campaignID, sessionID string, endedAt time.Time) (sessiondomain.Session, bool, error)
	// GetSession retrieves a session by campaign ID and session ID.
	GetSession(ctx context.Context, campaignID, sessionID string) (sessiondomain.Session, error)
	// GetActiveSession retrieves the active session for a campaign, if one exists.
	// Returns ErrNotFound if no active session exists.
	GetActiveSession(ctx context.Context, campaignID string) (sessiondomain.Session, error)
	// ListSessions returns a page of session records for a campaign starting after the page token.
	ListSessions(ctx context.Context, campaignID string, pageSize int, pageToken string) (SessionPage, error)
}

// SessionEventStore persists session event records.
type SessionEventStore interface {
	// AppendSessionEvent atomically appends a session event and returns the stored event with sequence set.
	AppendSessionEvent(ctx context.Context, event sessiondomain.SessionEvent) (sessiondomain.SessionEvent, error)
	// ListSessionEvents returns session events ordered by sequence ascending.
	ListSessionEvents(ctx context.Context, sessionID string, afterSeq uint64, limit int) ([]sessiondomain.SessionEvent, error)
}

// RollOutcomeDelta describes a per-character state change.
type RollOutcomeDelta struct {
	CharacterID string
	HopeDelta   int
	StressDelta int
}

// RollOutcomeApplyInput describes the outcome application request for storage.
type RollOutcomeApplyInput struct {
	CampaignID           string
	SessionID            string
	RollSeq              uint64
	Targets              []string
	RequiresComplication bool
	RequestID            string
	InvocationID         string
	ParticipantID        string
	CharacterID          string
	EventTimestamp       time.Time
	CharacterDeltas      []RollOutcomeDelta
	GMFearDelta          int
}

// RollOutcomeApplyResult describes the outcome application result from storage.
type RollOutcomeApplyResult struct {
	UpdatedCharacterStates []domain.CharacterState
	AppliedChanges         []sessiondomain.OutcomeAppliedChange
	GMFearChanged          bool
	GMFearBefore           int
	GMFearAfter            int
}

// RollOutcomeStore applies roll outcomes atomically.
type RollOutcomeStore interface {
	ApplyRollOutcome(ctx context.Context, input RollOutcomeApplyInput) (RollOutcomeApplyResult, error)
}

// SessionPage describes a page of session records.
type SessionPage struct {
	Sessions      []sessiondomain.Session
	NextPageToken string
}
