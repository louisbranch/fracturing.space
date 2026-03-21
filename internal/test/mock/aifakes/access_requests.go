package aifakes

import (
	"context"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// AccessRequestStore is an in-memory access-request repository fake.
type AccessRequestStore struct {
	AccessRequests map[string]storage.AccessRequestRecord

	ListAccessRequestsByRequesterErr   error
	ListAccessRequestsByRequesterCalls int
	GetApprovedInvokeAccessCalls       int
}

// NewAccessRequestStore creates an initialized access-request fake.
func NewAccessRequestStore() *AccessRequestStore {
	return &AccessRequestStore{AccessRequests: make(map[string]storage.AccessRequestRecord)}
}

// PutAccessRequest stores an access request record.
func (s *AccessRequestStore) PutAccessRequest(_ context.Context, record storage.AccessRequestRecord) error {
	s.AccessRequests[record.ID] = record
	return nil
}

// GetAccessRequest returns an access request by ID.
func (s *AccessRequestStore) GetAccessRequest(_ context.Context, accessRequestID string) (storage.AccessRequestRecord, error) {
	rec, ok := s.AccessRequests[accessRequestID]
	if !ok {
		return storage.AccessRequestRecord{}, storage.ErrNotFound
	}
	return rec, nil
}

// ListAccessRequestsByRequester lists requester-owned access requests.
func (s *AccessRequestStore) ListAccessRequestsByRequester(_ context.Context, requesterUserID string, _ int, _ string) (storage.AccessRequestPage, error) {
	s.ListAccessRequestsByRequesterCalls++
	if s.ListAccessRequestsByRequesterErr != nil {
		return storage.AccessRequestPage{}, s.ListAccessRequestsByRequesterErr
	}
	items := make([]storage.AccessRequestRecord, 0)
	for _, rec := range s.AccessRequests {
		if rec.RequesterUserID == requesterUserID {
			items = append(items, rec)
		}
	}
	return storage.AccessRequestPage{AccessRequests: items}, nil
}

// GetApprovedInvokeAccessByRequesterForAgent returns a matching approved invoke record.
func (s *AccessRequestStore) GetApprovedInvokeAccessByRequesterForAgent(_ context.Context, requesterUserID string, ownerUserID string, agentID string) (storage.AccessRequestRecord, error) {
	s.GetApprovedInvokeAccessCalls++
	requesterUserID = strings.TrimSpace(requesterUserID)
	ownerUserID = strings.TrimSpace(ownerUserID)
	agentID = strings.TrimSpace(agentID)
	for _, rec := range s.AccessRequests {
		if strings.TrimSpace(rec.RequesterUserID) != requesterUserID {
			continue
		}
		if strings.TrimSpace(rec.OwnerUserID) != ownerUserID {
			continue
		}
		if strings.TrimSpace(rec.AgentID) != agentID {
			continue
		}
		if strings.ToLower(strings.TrimSpace(rec.Scope)) != "invoke" {
			continue
		}
		if strings.ToLower(strings.TrimSpace(rec.Status)) != "approved" {
			continue
		}
		return rec, nil
	}
	return storage.AccessRequestRecord{}, storage.ErrNotFound
}

// ListApprovedInvokeAccessRequestsByRequester returns approved invoke requests for requester.
func (s *AccessRequestStore) ListApprovedInvokeAccessRequestsByRequester(_ context.Context, requesterUserID string, _ int, _ string) (storage.AccessRequestPage, error) {
	requesterUserID = strings.TrimSpace(requesterUserID)
	items := make([]storage.AccessRequestRecord, 0)
	for _, rec := range s.AccessRequests {
		if strings.TrimSpace(rec.RequesterUserID) != requesterUserID {
			continue
		}
		if strings.ToLower(strings.TrimSpace(rec.Scope)) != "invoke" {
			continue
		}
		if strings.ToLower(strings.TrimSpace(rec.Status)) != "approved" {
			continue
		}
		items = append(items, rec)
	}
	return storage.AccessRequestPage{AccessRequests: items}, nil
}

// ListAccessRequestsByOwner lists owner-owned access requests.
func (s *AccessRequestStore) ListAccessRequestsByOwner(_ context.Context, ownerUserID string, _ int, _ string) (storage.AccessRequestPage, error) {
	items := make([]storage.AccessRequestRecord, 0)
	for _, rec := range s.AccessRequests {
		if rec.OwnerUserID == ownerUserID {
			items = append(items, rec)
		}
	}
	return storage.AccessRequestPage{AccessRequests: items}, nil
}

// ReviewAccessRequest transitions a pending request to a reviewed status.
func (s *AccessRequestStore) ReviewAccessRequest(_ context.Context, input storage.ReviewAccessRequestInput) error {
	rec, ok := s.AccessRequests[input.AccessRequestID]
	if !ok || rec.OwnerUserID != input.OwnerUserID {
		return storage.ErrNotFound
	}
	if rec.Status != "pending" {
		return storage.ErrConflict
	}
	rec.Status = input.Status
	rec.ReviewerUserID = input.ReviewerUserID
	rec.ReviewNote = input.ReviewNote
	rec.UpdatedAt = input.ReviewedAt
	rec.ReviewedAt = &input.ReviewedAt
	s.AccessRequests[input.AccessRequestID] = rec
	return nil
}

// RevokeAccessRequest transitions an approved request to a revoked status.
func (s *AccessRequestStore) RevokeAccessRequest(_ context.Context, input storage.RevokeAccessRequestInput) error {
	rec, ok := s.AccessRequests[input.AccessRequestID]
	if !ok || rec.OwnerUserID != input.OwnerUserID {
		return storage.ErrNotFound
	}
	if rec.Status != "approved" {
		return storage.ErrConflict
	}
	rec.Status = input.Status
	rec.ReviewerUserID = input.ReviewerUserID
	rec.ReviewNote = input.ReviewNote
	rec.UpdatedAt = input.RevokedAt
	s.AccessRequests[input.AccessRequestID] = rec
	return nil
}
