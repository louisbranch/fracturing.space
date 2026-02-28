package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/notifications/storage"
)

func TestOpenRequiresPath(t *testing.T) {
	t.Parallel()

	if _, err := Open(""); err == nil {
		t.Fatal("expected empty path error")
	}
}

func TestPutGetListNotificationsAndMarkRead(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	now := time.Date(2026, 2, 21, 21, 0, 0, 0, time.UTC)

	inputs := []storage.NotificationRecord{
		{
			ID:              "notif-1",
			RecipientUserID: "user-1",
			MessageType:     "campaign.invite",
			PayloadJSON:     `{"invite_id":"inv-1"}`,
			DedupeKey:       "invite:inv-1",
			Source:          "game",
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			ID:              "notif-2",
			RecipientUserID: "user-1",
			MessageType:     "session.update",
			PayloadJSON:     `{"session_id":"sess-1"}`,
			DedupeKey:       "session:sess-1",
			Source:          "game",
			CreatedAt:       now.Add(2 * time.Minute),
			UpdatedAt:       now.Add(2 * time.Minute),
		},
		{
			ID:              "notif-hidden",
			RecipientUserID: "user-1",
			MessageType:     "auth.onboarding.welcome",
			PayloadJSON:     `{"signup_method":"passkey"}`,
			DedupeKey:       "welcome:user:user-1:v1",
			Source:          "system",
			CreatedAt:       now.Add(4 * time.Minute),
			UpdatedAt:       now.Add(4 * time.Minute),
		},
		{
			ID:              "notif-3",
			RecipientUserID: "user-2",
			MessageType:     "campaign.invite",
			PayloadJSON:     `{"invite_id":"inv-2"}`,
			DedupeKey:       "invite:inv-2",
			Source:          "game",
			CreatedAt:       now.Add(3 * time.Minute),
			UpdatedAt:       now.Add(3 * time.Minute),
		},
	}
	for _, input := range inputs {
		if err := store.PutNotification(context.Background(), input); err != nil {
			t.Fatalf("put notification %s: %v", input.ID, err)
		}
	}
	for _, delivery := range []storage.DeliveryRecord{
		{
			NotificationID: "notif-1",
			Channel:        storage.DeliveryChannelInApp,
			Status:         storage.DeliveryStatusDelivered,
			AttemptCount:   1,
			NextAttemptAt:  now,
			CreatedAt:      now,
			UpdatedAt:      now,
			DeliveredAt:    ptrTime(now),
		},
		{
			NotificationID: "notif-2",
			Channel:        storage.DeliveryChannelInApp,
			Status:         storage.DeliveryStatusDelivered,
			AttemptCount:   1,
			NextAttemptAt:  now.Add(2 * time.Minute),
			CreatedAt:      now.Add(2 * time.Minute),
			UpdatedAt:      now.Add(2 * time.Minute),
			DeliveredAt:    ptrTime(now.Add(2 * time.Minute)),
		},
		{
			NotificationID: "notif-hidden",
			Channel:        storage.DeliveryChannelEmail,
			Status:         storage.DeliveryStatusPending,
			AttemptCount:   0,
			NextAttemptAt:  now.Add(4 * time.Minute),
			CreatedAt:      now.Add(4 * time.Minute),
			UpdatedAt:      now.Add(4 * time.Minute),
		},
		{
			NotificationID: "notif-3",
			Channel:        storage.DeliveryChannelInApp,
			Status:         storage.DeliveryStatusDelivered,
			AttemptCount:   1,
			NextAttemptAt:  now.Add(3 * time.Minute),
			CreatedAt:      now.Add(3 * time.Minute),
			UpdatedAt:      now.Add(3 * time.Minute),
			DeliveredAt:    ptrTime(now.Add(3 * time.Minute)),
		},
	} {
		if err := store.PutDelivery(context.Background(), delivery); err != nil {
			t.Fatalf("put delivery %s/%s: %v", delivery.NotificationID, delivery.Channel, err)
		}
	}

	got, err := store.GetNotificationByRecipientAndDedupeKey(context.Background(), "user-1", "invite:inv-1")
	if err != nil {
		t.Fatalf("get by dedupe key: %v", err)
	}
	if got.ID != "notif-1" {
		t.Fatalf("dedupe lookup id = %q, want %q", got.ID, "notif-1")
	}

	pageOne, err := store.ListNotificationsByRecipient(context.Background(), "user-1", 1, "")
	if err != nil {
		t.Fatalf("list page one: %v", err)
	}
	if len(pageOne.Notifications) != 1 {
		t.Fatalf("page one size = %d, want 1", len(pageOne.Notifications))
	}
	if pageOne.Notifications[0].ID != "notif-2" {
		t.Fatalf("page one id = %q, want %q", pageOne.Notifications[0].ID, "notif-2")
	}
	if pageOne.NextPageToken == "" {
		t.Fatal("expected next page token")
	}

	pageTwo, err := store.ListNotificationsByRecipient(context.Background(), "user-1", 1, pageOne.NextPageToken)
	if err != nil {
		t.Fatalf("list page two: %v", err)
	}
	if len(pageTwo.Notifications) != 1 {
		t.Fatalf("page two size = %d, want 1", len(pageTwo.Notifications))
	}
	if pageTwo.Notifications[0].ID != "notif-1" {
		t.Fatalf("page two id = %q, want %q", pageTwo.Notifications[0].ID, "notif-1")
	}
	if pageTwo.NextPageToken != "" {
		t.Fatalf("page two next page token = %q, want empty", pageTwo.NextPageToken)
	}

	readAt := now.Add(5 * time.Minute)
	read, err := store.MarkNotificationRead(context.Background(), "user-1", "notif-1", readAt)
	if err != nil {
		t.Fatalf("mark read: %v", err)
	}
	if read.ReadAt == nil || !read.ReadAt.Equal(readAt) {
		t.Fatalf("read_at = %v, want %v", read.ReadAt, readAt)
	}
}

func TestCountUnreadNotificationsByRecipient(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	now := time.Date(2026, 2, 21, 21, 8, 0, 0, time.UTC)

	for _, input := range []storage.NotificationRecord{
		{
			ID:              "notif-1",
			RecipientUserID: "user-1",
			MessageType:     "campaign.invite",
			PayloadJSON:     "{}",
			DedupeKey:       "invite:1",
			Source:          "game",
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			ID:              "notif-2",
			RecipientUserID: "user-1",
			MessageType:     "session.update",
			PayloadJSON:     "{}",
			DedupeKey:       "session:1",
			Source:          "game",
			CreatedAt:       now.Add(time.Minute),
			UpdatedAt:       now.Add(time.Minute),
		},
		{
			ID:              "notif-3",
			RecipientUserID: "user-2",
			MessageType:     "campaign.invite",
			PayloadJSON:     "{}",
			DedupeKey:       "invite:2",
			Source:          "game",
			CreatedAt:       now.Add(2 * time.Minute),
			UpdatedAt:       now.Add(2 * time.Minute),
		},
		{
			ID:              "notif-4",
			RecipientUserID: "user-1",
			MessageType:     "auth.onboarding.welcome.v1",
			PayloadJSON:     "{}",
			DedupeKey:       "welcome:user:user-1:v1",
			Source:          "system",
			CreatedAt:       now.Add(3 * time.Minute),
			UpdatedAt:       now.Add(3 * time.Minute),
		},
	} {
		if err := store.PutNotification(context.Background(), input); err != nil {
			t.Fatalf("put notification %s: %v", input.ID, err)
		}
	}
	for _, delivery := range []storage.DeliveryRecord{
		{
			NotificationID: "notif-1",
			Channel:        storage.DeliveryChannelInApp,
			Status:         storage.DeliveryStatusDelivered,
			AttemptCount:   1,
			NextAttemptAt:  now,
			CreatedAt:      now,
			UpdatedAt:      now,
			DeliveredAt:    ptrTime(now),
		},
		{
			NotificationID: "notif-2",
			Channel:        storage.DeliveryChannelInApp,
			Status:         storage.DeliveryStatusDelivered,
			AttemptCount:   1,
			NextAttemptAt:  now.Add(time.Minute),
			CreatedAt:      now.Add(time.Minute),
			UpdatedAt:      now.Add(time.Minute),
			DeliveredAt:    ptrTime(now.Add(time.Minute)),
		},
		{
			NotificationID: "notif-3",
			Channel:        storage.DeliveryChannelInApp,
			Status:         storage.DeliveryStatusDelivered,
			AttemptCount:   1,
			NextAttemptAt:  now.Add(2 * time.Minute),
			CreatedAt:      now.Add(2 * time.Minute),
			UpdatedAt:      now.Add(2 * time.Minute),
			DeliveredAt:    ptrTime(now.Add(2 * time.Minute)),
		},
		{
			NotificationID: "notif-4",
			Channel:        storage.DeliveryChannelEmail,
			Status:         storage.DeliveryStatusPending,
			AttemptCount:   0,
			NextAttemptAt:  now.Add(3 * time.Minute),
			CreatedAt:      now.Add(3 * time.Minute),
			UpdatedAt:      now.Add(3 * time.Minute),
		},
	} {
		if err := store.PutDelivery(context.Background(), delivery); err != nil {
			t.Fatalf("put delivery %s/%s: %v", delivery.NotificationID, delivery.Channel, err)
		}
	}
	if _, err := store.MarkNotificationRead(context.Background(), "user-1", "notif-2", now.Add(3*time.Minute)); err != nil {
		t.Fatalf("mark read: %v", err)
	}

	userOneUnread, err := store.CountUnreadNotificationsByRecipient(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("count unread for user-1: %v", err)
	}
	if userOneUnread != 1 {
		t.Fatalf("user-1 unread = %d, want 1", userOneUnread)
	}

	userTwoUnread, err := store.CountUnreadNotificationsByRecipient(context.Background(), "user-2")
	if err != nil {
		t.Fatalf("count unread for user-2: %v", err)
	}
	if userTwoUnread != 1 {
		t.Fatalf("user-2 unread = %d, want 1", userTwoUnread)
	}

	missingUnread, err := store.CountUnreadNotificationsByRecipient(context.Background(), "user-missing")
	if err != nil {
		t.Fatalf("count unread for missing user: %v", err)
	}
	if missingUnread != 0 {
		t.Fatalf("user-missing unread = %d, want 0", missingUnread)
	}
}

func TestMarkNotificationRead_EmailOnlyNotificationReturnsNotFound(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	now := time.Date(2026, 2, 21, 21, 9, 0, 0, time.UTC)

	if err := store.PutNotification(context.Background(), storage.NotificationRecord{
		ID:              "notif-email-only",
		RecipientUserID: "user-1",
		MessageType:     "auth.onboarding.welcome",
		PayloadJSON:     "{}",
		DedupeKey:       "welcome:user:user-1:v1",
		Source:          "system",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("put notification: %v", err)
	}
	if err := store.PutDelivery(context.Background(), storage.DeliveryRecord{
		NotificationID: "notif-email-only",
		Channel:        storage.DeliveryChannelEmail,
		Status:         storage.DeliveryStatusPending,
		AttemptCount:   0,
		NextAttemptAt:  now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("put email delivery: %v", err)
	}

	_, err := store.MarkNotificationRead(context.Background(), "user-1", "notif-email-only", now.Add(time.Minute))
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("mark read err = %v, want %v", err, storage.ErrNotFound)
	}
}

func TestPutNotificationRejectsRecipientDedupeConflicts(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	now := time.Date(2026, 2, 21, 21, 10, 0, 0, time.UTC)

	if err := store.PutNotification(context.Background(), storage.NotificationRecord{
		ID:              "notif-1",
		RecipientUserID: "user-1",
		MessageType:     "campaign.invite",
		PayloadJSON:     "{}",
		DedupeKey:       "invite:dup",
		Source:          "game",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("put notification: %v", err)
	}

	err := store.PutNotification(context.Background(), storage.NotificationRecord{
		ID:              "notif-2",
		RecipientUserID: "user-1",
		MessageType:     "campaign.invite",
		PayloadJSON:     "{}",
		DedupeKey:       "invite:dup",
		Source:          "game",
		CreatedAt:       now.Add(time.Minute),
		UpdatedAt:       now.Add(time.Minute),
	})
	if !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestPutDeliveryRequiresExistingNotification(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	now := time.Date(2026, 2, 21, 21, 15, 0, 0, time.UTC)

	err := store.PutDelivery(context.Background(), storage.DeliveryRecord{
		NotificationID: "missing",
		Channel:        storage.DeliveryChannelEmail,
		Status:         storage.DeliveryStatusPending,
		AttemptCount:   0,
		NextAttemptAt:  now,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("expected ErrConflict for missing notification parent, got %v", err)
	}
}

func TestPutNotificationWithDeliveries_RollsBackOnDeliveryConflict(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	now := time.Date(2026, 2, 21, 21, 18, 0, 0, time.UTC)

	err := store.PutNotificationWithDeliveries(context.Background(), storage.NotificationRecord{
		ID:              "notif-rollback",
		RecipientUserID: "user-1",
		MessageType:     "campaign.invite",
		PayloadJSON:     "{}",
		DedupeKey:       "invite:rollback",
		Source:          "game",
		CreatedAt:       now,
		UpdatedAt:       now,
	}, []storage.DeliveryRecord{
		{
			NotificationID: "notif-rollback",
			Channel:        storage.DeliveryChannelInApp,
			Status:         storage.DeliveryStatusDelivered,
			AttemptCount:   1,
			NextAttemptAt:  now,
			CreatedAt:      now,
			UpdatedAt:      now,
			DeliveredAt:    &now,
		},
		{
			NotificationID: "missing-parent",
			Channel:        storage.DeliveryChannelEmail,
			Status:         storage.DeliveryStatusPending,
			AttemptCount:   0,
			NextAttemptAt:  now,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	})
	if !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("expected delivery conflict from transactional write, got %v", err)
	}

	if _, lookupErr := store.GetNotificationByRecipientAndDedupeKey(context.Background(), "user-1", "invite:rollback"); !errors.Is(lookupErr, storage.ErrNotFound) {
		t.Fatalf("expected notification rollback, lookup error = %v", lookupErr)
	}

	putErr := store.PutDelivery(context.Background(), storage.DeliveryRecord{
		NotificationID: "notif-rollback",
		Channel:        storage.DeliveryChannelEmail,
		Status:         storage.DeliveryStatusPending,
		AttemptCount:   0,
		NextAttemptAt:  now,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if !errors.Is(putErr, storage.ErrConflict) {
		t.Fatalf("expected parent notification to be absent after rollback, got %v", putErr)
	}
}

func TestDeliveryQueueRetryAndSuccess(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	now := time.Date(2026, 2, 21, 21, 20, 0, 0, time.UTC)
	if err := store.PutNotification(context.Background(), storage.NotificationRecord{
		ID:              "notif-1",
		RecipientUserID: "user-1",
		MessageType:     "campaign.invite",
		PayloadJSON:     "{}",
		DedupeKey:       "invite:inv-1",
		Source:          "game",
		CreatedAt:       now.Add(-2 * time.Minute),
		UpdatedAt:       now.Add(-2 * time.Minute),
	}); err != nil {
		t.Fatalf("put parent notification: %v", err)
	}

	if err := store.PutDelivery(context.Background(), storage.DeliveryRecord{
		NotificationID: "notif-1",
		Channel:        storage.DeliveryChannelEmail,
		Status:         storage.DeliveryStatusPending,
		AttemptCount:   0,
		NextAttemptAt:  now.Add(-time.Minute),
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("put delivery: %v", err)
	}

	pending, err := store.ListPendingDeliveries(context.Background(), storage.DeliveryChannelEmail, 10, now)
	if err != nil {
		t.Fatalf("list pending deliveries: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("pending deliveries = %d, want 1", len(pending))
	}

	nextAttempt := now.Add(5 * time.Minute)
	if err := store.MarkDeliveryRetry(context.Background(), "notif-1", storage.DeliveryChannelEmail, 1, nextAttempt, "temporary email provider failure"); err != nil {
		t.Fatalf("mark delivery retry: %v", err)
	}

	pending, err = store.ListPendingDeliveries(context.Background(), storage.DeliveryChannelEmail, 10, now)
	if err != nil {
		t.Fatalf("list pending deliveries after retry: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending deliveries after retry = %d, want 0", len(pending))
	}

	if err := store.MarkDeliverySucceeded(context.Background(), "notif-1", storage.DeliveryChannelEmail, now.Add(10*time.Minute)); err != nil {
		t.Fatalf("mark delivery succeeded: %v", err)
	}

	pending, err = store.ListPendingDeliveries(context.Background(), storage.DeliveryChannelEmail, 10, now.Add(15*time.Minute))
	if err != nil {
		t.Fatalf("list pending deliveries after success: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending deliveries after success = %d, want 0", len(pending))
	}
}

func ptrTime(value time.Time) *time.Time {
	v := value.UTC()
	return &v
}

func openTempStore(t *testing.T) *Store {
	t.Helper()
	storePath := filepath.Join(t.TempDir(), "notifications.db")
	store, err := Open(storePath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("close store: %v", closeErr)
		}
	})
	return store
}
