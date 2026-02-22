package aifakes

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// Sealer is an in-memory secret sealer fake for AI service tests.
type Sealer struct {
	SealErr error
	OpenErr error
}

// Seal returns a deterministic encrypted value unless configured to fail.
func (f *Sealer) Seal(value string) (string, error) {
	if f.SealErr != nil {
		return "", f.SealErr
	}
	return "enc:" + value, nil
}

// Open returns the plaintext value unless configured to fail.
func (f *Sealer) Open(sealed string) (string, error) {
	if f.OpenErr != nil {
		return "", f.OpenErr
	}
	const prefix = "enc:"
	if len(sealed) >= len(prefix) && sealed[:len(prefix)] == prefix {
		return sealed[len(prefix):], nil
	}
	return sealed, nil
}

// Store is an in-memory AI storage fake used by service-level tests.
type Store struct {
	Credentials     map[string]storage.CredentialRecord
	Agents          map[string]storage.AgentRecord
	AccessRequests  map[string]storage.AccessRequestRecord
	ProviderGrants  map[string]storage.ProviderGrantRecord
	ConnectSessions map[string]storage.ProviderConnectSessionRecord
	AuditEvents     []storage.AuditEventRecord
	AuditEventNames []string

	ListAccessRequestsByRequesterErr   error
	ListAccessRequestsByRequesterCalls int
	GetApprovedInvokeAccessCalls       int
}

// NewStore creates an initialized in-memory store fake.
func NewStore() *Store {
	return &Store{
		Credentials:     make(map[string]storage.CredentialRecord),
		Agents:          make(map[string]storage.AgentRecord),
		AccessRequests:  make(map[string]storage.AccessRequestRecord),
		ProviderGrants:  make(map[string]storage.ProviderGrantRecord),
		ConnectSessions: make(map[string]storage.ProviderConnectSessionRecord),
	}
}

// PutCredential stores a credential record.
func (s *Store) PutCredential(_ context.Context, record storage.CredentialRecord) error {
	s.Credentials[record.ID] = record
	return nil
}

// GetCredential returns a credential by ID.
func (s *Store) GetCredential(_ context.Context, credentialID string) (storage.CredentialRecord, error) {
	rec, ok := s.Credentials[credentialID]
	if !ok {
		return storage.CredentialRecord{}, storage.ErrNotFound
	}
	return rec, nil
}

// ListCredentialsByOwner lists credentials for an owner.
func (s *Store) ListCredentialsByOwner(_ context.Context, ownerUserID string, _ int, _ string) (storage.CredentialPage, error) {
	items := make([]storage.CredentialRecord, 0)
	for _, rec := range s.Credentials {
		if rec.OwnerUserID == ownerUserID {
			items = append(items, rec)
		}
	}
	return storage.CredentialPage{Credentials: items}, nil
}

// RevokeCredential marks a credential as revoked.
func (s *Store) RevokeCredential(_ context.Context, ownerUserID string, credentialID string, revokedAt time.Time) error {
	rec, ok := s.Credentials[credentialID]
	if !ok || rec.OwnerUserID != ownerUserID {
		return storage.ErrNotFound
	}
	rec.Status = "revoked"
	rec.RevokedAt = &revokedAt
	rec.UpdatedAt = revokedAt
	s.Credentials[credentialID] = rec
	return nil
}

// PutAgent stores an agent record.
func (s *Store) PutAgent(_ context.Context, record storage.AgentRecord) error {
	s.Agents[record.ID] = record
	return nil
}

// GetAgent returns an agent by ID.
func (s *Store) GetAgent(_ context.Context, agentID string) (storage.AgentRecord, error) {
	rec, ok := s.Agents[agentID]
	if !ok {
		return storage.AgentRecord{}, storage.ErrNotFound
	}
	return rec, nil
}

// ListAgentsByOwner lists agents for an owner.
func (s *Store) ListAgentsByOwner(_ context.Context, ownerUserID string, _ int, _ string) (storage.AgentPage, error) {
	items := make([]storage.AgentRecord, 0)
	for _, rec := range s.Agents {
		if rec.OwnerUserID == ownerUserID {
			items = append(items, rec)
		}
	}
	return storage.AgentPage{Agents: items}, nil
}

// DeleteAgent removes an owned agent.
func (s *Store) DeleteAgent(_ context.Context, ownerUserID string, agentID string) error {
	rec, ok := s.Agents[agentID]
	if !ok || rec.OwnerUserID != ownerUserID {
		return storage.ErrNotFound
	}
	delete(s.Agents, agentID)
	return nil
}

// PutAccessRequest stores an access request record.
func (s *Store) PutAccessRequest(_ context.Context, record storage.AccessRequestRecord) error {
	s.AccessRequests[record.ID] = record
	return nil
}

// GetAccessRequest returns an access request by ID.
func (s *Store) GetAccessRequest(_ context.Context, accessRequestID string) (storage.AccessRequestRecord, error) {
	rec, ok := s.AccessRequests[accessRequestID]
	if !ok {
		return storage.AccessRequestRecord{}, storage.ErrNotFound
	}
	return rec, nil
}

// ListAccessRequestsByRequester lists requester-owned access requests.
func (s *Store) ListAccessRequestsByRequester(_ context.Context, requesterUserID string, _ int, _ string) (storage.AccessRequestPage, error) {
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
func (s *Store) GetApprovedInvokeAccessByRequesterForAgent(_ context.Context, requesterUserID string, ownerUserID string, agentID string) (storage.AccessRequestRecord, error) {
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
func (s *Store) ListApprovedInvokeAccessRequestsByRequester(_ context.Context, requesterUserID string, _ int, _ string) (storage.AccessRequestPage, error) {
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
func (s *Store) ListAccessRequestsByOwner(_ context.Context, ownerUserID string, _ int, _ string) (storage.AccessRequestPage, error) {
	items := make([]storage.AccessRequestRecord, 0)
	for _, rec := range s.AccessRequests {
		if rec.OwnerUserID == ownerUserID {
			items = append(items, rec)
		}
	}
	return storage.AccessRequestPage{AccessRequests: items}, nil
}

// ReviewAccessRequest transitions a pending request to a reviewed status.
func (s *Store) ReviewAccessRequest(_ context.Context, ownerUserID string, accessRequestID string, status string, reviewerUserID string, reviewNote string, reviewedAt time.Time) error {
	rec, ok := s.AccessRequests[accessRequestID]
	if !ok || rec.OwnerUserID != ownerUserID {
		return storage.ErrNotFound
	}
	if rec.Status != "pending" {
		return storage.ErrConflict
	}
	rec.Status = status
	rec.ReviewerUserID = reviewerUserID
	rec.ReviewNote = reviewNote
	rec.UpdatedAt = reviewedAt
	rec.ReviewedAt = &reviewedAt
	s.AccessRequests[accessRequestID] = rec
	return nil
}

// RevokeAccessRequest transitions an approved request to a revoked status.
func (s *Store) RevokeAccessRequest(_ context.Context, ownerUserID string, accessRequestID string, status string, reviewerUserID string, reviewNote string, revokedAt time.Time) error {
	rec, ok := s.AccessRequests[accessRequestID]
	if !ok || rec.OwnerUserID != ownerUserID {
		return storage.ErrNotFound
	}
	if rec.Status != "approved" {
		return storage.ErrConflict
	}
	rec.Status = status
	rec.ReviewerUserID = reviewerUserID
	rec.ReviewNote = reviewNote
	rec.UpdatedAt = revokedAt
	s.AccessRequests[accessRequestID] = rec
	return nil
}

// PutAuditEvent appends an audit event record.
func (s *Store) PutAuditEvent(_ context.Context, record storage.AuditEventRecord) error {
	if strings.TrimSpace(record.ID) == "" {
		record.ID = fmt.Sprintf("%d", len(s.AuditEvents)+1)
	}
	s.AuditEvents = append(s.AuditEvents, record)
	s.AuditEventNames = append(s.AuditEventNames, record.EventName)
	return nil
}

// ListAuditEventsByOwner returns paginated audit events matching the filter.
func (s *Store) ListAuditEventsByOwner(_ context.Context, ownerUserID string, pageSize int, pageToken string, filter storage.AuditEventFilter) (storage.AuditEventPage, error) {
	if pageSize <= 0 {
		return storage.AuditEventPage{}, errors.New("page size must be greater than zero")
	}
	eventName := strings.TrimSpace(filter.EventName)
	agentID := strings.TrimSpace(filter.AgentID)
	var (
		createdAfter  *time.Time
		createdBefore *time.Time
	)
	if filter.CreatedAfter != nil {
		timestamp := filter.CreatedAfter.UTC()
		createdAfter = &timestamp
	}
	if filter.CreatedBefore != nil {
		timestamp := filter.CreatedBefore.UTC()
		createdBefore = &timestamp
	}
	items := make([]storage.AuditEventRecord, 0, len(s.AuditEvents))
	for _, rec := range s.AuditEvents {
		if rec.OwnerUserID == ownerUserID {
			if eventName != "" && rec.EventName != eventName {
				continue
			}
			if agentID != "" && rec.AgentID != agentID {
				continue
			}
			if createdAfter != nil && rec.CreatedAt.Before(*createdAfter) {
				continue
			}
			if createdBefore != nil && rec.CreatedAt.After(*createdBefore) {
				continue
			}
			items = append(items, rec)
		}
	}
	sort.Slice(items, func(i int, j int) bool {
		return compareAuditEventID(items[i].ID, items[j].ID) < 0
	})
	start := 0
	pageToken = strings.TrimSpace(pageToken)
	if pageToken != "" {
		start = len(items)
		for idx, rec := range items {
			if compareAuditEventID(rec.ID, pageToken) > 0 {
				start = idx
				break
			}
		}
	}
	if start >= len(items) {
		return storage.AuditEventPage{AuditEvents: []storage.AuditEventRecord{}}, nil
	}

	end := start + pageSize
	nextPageToken := ""
	if end < len(items) {
		nextPageToken = items[end-1].ID
	} else {
		end = len(items)
	}
	return storage.AuditEventPage{
		AuditEvents:   items[start:end],
		NextPageToken: nextPageToken,
	}, nil
}

func compareAuditEventID(left string, right string) int {
	leftID, leftErr := strconv.ParseInt(strings.TrimSpace(left), 10, 64)
	rightID, rightErr := strconv.ParseInt(strings.TrimSpace(right), 10, 64)
	if leftErr == nil && rightErr == nil {
		if leftID < rightID {
			return -1
		}
		if leftID > rightID {
			return 1
		}
		return 0
	}
	return strings.Compare(strings.TrimSpace(left), strings.TrimSpace(right))
}

// PutProviderGrant stores a provider grant record.
func (s *Store) PutProviderGrant(_ context.Context, record storage.ProviderGrantRecord) error {
	s.ProviderGrants[record.ID] = record
	return nil
}

// GetProviderGrant returns a provider grant by ID.
func (s *Store) GetProviderGrant(_ context.Context, providerGrantID string) (storage.ProviderGrantRecord, error) {
	rec, ok := s.ProviderGrants[providerGrantID]
	if !ok {
		return storage.ProviderGrantRecord{}, storage.ErrNotFound
	}
	return rec, nil
}

// ListProviderGrantsByOwner lists provider grants for an owner.
func (s *Store) ListProviderGrantsByOwner(_ context.Context, ownerUserID string, _ int, _ string, filter storage.ProviderGrantFilter) (storage.ProviderGrantPage, error) {
	provider := strings.ToLower(strings.TrimSpace(filter.Provider))
	status := strings.ToLower(strings.TrimSpace(filter.Status))
	items := make([]storage.ProviderGrantRecord, 0)
	for _, rec := range s.ProviderGrants {
		if rec.OwnerUserID == ownerUserID {
			if provider != "" && !strings.EqualFold(strings.TrimSpace(rec.Provider), provider) {
				continue
			}
			if status != "" && !strings.EqualFold(strings.TrimSpace(rec.Status), status) {
				continue
			}
			items = append(items, rec)
		}
	}
	return storage.ProviderGrantPage{ProviderGrants: items}, nil
}

// RevokeProviderGrant marks an owned provider grant as revoked.
func (s *Store) RevokeProviderGrant(_ context.Context, ownerUserID string, providerGrantID string, revokedAt time.Time) error {
	rec, ok := s.ProviderGrants[providerGrantID]
	if !ok || rec.OwnerUserID != ownerUserID {
		return storage.ErrNotFound
	}
	rec.Status = "revoked"
	rec.RevokedAt = &revokedAt
	rec.UpdatedAt = revokedAt
	s.ProviderGrants[providerGrantID] = rec
	return nil
}

// UpdateProviderGrantToken updates provider token metadata.
func (s *Store) UpdateProviderGrantToken(_ context.Context, ownerUserID string, providerGrantID string, tokenCiphertext string, refreshedAt time.Time, expiresAt *time.Time, status string, lastRefreshError string) error {
	rec, ok := s.ProviderGrants[providerGrantID]
	if !ok || rec.OwnerUserID != ownerUserID {
		return storage.ErrNotFound
	}
	rec.TokenCiphertext = tokenCiphertext
	rec.UpdatedAt = refreshedAt
	rec.LastRefreshedAt = &refreshedAt
	rec.ExpiresAt = expiresAt
	rec.Status = status
	rec.LastRefreshError = lastRefreshError
	s.ProviderGrants[providerGrantID] = rec
	return nil
}

// PutProviderConnectSession stores a provider connect session.
func (s *Store) PutProviderConnectSession(_ context.Context, record storage.ProviderConnectSessionRecord) error {
	s.ConnectSessions[record.ID] = record
	return nil
}

// GetProviderConnectSession returns a provider connect session by ID.
func (s *Store) GetProviderConnectSession(_ context.Context, connectSessionID string) (storage.ProviderConnectSessionRecord, error) {
	rec, ok := s.ConnectSessions[connectSessionID]
	if !ok {
		return storage.ProviderConnectSessionRecord{}, storage.ErrNotFound
	}
	return rec, nil
}

// CompleteProviderConnectSession marks a pending connect session completed.
func (s *Store) CompleteProviderConnectSession(_ context.Context, ownerUserID string, connectSessionID string, completedAt time.Time) error {
	rec, ok := s.ConnectSessions[connectSessionID]
	if !ok || rec.OwnerUserID != ownerUserID || rec.Status != "pending" {
		return storage.ErrNotFound
	}
	rec.Status = "completed"
	rec.CompletedAt = &completedAt
	rec.UpdatedAt = completedAt
	s.ConnectSessions[connectSessionID] = rec
	return nil
}
