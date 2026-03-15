package gametest

import (
	"context"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// FakeInviteStore is a test double for storage.InviteStore.
type FakeInviteStore struct {
	Invites   map[string]storage.InviteRecord
	PutErr    error
	GetErr    error
	ListErr   error
	UpdateErr error
}

// NewFakeInviteStore returns a ready-to-use invite store fake.
func NewFakeInviteStore() *FakeInviteStore {
	return &FakeInviteStore{Invites: make(map[string]storage.InviteRecord)}
}

func (s *FakeInviteStore) PutInvite(_ context.Context, inv storage.InviteRecord) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	s.Invites[inv.ID] = inv
	return nil
}

func (s *FakeInviteStore) GetInvite(_ context.Context, inviteID string) (storage.InviteRecord, error) {
	if s.GetErr != nil {
		return storage.InviteRecord{}, s.GetErr
	}
	inv, ok := s.Invites[inviteID]
	if !ok {
		return storage.InviteRecord{}, storage.ErrNotFound
	}
	return inv, nil
}

func (s *FakeInviteStore) ListInvites(_ context.Context, campaignID string, recipientUserID string, status invite.Status, pageSize int, pageToken string) (storage.InvitePage, error) {
	if s.ListErr != nil {
		return storage.InvitePage{}, s.ListErr
	}
	result := make([]storage.InviteRecord, 0)
	for _, inv := range s.Invites {
		if inv.CampaignID != campaignID {
			continue
		}
		if recipientUserID != "" && inv.RecipientUserID != recipientUserID {
			continue
		}
		if status != invite.StatusUnspecified && inv.Status != status {
			continue
		}
		result = append(result, inv)
	}
	return storage.InvitePage{Invites: result, NextPageToken: ""}, nil
}

func (s *FakeInviteStore) ListPendingInvites(_ context.Context, campaignID string, pageSize int, pageToken string) (storage.InvitePage, error) {
	if s.ListErr != nil {
		return storage.InvitePage{}, s.ListErr
	}
	result := make([]storage.InviteRecord, 0)
	for _, inv := range s.Invites {
		if inv.CampaignID == campaignID && inv.Status == invite.StatusPending {
			result = append(result, inv)
		}
	}
	return storage.InvitePage{Invites: result, NextPageToken: ""}, nil
}

func (s *FakeInviteStore) ListPendingInvitesForRecipient(_ context.Context, userID string, pageSize int, pageToken string) (storage.InvitePage, error) {
	if s.ListErr != nil {
		return storage.InvitePage{}, s.ListErr
	}
	result := make([]storage.InviteRecord, 0)
	for _, inv := range s.Invites {
		if inv.RecipientUserID == userID && inv.Status == invite.StatusPending {
			result = append(result, inv)
		}
	}
	return storage.InvitePage{Invites: result, NextPageToken: ""}, nil
}

func (s *FakeInviteStore) UpdateInviteStatus(_ context.Context, inviteID string, status invite.Status, updatedAt time.Time) error {
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
