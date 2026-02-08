package storage

import (
	"context"
	"time"

	"github.com/louisbranch/fracturing.space/internal/auth/user"
	"github.com/louisbranch/fracturing.space/internal/campaign"
	"github.com/louisbranch/fracturing.space/internal/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/campaign/invite"
	"github.com/louisbranch/fracturing.space/internal/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/campaign/session"
	apperrors "github.com/louisbranch/fracturing.space/internal/errors"
)

// ErrNotFound indicates a requested record is missing.
var ErrNotFound = apperrors.New(apperrors.CodeNotFound, "record not found")

// ErrActiveSessionExists indicates an active session already exists for a campaign.
var ErrActiveSessionExists = apperrors.New(apperrors.CodeActiveSessionExists, "active session already exists for campaign")

// CampaignStore persists campaign metadata records.
type CampaignStore interface {
	Put(ctx context.Context, c campaign.Campaign) error
	Get(ctx context.Context, id string) (campaign.Campaign, error)
	// List returns a page of campaign records starting after the page token.
	List(ctx context.Context, pageSize int, pageToken string) (CampaignPage, error)
}

// CampaignPage describes a page of campaign records.
type CampaignPage struct {
	Campaigns     []campaign.Campaign
	NextPageToken string
}

// ParticipantStore persists participant records.
type ParticipantStore interface {
	PutParticipant(ctx context.Context, p participant.Participant) error
	GetParticipant(ctx context.Context, campaignID, participantID string) (participant.Participant, error)
	DeleteParticipant(ctx context.Context, campaignID, participantID string) error
	// ListParticipantsByCampaign returns all participants for a campaign.
	ListParticipantsByCampaign(ctx context.Context, campaignID string) ([]participant.Participant, error)
	// ListParticipants returns a page of participant records for a campaign starting after the page token.
	ListParticipants(ctx context.Context, campaignID string, pageSize int, pageToken string) (ParticipantPage, error)
}

// ParticipantPage describes a page of participant records.
type ParticipantPage struct {
	Participants  []participant.Participant
	NextPageToken string
}

// UserStore persists auth user records.
type UserStore interface {
	PutUser(ctx context.Context, u user.User) error
	GetUser(ctx context.Context, userID string) (user.User, error)
	ListUsers(ctx context.Context, pageSize int, pageToken string) (UserPage, error)
}

// UserPage describes a page of user records.
type UserPage struct {
	Users         []user.User
	NextPageToken string
}

// InviteStore persists campaign invite records.
type InviteStore interface {
	PutInvite(ctx context.Context, inv invite.Invite) error
	GetInvite(ctx context.Context, inviteID string) (invite.Invite, error)
	ListInvites(ctx context.Context, campaignID string, pageSize int, pageToken string) (InvitePage, error)
	UpdateInviteStatus(ctx context.Context, inviteID string, status invite.Status, updatedAt time.Time) error
}

// InvitePage describes a page of invites.
type InvitePage struct {
	Invites       []invite.Invite
	NextPageToken string
}

// CharacterStore persists character records.
type CharacterStore interface {
	PutCharacter(ctx context.Context, c character.Character) error
	GetCharacter(ctx context.Context, campaignID, characterID string) (character.Character, error)
	DeleteCharacter(ctx context.Context, campaignID, characterID string) error
	// ListCharacters returns a page of character records for a campaign starting after the page token.
	ListCharacters(ctx context.Context, campaignID string, pageSize int, pageToken string) (CharacterPage, error)
}

// CharacterPage describes a page of character records.
type CharacterPage struct {
	Characters    []character.Character
	NextPageToken string
}

// ControlDefaultStore persists default controller assignments for characters.
type ControlDefaultStore interface {
	// PutControlDefault sets the default controller for a character in a campaign.
	// Overwrites any existing controller for the same (campaign_id, character_id) pair.
	PutControlDefault(ctx context.Context, campaignID, characterID string, controller character.CharacterController) error
	// GetControlDefault retrieves the default controller for a character in a campaign.
	GetControlDefault(ctx context.Context, campaignID, characterID string) (character.CharacterController, error)
}

// SessionStore persists session records.
type SessionStore interface {
	// PutSession atomically stores a session and sets it as the active session for the campaign.
	// Returns ErrActiveSessionExists if an active session already exists for the campaign.
	PutSession(ctx context.Context, s session.Session) error
	// EndSession marks a session as ended and clears it as active for the campaign.
	// The boolean return value reports whether the session transitioned to ENDED.
	EndSession(ctx context.Context, campaignID, sessionID string, endedAt time.Time) (session.Session, bool, error)
	// GetSession retrieves a session by campaign ID and session ID.
	GetSession(ctx context.Context, campaignID, sessionID string) (session.Session, error)
	// GetActiveSession retrieves the active session for a campaign, if one exists.
	// Returns ErrNotFound if no active session exists.
	GetActiveSession(ctx context.Context, campaignID string) (session.Session, error)
	// ListSessions returns a page of session records for a campaign starting after the page token.
	ListSessions(ctx context.Context, campaignID string, pageSize int, pageToken string) (SessionPage, error)
}

// EventStore persists events to the unified event journal.
type EventStore interface {
	// AppendEvent atomically appends an event and returns it with sequence and hash set.
	AppendEvent(ctx context.Context, evt event.Event) (event.Event, error)
	// GetEventByHash retrieves an event by its content hash.
	GetEventByHash(ctx context.Context, hash string) (event.Event, error)
	// GetEventBySeq retrieves a specific event by sequence number.
	GetEventBySeq(ctx context.Context, campaignID string, seq uint64) (event.Event, error)
	// ListEvents returns events ordered by sequence ascending.
	ListEvents(ctx context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error)
	// ListEventsBySession returns events for a specific session.
	ListEventsBySession(ctx context.Context, campaignID, sessionID string, afterSeq uint64, limit int) ([]event.Event, error)
	// GetLatestEventSeq returns the latest event sequence number for a campaign.
	// Returns 0 if no events exist.
	GetLatestEventSeq(ctx context.Context, campaignID string) (uint64, error)
	// ListEventsPage returns a paginated, filtered, and sorted list of events.
	ListEventsPage(ctx context.Context, req ListEventsPageRequest) (ListEventsPageResult, error)
}

// TelemetryEvent describes an operational telemetry record.
type TelemetryEvent struct {
	Timestamp      time.Time
	EventName      string
	Severity       string
	CampaignID     string
	SessionID      string
	ActorType      string
	ActorID        string
	RequestID      string
	InvocationID   string
	TraceID        string
	SpanID         string
	Attributes     map[string]any
	AttributesJSON []byte
}

// TelemetryStore persists operational telemetry events.
type TelemetryStore interface {
	AppendTelemetryEvent(ctx context.Context, evt TelemetryEvent) error
}

// ListEventsPageRequest describes the parameters for paginated event listing.
type ListEventsPageRequest struct {
	// CampaignID scopes the query to a specific campaign (required).
	CampaignID string
	// PageSize is the maximum number of events to return (default: 50, max: 200).
	PageSize int
	// CursorSeq is the sequence number to paginate from (0 for first page).
	CursorSeq uint64
	// CursorDir is the pagination direction ("fwd" = seq > cursor, "bwd" = seq < cursor).
	CursorDir string
	// CursorReverse indicates whether to temporarily reverse the sort order.
	// This is used for "previous page" navigation to fetch items nearest to the cursor.
	CursorReverse bool
	// Descending orders results by seq desc (newest first) when true.
	Descending bool
	// FilterClause is an optional SQL WHERE clause fragment.
	FilterClause string
	// FilterParams are the positional parameters for the filter clause.
	FilterParams []any
}

// ListEventsPageResult contains the paginated event results.
type ListEventsPageResult struct {
	// Events are the events matching the request.
	Events []event.Event
	// HasNextPage indicates whether more results exist in the forward direction.
	HasNextPage bool
	// HasPrevPage indicates whether more results exist in the backward direction.
	HasPrevPage bool
	// TotalCount is the total number of events matching the filter.
	TotalCount int
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
	UpdatedCharacterStates []DaggerheartCharacterState
	AppliedChanges         []session.OutcomeAppliedChange
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
	Sessions      []session.Session
	NextPageToken string
}

// Snapshot represents a precomputed state snapshot for a session as of an event sequence.
type Snapshot struct {
	CampaignID          string
	SessionID           string
	EventSeq            uint64
	CharacterStatesJSON []byte
	GMStateJSON         []byte
	SystemStateJSON     []byte
	CreatedAt           time.Time
}

// SnapshotStore persists session snapshots captured at event sequences.
type SnapshotStore interface {
	// PutSnapshot stores a snapshot.
	PutSnapshot(ctx context.Context, snap Snapshot) error
	// GetSnapshot retrieves a snapshot by campaign and session ID.
	GetSnapshot(ctx context.Context, campaignID, sessionID string) (Snapshot, error)
	// GetLatestSnapshot retrieves the most recent snapshot for a campaign.
	GetLatestSnapshot(ctx context.Context, campaignID string) (Snapshot, error)
	// ListSnapshots returns snapshots ordered by event sequence descending.
	ListSnapshots(ctx context.Context, campaignID string, limit int) ([]Snapshot, error)
}

// ForkMetadata contains fork-related campaign information.
type ForkMetadata struct {
	ParentCampaignID string
	ForkEventSeq     uint64
	OriginCampaignID string
}

// CampaignForkStore provides fork-related campaign operations.
type CampaignForkStore interface {
	// GetCampaignForkMetadata retrieves fork metadata for a campaign.
	GetCampaignForkMetadata(ctx context.Context, campaignID string) (ForkMetadata, error)
	// SetCampaignForkMetadata sets fork metadata for a campaign.
	SetCampaignForkMetadata(ctx context.Context, campaignID string, metadata ForkMetadata) error
}

// Store is a composite interface for all storage concerns.
// Storage backends (BoltDB, SQLite) implement this interface.
type Store interface {
	CampaignStore
	ParticipantStore
	CharacterStore
	ControlDefaultStore
	UserStore
	InviteStore
	DaggerheartStore
	SessionStore
	EventStore
	TelemetryStore
	RollOutcomeStore
	SnapshotStore
	CampaignForkStore
	Close() error
}

// DaggerheartCharacterProfile contains Daggerheart-specific character profile data.
type DaggerheartCharacterProfile struct {
	CampaignID      string
	CharacterID     string
	HpMax           int
	StressMax       int
	Evasion         int
	MajorThreshold  int
	SevereThreshold int
	// Daggerheart traits
	Agility   int
	Strength  int
	Finesse   int
	Instinct  int
	Presence  int
	Knowledge int
}

// DaggerheartCharacterState contains Daggerheart-specific character state data.
type DaggerheartCharacterState struct {
	CampaignID  string
	CharacterID string
	Hp          int
	Hope        int
	Stress      int
}

// DaggerheartSnapshot contains Daggerheart-specific campaign-level state.
type DaggerheartSnapshot struct {
	CampaignID string
	GMFear     int
}

// DaggerheartStore provides Daggerheart-specific storage operations.
// This interface is used for the Daggerheart game system extension tables.
type DaggerheartStore interface {
	// Character Profile Extensions
	PutDaggerheartCharacterProfile(ctx context.Context, profile DaggerheartCharacterProfile) error
	GetDaggerheartCharacterProfile(ctx context.Context, campaignID, characterID string) (DaggerheartCharacterProfile, error)

	// Character State Extensions
	PutDaggerheartCharacterState(ctx context.Context, state DaggerheartCharacterState) error
	GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (DaggerheartCharacterState, error)

	// Snapshot Extensions
	PutDaggerheartSnapshot(ctx context.Context, snap DaggerheartSnapshot) error
	GetDaggerheartSnapshot(ctx context.Context, campaignID string) (DaggerheartSnapshot, error)
}
