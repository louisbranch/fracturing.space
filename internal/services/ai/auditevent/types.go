package auditevent

import (
	"context"
	"time"
)

// Name identifies one persisted audit event type.
type Name string

const (
	NameAccessRequestCreated  Name = "access_request.created"
	NameAccessRequestReviewed Name = "access_request.reviewed"
	NameAccessRequestRevoked  Name = "access_request.revoked"
	NameAgentInvokeShared     Name = "agent.invoke.shared"
)

// Event stores one append-only AI audit event.
type Event struct {
	ID string

	EventName Name

	ActorUserID string

	OwnerUserID     string
	RequesterUserID string
	AgentID         string
	AccessRequestID string

	Outcome string

	CreatedAt time.Time
}

// Page is a paged set of audit events.
type Page struct {
	AuditEvents   []Event
	NextPageToken string
}

// Filter narrows owner-scoped audit event listing.
//
// Security note: owner scope is mandatory and enforced separately; these fields
// are optional in-memory constraints that must never broaden tenant visibility.
type Filter struct {
	EventName Name
	AgentID   string

	CreatedAfter  *time.Time
	CreatedBefore *time.Time
}

// Store persists append-only AI audit events.
type Store interface {
	PutAuditEvent(ctx context.Context, event Event) error
	ListAuditEventsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string, filter Filter) (Page, error)
}
