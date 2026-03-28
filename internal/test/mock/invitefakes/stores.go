// Package invitefakes provides lightweight in-memory fakes for the invite
// service storage interfaces so that transport-level tests can exercise
// handler logic without requiring SQLite.
package invitefakes

import (
	"context"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/invite/storage"
)

// Compile-time interface compliance.
var (
	_ storage.InviteStore = (*InviteStore)(nil)
	_ storage.OutboxStore = (*OutboxStore)(nil)
)

// InviteStore is an in-memory InviteStore fake with error injection.
type InviteStore struct {
	Invites   map[string]storage.InviteRecord
	PutErr    error
	GetErr    error
	ListErr   error
	UpdateErr error
}

// NewInviteStore constructs an InviteStore fake with initialized state.
func NewInviteStore() *InviteStore {
	return &InviteStore{Invites: make(map[string]storage.InviteRecord)}
}

func (s *InviteStore) GetInvite(_ context.Context, inviteID string) (storage.InviteRecord, error) {
	if s.GetErr != nil {
		return storage.InviteRecord{}, s.GetErr
	}
	inv, ok := s.Invites[inviteID]
	if !ok {
		return storage.InviteRecord{}, storage.ErrNotFound
	}
	return inv, nil
}

func (s *InviteStore) ListInvites(_ context.Context, campaignID, recipientUserID string, status storage.Status, pageSize int, _ string) (storage.InvitePage, error) {
	if s.ListErr != nil {
		return storage.InvitePage{}, s.ListErr
	}
	var out []storage.InviteRecord
	for _, inv := range s.Invites {
		if campaignID != "" && inv.CampaignID != campaignID {
			continue
		}
		if recipientUserID != "" && inv.RecipientUserID != recipientUserID {
			continue
		}
		if status != "" && inv.Status != status {
			continue
		}
		out = append(out, inv)
		if pageSize > 0 && len(out) >= pageSize {
			break
		}
	}
	return storage.InvitePage{Invites: out}, nil
}

func (s *InviteStore) ListPendingInvites(_ context.Context, campaignID string, pageSize int, _ string) (storage.InvitePage, error) {
	if s.ListErr != nil {
		return storage.InvitePage{}, s.ListErr
	}
	var out []storage.InviteRecord
	for _, inv := range s.Invites {
		if inv.CampaignID != campaignID || inv.Status != storage.StatusPending {
			continue
		}
		out = append(out, inv)
		if pageSize > 0 && len(out) >= pageSize {
			break
		}
	}
	return storage.InvitePage{Invites: out}, nil
}

func (s *InviteStore) ListPendingInvitesForRecipient(_ context.Context, userID string, pageSize int, _ string) (storage.InvitePage, error) {
	if s.ListErr != nil {
		return storage.InvitePage{}, s.ListErr
	}
	var out []storage.InviteRecord
	for _, inv := range s.Invites {
		if inv.Status != storage.StatusPending || !strings.EqualFold(inv.RecipientUserID, userID) {
			continue
		}
		out = append(out, inv)
		if pageSize > 0 && len(out) >= pageSize {
			break
		}
	}
	return storage.InvitePage{Invites: out}, nil
}

func (s *InviteStore) PutInvite(_ context.Context, inv storage.InviteRecord) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	s.Invites[inv.ID] = inv
	return nil
}

func (s *InviteStore) UpdateInviteStatus(_ context.Context, inviteID string, status storage.Status, updatedAt time.Time) error {
	if s.UpdateErr != nil {
		return s.UpdateErr
	}
	inv, ok := s.Invites[inviteID]
	if !ok {
		return storage.ErrNotFound
	}
	inv.Status = status
	inv.UpdatedAt = updatedAt
	s.Invites[inviteID] = inv
	return nil
}

// OutboxStore is an in-memory OutboxStore fake with error injection.
type OutboxStore struct {
	Events     []storage.OutboxEvent
	EnqueueErr error
	LeaseErr   error
	AckErr     error
}

// NewOutboxStore constructs an OutboxStore fake with initialized state.
func NewOutboxStore() *OutboxStore {
	return &OutboxStore{}
}

func (s *OutboxStore) Enqueue(_ context.Context, evt storage.OutboxEvent) error {
	if s.EnqueueErr != nil {
		return s.EnqueueErr
	}
	s.Events = append(s.Events, evt)
	return nil
}

func (s *OutboxStore) LeaseOutboxEvents(_ context.Context, _ string, limit int, _ time.Duration, _ time.Time) ([]storage.LeasedOutboxEvent, error) {
	if s.LeaseErr != nil {
		return nil, s.LeaseErr
	}
	var out []storage.LeasedOutboxEvent
	for _, evt := range s.Events {
		out = append(out, storage.LeasedOutboxEvent{
			ID:          evt.ID,
			EventType:   evt.EventType,
			PayloadJSON: string(evt.PayloadJSON),
			DedupeKey:   evt.DedupeKey,
			CreatedAt:   evt.CreatedAt,
		})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *OutboxStore) AckOutboxEvent(_ context.Context, _ string, _ string, _ string, _ time.Time, _ string, _ time.Time) error {
	return s.AckErr
}
