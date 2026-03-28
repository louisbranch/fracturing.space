package service

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/accessrequest"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/auditevent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/test/mock/aifakes"
)

func TestAccessRequestServiceCreatePersistsAndWritesAuditEvent(t *testing.T) {
	agentStore := aifakes.NewAgentStore()
	accessRequestStore := aifakes.NewAccessRequestStore()
	auditEventStore := aifakes.NewAuditEventStore()
	now := time.Date(2026, 3, 23, 16, 0, 0, 0, time.UTC)
	agentStore.Agents["agent-1"] = agent.Agent{
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

	svc, err := NewAccessRequestService(AccessRequestServiceConfig{
		AgentStore:         agentStore,
		AccessRequestStore: accessRequestStore,
		AuditEventStore:    auditEventStore,
		Clock:              func() time.Time { return now },
		IDGenerator:        func() (string, error) { return "request-1", nil },
	})
	if err != nil {
		t.Fatalf("NewAccessRequestService: %v", err)
	}

	record, err := svc.Create(context.Background(), CreateAccessRequestInput{
		RequesterUserID: "user-1",
		AgentID:         "agent-1",
		Scope:           "invoke",
		RequestNote:     "please allow",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if record.ID != "request-1" {
		t.Fatalf("record.ID = %q, want %q", record.ID, "request-1")
	}
	if record.Status != accessrequest.StatusPending {
		t.Fatalf("record.Status = %q, want %q", record.Status, accessrequest.StatusPending)
	}
	if got := len(auditEventStore.AuditEvents); got != 1 {
		t.Fatalf("len(auditEventStore.AuditEvents) = %d, want 1", got)
	}
	if got := auditEventStore.AuditEvents[0].EventName; got != auditevent.NameAccessRequestCreated {
		t.Fatalf("auditEventStore.AuditEvents[0].EventName = %q, want %q", got, auditevent.NameAccessRequestCreated)
	}
}

func TestAccessRequestServiceCreateMapsValidationFailures(t *testing.T) {
	now := time.Date(2026, 3, 23, 16, 5, 0, 0, time.UTC)

	t.Run("own agent", func(t *testing.T) {
		agentStore := aifakes.NewAgentStore()
		agentStore.Agents["agent-1"] = agent.Agent{
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

		svc, err := NewAccessRequestService(AccessRequestServiceConfig{
			AgentStore:         agentStore,
			AccessRequestStore: aifakes.NewAccessRequestStore(),
			AuditEventStore:    aifakes.NewAuditEventStore(),
			Clock:              func() time.Time { return now },
		})
		if err != nil {
			t.Fatalf("NewAccessRequestService: %v", err)
		}

		_, err = svc.Create(context.Background(), CreateAccessRequestInput{
			RequesterUserID: "user-1",
			AgentID:         "agent-1",
			Scope:           "invoke",
		})
		if got := ErrorKindOf(err); got != ErrKindInvalidArgument {
			t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindInvalidArgument)
		}
	})

	t.Run("unsupported scope", func(t *testing.T) {
		agentStore := aifakes.NewAgentStore()
		agentStore.Agents["agent-1"] = agent.Agent{
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

		svc, err := NewAccessRequestService(AccessRequestServiceConfig{
			AgentStore:         agentStore,
			AccessRequestStore: aifakes.NewAccessRequestStore(),
			AuditEventStore:    aifakes.NewAuditEventStore(),
			Clock:              func() time.Time { return now },
		})
		if err != nil {
			t.Fatalf("NewAccessRequestService: %v", err)
		}

		_, err = svc.Create(context.Background(), CreateAccessRequestInput{
			RequesterUserID: "user-1",
			AgentID:         "agent-1",
			Scope:           "admin",
		})
		if got := ErrorKindOf(err); got != ErrKindInvalidArgument {
			t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindInvalidArgument)
		}
	})
}

func TestAccessRequestServiceListByRole(t *testing.T) {
	store := aifakes.NewAccessRequestStore()
	now := time.Date(2026, 3, 23, 16, 10, 0, 0, time.UTC)
	store.AccessRequests["request-1"] = accessrequest.AccessRequest{ID: "request-1", RequesterUserID: "user-1", OwnerUserID: "owner-1", AgentID: "agent-1", Scope: accessrequest.ScopeInvoke, Status: accessrequest.StatusPending, CreatedAt: now, UpdatedAt: now}
	store.AccessRequests["request-2"] = accessrequest.AccessRequest{ID: "request-2", RequesterUserID: "user-1", OwnerUserID: "owner-2", AgentID: "agent-2", Scope: accessrequest.ScopeInvoke, Status: accessrequest.StatusPending, CreatedAt: now, UpdatedAt: now}
	store.AccessRequests["request-3"] = accessrequest.AccessRequest{ID: "request-3", RequesterUserID: "user-3", OwnerUserID: "owner-1", AgentID: "agent-3", Scope: accessrequest.ScopeInvoke, Status: accessrequest.StatusApproved, CreatedAt: now, UpdatedAt: now}

	svc, err := NewAccessRequestService(AccessRequestServiceConfig{
		AgentStore:         aifakes.NewAgentStore(),
		AccessRequestStore: store,
		AuditEventStore:    aifakes.NewAuditEventStore(),
	})
	if err != nil {
		t.Fatalf("NewAccessRequestService: %v", err)
	}

	requesterPage, err := svc.List(context.Background(), "user-1", ListAccessRequestRoleRequester, 10, "")
	if err != nil {
		t.Fatalf("List requester: %v", err)
	}
	if got := len(requesterPage.AccessRequests); got != 2 {
		t.Fatalf("len(requesterPage.AccessRequests) = %d, want 2", got)
	}

	ownerPage, err := svc.List(context.Background(), "owner-1", ListAccessRequestRoleOwner, 10, "")
	if err != nil {
		t.Fatalf("List owner: %v", err)
	}
	if got := len(ownerPage.AccessRequests); got != 2 {
		t.Fatalf("len(ownerPage.AccessRequests) = %d, want 2", got)
	}
}

func TestAccessRequestServiceListAuditEvents(t *testing.T) {
	auditEventStore := aifakes.NewAuditEventStore()
	now := time.Date(2026, 3, 23, 16, 12, 0, 0, time.UTC)
	auditEventStore.AuditEvents = []auditevent.Event{
		{ID: "event-1", OwnerUserID: "owner-1", EventName: auditevent.NameAccessRequestCreated, CreatedAt: now},
		{ID: "event-2", OwnerUserID: "owner-2", EventName: auditevent.NameAccessRequestReviewed, CreatedAt: now},
	}

	svc, err := NewAccessRequestService(AccessRequestServiceConfig{
		AgentStore:         aifakes.NewAgentStore(),
		AccessRequestStore: aifakes.NewAccessRequestStore(),
		AuditEventStore:    auditEventStore,
	})
	if err != nil {
		t.Fatalf("NewAccessRequestService: %v", err)
	}

	page, err := svc.ListAuditEvents(context.Background(), ListAuditEventsInput{OwnerUserID: "owner-1", PageSize: 10})
	if err != nil {
		t.Fatalf("ListAuditEvents: %v", err)
	}
	if len(page.AuditEvents) != 1 || page.AuditEvents[0].ID != "event-1" {
		t.Fatalf("page.AuditEvents = %+v, want event-1 only", page.AuditEvents)
	}
}

func TestAccessRequestServiceReviewWritesAuditEvent(t *testing.T) {
	store := aifakes.NewAccessRequestStore()
	auditEventStore := aifakes.NewAuditEventStore()
	now := time.Date(2026, 3, 23, 16, 15, 0, 0, time.UTC)
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

	svc, err := NewAccessRequestService(AccessRequestServiceConfig{
		AgentStore:         aifakes.NewAgentStore(),
		AccessRequestStore: store,
		AuditEventStore:    auditEventStore,
		Clock:              func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewAccessRequestService: %v", err)
	}

	record, err := svc.Review(context.Background(), ReviewAccessRequestInput{
		OwnerUserID:     "owner-1",
		AccessRequestID: "request-1",
		Decision:        accessrequest.DecisionApprove,
		ReviewNote:      "approved",
	})
	if err != nil {
		t.Fatalf("Review: %v", err)
	}
	if record.Status != accessrequest.StatusApproved {
		t.Fatalf("record.Status = %q, want %q", record.Status, accessrequest.StatusApproved)
	}
	if got := len(auditEventStore.AuditEvents); got != 1 {
		t.Fatalf("len(auditEventStore.AuditEvents) = %d, want 1", got)
	}
	if got := auditEventStore.AuditEvents[0].EventName; got != auditevent.NameAccessRequestReviewed {
		t.Fatalf("auditEventStore.AuditEvents[0].EventName = %q, want %q", got, auditevent.NameAccessRequestReviewed)
	}
}

func TestAccessRequestServiceReviewRejectsNonOwner(t *testing.T) {
	store := aifakes.NewAccessRequestStore()
	now := time.Date(2026, 3, 23, 16, 20, 0, 0, time.UTC)
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

	svc, err := NewAccessRequestService(AccessRequestServiceConfig{
		AgentStore:         aifakes.NewAgentStore(),
		AccessRequestStore: store,
		AuditEventStore:    aifakes.NewAuditEventStore(),
		Clock:              func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewAccessRequestService: %v", err)
	}

	_, err = svc.Review(context.Background(), ReviewAccessRequestInput{
		OwnerUserID:     "owner-2",
		AccessRequestID: "request-1",
		Decision:        accessrequest.DecisionDeny,
		ReviewNote:      "no",
	})
	if got := ErrorKindOf(err); got != ErrKindNotFound {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindNotFound)
	}
}

func TestAccessRequestServiceRevokeWritesAuditEvent(t *testing.T) {
	store := aifakes.NewAccessRequestStore()
	auditEventStore := aifakes.NewAuditEventStore()
	now := time.Date(2026, 3, 23, 16, 25, 0, 0, time.UTC)
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

	svc, err := NewAccessRequestService(AccessRequestServiceConfig{
		AgentStore:         aifakes.NewAgentStore(),
		AccessRequestStore: store,
		AuditEventStore:    auditEventStore,
		Clock:              func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewAccessRequestService: %v", err)
	}

	record, err := svc.Revoke(context.Background(), RevokeAccessRequestInput{
		OwnerUserID:     "owner-1",
		AccessRequestID: "request-1",
		RevokeNote:      "removed",
	})
	if err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if record.Status != accessrequest.StatusRevoked {
		t.Fatalf("record.Status = %q, want %q", record.Status, accessrequest.StatusRevoked)
	}
	if record.ReviewNote != "approved" {
		t.Fatalf("record.ReviewNote = %q, want %q", record.ReviewNote, "approved")
	}
	if record.RevokerUserID != "owner-1" {
		t.Fatalf("record.RevokerUserID = %q, want %q", record.RevokerUserID, "owner-1")
	}
	if record.RevokeNote != "removed" {
		t.Fatalf("record.RevokeNote = %q, want %q", record.RevokeNote, "removed")
	}
	if record.RevokedAt == nil || !record.RevokedAt.Equal(now) {
		t.Fatalf("record.RevokedAt = %v, want %v", record.RevokedAt, now)
	}
	if got := len(auditEventStore.AuditEvents); got != 1 {
		t.Fatalf("len(auditEventStore.AuditEvents) = %d, want 1", got)
	}
	if got := auditEventStore.AuditEvents[0].EventName; got != auditevent.NameAccessRequestRevoked {
		t.Fatalf("auditEventStore.AuditEvents[0].EventName = %q, want %q", got, auditevent.NameAccessRequestRevoked)
	}
}

func TestAccessRequestServiceRevokeRejectsInvalidOwnershipOrState(t *testing.T) {
	now := time.Date(2026, 3, 23, 16, 30, 0, 0, time.UTC)

	t.Run("non owner", func(t *testing.T) {
		store := aifakes.NewAccessRequestStore()
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

		svc, err := NewAccessRequestService(AccessRequestServiceConfig{
			AgentStore:         aifakes.NewAgentStore(),
			AccessRequestStore: store,
			AuditEventStore:    aifakes.NewAuditEventStore(),
			Clock:              func() time.Time { return now },
		})
		if err != nil {
			t.Fatalf("NewAccessRequestService: %v", err)
		}

		_, err = svc.Revoke(context.Background(), RevokeAccessRequestInput{
			OwnerUserID:     "owner-2",
			AccessRequestID: "request-1",
			RevokeNote:      "no",
		})
		if got := ErrorKindOf(err); got != ErrKindNotFound {
			t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindNotFound)
		}
	})

	t.Run("not approved", func(t *testing.T) {
		store := aifakes.NewAccessRequestStore()
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

		svc, err := NewAccessRequestService(AccessRequestServiceConfig{
			AgentStore:         aifakes.NewAgentStore(),
			AccessRequestStore: store,
			AuditEventStore:    aifakes.NewAuditEventStore(),
			Clock:              func() time.Time { return now },
		})
		if err != nil {
			t.Fatalf("NewAccessRequestService: %v", err)
		}

		_, err = svc.Revoke(context.Background(), RevokeAccessRequestInput{
			OwnerUserID:     "owner-1",
			AccessRequestID: "request-1",
			RevokeNote:      "not approved",
		})
		if got := ErrorKindOf(err); got != ErrKindFailedPrecondition {
			t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindFailedPrecondition)
		}
	})
}
