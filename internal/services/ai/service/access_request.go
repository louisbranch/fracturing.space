package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/ai/accessrequest"
	"github.com/louisbranch/fracturing.space/internal/services/ai/auditevent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// AccessRequestService handles access-request lifecycle operations including
// creation, listing, review, revocation, and related audit events.
type AccessRequestService struct {
	agentStore         storage.AgentStore
	accessRequestStore storage.AccessRequestStore
	auditEventStore    auditevent.Store
	clock              Clock
	idGenerator        IDGenerator
}

// AccessRequestServiceConfig declares dependencies for the access-request service.
type AccessRequestServiceConfig struct {
	AgentStore         storage.AgentStore
	AccessRequestStore storage.AccessRequestStore
	AuditEventStore    auditevent.Store
	Clock              Clock
	IDGenerator        IDGenerator
}

// NewAccessRequestService builds an access-request service from explicit deps.
func NewAccessRequestService(cfg AccessRequestServiceConfig) (*AccessRequestService, error) {
	if cfg.AgentStore == nil {
		return nil, fmt.Errorf("ai: NewAccessRequestService: agent store is required")
	}
	if cfg.AccessRequestStore == nil {
		return nil, fmt.Errorf("ai: NewAccessRequestService: access request store is required")
	}
	if cfg.AuditEventStore == nil {
		return nil, fmt.Errorf("ai: NewAccessRequestService: audit event store is required")
	}
	return &AccessRequestService{
		agentStore:         cfg.AgentStore,
		accessRequestStore: cfg.AccessRequestStore,
		auditEventStore:    cfg.AuditEventStore,
		clock:              withDefaultClock(cfg.Clock),
		idGenerator:        withDefaultIDGenerator(cfg.IDGenerator),
	}, nil
}

// CreateAccessRequestInput is the domain input for creating an access request.
type CreateAccessRequestInput struct {
	RequesterUserID string
	AgentID         string
	Scope           string
	RequestNote     string
}

// Create creates a pending access request for the given agent, validates that
// the agent exists and is active, persists the record, and writes an audit event.
func (s *AccessRequestService) Create(ctx context.Context, input CreateAccessRequestInput) (accessrequest.AccessRequest, error) {
	agentRecord, err := s.agentStore.GetAgent(ctx, input.AgentID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return accessrequest.AccessRequest{}, Errorf(ErrKindFailedPrecondition, "agent is unavailable")
		}
		return accessrequest.AccessRequest{}, Wrapf(ErrKindInternal, err, "get agent")
	}
	if !agentRecord.Status.IsActive() {
		return accessrequest.AccessRequest{}, Errorf(ErrKindFailedPrecondition, "agent is unavailable")
	}

	createInput, err := accessrequest.NormalizeCreateInput(accessrequest.CreateInput{
		RequesterUserID: input.RequesterUserID,
		OwnerUserID:     agentRecord.OwnerUserID,
		AgentID:         agentRecord.ID,
		Scope:           accessrequest.Scope(input.Scope),
		RequestNote:     input.RequestNote,
	})
	if err != nil {
		return accessrequest.AccessRequest{}, Errorf(ErrKindInvalidArgument, "%s", err)
	}

	created, err := accessrequest.Create(createInput, s.clock, s.idGenerator)
	if err != nil {
		return accessrequest.AccessRequest{}, Errorf(ErrKindInvalidArgument, "%s", err)
	}

	if err := s.accessRequestStore.PutAccessRequest(ctx, created); err != nil {
		return accessrequest.AccessRequest{}, Wrapf(ErrKindInternal, err, "put access request")
	}
	if err := s.putAuditEvent(ctx, auditevent.Event{
		EventName:       auditevent.NameAccessRequestCreated,
		ActorUserID:     input.RequesterUserID,
		OwnerUserID:     created.OwnerUserID,
		RequesterUserID: created.RequesterUserID,
		AgentID:         created.AgentID,
		AccessRequestID: created.ID,
		Outcome:         string(created.Status),
		CreatedAt:       created.CreatedAt,
	}); err != nil {
		return accessrequest.AccessRequest{}, Wrapf(ErrKindInternal, err, "put audit event")
	}

	return created, nil
}

// ListAccessRequestRole selects the caller's perspective when listing access
// requests: requester-scoped or owner-scoped.
type ListAccessRequestRole int

const (
	// ListAccessRequestRoleRequester lists requests where the caller is the requester.
	ListAccessRequestRoleRequester ListAccessRequestRole = iota + 1
	// ListAccessRequestRoleOwner lists requests where the caller is the agent owner.
	ListAccessRequestRoleOwner
)

// List returns a page of access requests scoped by the caller's role.
func (s *AccessRequestService) List(ctx context.Context, userID string, role ListAccessRequestRole, pageSize int, pageToken string) (accessrequest.Page, error) {
	var (
		page accessrequest.Page
		err  error
	)
	switch role {
	case ListAccessRequestRoleRequester:
		page, err = s.accessRequestStore.ListAccessRequestsByRequester(ctx, userID, pageSize, pageToken)
	case ListAccessRequestRoleOwner:
		page, err = s.accessRequestStore.ListAccessRequestsByOwner(ctx, userID, pageSize, pageToken)
	default:
		return accessrequest.Page{}, Errorf(ErrKindInvalidArgument, "role is required")
	}
	if err != nil {
		return accessrequest.Page{}, Wrapf(ErrKindInternal, err, "list access requests")
	}
	return page, nil
}

// ListAuditEventsInput is the domain input for listing audit events.
type ListAuditEventsInput struct {
	OwnerUserID string
	PageSize    int
	PageToken   string
	Filter      auditevent.Filter
}

// ListAuditEvents returns a page of audit events scoped to the given owner.
func (s *AccessRequestService) ListAuditEvents(ctx context.Context, input ListAuditEventsInput) (auditevent.Page, error) {
	page, err := s.auditEventStore.ListAuditEventsByOwner(ctx, input.OwnerUserID, input.PageSize, input.PageToken, input.Filter)
	if err != nil {
		return auditevent.Page{}, Wrapf(ErrKindInternal, err, "list audit events")
	}
	return page, nil
}

// ReviewAccessRequestInput is the domain input for reviewing an access request.
type ReviewAccessRequestInput struct {
	OwnerUserID     string
	AccessRequestID string
	Decision        accessrequest.Decision
	ReviewNote      string
}

// Review applies one owner decision to a pending access request, persists the
// update, and writes an audit event.
func (s *AccessRequestService) Review(ctx context.Context, input ReviewAccessRequestInput) (accessrequest.AccessRequest, error) {
	if input.AccessRequestID == "" {
		return accessrequest.AccessRequest{}, Errorf(ErrKindInvalidArgument, "access_request_id is required")
	}

	existing, err := s.accessRequestStore.GetAccessRequest(ctx, input.AccessRequestID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return accessrequest.AccessRequest{}, Errorf(ErrKindNotFound, "access request not found")
		}
		return accessrequest.AccessRequest{}, Wrapf(ErrKindInternal, err, "get access request")
	}
	if existing.OwnerUserID != input.OwnerUserID {
		return accessrequest.AccessRequest{}, Errorf(ErrKindNotFound, "access request not found")
	}

	reviewed, err := accessrequest.Review(existing, accessrequest.ReviewInput{
		ID:             existing.ID,
		ReviewerUserID: input.OwnerUserID,
		Decision:       input.Decision,
		ReviewNote:     input.ReviewNote,
	}, s.clock)
	if err != nil {
		if errors.Is(err, accessrequest.ErrNotPending) {
			return accessrequest.AccessRequest{}, Errorf(ErrKindFailedPrecondition, "access request is already reviewed")
		}
		if errors.Is(err, accessrequest.ErrRevokerNotOwner) {
			return accessrequest.AccessRequest{}, Errorf(ErrKindNotFound, "access request not found")
		}
		return accessrequest.AccessRequest{}, Errorf(ErrKindInvalidArgument, "%s", err)
	}

	if reviewed.ReviewedAt == nil {
		return accessrequest.AccessRequest{}, Errorf(ErrKindInternal, "review timestamp is unavailable")
	}
	if err := s.accessRequestStore.ReviewAccessRequest(ctx, reviewed); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return accessrequest.AccessRequest{}, Errorf(ErrKindNotFound, "access request not found")
		}
		if errors.Is(err, storage.ErrConflict) {
			return accessrequest.AccessRequest{}, Errorf(ErrKindFailedPrecondition, "access request is already reviewed")
		}
		return accessrequest.AccessRequest{}, Wrapf(ErrKindInternal, err, "review access request")
	}

	if err := s.putAuditEvent(ctx, auditevent.Event{
		EventName:       auditevent.NameAccessRequestReviewed,
		ActorUserID:     input.OwnerUserID,
		OwnerUserID:     reviewed.OwnerUserID,
		RequesterUserID: reviewed.RequesterUserID,
		AgentID:         reviewed.AgentID,
		AccessRequestID: reviewed.ID,
		Outcome:         string(reviewed.Status),
		CreatedAt:       reviewed.UpdatedAt,
	}); err != nil {
		return accessrequest.AccessRequest{}, Wrapf(ErrKindInternal, err, "put audit event")
	}

	return reviewed, nil
}

// RevokeAccessRequestInput is the domain input for revoking an access request.
type RevokeAccessRequestInput struct {
	OwnerUserID     string
	AccessRequestID string
	RevokeNote      string
}

// Revoke removes delegated access for one approved request, persists the update,
// and writes an audit event.
func (s *AccessRequestService) Revoke(ctx context.Context, input RevokeAccessRequestInput) (accessrequest.AccessRequest, error) {
	if input.AccessRequestID == "" {
		return accessrequest.AccessRequest{}, Errorf(ErrKindInvalidArgument, "access_request_id is required")
	}

	existing, err := s.accessRequestStore.GetAccessRequest(ctx, input.AccessRequestID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return accessrequest.AccessRequest{}, Errorf(ErrKindNotFound, "access request not found")
		}
		return accessrequest.AccessRequest{}, Wrapf(ErrKindInternal, err, "get access request")
	}
	if existing.OwnerUserID != input.OwnerUserID {
		return accessrequest.AccessRequest{}, Errorf(ErrKindNotFound, "access request not found")
	}

	revoked, err := accessrequest.Revoke(existing, accessrequest.RevokeInput{
		ID:            existing.ID,
		RevokerUserID: input.OwnerUserID,
		RevokeNote:    input.RevokeNote,
	}, s.clock)
	if err != nil {
		if errors.Is(err, accessrequest.ErrNotApproved) {
			return accessrequest.AccessRequest{}, Errorf(ErrKindFailedPrecondition, "access request is not approved")
		}
		if errors.Is(err, accessrequest.ErrReviewerNotOwner) {
			return accessrequest.AccessRequest{}, Errorf(ErrKindNotFound, "access request not found")
		}
		return accessrequest.AccessRequest{}, Errorf(ErrKindInvalidArgument, "%s", err)
	}
	if err := s.accessRequestStore.RevokeAccessRequest(ctx, revoked); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return accessrequest.AccessRequest{}, Errorf(ErrKindNotFound, "access request not found")
		}
		if errors.Is(err, storage.ErrConflict) {
			return accessrequest.AccessRequest{}, Errorf(ErrKindFailedPrecondition, "access request is not approved")
		}
		return accessrequest.AccessRequest{}, Wrapf(ErrKindInternal, err, "revoke access request")
	}

	if err := s.putAuditEvent(ctx, auditevent.Event{
		EventName:       auditevent.NameAccessRequestRevoked,
		ActorUserID:     input.OwnerUserID,
		OwnerUserID:     revoked.OwnerUserID,
		RequesterUserID: revoked.RequesterUserID,
		AgentID:         revoked.AgentID,
		AccessRequestID: revoked.ID,
		Outcome:         string(revoked.Status),
		CreatedAt:       revoked.UpdatedAt,
	}); err != nil {
		return accessrequest.AccessRequest{}, Wrapf(ErrKindInternal, err, "put audit event")
	}

	return revoked, nil
}

// putAuditEvent persists one audit event record.
func (s *AccessRequestService) putAuditEvent(ctx context.Context, record auditevent.Event) error {
	record.CreatedAt = record.CreatedAt.UTC()
	return s.auditEventStore.PutAuditEvent(ctx, record)
}
