package sqlite

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/invite/storage"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "invite-test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open(%q): %v", path, err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

var testTime = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

// --- InviteStore contract tests ---

func TestPutAndGetInvite(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	inv := storage.InviteRecord{
		ID:                     "inv-1",
		CampaignID:             "camp-1",
		ParticipantID:          "part-1",
		RecipientUserID:        "user-1",
		Status:                 storage.StatusPending,
		CreatedByParticipantID: "part-gm",
		CreatedAt:              testTime,
		UpdatedAt:              testTime,
	}
	if err := s.PutInvite(ctx, inv); err != nil {
		t.Fatalf("PutInvite: %v", err)
	}

	got, err := s.GetInvite(ctx, "inv-1")
	if err != nil {
		t.Fatalf("GetInvite: %v", err)
	}
	if got.ID != inv.ID {
		t.Fatalf("ID = %q, want %q", got.ID, inv.ID)
	}
	if got.CampaignID != inv.CampaignID {
		t.Fatalf("CampaignID = %q, want %q", got.CampaignID, inv.CampaignID)
	}
	if got.Status != storage.StatusPending {
		t.Fatalf("Status = %q, want %q", got.Status, storage.StatusPending)
	}
	if got.RecipientUserID != inv.RecipientUserID {
		t.Fatalf("RecipientUserID = %q, want %q", got.RecipientUserID, inv.RecipientUserID)
	}
}

func TestGetInvite_NotFound(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)

	_, err := s.GetInvite(context.Background(), "nonexistent")
	if err != storage.ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestPutInvite_Upsert(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	inv := storage.InviteRecord{
		ID:            "inv-up",
		CampaignID:    "camp-1",
		ParticipantID: "part-1",
		Status:        storage.StatusPending,
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	if err := s.PutInvite(ctx, inv); err != nil {
		t.Fatalf("PutInvite: %v", err)
	}

	// Upsert with claimed status.
	inv.Status = storage.StatusClaimed
	inv.UpdatedAt = testTime.Add(time.Hour)
	if err := s.PutInvite(ctx, inv); err != nil {
		t.Fatalf("PutInvite upsert: %v", err)
	}

	got, _ := s.GetInvite(ctx, "inv-up")
	if got.Status != storage.StatusClaimed {
		t.Fatalf("Status after upsert = %q, want %q", got.Status, storage.StatusClaimed)
	}
}

func TestUpdateInviteStatus(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	inv := storage.InviteRecord{
		ID:            "inv-status",
		CampaignID:    "camp-1",
		ParticipantID: "part-1",
		Status:        storage.StatusPending,
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	if err := s.PutInvite(ctx, inv); err != nil {
		t.Fatalf("PutInvite: %v", err)
	}

	updatedAt := testTime.Add(2 * time.Hour)
	if err := s.UpdateInviteStatus(ctx, "inv-status", storage.StatusRevoked, updatedAt); err != nil {
		t.Fatalf("UpdateInviteStatus: %v", err)
	}

	got, _ := s.GetInvite(ctx, "inv-status")
	if got.Status != storage.StatusRevoked {
		t.Fatalf("Status = %q, want %q", got.Status, storage.StatusRevoked)
	}
}

func TestUpdateInviteStatus_NotFound(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)

	err := s.UpdateInviteStatus(context.Background(), "nonexistent", storage.StatusClaimed, testTime)
	if err != storage.ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestListInvites_ByCampaign(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	for _, id := range []string{"a", "b", "c"} {
		_ = s.PutInvite(ctx, storage.InviteRecord{
			ID:         "inv-" + id,
			CampaignID: "camp-1",
			Status:     storage.StatusPending,
			CreatedAt:  testTime,
			UpdatedAt:  testTime,
		})
	}
	_ = s.PutInvite(ctx, storage.InviteRecord{
		ID:         "inv-other",
		CampaignID: "camp-2",
		Status:     storage.StatusPending,
		CreatedAt:  testTime,
		UpdatedAt:  testTime,
	})

	page, err := s.ListInvites(ctx, "camp-1", "", "", 10, "")
	if err != nil {
		t.Fatalf("ListInvites: %v", err)
	}
	if len(page.Invites) != 3 {
		t.Fatalf("len(Invites) = %d, want 3", len(page.Invites))
	}
}

func TestListInvites_ByStatus(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	_ = s.PutInvite(ctx, storage.InviteRecord{
		ID: "inv-p", CampaignID: "camp-1", Status: storage.StatusPending,
		CreatedAt: testTime, UpdatedAt: testTime,
	})
	_ = s.PutInvite(ctx, storage.InviteRecord{
		ID: "inv-c", CampaignID: "camp-1", Status: storage.StatusClaimed,
		CreatedAt: testTime, UpdatedAt: testTime,
	})

	page, err := s.ListInvites(ctx, "camp-1", "", storage.StatusClaimed, 10, "")
	if err != nil {
		t.Fatalf("ListInvites: %v", err)
	}
	if len(page.Invites) != 1 {
		t.Fatalf("len(Invites) = %d, want 1", len(page.Invites))
	}
	if page.Invites[0].ID != "inv-c" {
		t.Fatalf("ID = %q, want %q", page.Invites[0].ID, "inv-c")
	}
}

func TestListInvites_Pagination(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_ = s.PutInvite(ctx, storage.InviteRecord{
			ID:         fmt.Sprintf("inv-%02d", i),
			CampaignID: "camp-1",
			Status:     storage.StatusPending,
			CreatedAt:  testTime,
			UpdatedAt:  testTime,
		})
	}

	// Page 1 — get 2 records.
	page1, err := s.ListInvites(ctx, "camp-1", "", "", 2, "")
	if err != nil {
		t.Fatalf("ListInvites page1: %v", err)
	}
	if len(page1.Invites) != 2 {
		t.Fatalf("page1 len = %d, want 2", len(page1.Invites))
	}
	if page1.NextPageToken == "" {
		t.Fatal("expected NextPageToken for page1")
	}

	// Page 2 — use token.
	page2, err := s.ListInvites(ctx, "camp-1", "", "", 2, page1.NextPageToken)
	if err != nil {
		t.Fatalf("ListInvites page2: %v", err)
	}
	if len(page2.Invites) != 2 {
		t.Fatalf("page2 len = %d, want 2", len(page2.Invites))
	}

	// Page 3 — last record.
	page3, err := s.ListInvites(ctx, "camp-1", "", "", 2, page2.NextPageToken)
	if err != nil {
		t.Fatalf("ListInvites page3: %v", err)
	}
	if len(page3.Invites) != 1 {
		t.Fatalf("page3 len = %d, want 1", len(page3.Invites))
	}
	if page3.NextPageToken != "" {
		t.Fatalf("expected empty NextPageToken for last page, got %q", page3.NextPageToken)
	}
}

func TestListPendingInvites(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	_ = s.PutInvite(ctx, storage.InviteRecord{
		ID: "inv-p1", CampaignID: "camp-1", Status: storage.StatusPending,
		CreatedAt: testTime, UpdatedAt: testTime,
	})
	_ = s.PutInvite(ctx, storage.InviteRecord{
		ID: "inv-c1", CampaignID: "camp-1", Status: storage.StatusClaimed,
		CreatedAt: testTime, UpdatedAt: testTime,
	})
	_ = s.PutInvite(ctx, storage.InviteRecord{
		ID: "inv-p2", CampaignID: "camp-1", Status: storage.StatusPending,
		CreatedAt: testTime, UpdatedAt: testTime,
	})

	page, err := s.ListPendingInvites(ctx, "camp-1", 10, "")
	if err != nil {
		t.Fatalf("ListPendingInvites: %v", err)
	}
	if len(page.Invites) != 2 {
		t.Fatalf("len(Invites) = %d, want 2", len(page.Invites))
	}
}

func TestListPendingInvitesForRecipient(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	_ = s.PutInvite(ctx, storage.InviteRecord{
		ID: "inv-r1", CampaignID: "camp-1", RecipientUserID: "user-1",
		Status: storage.StatusPending, CreatedAt: testTime, UpdatedAt: testTime,
	})
	_ = s.PutInvite(ctx, storage.InviteRecord{
		ID: "inv-r2", CampaignID: "camp-2", RecipientUserID: "user-1",
		Status: storage.StatusClaimed, CreatedAt: testTime, UpdatedAt: testTime,
	})
	_ = s.PutInvite(ctx, storage.InviteRecord{
		ID: "inv-r3", CampaignID: "camp-3", RecipientUserID: "user-1",
		Status: storage.StatusPending, CreatedAt: testTime, UpdatedAt: testTime,
	})

	page, err := s.ListPendingInvitesForRecipient(ctx, "user-1", 10, "")
	if err != nil {
		t.Fatalf("ListPendingInvitesForRecipient: %v", err)
	}
	if len(page.Invites) != 2 {
		t.Fatalf("len(Invites) = %d, want 2", len(page.Invites))
	}
}

// --- OutboxStore contract tests ---

func TestEnqueueAndLease(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()
	now := testTime

	evt := storage.OutboxEvent{
		ID:          "evt-1",
		EventType:   "invite.created",
		PayloadJSON: []byte(`{"id":"inv-1"}`),
		DedupeKey:   "inv-1",
		CreatedAt:   now,
	}
	if err := s.Enqueue(ctx, evt); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	leased, err := s.LeaseOutboxEvents(ctx, "worker-1", 10, 5*time.Minute, now)
	if err != nil {
		t.Fatalf("LeaseOutboxEvents: %v", err)
	}
	if len(leased) != 1 {
		t.Fatalf("len(leased) = %d, want 1", len(leased))
	}
	if leased[0].EventType != "invite.created" {
		t.Fatalf("EventType = %q, want %q", leased[0].EventType, "invite.created")
	}
	if leased[0].LeaseOwner != "worker-1" {
		t.Fatalf("LeaseOwner = %q, want %q", leased[0].LeaseOwner, "worker-1")
	}
}

func TestEnqueue_Dedupe(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	evt := storage.OutboxEvent{
		ID: "evt-dup", EventType: "invite.created",
		PayloadJSON: []byte(`{}`), DedupeKey: "dedup-1", CreatedAt: testTime,
	}
	if err := s.Enqueue(ctx, evt); err != nil {
		t.Fatalf("Enqueue first: %v", err)
	}
	// Duplicate should not error (ON CONFLICT DO NOTHING).
	if err := s.Enqueue(ctx, evt); err != nil {
		t.Fatalf("Enqueue duplicate: %v", err)
	}

	leased, err := s.LeaseOutboxEvents(ctx, "w", 10, 5*time.Minute, testTime)
	if err != nil {
		t.Fatalf("LeaseOutboxEvents: %v", err)
	}
	if len(leased) != 1 {
		t.Fatalf("len(leased) = %d, want 1 (dedup should prevent second insert)", len(leased))
	}
}

func TestAckOutboxEvent_Succeeded(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()
	now := testTime

	_ = s.Enqueue(ctx, storage.OutboxEvent{
		ID: "evt-ack", EventType: "invite.claimed",
		PayloadJSON: []byte(`{}`), CreatedAt: now,
	})
	_, _ = s.LeaseOutboxEvents(ctx, "w1", 10, 5*time.Minute, now)

	if err := s.AckOutboxEvent(ctx, "evt-ack", "w1", "succeeded", time.Time{}, "", now); err != nil {
		t.Fatalf("AckOutboxEvent: %v", err)
	}

	// Succeeded events should not be re-leased.
	leased, _ := s.LeaseOutboxEvents(ctx, "w1", 10, 5*time.Minute, now.Add(time.Hour))
	if len(leased) != 0 {
		t.Fatalf("len(leased) = %d, want 0 (event was acked)", len(leased))
	}
}

func TestAckOutboxEvent_Retry(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()
	now := testTime

	_ = s.Enqueue(ctx, storage.OutboxEvent{
		ID: "evt-retry", EventType: "invite.created",
		PayloadJSON: []byte(`{}`), CreatedAt: now,
	})
	_, _ = s.LeaseOutboxEvents(ctx, "w1", 10, 5*time.Minute, now)

	retryAt := now.Add(10 * time.Minute)
	if err := s.AckOutboxEvent(ctx, "evt-retry", "w1", "retry", retryAt, "timeout", now); err != nil {
		t.Fatalf("AckOutboxEvent retry: %v", err)
	}

	// Before retry time — not available.
	leased, _ := s.LeaseOutboxEvents(ctx, "w1", 10, 5*time.Minute, now.Add(5*time.Minute))
	if len(leased) != 0 {
		t.Fatalf("len(leased) = %d before retry time, want 0", len(leased))
	}

	// After retry time — available again.
	leased, _ = s.LeaseOutboxEvents(ctx, "w1", 10, 5*time.Minute, now.Add(15*time.Minute))
	if len(leased) != 1 {
		t.Fatalf("len(leased) = %d after retry time, want 1", len(leased))
	}
}

func TestAckOutboxEvent_Dead(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()
	now := testTime

	_ = s.Enqueue(ctx, storage.OutboxEvent{
		ID: "evt-dead", EventType: "invite.created",
		PayloadJSON: []byte(`{}`), CreatedAt: now,
	})
	_, _ = s.LeaseOutboxEvents(ctx, "w1", 10, 5*time.Minute, now)

	if err := s.AckOutboxEvent(ctx, "evt-dead", "w1", "dead", time.Time{}, "exhausted retries", now); err != nil {
		t.Fatalf("AckOutboxEvent dead: %v", err)
	}

	// Dead events should not be re-leased.
	leased, _ := s.LeaseOutboxEvents(ctx, "w1", 10, 5*time.Minute, now.Add(24*time.Hour))
	if len(leased) != 0 {
		t.Fatalf("len(leased) = %d, want 0 (event is dead)", len(leased))
	}
}

func TestLeaseOutboxEvents_Empty(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)

	leased, err := s.LeaseOutboxEvents(context.Background(), "w1", 10, 5*time.Minute, testTime)
	if err != nil {
		t.Fatalf("LeaseOutboxEvents: %v", err)
	}
	if len(leased) != 0 {
		t.Fatalf("len(leased) = %d, want 0", len(leased))
	}
}

func TestOpen_EmptyPath(t *testing.T) {
	t.Parallel()
	_, err := Open("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}
