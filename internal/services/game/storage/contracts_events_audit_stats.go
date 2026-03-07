package storage

import (
	"context"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

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
	// Filter captures typed event-query filter constraints.
	Filter EventQueryFilter
}

// EventQueryFilter captures storage-level filter intent without leaking SQL.
//
// Expression supports AIP-160 event filtering syntax. Field filters are exact
// matches that are applied in addition to Expression constraints.
type EventQueryFilter struct {
	Expression    string
	EventType     string
	SessionID     string
	SceneID       string
	RequestID     string
	InvocationID  string
	ActorType     string
	ActorID       string
	SystemID      string
	SystemVersion string
	EntityType    string
	EntityID      string
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

// AuditEvent captures operational observations emitted during command execution.
type AuditEvent struct {
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

// AuditEventStore persists operational audit records for audits and incident analysis.
type AuditEventStore interface {
	AppendAuditEvent(ctx context.Context, evt AuditEvent) error
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
