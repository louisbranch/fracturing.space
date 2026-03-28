package ai

import (
	"context"
	"fmt"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/auditevent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AccessRequestHandlers serves access-request and audit-event RPCs as thin
// transport wrappers over the access-request service.
type AccessRequestHandlers struct {
	aiv1.UnimplementedAccessRequestServiceServer
	svc *service.AccessRequestService
}

// AccessRequestHandlersConfig declares the dependencies for access-request RPCs.
type AccessRequestHandlersConfig struct {
	AccessRequestService *service.AccessRequestService
}

// NewAccessRequestHandlers builds an access-request RPC server from a service.
func NewAccessRequestHandlers(cfg AccessRequestHandlersConfig) (*AccessRequestHandlers, error) {
	if cfg.AccessRequestService == nil {
		return nil, fmt.Errorf("ai: NewAccessRequestHandlers: access request service is required")
	}
	return &AccessRequestHandlers{svc: cfg.AccessRequestService}, nil
}

// CreateAccessRequest creates a pending access request for an agent.
func (h *AccessRequestHandlers) CreateAccessRequest(ctx context.Context, in *aiv1.CreateAccessRequestRequest) (*aiv1.CreateAccessRequestResponse, error) {
	userID, err := requireUserScopedUnaryRequest(ctx, in, "create access request is required")
	if err != nil {
		return nil, err
	}

	record, err := h.svc.Create(ctx, service.CreateAccessRequestInput{
		RequesterUserID: userID,
		AgentID:         in.GetAgentId(),
		Scope:           in.GetScope(),
		RequestNote:     in.GetRequestNote(),
	})
	if err != nil {
		return nil, serviceErrorToStatus(err)
	}
	return &aiv1.CreateAccessRequestResponse{AccessRequest: accessRequestToProto(record)}, nil
}

// ListAccessRequests returns one role-scoped page of access requests.
func (h *AccessRequestHandlers) ListAccessRequests(ctx context.Context, in *aiv1.ListAccessRequestsRequest) (*aiv1.ListAccessRequestsResponse, error) {
	userID, err := requireUserScopedUnaryRequest(ctx, in, "list access requests request is required")
	if err != nil {
		return nil, err
	}

	role, err := listAccessRequestRoleFromProto(in.GetRole())
	if err != nil {
		return nil, err
	}

	page, err := h.svc.List(ctx, userID, role, clampPageSize(in.GetPageSize()), in.GetPageToken())
	if err != nil {
		return nil, serviceErrorToStatus(err)
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
func (h *AccessRequestHandlers) ListAuditEvents(ctx context.Context, in *aiv1.ListAuditEventsRequest) (*aiv1.ListAuditEventsResponse, error) {
	userID, err := requireUserScopedUnaryRequest(ctx, in, "list audit events request is required")
	if err != nil {
		return nil, err
	}

	filter, err := listAuditEventFilterFromRequest(in)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	page, err := h.svc.ListAuditEvents(ctx, service.ListAuditEventsInput{
		OwnerUserID: userID,
		PageSize:    clampPageSize(in.GetPageSize()),
		PageToken:   in.GetPageToken(),
		Filter:      filter,
	})
	if err != nil {
		return nil, serviceErrorToStatus(err)
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
func (h *AccessRequestHandlers) ReviewAccessRequest(ctx context.Context, in *aiv1.ReviewAccessRequestRequest) (*aiv1.ReviewAccessRequestResponse, error) {
	userID, err := requireUserScopedUnaryRequest(ctx, in, "review access request is required")
	if err != nil {
		return nil, err
	}

	decision, err := accessRequestDecisionFromProto(in.GetDecision())
	if err != nil {
		return nil, err
	}

	record, err := h.svc.Review(ctx, service.ReviewAccessRequestInput{
		OwnerUserID:     userID,
		AccessRequestID: strings.TrimSpace(in.GetAccessRequestId()),
		Decision:        decision,
		ReviewNote:      in.GetReviewNote(),
	})
	if err != nil {
		return nil, serviceErrorToStatus(err)
	}
	return &aiv1.ReviewAccessRequestResponse{AccessRequest: accessRequestToProto(record)}, nil
}

// RevokeAccessRequest removes delegated access for one approved request.
func (h *AccessRequestHandlers) RevokeAccessRequest(ctx context.Context, in *aiv1.RevokeAccessRequestRequest) (*aiv1.RevokeAccessRequestResponse, error) {
	userID, err := requireUserScopedUnaryRequest(ctx, in, "revoke access request is required")
	if err != nil {
		return nil, err
	}

	record, err := h.svc.Revoke(ctx, service.RevokeAccessRequestInput{
		OwnerUserID:     userID,
		AccessRequestID: strings.TrimSpace(in.GetAccessRequestId()),
		RevokeNote:      in.GetRevokeNote(),
	})
	if err != nil {
		return nil, serviceErrorToStatus(err)
	}
	return &aiv1.RevokeAccessRequestResponse{AccessRequest: accessRequestToProto(record)}, nil
}

// listAccessRequestRoleFromProto converts a proto role to the service-layer enum.
func listAccessRequestRoleFromProto(role aiv1.AccessRequestRole) (service.ListAccessRequestRole, error) {
	switch role {
	case aiv1.AccessRequestRole_ACCESS_REQUEST_ROLE_REQUESTER:
		return service.ListAccessRequestRoleRequester, nil
	case aiv1.AccessRequestRole_ACCESS_REQUEST_ROLE_OWNER:
		return service.ListAccessRequestRoleOwner, nil
	default:
		return 0, status.Error(codes.InvalidArgument, "role is required")
	}
}

func listAuditEventFilterFromRequest(in *aiv1.ListAuditEventsRequest) (auditevent.Filter, error) {
	filter := auditevent.Filter{
		EventName: auditevent.Name(strings.TrimSpace(in.GetEventName())),
		AgentID:   strings.TrimSpace(in.GetAgentId()),
	}
	if in.GetCreatedAfter() != nil {
		if err := in.GetCreatedAfter().CheckValid(); err != nil {
			return auditevent.Filter{}, fmt.Errorf("created_after is invalid")
		}
		createdAfter := in.GetCreatedAfter().AsTime().UTC()
		filter.CreatedAfter = &createdAfter
	}
	if in.GetCreatedBefore() != nil {
		if err := in.GetCreatedBefore().CheckValid(); err != nil {
			return auditevent.Filter{}, fmt.Errorf("created_before is invalid")
		}
		createdBefore := in.GetCreatedBefore().AsTime().UTC()
		filter.CreatedBefore = &createdBefore
	}
	if filter.CreatedAfter != nil && filter.CreatedBefore != nil && filter.CreatedAfter.After(*filter.CreatedBefore) {
		return auditevent.Filter{}, fmt.Errorf("created_after must be before or equal to created_before")
	}
	return filter, nil
}
