package aifakes

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/ai/accessrequest"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// AccessRequestStore is an in-memory access-request repository fake.
type AccessRequestStore struct {
	AccessRequests map[string]accessrequest.AccessRequest
}

// NewAccessRequestStore creates an initialized access-request fake.
func NewAccessRequestStore() *AccessRequestStore {
	return &AccessRequestStore{AccessRequests: make(map[string]accessrequest.AccessRequest)}
}

// PutAccessRequest stores an access request.
func (s *AccessRequestStore) PutAccessRequest(_ context.Context, request accessrequest.AccessRequest) error {
	s.AccessRequests[request.ID] = request
	return nil
}

// GetAccessRequest returns an access request by ID.
func (s *AccessRequestStore) GetAccessRequest(_ context.Context, accessRequestID string) (accessrequest.AccessRequest, error) {
	rec, ok := s.AccessRequests[accessRequestID]
	if !ok {
		return accessrequest.AccessRequest{}, storage.ErrNotFound
	}
	return rec, nil
}

// ListAccessRequestsByRequester lists requester-owned access requests.
func (s *AccessRequestStore) ListAccessRequestsByRequester(_ context.Context, requesterUserID string, _ int, _ string) (accessrequest.Page, error) {
	items := make([]accessrequest.AccessRequest, 0)
	for _, ar := range s.AccessRequests {
		if ar.RequesterUserID == requesterUserID {
			items = append(items, ar)
		}
	}
	return accessrequest.Page{AccessRequests: items}, nil
}

// GetApprovedInvokeAccessByRequesterForAgent returns a matching approved invoke record.
func (s *AccessRequestStore) GetApprovedInvokeAccessByRequesterForAgent(_ context.Context, requesterUserID string, ownerUserID string, agentID string) (accessrequest.AccessRequest, error) {
	for _, ar := range s.AccessRequests {
		if ar.RequesterUserID != requesterUserID {
			continue
		}
		if ar.OwnerUserID != ownerUserID {
			continue
		}
		if ar.AgentID != agentID {
			continue
		}
		if ar.Scope != accessrequest.ScopeInvoke {
			continue
		}
		if ar.Status != accessrequest.StatusApproved {
			continue
		}
		return ar, nil
	}
	return accessrequest.AccessRequest{}, storage.ErrNotFound
}

// ListApprovedInvokeAccessRequestsByRequester returns approved invoke requests for requester.
func (s *AccessRequestStore) ListApprovedInvokeAccessRequestsByRequester(_ context.Context, requesterUserID string, _ int, _ string) (accessrequest.Page, error) {
	items := make([]accessrequest.AccessRequest, 0)
	for _, ar := range s.AccessRequests {
		if ar.RequesterUserID != requesterUserID {
			continue
		}
		if ar.Scope != accessrequest.ScopeInvoke {
			continue
		}
		if ar.Status != accessrequest.StatusApproved {
			continue
		}
		items = append(items, ar)
	}
	return accessrequest.Page{AccessRequests: items}, nil
}

// ListAccessRequestsByOwner lists owner-owned access requests.
func (s *AccessRequestStore) ListAccessRequestsByOwner(_ context.Context, ownerUserID string, _ int, _ string) (accessrequest.Page, error) {
	items := make([]accessrequest.AccessRequest, 0)
	for _, ar := range s.AccessRequests {
		if ar.OwnerUserID == ownerUserID {
			items = append(items, ar)
		}
	}
	return accessrequest.Page{AccessRequests: items}, nil
}

// ReviewAccessRequest transitions a pending request to a reviewed status.
func (s *AccessRequestStore) ReviewAccessRequest(_ context.Context, reviewed accessrequest.AccessRequest) error {
	existing, ok := s.AccessRequests[reviewed.ID]
	if !ok || existing.OwnerUserID != reviewed.OwnerUserID {
		return storage.ErrNotFound
	}
	if existing.Status != accessrequest.StatusPending {
		return storage.ErrConflict
	}
	existing.Status = reviewed.Status
	existing.ReviewerUserID = reviewed.ReviewerUserID
	existing.ReviewNote = reviewed.ReviewNote
	existing.UpdatedAt = reviewed.UpdatedAt
	existing.ReviewedAt = reviewed.ReviewedAt
	s.AccessRequests[reviewed.ID] = existing
	return nil
}

// RevokeAccessRequest transitions an approved request to a revoked status.
func (s *AccessRequestStore) RevokeAccessRequest(_ context.Context, revoked accessrequest.AccessRequest) error {
	existing, ok := s.AccessRequests[revoked.ID]
	if !ok || existing.OwnerUserID != revoked.OwnerUserID {
		return storage.ErrNotFound
	}
	if existing.Status != accessrequest.StatusApproved {
		return storage.ErrConflict
	}
	existing.Status = revoked.Status
	existing.RevokerUserID = revoked.RevokerUserID
	existing.RevokeNote = revoked.RevokeNote
	existing.RevokedAt = revoked.RevokedAt
	existing.UpdatedAt = revoked.UpdatedAt
	s.AccessRequests[revoked.ID] = existing
	return nil
}
