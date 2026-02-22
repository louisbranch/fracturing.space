package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/accessrequest"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateAccessRequest stores a requester-owned pending access request for an agent.
func (s *Service) CreateAccessRequest(ctx context.Context, in *aiv1.CreateAccessRequestRequest) (*aiv1.CreateAccessRequestResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create access request is required")
	}
	if s.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}
	if s.accessRequestStore == nil {
		return nil, status.Error(codes.Internal, "access request store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	agentID := strings.TrimSpace(in.GetAgentId())
	agentRecord, err := s.agentStore.GetAgent(ctx, agentID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			// Return generic unavailability so callers cannot infer resource ownership.
			return nil, status.Error(codes.FailedPrecondition, "agent is unavailable")
		}
		return nil, status.Errorf(codes.Internal, "get agent: %v", err)
	}
	if !strings.EqualFold(strings.TrimSpace(agentRecord.Status), "active") {
		return nil, status.Error(codes.FailedPrecondition, "agent is unavailable")
	}

	createInput, err := accessrequest.NormalizeCreateInput(accessrequest.CreateInput{
		RequesterUserID: userID,
		OwnerUserID:     agentRecord.OwnerUserID,
		AgentID:         agentRecord.ID,
		Scope:           accessrequest.Scope(in.GetScope()),
		RequestNote:     in.GetRequestNote(),
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	created, err := accessrequest.Create(createInput, s.clock, s.idGenerator)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	record := storage.AccessRequestRecord{
		ID:              created.ID,
		RequesterUserID: created.RequesterUserID,
		OwnerUserID:     created.OwnerUserID,
		AgentID:         created.AgentID,
		Scope:           string(created.Scope),
		RequestNote:     created.RequestNote,
		Status:          string(created.Status),
		ReviewerUserID:  created.ReviewerUserID,
		ReviewNote:      created.ReviewNote,
		CreatedAt:       created.CreatedAt,
		UpdatedAt:       created.UpdatedAt,
		ReviewedAt:      created.ReviewedAt,
	}
	if err := s.accessRequestStore.PutAccessRequest(ctx, record); err != nil {
		return nil, status.Errorf(codes.Internal, "put access request: %v", err)
	}
	if err := s.putAuditEvent(ctx, storage.AuditEventRecord{
		EventName:       "access_request.created",
		ActorUserID:     userID,
		OwnerUserID:     record.OwnerUserID,
		RequesterUserID: record.RequesterUserID,
		AgentID:         record.AgentID,
		AccessRequestID: record.ID,
		Outcome:         record.Status,
		CreatedAt:       record.CreatedAt,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "put audit event: %v", err)
	}

	return &aiv1.CreateAccessRequestResponse{AccessRequest: accessRequestToProto(record)}, nil
}

// ListAccessRequests returns one role-scoped page of access requests.
func (s *Service) ListAccessRequests(ctx context.Context, in *aiv1.ListAccessRequestsRequest) (*aiv1.ListAccessRequestsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list access requests request is required")
	}
	if s.accessRequestStore == nil {
		return nil, status.Error(codes.Internal, "access request store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	var (
		page storage.AccessRequestPage
		err  error
	)
	switch in.GetRole() {
	case aiv1.AccessRequestRole_ACCESS_REQUEST_ROLE_REQUESTER:
		page, err = s.accessRequestStore.ListAccessRequestsByRequester(ctx, userID, clampPageSize(in.GetPageSize()), in.GetPageToken())
	case aiv1.AccessRequestRole_ACCESS_REQUEST_ROLE_OWNER:
		page, err = s.accessRequestStore.ListAccessRequestsByOwner(ctx, userID, clampPageSize(in.GetPageSize()), in.GetPageToken())
	default:
		return nil, status.Error(codes.InvalidArgument, "role is required")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list access requests: %v", err)
	}

	resp := &aiv1.ListAccessRequestsResponse{
		NextPageToken:  page.NextPageToken,
		AccessRequests: make([]*aiv1.AccessRequest, 0, len(page.AccessRequests)),
	}
	for _, record := range page.AccessRequests {
		resp.AccessRequests = append(resp.AccessRequests, accessRequestToProto(record))
	}
	return resp, nil
}

// ListAuditEvents returns one owner-scoped page of AI audit events.
func (s *Service) ListAuditEvents(ctx context.Context, in *aiv1.ListAuditEventsRequest) (*aiv1.ListAuditEventsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list audit events request is required")
	}
	if s.auditEventStore == nil {
		return nil, status.Error(codes.Internal, "audit event store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	// Filters are caller-supplied and only narrow rows within this authenticated
	// owner scope. Ownership comes from trusted auth metadata, never from input.
	filter, err := listAuditEventFilterFromRequest(in)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	page, err := s.auditEventStore.ListAuditEventsByOwner(ctx, userID, clampPageSize(in.GetPageSize()), in.GetPageToken(), filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list audit events: %v", err)
	}
	resp := &aiv1.ListAuditEventsResponse{
		NextPageToken: page.NextPageToken,
		AuditEvents:   make([]*aiv1.AuditEvent, 0, len(page.AuditEvents)),
	}
	for _, record := range page.AuditEvents {
		resp.AuditEvents = append(resp.AuditEvents, auditEventToProto(record))
	}
	return resp, nil
}

// ReviewAccessRequest applies one owner decision to a pending access request.
func (s *Service) ReviewAccessRequest(ctx context.Context, in *aiv1.ReviewAccessRequestRequest) (*aiv1.ReviewAccessRequestResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "review access request is required")
	}
	if s.accessRequestStore == nil {
		return nil, status.Error(codes.Internal, "access request store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	accessRequestID := strings.TrimSpace(in.GetAccessRequestId())
	if accessRequestID == "" {
		return nil, status.Error(codes.InvalidArgument, "access_request_id is required")
	}

	existing, err := s.accessRequestStore.GetAccessRequest(ctx, accessRequestID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "access request not found")
		}
		return nil, status.Errorf(codes.Internal, "get access request: %v", err)
	}
	if strings.TrimSpace(existing.OwnerUserID) != userID {
		// Hide unauthorized resources to avoid cross-tenant enumeration.
		return nil, status.Error(codes.NotFound, "access request not found")
	}

	decision, err := accessRequestDecisionFromProto(in.GetDecision())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	updatedDomain, err := accessrequest.Review(accessrequest.AccessRequest{
		ID:              existing.ID,
		RequesterUserID: existing.RequesterUserID,
		OwnerUserID:     existing.OwnerUserID,
		AgentID:         existing.AgentID,
		Scope:           accessrequest.Scope(existing.Scope),
		RequestNote:     existing.RequestNote,
		Status:          accessrequest.Status(existing.Status),
		ReviewerUserID:  existing.ReviewerUserID,
		ReviewNote:      existing.ReviewNote,
		CreatedAt:       existing.CreatedAt,
		UpdatedAt:       existing.UpdatedAt,
		ReviewedAt:      existing.ReviewedAt,
	}, accessrequest.ReviewInput{
		ID:             existing.ID,
		ReviewerUserID: userID,
		Decision:       decision,
		ReviewNote:     in.GetReviewNote(),
	}, s.clock)
	if err != nil {
		if errors.Is(err, accessrequest.ErrNotPending) {
			return nil, status.Error(codes.FailedPrecondition, "access request is already reviewed")
		}
		if errors.Is(err, accessrequest.ErrReviewerNotOwner) {
			return nil, status.Error(codes.NotFound, "access request not found")
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if updatedDomain.ReviewedAt == nil {
		return nil, status.Error(codes.Internal, "review timestamp is unavailable")
	}
	if err := s.accessRequestStore.ReviewAccessRequest(
		ctx,
		userID,
		existing.ID,
		string(updatedDomain.Status),
		updatedDomain.ReviewerUserID,
		updatedDomain.ReviewNote,
		*updatedDomain.ReviewedAt,
	); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "access request not found")
		}
		if errors.Is(err, storage.ErrConflict) {
			return nil, status.Error(codes.FailedPrecondition, "access request is already reviewed")
		}
		return nil, status.Errorf(codes.Internal, "review access request: %v", err)
	}

	updatedRecord, err := s.accessRequestStore.GetAccessRequest(ctx, existing.ID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "access request not found")
		}
		return nil, status.Errorf(codes.Internal, "get access request: %v", err)
	}
	if err := s.putAuditEvent(ctx, storage.AuditEventRecord{
		EventName:       "access_request.reviewed",
		ActorUserID:     userID,
		OwnerUserID:     updatedRecord.OwnerUserID,
		RequesterUserID: updatedRecord.RequesterUserID,
		AgentID:         updatedRecord.AgentID,
		AccessRequestID: updatedRecord.ID,
		Outcome:         updatedRecord.Status,
		CreatedAt:       updatedRecord.UpdatedAt,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "put audit event: %v", err)
	}
	return &aiv1.ReviewAccessRequestResponse{AccessRequest: accessRequestToProto(updatedRecord)}, nil
}

// RevokeAccessRequest removes delegated access for one approved request.
func (s *Service) RevokeAccessRequest(ctx context.Context, in *aiv1.RevokeAccessRequestRequest) (*aiv1.RevokeAccessRequestResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "revoke access request is required")
	}
	if s.accessRequestStore == nil {
		return nil, status.Error(codes.Internal, "access request store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}
	accessRequestID := strings.TrimSpace(in.GetAccessRequestId())
	if accessRequestID == "" {
		return nil, status.Error(codes.InvalidArgument, "access_request_id is required")
	}

	existing, err := s.accessRequestStore.GetAccessRequest(ctx, accessRequestID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "access request not found")
		}
		return nil, status.Errorf(codes.Internal, "get access request: %v", err)
	}
	if strings.TrimSpace(existing.OwnerUserID) != userID {
		return nil, status.Error(codes.NotFound, "access request not found")
	}

	updatedDomain, err := accessrequest.Revoke(accessrequest.AccessRequest{
		ID:              existing.ID,
		RequesterUserID: existing.RequesterUserID,
		OwnerUserID:     existing.OwnerUserID,
		AgentID:         existing.AgentID,
		Scope:           accessrequest.Scope(existing.Scope),
		RequestNote:     existing.RequestNote,
		Status:          accessrequest.Status(existing.Status),
		ReviewerUserID:  existing.ReviewerUserID,
		ReviewNote:      existing.ReviewNote,
		CreatedAt:       existing.CreatedAt,
		UpdatedAt:       existing.UpdatedAt,
		ReviewedAt:      existing.ReviewedAt,
	}, accessrequest.RevokeInput{
		ID:            existing.ID,
		RevokerUserID: userID,
		RevokeNote:    in.GetRevokeNote(),
	}, s.clock)
	if err != nil {
		if errors.Is(err, accessrequest.ErrNotApproved) {
			return nil, status.Error(codes.FailedPrecondition, "access request is not approved")
		}
		if errors.Is(err, accessrequest.ErrReviewerNotOwner) {
			return nil, status.Error(codes.NotFound, "access request not found")
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.accessRequestStore.RevokeAccessRequest(
		ctx,
		userID,
		existing.ID,
		string(updatedDomain.Status),
		updatedDomain.ReviewerUserID,
		updatedDomain.ReviewNote,
		updatedDomain.UpdatedAt,
	); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "access request not found")
		}
		if errors.Is(err, storage.ErrConflict) {
			return nil, status.Error(codes.FailedPrecondition, "access request is not approved")
		}
		return nil, status.Errorf(codes.Internal, "revoke access request: %v", err)
	}

	updatedRecord, err := s.accessRequestStore.GetAccessRequest(ctx, existing.ID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "access request not found")
		}
		return nil, status.Errorf(codes.Internal, "get access request: %v", err)
	}
	if err := s.putAuditEvent(ctx, storage.AuditEventRecord{
		EventName:       "access_request.revoked",
		ActorUserID:     userID,
		OwnerUserID:     updatedRecord.OwnerUserID,
		RequesterUserID: updatedRecord.RequesterUserID,
		AgentID:         updatedRecord.AgentID,
		AccessRequestID: updatedRecord.ID,
		Outcome:         updatedRecord.Status,
		CreatedAt:       updatedRecord.UpdatedAt,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "put audit event: %v", err)
	}
	return &aiv1.RevokeAccessRequestResponse{AccessRequest: accessRequestToProto(updatedRecord)}, nil
}
func (s *Service) putAuditEvent(ctx context.Context, record storage.AuditEventRecord) error {
	if s.auditEventStore == nil {
		return fmt.Errorf("audit event store is not configured")
	}
	record.CreatedAt = record.CreatedAt.UTC()
	return s.auditEventStore.PutAuditEvent(ctx, record)
}
func listAuditEventFilterFromRequest(in *aiv1.ListAuditEventsRequest) (storage.AuditEventFilter, error) {
	filter := storage.AuditEventFilter{
		EventName: strings.TrimSpace(in.GetEventName()),
		AgentID:   strings.TrimSpace(in.GetAgentId()),
	}
	if in.GetCreatedAfter() != nil {
		if err := in.GetCreatedAfter().CheckValid(); err != nil {
			return storage.AuditEventFilter{}, fmt.Errorf("created_after is invalid")
		}
		createdAfter := in.GetCreatedAfter().AsTime().UTC()
		filter.CreatedAfter = &createdAfter
	}
	if in.GetCreatedBefore() != nil {
		if err := in.GetCreatedBefore().CheckValid(); err != nil {
			return storage.AuditEventFilter{}, fmt.Errorf("created_before is invalid")
		}
		createdBefore := in.GetCreatedBefore().AsTime().UTC()
		filter.CreatedBefore = &createdBefore
	}
	if filter.CreatedAfter != nil && filter.CreatedBefore != nil && filter.CreatedAfter.After(*filter.CreatedBefore) {
		return storage.AuditEventFilter{}, fmt.Errorf("created_after must be before or equal to created_before")
	}
	return filter, nil
}
