package game

import (
	"context"
	"testing"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestLeaseIntegrationOutboxEvents_Success(t *testing.T) {
	store := newFakeIntegrationOutboxEventStore()
	now := time.Date(2026, 3, 9, 14, 0, 0, 0, time.UTC)
	store.outbox["evt-1"] = storage.IntegrationOutboxEvent{
		ID:            "evt-1",
		EventType:     "game.invite.created.v1",
		PayloadJSON:   `{"invite_id":"invite-1","campaign_id":"campaign-1","recipient_user_id":"user-2"}`,
		DedupeKey:     "invite:invite-1:created",
		Status:        storage.IntegrationOutboxStatusPending,
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	svc := NewIntegrationService(store)
	resp, err := svc.LeaseIntegrationOutboxEvents(context.Background(), &gamev1.LeaseIntegrationOutboxEventsRequest{
		Consumer:   "worker-1",
		Limit:      10,
		LeaseTtlMs: int64((5 * time.Minute) / time.Millisecond),
		Now:        timestamppb.New(now),
	})
	if err != nil {
		t.Fatalf("lease integration outbox events: %v", err)
	}
	if len(resp.GetEvents()) != 1 {
		t.Fatalf("events len = %d, want 1", len(resp.GetEvents()))
	}
	if resp.GetEvents()[0].GetId() != "evt-1" {
		t.Fatalf("event id = %q, want %q", resp.GetEvents()[0].GetId(), "evt-1")
	}
	if resp.GetEvents()[0].GetStatus() != storage.IntegrationOutboxStatusLeased {
		t.Fatalf("event status = %q, want %q", resp.GetEvents()[0].GetStatus(), storage.IntegrationOutboxStatusLeased)
	}
}
