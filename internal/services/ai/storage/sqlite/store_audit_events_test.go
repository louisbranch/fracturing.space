package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func TestPutAuditEvent(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 2, 30, 0, 0, time.UTC)

	err := store.PutAuditEvent(context.Background(), storage.AuditEventRecord{
		EventName:       "access_request.created",
		ActorUserID:     "",
		OwnerUserID:     "owner-1",
		RequesterUserID: "requester-1",
		AgentID:         "agent-1",
		AccessRequestID: "request-1",
		Outcome:         "pending",
		CreatedAt:       now,
	})
	if err == nil {
		t.Fatal("expected validation error for empty actor_user_id")
	}

	if err := store.PutAuditEvent(context.Background(), storage.AuditEventRecord{
		EventName:       "access_request.created",
		ActorUserID:     "requester-1",
		OwnerUserID:     "owner-1",
		RequesterUserID: "requester-1",
		AgentID:         "agent-1",
		AccessRequestID: "request-1",
		Outcome:         "pending",
		CreatedAt:       now,
	}); err != nil {
		t.Fatalf("put audit event: %v", err)
	}

	var (
		eventName       string
		actorUserID     string
		ownerUserID     string
		requesterUserID string
		agentID         string
		accessRequestID string
		outcome         string
		createdAt       int64
	)
	row := store.DB().QueryRowContext(context.Background(), `
SELECT event_name, actor_user_id, owner_user_id, requester_user_id, agent_id, access_request_id, outcome, created_at
FROM ai_audit_events
WHERE actor_user_id = ?
ORDER BY id DESC
LIMIT 1
`, "requester-1")
	if err := row.Scan(&eventName, &actorUserID, &ownerUserID, &requesterUserID, &agentID, &accessRequestID, &outcome, &createdAt); err != nil {
		t.Fatalf("scan audit row: %v", err)
	}
	if eventName != "access_request.created" {
		t.Fatalf("event_name = %q, want %q", eventName, "access_request.created")
	}
	if actorUserID != "requester-1" {
		t.Fatalf("actor_user_id = %q, want %q", actorUserID, "requester-1")
	}
	if ownerUserID != "owner-1" {
		t.Fatalf("owner_user_id = %q, want %q", ownerUserID, "owner-1")
	}
	if requesterUserID != "requester-1" {
		t.Fatalf("requester_user_id = %q, want %q", requesterUserID, "requester-1")
	}
	if agentID != "agent-1" {
		t.Fatalf("agent_id = %q, want %q", agentID, "agent-1")
	}
	if accessRequestID != "request-1" {
		t.Fatalf("access_request_id = %q, want %q", accessRequestID, "request-1")
	}
	if outcome != "pending" {
		t.Fatalf("outcome = %q, want %q", outcome, "pending")
	}
	if createdAt != now.UnixMilli() {
		t.Fatalf("created_at = %d, want %d", createdAt, now.UnixMilli())
	}
}

func TestListAuditEventsByOwner(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 3, 15, 0, 0, time.UTC)

	records := []storage.AuditEventRecord{
		{EventName: "access_request.created", ActorUserID: "user-1", OwnerUserID: "owner-1", RequesterUserID: "user-1", AgentID: "agent-1", AccessRequestID: "request-1", Outcome: "pending", CreatedAt: now},
		{EventName: "access_request.reviewed", ActorUserID: "owner-1", OwnerUserID: "owner-1", RequesterUserID: "user-1", AgentID: "agent-1", AccessRequestID: "request-1", Outcome: "approved", CreatedAt: now.Add(time.Minute)},
		{EventName: "access_request.created", ActorUserID: "user-2", OwnerUserID: "owner-2", RequesterUserID: "user-2", AgentID: "agent-2", AccessRequestID: "request-2", Outcome: "pending", CreatedAt: now},
		{EventName: "access_request.revoked", ActorUserID: "owner-1", OwnerUserID: "owner-1", RequesterUserID: "user-1", AgentID: "agent-1", AccessRequestID: "request-1", Outcome: "revoked", CreatedAt: now.Add(2 * time.Minute)},
	}
	for _, record := range records {
		if err := store.PutAuditEvent(context.Background(), record); err != nil {
			t.Fatalf("put audit event: %v", err)
		}
	}

	first, err := store.ListAuditEventsByOwner(context.Background(), "owner-1", 2, "", storage.AuditEventFilter{})
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if len(first.AuditEvents) != 2 {
		t.Fatalf("first page len = %d, want 2", len(first.AuditEvents))
	}
	if first.NextPageToken == "" {
		t.Fatal("expected next page token")
	}
	if first.AuditEvents[0].OwnerUserID != "owner-1" || first.AuditEvents[1].OwnerUserID != "owner-1" {
		t.Fatalf("unexpected owner ids: %+v", first.AuditEvents)
	}

	second, err := store.ListAuditEventsByOwner(context.Background(), "owner-1", 2, first.NextPageToken, storage.AuditEventFilter{})
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if len(second.AuditEvents) != 1 {
		t.Fatalf("second page len = %d, want 1", len(second.AuditEvents))
	}
	if second.AuditEvents[0].Outcome != "revoked" {
		t.Fatalf("second page outcome = %q, want %q", second.AuditEvents[0].Outcome, "revoked")
	}
	if second.NextPageToken != "" {
		t.Fatalf("second next page token = %q, want empty", second.NextPageToken)
	}
}

func TestListAuditEventsByOwnerWithFilters(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 4, 0, 0, 0, time.UTC)
	records := []storage.AuditEventRecord{
		{
			EventName:       "access_request.created",
			ActorUserID:     "requester-1",
			OwnerUserID:     "owner-1",
			RequesterUserID: "requester-1",
			AgentID:         "agent-1",
			AccessRequestID: "request-1",
			Outcome:         "pending",
			CreatedAt:       now,
		},
		{
			EventName:       "access_request.reviewed",
			ActorUserID:     "owner-1",
			OwnerUserID:     "owner-1",
			RequesterUserID: "requester-1",
			AgentID:         "agent-1",
			AccessRequestID: "request-1",
			Outcome:         "approved",
			CreatedAt:       now.Add(2 * time.Minute),
		},
		{
			EventName:       "access_request.reviewed",
			ActorUserID:     "owner-1",
			OwnerUserID:     "owner-1",
			RequesterUserID: "requester-2",
			AgentID:         "agent-2",
			AccessRequestID: "request-2",
			Outcome:         "approved",
			CreatedAt:       now.Add(4 * time.Minute),
		},
		{
			EventName:       "access_request.reviewed",
			ActorUserID:     "owner-2",
			OwnerUserID:     "owner-2",
			RequesterUserID: "requester-3",
			AgentID:         "agent-3",
			AccessRequestID: "request-3",
			Outcome:         "approved",
			CreatedAt:       now.Add(5 * time.Minute),
		},
	}
	for _, record := range records {
		if err := store.PutAuditEvent(context.Background(), record); err != nil {
			t.Fatalf("put audit event: %v", err)
		}
	}

	eventNameOnly, err := store.ListAuditEventsByOwner(context.Background(), "owner-1", 10, "", storage.AuditEventFilter{
		EventName: "access_request.reviewed",
	})
	if err != nil {
		t.Fatalf("list by event name: %v", err)
	}
	if len(eventNameOnly.AuditEvents) != 2 {
		t.Fatalf("event name len = %d, want 2", len(eventNameOnly.AuditEvents))
	}
	for _, event := range eventNameOnly.AuditEvents {
		if event.EventName != "access_request.reviewed" {
			t.Fatalf("event_name = %q, want %q", event.EventName, "access_request.reviewed")
		}
	}

	agentOnly, err := store.ListAuditEventsByOwner(context.Background(), "owner-1", 10, "", storage.AuditEventFilter{
		AgentID: "agent-2",
	})
	if err != nil {
		t.Fatalf("list by agent id: %v", err)
	}
	if len(agentOnly.AuditEvents) != 1 {
		t.Fatalf("agent filter len = %d, want 1", len(agentOnly.AuditEvents))
	}
	if got := agentOnly.AuditEvents[0].AgentID; got != "agent-2" {
		t.Fatalf("agent id = %q, want %q", got, "agent-2")
	}

	createdAfter := now.Add(time.Minute)
	createdBefore := now.Add(3 * time.Minute)
	timeWindow, err := store.ListAuditEventsByOwner(context.Background(), "owner-1", 10, "", storage.AuditEventFilter{
		CreatedAfter:  &createdAfter,
		CreatedBefore: &createdBefore,
	})
	if err != nil {
		t.Fatalf("list by time window: %v", err)
	}
	if len(timeWindow.AuditEvents) != 1 {
		t.Fatalf("time window len = %d, want 1", len(timeWindow.AuditEvents))
	}
	if got := timeWindow.AuditEvents[0].AccessRequestID; got != "request-1" {
		t.Fatalf("time window access_request_id = %q, want %q", got, "request-1")
	}
}
