package sqlite

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/accessrequest"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func TestPutGetAccessRequestRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 0, 55, 0, 0, time.UTC)

	input := accessrequest.AccessRequest{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "user-2",
		AgentID:         "agent-1",
		Scope:           accessrequest.ScopeInvoke,
		RequestNote:     "please allow",
		Status:          accessrequest.StatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := store.PutAccessRequest(context.Background(), input); err != nil {
		t.Fatalf("put access request: %v", err)
	}

	got, err := store.GetAccessRequest(context.Background(), "request-1")
	if err != nil {
		t.Fatalf("get access request: %v", err)
	}
	if got.ID != input.ID || got.RequesterUserID != input.RequesterUserID || got.OwnerUserID != input.OwnerUserID {
		t.Fatalf("unexpected access request: %+v", got)
	}
}

func TestListAccessRequestsByRequesterAndOwner(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 0, 56, 0, 0, time.UTC)

	records := []accessrequest.AccessRequest{
		{ID: "request-1", RequesterUserID: "user-1", OwnerUserID: "user-2", AgentID: "agent-1", Scope: accessrequest.ScopeInvoke, Status: accessrequest.StatusPending, CreatedAt: now, UpdatedAt: now},
		{ID: "request-2", RequesterUserID: "user-1", OwnerUserID: "user-3", AgentID: "agent-2", Scope: accessrequest.ScopeInvoke, Status: accessrequest.StatusPending, CreatedAt: now, UpdatedAt: now},
		{ID: "request-3", RequesterUserID: "user-4", OwnerUserID: "user-2", AgentID: "agent-3", Scope: accessrequest.ScopeInvoke, Status: accessrequest.StatusPending, CreatedAt: now, UpdatedAt: now},
	}
	for _, record := range records {
		if err := store.PutAccessRequest(context.Background(), record); err != nil {
			t.Fatalf("put access request %s: %v", record.ID, err)
		}
	}

	requesterPage, err := store.ListAccessRequestsByRequester(context.Background(), "user-1", 10, "")
	if err != nil {
		t.Fatalf("list by requester: %v", err)
	}
	if len(requesterPage.AccessRequests) != 2 {
		t.Fatalf("requester page len = %d, want 2", len(requesterPage.AccessRequests))
	}

	ownerPage, err := store.ListAccessRequestsByOwner(context.Background(), "user-2", 10, "")
	if err != nil {
		t.Fatalf("list by owner: %v", err)
	}
	if len(ownerPage.AccessRequests) != 2 {
		t.Fatalf("owner page len = %d, want 2", len(ownerPage.AccessRequests))
	}
}

func TestGetApprovedInvokeAccessByRequesterForAgent(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 1, 5, 0, 0, time.UTC)

	records := []accessrequest.AccessRequest{
		{ID: "request-1", RequesterUserID: "user-1", OwnerUserID: "owner-1", AgentID: "agent-1", Scope: accessrequest.ScopeInvoke, Status: accessrequest.StatusApproved, CreatedAt: now, UpdatedAt: now},
		{ID: "request-2", RequesterUserID: "user-1", OwnerUserID: "owner-1", AgentID: "agent-1", Scope: accessrequest.ScopeInvoke, Status: accessrequest.StatusPending, CreatedAt: now, UpdatedAt: now},
		{ID: "request-3", RequesterUserID: "user-1", OwnerUserID: "owner-1", AgentID: "agent-2", Scope: accessrequest.ScopeInvoke, Status: accessrequest.StatusApproved, CreatedAt: now, UpdatedAt: now},
		{ID: "request-4", RequesterUserID: "user-1", OwnerUserID: "owner-2", AgentID: "agent-1", Scope: accessrequest.ScopeInvoke, Status: accessrequest.StatusApproved, CreatedAt: now, UpdatedAt: now},
		{ID: "request-5", RequesterUserID: "user-1", OwnerUserID: "owner-1", AgentID: "agent-1", Scope: accessrequest.Scope("observe"), Status: accessrequest.StatusApproved, CreatedAt: now, UpdatedAt: now},
	}
	for _, record := range records {
		if err := store.PutAccessRequest(context.Background(), record); err != nil {
			t.Fatalf("put access request %s: %v", record.ID, err)
		}
	}

	got, err := store.GetApprovedInvokeAccessByRequesterForAgent(context.Background(), "user-1", "owner-1", "agent-1")
	if err != nil {
		t.Fatalf("get approved invoke access: %v", err)
	}
	if got.ID != "request-1" {
		t.Fatalf("id = %q, want %q", got.ID, "request-1")
	}

	_, err = store.GetApprovedInvokeAccessByRequesterForAgent(context.Background(), "user-1", "owner-1", "agent-missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("missing access error = %v, want %v", err, storage.ErrNotFound)
	}
}

func TestListApprovedInvokeAccessRequestsByRequester(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 1, 6, 0, 0, time.UTC)

	records := []accessrequest.AccessRequest{
		{ID: "request-1", RequesterUserID: "user-1", OwnerUserID: "owner-1", AgentID: "agent-1", Scope: accessrequest.ScopeInvoke, Status: accessrequest.StatusApproved, CreatedAt: now, UpdatedAt: now},
		{ID: "request-2", RequesterUserID: "user-1", OwnerUserID: "owner-1", AgentID: "agent-2", Scope: accessrequest.ScopeInvoke, Status: accessrequest.StatusApproved, CreatedAt: now, UpdatedAt: now},
		{ID: "request-3", RequesterUserID: "user-1", OwnerUserID: "owner-1", AgentID: "agent-3", Scope: accessrequest.ScopeInvoke, Status: accessrequest.StatusPending, CreatedAt: now, UpdatedAt: now},
		{ID: "request-4", RequesterUserID: "user-1", OwnerUserID: "owner-1", AgentID: "agent-4", Scope: accessrequest.Scope("observe"), Status: accessrequest.StatusApproved, CreatedAt: now, UpdatedAt: now},
		{ID: "request-5", RequesterUserID: "user-2", OwnerUserID: "owner-1", AgentID: "agent-5", Scope: accessrequest.ScopeInvoke, Status: accessrequest.StatusApproved, CreatedAt: now, UpdatedAt: now},
	}
	for _, record := range records {
		if err := store.PutAccessRequest(context.Background(), record); err != nil {
			t.Fatalf("put access request %s: %v", record.ID, err)
		}
	}

	first, err := store.ListApprovedInvokeAccessRequestsByRequester(context.Background(), "user-1", 1, "")
	if err != nil {
		t.Fatalf("list first approved invoke page: %v", err)
	}
	if len(first.AccessRequests) != 1 {
		t.Fatalf("first page len = %d, want 1", len(first.AccessRequests))
	}
	if got := first.AccessRequests[0].ID; got != "request-1" {
		t.Fatalf("first page id = %q, want %q", got, "request-1")
	}
	if first.NextPageToken == "" {
		t.Fatal("expected next page token")
	}

	second, err := store.ListApprovedInvokeAccessRequestsByRequester(context.Background(), "user-1", 1, first.NextPageToken)
	if err != nil {
		t.Fatalf("list second approved invoke page: %v", err)
	}
	if len(second.AccessRequests) != 1 {
		t.Fatalf("second page len = %d, want 1", len(second.AccessRequests))
	}
	if got := second.AccessRequests[0].ID; got != "request-2" {
		t.Fatalf("second page id = %q, want %q", got, "request-2")
	}
}

func TestReviewAccessRequest(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 0, 57, 0, 0, time.UTC)

	if err := store.PutAccessRequest(context.Background(), accessrequest.AccessRequest{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "user-2",
		AgentID:         "agent-1",
		Scope:           accessrequest.ScopeInvoke,
		RequestNote:     "please allow",
		Status:          accessrequest.StatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("put access request: %v", err)
	}

	reviewedAt := now.Add(time.Minute)
	if err := store.ReviewAccessRequest(context.Background(), accessrequest.AccessRequest{
		ID:             "request-1",
		OwnerUserID:    "user-2",
		Status:         accessrequest.StatusApproved,
		ReviewerUserID: "user-2",
		ReviewNote:     "approved",
		ReviewedAt:     &reviewedAt,
	}); err != nil {
		t.Fatalf("review access request: %v", err)
	}

	got, err := store.GetAccessRequest(context.Background(), "request-1")
	if err != nil {
		t.Fatalf("get access request: %v", err)
	}
	if got.Status != accessrequest.StatusApproved {
		t.Fatalf("status = %q, want %q", got.Status, accessrequest.StatusApproved)
	}
	if got.ReviewerUserID != "user-2" {
		t.Fatalf("reviewer_user_id = %q, want %q", got.ReviewerUserID, "user-2")
	}
	if got.ReviewedAt == nil || !got.ReviewedAt.Equal(reviewedAt) {
		t.Fatalf("reviewed_at = %v, want %v", got.ReviewedAt, reviewedAt)
	}
}

func TestReviewAccessRequestRejectsNonPending(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 0, 58, 0, 0, time.UTC)

	if err := store.PutAccessRequest(context.Background(), accessrequest.AccessRequest{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "user-2",
		AgentID:         "agent-1",
		Scope:           accessrequest.ScopeInvoke,
		Status:          accessrequest.StatusDenied,
		CreatedAt:       now,
		UpdatedAt:       now,
		ReviewerUserID:  "user-2",
		ReviewNote:      "already denied",
		ReviewedAt:      &now,
	}); err != nil {
		t.Fatalf("put access request: %v", err)
	}

	reviewedAt := now.Add(time.Minute)
	err := store.ReviewAccessRequest(context.Background(), accessrequest.AccessRequest{
		ID:             "request-1",
		OwnerUserID:    "user-2",
		Status:         accessrequest.StatusApproved,
		ReviewerUserID: "user-2",
		ReviewNote:     "retry",
		ReviewedAt:     &reviewedAt,
	})
	if !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("review error = %v, want %v", err, storage.ErrConflict)
	}
}

func TestReviewAccessRequestRejectsReviewerMismatch(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 0, 58, 0, 0, time.UTC)

	if err := store.PutAccessRequest(context.Background(), accessrequest.AccessRequest{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "user-2",
		AgentID:         "agent-1",
		Scope:           accessrequest.ScopeInvoke,
		Status:          accessrequest.StatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("put access request: %v", err)
	}

	reviewedAt := now.Add(time.Minute)
	err := store.ReviewAccessRequest(context.Background(), accessrequest.AccessRequest{
		ID:             "request-1",
		OwnerUserID:    "user-2",
		Status:         accessrequest.StatusApproved,
		ReviewerUserID: "user-3",
		ReviewNote:     "retry",
		ReviewedAt:     &reviewedAt,
	})
	if err == nil {
		t.Fatal("expected review error for reviewer mismatch")
	}
	if !strings.Contains(err.Error(), "reviewer user id must match owner user id") {
		t.Fatalf("review error = %v, want reviewer mismatch", err)
	}
}

func TestRevokeAccessRequestTransition(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 3, 0, 0, 0, time.UTC)

	if err := store.PutAccessRequest(context.Background(), accessrequest.AccessRequest{
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
	}); err != nil {
		t.Fatalf("put access request: %v", err)
	}

	revokedAt := now
	if err := store.RevokeAccessRequest(context.Background(), accessrequest.AccessRequest{
		ID:             "request-1",
		OwnerUserID:    "owner-1",
		Status:         accessrequest.StatusRevoked,
		ReviewerUserID: "owner-1",
		ReviewNote:     "removed",
		UpdatedAt:      revokedAt,
	}); err != nil {
		t.Fatalf("revoke access request: %v", err)
	}

	got, err := store.GetAccessRequest(context.Background(), "request-1")
	if err != nil {
		t.Fatalf("get access request: %v", err)
	}
	if got.Status != accessrequest.StatusRevoked {
		t.Fatalf("status = %q, want %q", got.Status, accessrequest.StatusRevoked)
	}
	if got.ReviewerUserID != "owner-1" {
		t.Fatalf("reviewer_user_id = %q, want %q", got.ReviewerUserID, "owner-1")
	}
	if got.ReviewNote != "removed" {
		t.Fatalf("review_note = %q, want %q", got.ReviewNote, "removed")
	}
	if !got.UpdatedAt.Equal(revokedAt) {
		t.Fatalf("updated_at = %v, want %v", got.UpdatedAt, revokedAt)
	}

	if err := store.RevokeAccessRequest(context.Background(), accessrequest.AccessRequest{
		ID:             "request-1",
		OwnerUserID:    "owner-1",
		Status:         accessrequest.StatusRevoked,
		ReviewerUserID: "owner-1",
		ReviewNote:     "again",
		UpdatedAt:      revokedAt.Add(time.Minute),
	}); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("second revoke error = %v, want %v", err, storage.ErrConflict)
	}
}

func TestRevokeAccessRequestRejectsReviewerMismatch(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 3, 0, 0, 0, time.UTC)

	if err := store.PutAccessRequest(context.Background(), accessrequest.AccessRequest{
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
	}); err != nil {
		t.Fatalf("put access request: %v", err)
	}

	err := store.RevokeAccessRequest(context.Background(), accessrequest.AccessRequest{
		ID:             "request-1",
		OwnerUserID:    "owner-1",
		Status:         accessrequest.StatusRevoked,
		ReviewerUserID: "owner-2",
		ReviewNote:     "removed",
		UpdatedAt:      now,
	})
	if err == nil {
		t.Fatal("expected revoke error for reviewer mismatch")
	}
	if !strings.Contains(err.Error(), "reviewer user id must match owner user id") {
		t.Fatalf("revoke error = %v, want reviewer mismatch", err)
	}
}
