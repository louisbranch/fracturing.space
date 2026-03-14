package sqlite

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	gameintegration "github.com/louisbranch/fracturing.space/internal/services/game/integration"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestAppendEvent_EnqueuesInviteIntegrationOutbox(t *testing.T) {
	store := openTestEventsStore(t)
	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	payloadJSON, err := json.Marshal(invite.CreatePayload{
		InviteID:               ids.InviteID("invite-1"),
		ParticipantID:          ids.ParticipantID("seat-1"),
		RecipientUserID:        ids.UserID("user-2"),
		CreatedByParticipantID: ids.ParticipantID("owner-1"),
		Status:                 string(invite.StatusPending),
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	storedEvent, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  ids.CampaignID("campaign-1"),
		Type:        invite.EventTypeCreated,
		Timestamp:   now,
		ActorType:   event.ActorTypeParticipant,
		ActorID:     "owner-1",
		EntityType:  "invite",
		EntityID:    "invite-1",
		PayloadJSON: payloadJSON,
	})
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	leased, err := store.LeaseIntegrationOutboxEvents(context.Background(), "worker-1", 1, storedEvent.Timestamp, time.Minute)
	if err != nil {
		t.Fatalf("lease integration outbox events: %v", err)
	}
	if len(leased) != 1 {
		t.Fatalf("leased len = %d, want 1", len(leased))
	}
	outboxEvent := leased[0]
	if outboxEvent.EventType != gameintegration.InviteNotificationCreatedOutboxEventType {
		t.Fatalf("event type = %q, want %q", outboxEvent.EventType, gameintegration.InviteNotificationCreatedOutboxEventType)
	}
	if outboxEvent.DedupeKey != gameintegration.InviteCreatedNotificationDedupeKey("invite-1") {
		t.Fatalf("dedupe key = %q, want %q", outboxEvent.DedupeKey, gameintegration.InviteCreatedNotificationDedupeKey("invite-1"))
	}
	if outboxEvent.Status != storage.IntegrationOutboxStatusLeased {
		t.Fatalf("status = %q, want %q", outboxEvent.Status, storage.IntegrationOutboxStatusLeased)
	}
	if !outboxEvent.NextAttemptAt.Equal(storedEvent.Timestamp) {
		t.Fatalf("next attempt at = %v, want %v", outboxEvent.NextAttemptAt, storedEvent.Timestamp)
	}

	var payload gameintegration.InviteNotificationOutboxPayload
	if err := json.Unmarshal([]byte(outboxEvent.PayloadJSON), &payload); err != nil {
		t.Fatalf("unmarshal outbox payload: %v", err)
	}
	if payload.InviteID != "invite-1" || payload.RecipientUserID != "user-2" {
		t.Fatalf("payload = %+v, want invite-1/user-2", payload)
	}
}

func TestAppendEvent_SkipsUntargetedInviteCreatedIntegrationOutbox(t *testing.T) {
	store := openTestEventsStore(t)
	payloadJSON, err := json.Marshal(invite.CreatePayload{
		InviteID:               ids.InviteID("invite-1"),
		ParticipantID:          ids.ParticipantID("seat-1"),
		CreatedByParticipantID: ids.ParticipantID("owner-1"),
		Status:                 string(invite.StatusPending),
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	if _, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  ids.CampaignID("campaign-1"),
		Type:        invite.EventTypeCreated,
		Timestamp:   time.Date(2026, 3, 9, 12, 5, 0, 0, time.UTC),
		ActorType:   event.ActorTypeParticipant,
		ActorID:     "owner-1",
		EntityType:  "invite",
		EntityID:    "invite-1",
		PayloadJSON: payloadJSON,
	}); err != nil {
		t.Fatalf("append event: %v", err)
	}

	leased, err := store.LeaseIntegrationOutboxEvents(context.Background(), "worker-1", 1, time.Date(2026, 3, 9, 12, 5, 0, 0, time.UTC), time.Minute)
	if err != nil {
		t.Fatalf("lease integration outbox events: %v", err)
	}
	if len(leased) != 0 {
		t.Fatalf("leased len = %d, want 0", len(leased))
	}
}

func TestIntegrationOutbox_LeaseRetryAndSucceed(t *testing.T) {
	store := openTestEventsStore(t)
	now := time.Date(2026, 3, 9, 12, 10, 0, 0, time.UTC)
	err := store.EnqueueIntegrationOutboxEvent(context.Background(), storage.IntegrationOutboxEvent{
		ID:            "evt-1",
		EventType:     gameintegration.InviteNotificationClaimedOutboxEventType,
		PayloadJSON:   `{"invite_id":"invite-1","user_id":"user-2"}`,
		DedupeKey:     gameintegration.InviteAcceptedNotificationDedupeKey("invite-1"),
		Status:        storage.IntegrationOutboxStatusPending,
		AttemptCount:  0,
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		t.Fatalf("enqueue integration outbox event: %v", err)
	}

	leased, err := store.LeaseIntegrationOutboxEvents(context.Background(), "worker-1", 10, now, time.Minute)
	if err != nil {
		t.Fatalf("lease integration outbox events: %v", err)
	}
	if len(leased) != 1 {
		t.Fatalf("leased len = %d, want 1", len(leased))
	}
	if leased[0].Status != storage.IntegrationOutboxStatusLeased {
		t.Fatalf("leased status = %q, want %q", leased[0].Status, storage.IntegrationOutboxStatusLeased)
	}

	retryAt := now.Add(30 * time.Second)
	if err := store.MarkIntegrationOutboxRetry(context.Background(), "evt-1", "worker-1", retryAt, "temporary"); err != nil {
		t.Fatalf("mark retry: %v", err)
	}
	retried, err := store.GetIntegrationOutboxEvent(context.Background(), "evt-1")
	if err != nil {
		t.Fatalf("get retried event: %v", err)
	}
	if retried.Status != storage.IntegrationOutboxStatusPending {
		t.Fatalf("status after retry = %q, want %q", retried.Status, storage.IntegrationOutboxStatusPending)
	}
	if retried.AttemptCount != 1 {
		t.Fatalf("attempt count after retry = %d, want 1", retried.AttemptCount)
	}

	leased, err = store.LeaseIntegrationOutboxEvents(context.Background(), "worker-1", 10, retryAt, time.Minute)
	if err != nil {
		t.Fatalf("lease integration outbox events again: %v", err)
	}
	if len(leased) != 1 {
		t.Fatalf("leased len after retry = %d, want 1", len(leased))
	}

	if err := store.MarkIntegrationOutboxSucceeded(context.Background(), "evt-1", "worker-1", retryAt.Add(time.Second)); err != nil {
		t.Fatalf("mark succeeded: %v", err)
	}
	done, err := store.GetIntegrationOutboxEvent(context.Background(), "evt-1")
	if err != nil {
		t.Fatalf("get succeeded event: %v", err)
	}
	if done.Status != storage.IntegrationOutboxStatusSucceeded {
		t.Fatalf("status after success = %q, want %q", done.Status, storage.IntegrationOutboxStatusSucceeded)
	}
	if done.ProcessedAt == nil {
		t.Fatal("expected processed at to be set")
	}
}
