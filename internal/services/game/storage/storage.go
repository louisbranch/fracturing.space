package storage

import (
	"context"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

// ErrNotFound indicates a requested persistence record is missing.
// Callers use this to differentiate between legitimate "no such entity" states
// and transport or data corruption failures.
var ErrNotFound = apperrors.New(apperrors.CodeNotFound, "record not found")

// ErrActiveSessionExists indicates a command tried to start a second active session
// for the same campaign, which would violate the single-active-session domain rule.
var ErrActiveSessionExists = apperrors.New(apperrors.CodeActiveSessionExists, "active session already exists for campaign")

// CampaignRecord captures the projection-oriented campaign metadata that APIs read.
type CampaignRecord struct {
	ID               string
	Name             string
	Locale           commonv1.Locale
	System           commonv1.GameSystem
	Status           campaign.Status
	GmMode           campaign.GmMode
	Intent           campaign.Intent
	AccessPolicy     campaign.AccessPolicy
	ParticipantCount int
	CharacterCount   int
	ThemePrompt      string
	CoverAssetID     string
	CoverSetID       string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CompletedAt      *time.Time
	ArchivedAt       *time.Time
}

// ParticipantRecord captures participation state used by campaign membership queries.
type ParticipantRecord struct {
	ID             string
	CampaignID     string
	UserID         string
	Name           string
	Role           participant.Role
	Controller     participant.Controller
	CampaignAccess participant.CampaignAccess
	AvatarSetID    string
	AvatarAssetID  string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// InviteRecord captures invite state used for invitation lifecycle and UX decisions.
type InviteRecord struct {
	ID                     string
	CampaignID             string
	ParticipantID          string
	RecipientUserID        string
	Status                 invite.Status
	CreatedByParticipantID string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// CharacterRecord captures character identity/state metadata for campaign read views.
type CharacterRecord struct {
	ID            string
	CampaignID    string
	ParticipantID string
	Name          string
	Kind          character.Kind
	Notes         string
	AvatarSetID   string
	AvatarAssetID string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// SessionRecord captures session lifecycle metadata that defines active session boundaries.
type SessionRecord struct {
	ID         string
	CampaignID string
	Name       string
	Status     session.Status
	StartedAt  time.Time
	UpdatedAt  time.Time
	EndedAt    *time.Time
}

// CampaignStore owns the campaign-level projection used by list/detail screens and
// status transitions.
type CampaignStore interface {
	Put(ctx context.Context, c CampaignRecord) error
	Get(ctx context.Context, id string) (CampaignRecord, error)
	// List returns a page of campaign records starting after the page token.
	List(ctx context.Context, pageSize int, pageToken string) (CampaignPage, error)
}

// CampaignPage describes a page of campaign records.
type CampaignPage struct {
	Campaigns     []CampaignRecord
	NextPageToken string
}

// ParticipantStore owns membership read state, including seat ownership and ordering.
type ParticipantStore interface {
	PutParticipant(ctx context.Context, p ParticipantRecord) error
	GetParticipant(ctx context.Context, campaignID, participantID string) (ParticipantRecord, error)
	DeleteParticipant(ctx context.Context, campaignID, participantID string) error
	// ListParticipantsByCampaign returns all participants for a campaign.
	ListParticipantsByCampaign(ctx context.Context, campaignID string) ([]ParticipantRecord, error)
	// ListCampaignIDsByUser returns campaign IDs for a participant user.
	ListCampaignIDsByUser(ctx context.Context, userID string) ([]string, error)
	// ListCampaignIDsByParticipant returns campaign IDs for a participant id.
	ListCampaignIDsByParticipant(ctx context.Context, participantID string) ([]string, error)
	// ListParticipants returns a page of participant records for a campaign starting after the page token.
	ListParticipants(ctx context.Context, campaignID string, pageSize int, pageToken string) (ParticipantPage, error)
}

// ParticipantPage describes a page of participant records.
type ParticipantPage struct {
	Participants  []ParticipantRecord
	NextPageToken string
}

// InviteStore owns invite lifecycle read data (created/claimed/revoked flows).
type InviteStore interface {
	PutInvite(ctx context.Context, inv InviteRecord) error
	GetInvite(ctx context.Context, inviteID string) (InviteRecord, error)
	ListInvites(ctx context.Context, campaignID string, recipientUserID string, status invite.Status, pageSize int, pageToken string) (InvitePage, error)
	ListPendingInvites(ctx context.Context, campaignID string, pageSize int, pageToken string) (InvitePage, error)
	ListPendingInvitesForRecipient(ctx context.Context, userID string, pageSize int, pageToken string) (InvitePage, error)
	UpdateInviteStatus(ctx context.Context, inviteID string, status invite.Status, updatedAt time.Time) error
}

// InvitePage describes a page of invites.
type InvitePage struct {
	Invites       []InviteRecord
	NextPageToken string
}

// CharacterStore owns character listing and identity metadata for campaign views.
type CharacterStore interface {
	PutCharacter(ctx context.Context, c CharacterRecord) error
	GetCharacter(ctx context.Context, campaignID, characterID string) (CharacterRecord, error)
	DeleteCharacter(ctx context.Context, campaignID, characterID string) error
	// ListCharacters returns a page of character records for a campaign starting after the page token.
	ListCharacters(ctx context.Context, campaignID string, pageSize int, pageToken string) (CharacterPage, error)
}

// CharacterPage describes a page of character records.
type CharacterPage struct {
	Characters    []CharacterRecord
	NextPageToken string
}

// SessionStore owns active/completed session state used by replay, API, and CLI flows.
type SessionStore interface {
	// PutSession atomically stores a session and sets it as the active session for the campaign.
	// Returns ErrActiveSessionExists if an active session already exists for the campaign.
	PutSession(ctx context.Context, s SessionRecord) error
	// EndSession marks a session as ended and clears it as active for the campaign.
	// The boolean return value reports whether the session transitioned to ENDED.
	EndSession(ctx context.Context, campaignID, sessionID string, endedAt time.Time) (SessionRecord, bool, error)
	// GetSession retrieves a session by campaign ID and session ID.
	GetSession(ctx context.Context, campaignID, sessionID string) (SessionRecord, error)
	// GetActiveSession retrieves the active session for a campaign, if one exists.
	// Returns ErrNotFound if no active session exists.
	GetActiveSession(ctx context.Context, campaignID string) (SessionRecord, error)
	// ListSessions returns a page of session records for a campaign starting after the page token.
	ListSessions(ctx context.Context, campaignID string, pageSize int, pageToken string) (SessionPage, error)
}

// EventStore owns the event stream boundary that drives replay and command
// rehydration; this is the source of truth for state reconstruction.
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

// TelemetryEvent captures operational observations emitted during command execution.
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

// TelemetryStore persists operational telemetry records for audits and incident analysis.
type TelemetryStore interface {
	AppendTelemetryEvent(ctx context.Context, evt TelemetryEvent) error
}

// GameStatistics contains aggregate counters used by dashboards and housekeeping.
type GameStatistics struct {
	CampaignCount    int64
	SessionCount     int64
	CharacterCount   int64
	ParticipantCount int64
}

// StatisticsStore centralizes aggregate count queries for operational observability.
type StatisticsStore interface {
	// GetGameStatistics returns aggregate counts.
	// When since is nil, counts are for all time.
	GetGameStatistics(ctx context.Context, since *time.Time) (GameStatistics, error)
}

// ListEventsPageRequest describes request filters for operator and UI event history views.
type ListEventsPageRequest struct {
	// CampaignID scopes the query to a specific campaign (required).
	CampaignID string
	// AfterSeq returns only events with seq greater than this value.
	AfterSeq uint64
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

// ListEventsPageResult contains paginated event history for introspection tooling.
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

// SessionPage describes a page of session records.
type SessionPage struct {
	Sessions      []SessionRecord
	NextPageToken string
}

// SessionGate describes one gate and its resolution lifecycle within a session.
type SessionGate struct {
	CampaignID          string
	SessionID           string
	GateID              string
	GateType            string
	Status              session.GateStatus
	Reason              string
	CreatedAt           time.Time
	CreatedByActorType  string
	CreatedByActorID    string
	ResolvedAt          *time.Time
	ResolvedByActorType string
	ResolvedByActorID   string
	MetadataJSON        []byte
	ResolutionJSON      []byte
}

// SessionGateStore persists gate state for the same lifecycle rules the game engine enforces.
type SessionGateStore interface {
	// PutSessionGate stores a gate record.
	PutSessionGate(ctx context.Context, gate SessionGate) error
	// GetSessionGate retrieves a gate by id.
	GetSessionGate(ctx context.Context, campaignID, sessionID, gateID string) (SessionGate, error)
	// GetOpenSessionGate retrieves the currently open gate for a session.
	GetOpenSessionGate(ctx context.Context, campaignID, sessionID string) (SessionGate, error)
}

// SessionSpotlight captures spotlight turn ownership so clients can read turn-order intent.
type SessionSpotlight struct {
	CampaignID         string
	SessionID          string
	SpotlightType      session.SpotlightType
	CharacterID        string
	UpdatedAt          time.Time
	UpdatedByActorType string
	UpdatedByActorID   string
}

// SessionSpotlightStore persists current spotlight state for session-facing APIs.
type SessionSpotlightStore interface {
	// PutSessionSpotlight stores the current spotlight for a session.
	PutSessionSpotlight(ctx context.Context, spotlight SessionSpotlight) error
	// GetSessionSpotlight retrieves the current spotlight for a session.
	GetSessionSpotlight(ctx context.Context, campaignID, sessionID string) (SessionSpotlight, error)
	// ClearSessionSpotlight removes the spotlight for a session.
	ClearSessionSpotlight(ctx context.Context, campaignID, sessionID string) error
}

// Snapshot is a materialized campaign/session state checkpoint derived from the event journal.
// Snapshots are accelerators for replay, not the source of authority.
type Snapshot struct {
	CampaignID          string
	SessionID           string
	EventSeq            uint64
	CharacterStatesJSON []byte
	GMStateJSON         []byte
	SystemStateJSON     []byte
	CreatedAt           time.Time
}

// SnapshotStore persists replay checkpoints used to jump event replay work.
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

// ParticipantClaim describes enforced uniqueness of user-to-seat binding.
type ParticipantClaim struct {
	CampaignID    string
	UserID        string
	ParticipantID string
	ClaimedAt     time.Time
}

// ClaimIndexStore keeps seat claim uniqueness from drifting during concurrent joins.
type ClaimIndexStore interface {
	// PutParticipantClaim stores a user claim for a participant seat.
	PutParticipantClaim(ctx context.Context, campaignID, userID, participantID string, claimedAt time.Time) error
	// GetParticipantClaim returns the claim for a user in a campaign.
	GetParticipantClaim(ctx context.Context, campaignID, userID string) (ParticipantClaim, error)
	// DeleteParticipantClaim removes a claim by user.
	DeleteParticipantClaim(ctx context.Context, campaignID, userID string) error
}

// ForkMetadata tracks campaign lineage needed for fork navigation and support tooling.
type ForkMetadata struct {
	ParentCampaignID string
	ForkEventSeq     uint64
	OriginCampaignID string
}

// CampaignForkStore persists fork lineage metadata for derived-campaign workflows.
type CampaignForkStore interface {
	// GetCampaignForkMetadata retrieves fork metadata for a campaign.
	GetCampaignForkMetadata(ctx context.Context, campaignID string) (ForkMetadata, error)
	// SetCampaignForkMetadata sets fork metadata for a campaign.
	SetCampaignForkMetadata(ctx context.Context, campaignID string, metadata ForkMetadata) error
}

// ProjectionStore groups read-model-oriented stores consumed by APIs and queries.
type ProjectionStore interface {
	CampaignStore
	ParticipantStore
	ClaimIndexStore
	InviteStore
	CharacterStore
	DaggerheartStore
	SessionStore
	SnapshotStore
	CampaignForkStore
	StatisticsStore
}

// Store is a composite interface for all persistence concerns used across event
// sourcing, projection application, and queries.
type Store interface {
	CampaignStore
	ParticipantStore
	ClaimIndexStore
	CharacterStore
	InviteStore
	DaggerheartStore
	SessionStore
	EventStore
	TelemetryStore
	StatisticsStore
	SnapshotStore
	CampaignForkStore
	Close() error
}
