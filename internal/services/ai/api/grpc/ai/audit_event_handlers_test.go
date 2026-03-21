package ai

import (
	"context"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestListAuditEventsRequiresUserID(t *testing.T) {
	th := newAccessRequestHandlersWithStores(t, newFakeStore(), newFakeStore(), newFakeStore())
	_, err := th.ListAuditEvents(context.Background(), &aiv1.ListAuditEventsRequest{PageSize: 10})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListAuditEventsOwnerScoped(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 3, 10, 0, 0, time.UTC)
	store.AuditEvents = []storage.AuditEventRecord{
		{ID: "1", EventName: "access_request.created", ActorUserID: "user-1", OwnerUserID: "owner-1", RequesterUserID: "user-1", AgentID: "agent-1", AccessRequestID: "request-1", Outcome: "pending", CreatedAt: now},
		{ID: "2", EventName: "access_request.reviewed", ActorUserID: "owner-1", OwnerUserID: "owner-1", RequesterUserID: "user-1", AgentID: "agent-1", AccessRequestID: "request-1", Outcome: "approved", CreatedAt: now.Add(time.Minute)},
		{ID: "3", EventName: "access_request.created", ActorUserID: "user-2", OwnerUserID: "owner-2", RequesterUserID: "user-2", AgentID: "agent-2", AccessRequestID: "request-2", Outcome: "pending", CreatedAt: now},
	}

	th := newAccessRequestHandlersWithStores(t, store, store, store)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))
	resp, err := th.ListAuditEvents(ctx, &aiv1.ListAuditEventsRequest{PageSize: 10})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(resp.GetAuditEvents()) != 2 {
		t.Fatalf("events len = %d, want 2", len(resp.GetAuditEvents()))
	}
	if got := resp.GetAuditEvents()[0].GetId(); got != "1" {
		t.Fatalf("event[0].id = %q, want %q", got, "1")
	}
	if got := resp.GetAuditEvents()[1].GetId(); got != "2" {
		t.Fatalf("event[1].id = %q, want %q", got, "2")
	}
}

func TestListAuditEventsPaginates(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 3, 10, 0, 0, time.UTC)
	store.AuditEvents = []storage.AuditEventRecord{
		{ID: "1", EventName: "access_request.created", ActorUserID: "user-1", OwnerUserID: "owner-1", RequesterUserID: "user-1", AgentID: "agent-1", AccessRequestID: "request-1", Outcome: "pending", CreatedAt: now},
		{ID: "2", EventName: "access_request.reviewed", ActorUserID: "owner-1", OwnerUserID: "owner-1", RequesterUserID: "user-1", AgentID: "agent-1", AccessRequestID: "request-1", Outcome: "approved", CreatedAt: now.Add(time.Minute)},
		{ID: "3", EventName: "access_request.revoked", ActorUserID: "owner-1", OwnerUserID: "owner-1", RequesterUserID: "user-1", AgentID: "agent-1", AccessRequestID: "request-1", Outcome: "revoked", CreatedAt: now.Add(2 * time.Minute)},
	}

	th := newAccessRequestHandlersWithStores(t, store, store, store)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))

	first, err := th.ListAuditEvents(ctx, &aiv1.ListAuditEventsRequest{PageSize: 2})
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(first.GetAuditEvents()) != 2 {
		t.Fatalf("first page len = %d, want 2", len(first.GetAuditEvents()))
	}
	if got := first.GetNextPageToken(); got != "2" {
		t.Fatalf("first next token = %q, want %q", got, "2")
	}

	second, err := th.ListAuditEvents(ctx, &aiv1.ListAuditEventsRequest{
		PageSize:  2,
		PageToken: first.GetNextPageToken(),
	})
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(second.GetAuditEvents()) != 1 {
		t.Fatalf("second page len = %d, want 1", len(second.GetAuditEvents()))
	}
	if got := second.GetAuditEvents()[0].GetId(); got != "3" {
		t.Fatalf("second page id = %q, want %q", got, "3")
	}
	if got := second.GetNextPageToken(); got != "" {
		t.Fatalf("second next token = %q, want empty", got)
	}
}

func TestListAuditEventsFiltersByEventName(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 3, 20, 0, 0, time.UTC)
	store.AuditEvents = []storage.AuditEventRecord{
		{ID: "1", EventName: "access_request.created", OwnerUserID: "owner-1", AgentID: "agent-1", CreatedAt: now},
		{ID: "2", EventName: "access_request.reviewed", OwnerUserID: "owner-1", AgentID: "agent-1", CreatedAt: now.Add(time.Minute)},
	}

	th := newAccessRequestHandlersWithStores(t, store, store, store)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))
	resp, err := th.ListAuditEvents(ctx, &aiv1.ListAuditEventsRequest{
		PageSize:  10,
		EventName: "access_request.reviewed",
	})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(resp.GetAuditEvents()) != 1 {
		t.Fatalf("events len = %d, want 1", len(resp.GetAuditEvents()))
	}
	if got := resp.GetAuditEvents()[0].GetId(); got != "2" {
		t.Fatalf("event id = %q, want %q", got, "2")
	}
}

func TestListAuditEventsFiltersByAgentID(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 3, 20, 0, 0, time.UTC)
	store.AuditEvents = []storage.AuditEventRecord{
		{ID: "1", EventName: "access_request.created", OwnerUserID: "owner-1", AgentID: "agent-1", CreatedAt: now},
		{ID: "2", EventName: "access_request.reviewed", OwnerUserID: "owner-1", AgentID: "agent-2", CreatedAt: now.Add(time.Minute)},
	}

	th := newAccessRequestHandlersWithStores(t, store, store, store)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))
	resp, err := th.ListAuditEvents(ctx, &aiv1.ListAuditEventsRequest{
		PageSize: 10,
		AgentId:  "agent-2",
	})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(resp.GetAuditEvents()) != 1 {
		t.Fatalf("events len = %d, want 1", len(resp.GetAuditEvents()))
	}
	if got := resp.GetAuditEvents()[0].GetId(); got != "2" {
		t.Fatalf("event id = %q, want %q", got, "2")
	}
}

func TestListAuditEventsFiltersByTimeWindow(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 3, 20, 0, 0, time.UTC)
	store.AuditEvents = []storage.AuditEventRecord{
		{ID: "1", EventName: "access_request.created", OwnerUserID: "owner-1", AgentID: "agent-1", CreatedAt: now},
		{ID: "2", EventName: "access_request.reviewed", OwnerUserID: "owner-1", AgentID: "agent-1", CreatedAt: now.Add(2 * time.Minute)},
		{ID: "3", EventName: "access_request.revoked", OwnerUserID: "owner-1", AgentID: "agent-1", CreatedAt: now.Add(4 * time.Minute)},
	}

	th := newAccessRequestHandlersWithStores(t, store, store, store)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))
	resp, err := th.ListAuditEvents(ctx, &aiv1.ListAuditEventsRequest{
		PageSize:      10,
		CreatedAfter:  timestamppb.New(now.Add(time.Minute)),
		CreatedBefore: timestamppb.New(now.Add(3 * time.Minute)),
	})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(resp.GetAuditEvents()) != 1 {
		t.Fatalf("events len = %d, want 1", len(resp.GetAuditEvents()))
	}
	if got := resp.GetAuditEvents()[0].GetId(); got != "2" {
		t.Fatalf("event id = %q, want %q", got, "2")
	}
}

func TestListAuditEventsRejectsInvalidTimeWindow(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 3, 20, 0, 0, time.UTC)
	th := newAccessRequestHandlersWithStores(t, store, store, store)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))
	_, err := th.ListAuditEvents(ctx, &aiv1.ListAuditEventsRequest{
		PageSize:      10,
		CreatedAfter:  timestamppb.New(now.Add(2 * time.Minute)),
		CreatedBefore: timestamppb.New(now),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}
