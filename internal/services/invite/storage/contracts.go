// Package storage defines the invite service storage contracts.
package storage

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when a record does not exist.
var ErrNotFound = errors.New("not found")

// Status represents the invite lifecycle state.
type Status string

const (
	StatusPending  Status = "pending"
	StatusClaimed  Status = "claimed"
	StatusRevoked  Status = "revoked"
	StatusDeclined Status = "declined"
)

// InviteRecord captures invite state.
type InviteRecord struct {
	ID                     string
	CampaignID             string
	ParticipantID          string
	RecipientUserID        string
	Status                 Status
	CreatedByParticipantID string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// InvitePage describes a page of invite records.
type InvitePage struct {
	Invites       []InviteRecord
	NextPageToken string
}

// InviteStore owns invite lifecycle persistence.
type InviteStore interface {
	GetInvite(ctx context.Context, inviteID string) (InviteRecord, error)
	ListInvites(ctx context.Context, campaignID, recipientUserID string, status Status, pageSize int, pageToken string) (InvitePage, error)
	ListPendingInvites(ctx context.Context, campaignID string, pageSize int, pageToken string) (InvitePage, error)
	ListPendingInvitesForRecipient(ctx context.Context, userID string, pageSize int, pageToken string) (InvitePage, error)
	PutInvite(ctx context.Context, inv InviteRecord) error
	UpdateInviteStatus(ctx context.Context, inviteID string, status Status, updatedAt time.Time) error
}

// OutboxEvent represents a pending outbox event.
type OutboxEvent struct {
	ID          string
	EventType   string
	PayloadJSON []byte
	DedupeKey   string
	CreatedAt   time.Time
}

// LeasedOutboxEvent represents a leased outbox event returned to consumers.
type LeasedOutboxEvent struct {
	ID           string
	EventType    string
	PayloadJSON  string
	DedupeKey    string
	Status       string
	AttemptCount int
	LeaseOwner   string
	CreatedAt    time.Time
}

// OutboxStore manages outbox events for downstream notification processing.
type OutboxStore interface {
	Enqueue(ctx context.Context, evt OutboxEvent) error
	LeaseOutboxEvents(ctx context.Context, consumer string, limit int, leaseTTL time.Duration, now time.Time) ([]LeasedOutboxEvent, error)
	AckOutboxEvent(ctx context.Context, eventID, consumer string, outcome string, nextAttemptAt time.Time, lastError string, now time.Time) error
}
