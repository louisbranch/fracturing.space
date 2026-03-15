package transcript

import (
	"context"
	"errors"
	"math"
	"strings"
)

const (
	// DefaultHistoryLimit is the default number of older messages returned when
	// callers do not request an explicit page size.
	DefaultHistoryLimit = 50
	// MaxHistoryLimit caps history pagination so one browser request cannot ask
	// the store for an unbounded transcript slice.
	MaxHistoryLimit = 200
)

var (
	// ErrInvalidScope reports that a campaign/session scope was missing.
	ErrInvalidScope = errors.New("transcript scope requires campaign and session")
	// ErrEmptyBody reports that an append request had no human message body.
	ErrEmptyBody = errors.New("transcript message body is required")
)

// MessageActor captures the participant identity attached to one human message.
type MessageActor struct {
	ParticipantID string
	Name          string
}

// Normalize trims transport whitespace and preserves the actor fallback policy
// expected by transcript storage.
func (a MessageActor) Normalize() MessageActor {
	a.ParticipantID = strings.TrimSpace(a.ParticipantID)
	a.Name = strings.TrimSpace(a.Name)
	return a
}

// Scope identifies one play transcript stream for a campaign's active session.
type Scope struct {
	CampaignID string
	SessionID  string
}

// Normalize trims transport whitespace so callers and adapters share one
// canonical scope shape.
func (s Scope) Normalize() Scope {
	s.CampaignID = strings.TrimSpace(s.CampaignID)
	s.SessionID = strings.TrimSpace(s.SessionID)
	return s
}

// Validate rejects missing campaign/session identifiers at the contract edge.
func (s Scope) Validate() error {
	s = s.Normalize()
	if s.CampaignID == "" || s.SessionID == "" {
		return ErrInvalidScope
	}
	return nil
}

// Message is one persisted human transcript row for the active-play surface.
type Message struct {
	MessageID       string
	CampaignID      string
	SessionID       string
	SequenceID      int64
	SentAt          string
	Actor           MessageActor
	Body            string
	ClientMessageID string
}

// AppendRequest describes one transcript append attempt, including the
// idempotency key supplied by the browser when present.
type AppendRequest struct {
	Scope           Scope
	Actor           MessageActor
	Body            string
	ClientMessageID string
}

// Normalize trims transport whitespace and preserves actor/body defaults in one
// place so adapters do not each reinvent request cleanup.
func (r AppendRequest) Normalize() AppendRequest {
	r.Scope = r.Scope.Normalize()
	r.Actor = r.Actor.Normalize()
	r.Body = strings.TrimSpace(r.Body)
	r.ClientMessageID = strings.TrimSpace(r.ClientMessageID)
	return r
}

// Validate enforces transcript-level invariants before the storage adapter runs.
func (r AppendRequest) Validate() error {
	r = r.Normalize()
	if err := r.Scope.Validate(); err != nil {
		return err
	}
	if r.Body == "" {
		return ErrEmptyBody
	}
	return nil
}

// AppendResult reports the stored message and whether it was returned through
// the client-message idempotency path.
type AppendResult struct {
	Message   Message
	Duplicate bool
}

// HistoryAfterQuery lists messages strictly after one known sequence ID.
type HistoryAfterQuery struct {
	Scope           Scope
	AfterSequenceID int64
}

// Normalize trims the scope and clamps negative cursors to the empty baseline.
func (q HistoryAfterQuery) Normalize() HistoryAfterQuery {
	q.Scope = q.Scope.Normalize()
	if q.AfterSequenceID < 0 {
		q.AfterSequenceID = 0
	}
	return q
}

// Validate rejects missing transcript scope.
func (q HistoryAfterQuery) Validate() error {
	return q.Scope.Validate()
}

// HistoryBeforeQuery lists older messages before one known sequence boundary.
type HistoryBeforeQuery struct {
	Scope            Scope
	BeforeSequenceID int64
	Limit            int
}

// Normalize trims the scope, turns an unset boundary into "latest", and clamps
// the page size into the supported transcript range.
func (q HistoryBeforeQuery) Normalize() HistoryBeforeQuery {
	q.Scope = q.Scope.Normalize()
	if q.BeforeSequenceID <= 0 {
		q.BeforeSequenceID = math.MaxInt64
	}
	switch {
	case q.Limit <= 0:
		q.Limit = DefaultHistoryLimit
	case q.Limit > MaxHistoryLimit:
		q.Limit = MaxHistoryLimit
	}
	return q
}

// Validate rejects missing transcript scope.
func (q HistoryBeforeQuery) Validate() error {
	return q.Scope.Validate()
}

// Store persists and replays play-owned human transcript messages.
type Store interface {
	LatestSequence(ctx context.Context, scope Scope) (int64, error)
	AppendMessage(ctx context.Context, req AppendRequest) (AppendResult, error)
	HistoryAfter(ctx context.Context, query HistoryAfterQuery) ([]Message, error)
	HistoryBefore(ctx context.Context, query HistoryBeforeQuery) ([]Message, error)
}
