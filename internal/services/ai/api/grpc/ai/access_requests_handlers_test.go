package ai

import (
	"context"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/accessrequest"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestCreateAccessRequestSuccess(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 1, 5, 0, 0, time.UTC)
	store.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "owner-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now.Add(-time.Hour),
		UpdatedAt:     now.Add(-time.Hour),
	}

	th := newAccessRequestHandlersWithStores(t, store, store, store)
	th.svc.SetClock(func() time.Time { return now })
	th.svc.SetIDGenerator(func() (string, error) { return "request-1", nil })

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := th.CreateAccessRequest(ctx, &aiv1.CreateAccessRequestRequest{
		AgentId:     "agent-1",
		Scope:       "invoke",
		RequestNote: "please allow",
	})
	if err != nil {
		t.Fatalf("create access request: %v", err)
	}
	if got := resp.GetAccessRequest().GetId(); got != "request-1" {
		t.Fatalf("id = %q, want %q", got, "request-1")
	}
	if got := resp.GetAccessRequest().GetStatus(); got != aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_PENDING {
		t.Fatalf("status = %v, want pending", got)
	}
}

func TestCreateAccessRequestWritesAuditEvent(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 1, 5, 0, 0, time.UTC)
	store.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "owner-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now.Add(-time.Hour),
		UpdatedAt:     now.Add(-time.Hour),
	}

	th := newAccessRequestHandlersWithStores(t, store, store, store)
	th.svc.SetClock(func() time.Time { return now })
	th.svc.SetIDGenerator(func() (string, error) { return "request-1", nil })

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	if _, err := th.CreateAccessRequest(ctx, &aiv1.CreateAccessRequestRequest{
		AgentId:     "agent-1",
		Scope:       "invoke",
		RequestNote: "please allow",
	}); err != nil {
		t.Fatalf("create access request: %v", err)
	}
	if len(store.AuditEventNames) != 1 {
		t.Fatalf("audit events len = %d, want 1", len(store.AuditEventNames))
	}
	if got := store.AuditEventNames[0]; got != "access_request.created" {
		t.Fatalf("audit event = %q, want %q", got, "access_request.created")
	}
}

func TestCreateAccessRequestRejectsOwnAgent(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	th := newAccessRequestHandlersWithStores(t, store, store, store)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := th.CreateAccessRequest(ctx, &aiv1.CreateAccessRequestRequest{
		AgentId: "agent-1",
		Scope:   "invoke",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateAccessRequestRejectsUnsupportedScope(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "owner-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	th := newAccessRequestHandlersWithStores(t, store, store, store)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := th.CreateAccessRequest(ctx, &aiv1.CreateAccessRequestRequest{
		AgentId: "agent-1",
		Scope:   "admin",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListAccessRequestsByRole(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.AccessRequests["request-1"] = accessrequest.AccessRequest{ID: "request-1", RequesterUserID: "user-1", OwnerUserID: "owner-1", AgentID: "agent-1", Scope: accessrequest.ScopeInvoke, Status: accessrequest.StatusPending, CreatedAt: now, UpdatedAt: now}
	store.AccessRequests["request-2"] = accessrequest.AccessRequest{ID: "request-2", RequesterUserID: "user-1", OwnerUserID: "owner-2", AgentID: "agent-2", Scope: accessrequest.ScopeInvoke, Status: accessrequest.StatusPending, CreatedAt: now, UpdatedAt: now}
	store.AccessRequests["request-3"] = accessrequest.AccessRequest{ID: "request-3", RequesterUserID: "user-3", OwnerUserID: "owner-1", AgentID: "agent-3", Scope: accessrequest.ScopeInvoke, Status: accessrequest.StatusApproved, CreatedAt: now, UpdatedAt: now}

	th := newAccessRequestHandlersWithStores(t, store, store, store)

	requesterCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	requesterResp, err := th.ListAccessRequests(requesterCtx, &aiv1.ListAccessRequestsRequest{
		Role: aiv1.AccessRequestRole_ACCESS_REQUEST_ROLE_REQUESTER,
	})
	if err != nil {
		t.Fatalf("list requester access requests: %v", err)
	}
	if len(requesterResp.GetAccessRequests()) != 2 {
		t.Fatalf("requester len = %d, want 2", len(requesterResp.GetAccessRequests()))
	}

	ownerCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))
	ownerResp, err := th.ListAccessRequests(ownerCtx, &aiv1.ListAccessRequestsRequest{
		Role: aiv1.AccessRequestRole_ACCESS_REQUEST_ROLE_OWNER,
	})
	if err != nil {
		t.Fatalf("list owner access requests: %v", err)
	}
	if len(ownerResp.GetAccessRequests()) != 2 {
		t.Fatalf("owner len = %d, want 2", len(ownerResp.GetAccessRequests()))
	}
}

func TestReviewAccessRequestByOwner(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 1, 7, 0, 0, time.UTC)
	store.AccessRequests["request-1"] = accessrequest.AccessRequest{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-1",
		Scope:           accessrequest.ScopeInvoke,
		Status:          accessrequest.StatusPending,
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}

	th := newAccessRequestHandlersWithStores(t, store, store, store)
	th.svc.SetClock(func() time.Time { return now })
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))
	resp, err := th.ReviewAccessRequest(ctx, &aiv1.ReviewAccessRequestRequest{
		AccessRequestId: "request-1",
		Decision:        aiv1.AccessRequestDecision_ACCESS_REQUEST_DECISION_APPROVE,
		ReviewNote:      "approved",
	})
	if err != nil {
		t.Fatalf("review access request: %v", err)
	}
	if got := resp.GetAccessRequest().GetStatus(); got != aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_APPROVED {
		t.Fatalf("status = %v, want approved", got)
	}
}

func TestReviewAccessRequestWritesAuditEvent(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 1, 7, 0, 0, time.UTC)
	store.AccessRequests["request-1"] = accessrequest.AccessRequest{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-1",
		Scope:           accessrequest.ScopeInvoke,
		Status:          accessrequest.StatusPending,
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}

	th := newAccessRequestHandlersWithStores(t, store, store, store)
	th.svc.SetClock(func() time.Time { return now })
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))
	if _, err := th.ReviewAccessRequest(ctx, &aiv1.ReviewAccessRequestRequest{
		AccessRequestId: "request-1",
		Decision:        aiv1.AccessRequestDecision_ACCESS_REQUEST_DECISION_APPROVE,
		ReviewNote:      "approved",
	}); err != nil {
		t.Fatalf("review access request: %v", err)
	}
	if len(store.AuditEventNames) != 1 {
		t.Fatalf("audit events len = %d, want 1", len(store.AuditEventNames))
	}
	if got := store.AuditEventNames[0]; got != "access_request.reviewed" {
		t.Fatalf("audit event = %q, want %q", got, "access_request.reviewed")
	}
}

func TestReviewAccessRequestRejectsNonOwner(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.AccessRequests["request-1"] = accessrequest.AccessRequest{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-1",
		Scope:           accessrequest.ScopeInvoke,
		Status:          accessrequest.StatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	th := newAccessRequestHandlersWithStores(t, store, store, store)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-2"))
	_, err := th.ReviewAccessRequest(ctx, &aiv1.ReviewAccessRequestRequest{
		AccessRequestId: "request-1",
		Decision:        aiv1.AccessRequestDecision_ACCESS_REQUEST_DECISION_DENY,
		ReviewNote:      "no",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestRevokeAccessRequestByOwner(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 1, 9, 0, 0, time.UTC)
	store.AccessRequests["request-1"] = accessrequest.AccessRequest{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-1",
		Scope:           accessrequest.ScopeInvoke,
		Status:          accessrequest.StatusApproved,
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Minute),
		ReviewerUserID:  "owner-1",
		ReviewNote:      "approved",
		ReviewedAt:      ptrTime(now.Add(-time.Minute)),
	}

	th := newAccessRequestHandlersWithStores(t, store, store, store)
	th.svc.SetClock(func() time.Time { return now })
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))
	resp, err := th.RevokeAccessRequest(ctx, &aiv1.RevokeAccessRequestRequest{
		AccessRequestId: "request-1",
		RevokeNote:      "removed",
	})
	if err != nil {
		t.Fatalf("revoke access request: %v", err)
	}
	if got := resp.GetAccessRequest().GetStatus(); got != aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_REVOKED {
		t.Fatalf("status = %v, want revoked", got)
	}
	if len(store.AuditEventNames) != 1 {
		t.Fatalf("audit events len = %d, want 1", len(store.AuditEventNames))
	}
	if got := store.AuditEventNames[0]; got != "access_request.revoked" {
		t.Fatalf("audit event = %q, want %q", got, "access_request.revoked")
	}
}

func TestRevokeAccessRequestRejectsNonOwner(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.AccessRequests["request-1"] = accessrequest.AccessRequest{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-1",
		Scope:           accessrequest.ScopeInvoke,
		Status:          accessrequest.StatusApproved,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	th := newAccessRequestHandlersWithStores(t, store, store, store)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-2"))
	_, err := th.RevokeAccessRequest(ctx, &aiv1.RevokeAccessRequestRequest{
		AccessRequestId: "request-1",
		RevokeNote:      "no",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestRevokeAccessRequestRejectsNonApproved(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.AccessRequests["request-1"] = accessrequest.AccessRequest{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-1",
		Scope:           accessrequest.ScopeInvoke,
		Status:          accessrequest.StatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	th := newAccessRequestHandlersWithStores(t, store, store, store)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))
	_, err := th.RevokeAccessRequest(ctx, &aiv1.RevokeAccessRequestRequest{
		AccessRequestId: "request-1",
		RevokeNote:      "not approved",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}
