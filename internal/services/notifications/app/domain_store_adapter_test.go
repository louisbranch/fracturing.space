package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/notifications/domain"
	"github.com/louisbranch/fracturing.space/internal/services/notifications/storage"
)

func TestPutNotification_UsesAtomicBootstrapWriteWhenAvailable(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 21, 21, 55, 0, 0, time.UTC)
	store := newAtomicCapableStore()
	store.atomicWriteErr = errors.New("atomic write failed")
	store.putDeliveryErr = errors.New("delivery write failed")
	adapter := newDomainStoreAdapter(store, store, true)

	err := adapter.PutNotification(context.Background(), domain.Notification{
		ID:              "notif-1",
		RecipientUserID: "user-1",
		MessageType:     "campaign.invite",
		PayloadJSON:     `{"invite_id":"inv-1"}`,
		DedupeKey:       "invite:inv-1",
		Source:          "game",
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err == nil {
		t.Fatal("expected put notification to fail")
	}
	if store.atomicWriteCalls != 1 {
		t.Fatalf("atomic write calls = %d, want 1", store.atomicWriteCalls)
	}
	if got := len(store.notifications); got != 0 {
		t.Fatalf("notifications persisted = %d, want 0 on atomic failure", got)
	}
	if got := len(store.deliveries); got != 0 {
		t.Fatalf("deliveries persisted = %d, want 0 on atomic failure", got)
	}
}

func TestPutNotification_GenericMessageTypeCreatesInAppDeliveryOnly(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 21, 22, 5, 0, 0, time.UTC)
	store := newAtomicCapableStore()
	adapter := newDomainStoreAdapter(store, store, true)

	err := adapter.PutNotification(context.Background(), domain.Notification{
		ID:              "notif-generic",
		RecipientUserID: "user-1",
		MessageType:     "campaign.invite",
		PayloadJSON:     `{"invite_id":"inv-1"}`,
		DedupeKey:       "invite:inv-1",
		Source:          "game",
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		t.Fatalf("put generic notification: %v", err)
	}
	if store.atomicWriteCalls != 1 {
		t.Fatalf("atomic write calls = %d, want 1", store.atomicWriteCalls)
	}
	if len(store.lastAtomicDeliveries) != 1 {
		t.Fatalf("delivery rows = %d, want 1", len(store.lastAtomicDeliveries))
	}
	delivery := store.lastAtomicDeliveries[0]
	if delivery.Channel != storage.DeliveryChannelInApp {
		t.Fatalf("channel = %q, want %q", delivery.Channel, storage.DeliveryChannelInApp)
	}
	if delivery.Status != storage.DeliveryStatusDelivered {
		t.Fatalf("status = %q, want %q", delivery.Status, storage.DeliveryStatusDelivered)
	}
}

func TestPutNotification_OnboardingMessageTypeCreatesEmailDeliveryOnly(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 21, 22, 10, 0, 0, time.UTC)
	store := newAtomicCapableStore()
	adapter := newDomainStoreAdapter(store, store, true)

	err := adapter.PutNotification(context.Background(), domain.Notification{
		ID:              "notif-onboarding-email",
		RecipientUserID: "user-1",
		MessageType:     domain.MessageTypeOnboardingWelcome,
		PayloadJSON:     `{"signup_method":"passkey"}`,
		DedupeKey:       "welcome:user:user-1:v1",
		Source:          "system",
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		t.Fatalf("put onboarding notification: %v", err)
	}
	if len(store.lastAtomicDeliveries) != 1 {
		t.Fatalf("delivery rows = %d, want 1", len(store.lastAtomicDeliveries))
	}
	delivery := store.lastAtomicDeliveries[0]
	if delivery.Channel != storage.DeliveryChannelEmail {
		t.Fatalf("channel = %q, want %q", delivery.Channel, storage.DeliveryChannelEmail)
	}
	if delivery.Status != storage.DeliveryStatusPending {
		t.Fatalf("status = %q, want %q", delivery.Status, storage.DeliveryStatusPending)
	}
}

func TestPutNotification_OnboardingMessageTypeSkippedWhenEmailDisabled(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 21, 22, 15, 0, 0, time.UTC)
	store := newAtomicCapableStore()
	adapter := newDomainStoreAdapter(store, store, false)

	err := adapter.PutNotification(context.Background(), domain.Notification{
		ID:              "notif-onboarding-skipped",
		RecipientUserID: "user-1",
		MessageType:     domain.MessageTypeOnboardingWelcomeV1,
		PayloadJSON:     `{"signup_method":"magic_link"}`,
		DedupeKey:       "welcome:user:user-1:v1",
		Source:          "system",
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		t.Fatalf("put onboarding notification with email disabled: %v", err)
	}
	if len(store.lastAtomicDeliveries) != 1 {
		t.Fatalf("delivery rows = %d, want 1", len(store.lastAtomicDeliveries))
	}
	delivery := store.lastAtomicDeliveries[0]
	if delivery.Channel != storage.DeliveryChannelEmail {
		t.Fatalf("channel = %q, want %q", delivery.Channel, storage.DeliveryChannelEmail)
	}
	if delivery.Status != storage.DeliveryStatusSkipped {
		t.Fatalf("status = %q, want %q", delivery.Status, storage.DeliveryStatusSkipped)
	}
}

type atomicCapableStore struct {
	notifications map[string]storage.NotificationRecord
	deliveries    []storage.DeliveryRecord

	lastAtomicNotification storage.NotificationRecord
	lastAtomicDeliveries   []storage.DeliveryRecord

	putDeliveryErr error
	atomicWriteErr error

	atomicWriteCalls int
}

func newAtomicCapableStore() *atomicCapableStore {
	return &atomicCapableStore{
		notifications: make(map[string]storage.NotificationRecord),
	}
}

func (s *atomicCapableStore) PutNotification(_ context.Context, record storage.NotificationRecord) error {
	s.notifications[record.ID] = record
	return nil
}

func (s *atomicCapableStore) GetNotificationByRecipientAndDedupeKey(_ context.Context, recipientUserID string, dedupeKey string) (storage.NotificationRecord, error) {
	for _, notification := range s.notifications {
		if notification.RecipientUserID == recipientUserID && notification.DedupeKey == dedupeKey {
			return notification, nil
		}
	}
	return storage.NotificationRecord{}, storage.ErrNotFound
}

func (s *atomicCapableStore) GetNotificationByRecipientAndID(_ context.Context, recipientUserID string, notificationID string) (storage.NotificationRecord, error) {
	for _, notification := range s.notifications {
		if notification.ID == notificationID && notification.RecipientUserID == recipientUserID {
			return notification, nil
		}
	}
	return storage.NotificationRecord{}, storage.ErrNotFound
}

func (s *atomicCapableStore) ListNotificationsByRecipient(_ context.Context, _ string, _ int, _ string) (storage.NotificationPage, error) {
	return storage.NotificationPage{}, nil
}

func (s *atomicCapableStore) CountUnreadNotificationsByRecipient(_ context.Context, recipientUserID string) (int, error) {
	unreadCount := 0
	for _, notification := range s.notifications {
		if notification.RecipientUserID != recipientUserID {
			continue
		}
		if notification.ReadAt == nil {
			unreadCount++
		}
	}
	return unreadCount, nil
}

func (s *atomicCapableStore) MarkNotificationRead(_ context.Context, _, _ string, _ time.Time) (storage.NotificationRecord, error) {
	return storage.NotificationRecord{}, storage.ErrNotFound
}

func (s *atomicCapableStore) PutDelivery(_ context.Context, record storage.DeliveryRecord) error {
	if s.putDeliveryErr != nil {
		return s.putDeliveryErr
	}
	s.deliveries = append(s.deliveries, record)
	return nil
}

func (s *atomicCapableStore) ListPendingDeliveries(_ context.Context, _ storage.DeliveryChannel, _ int, _ time.Time) ([]storage.DeliveryRecord, error) {
	return nil, nil
}

func (s *atomicCapableStore) MarkDeliveryRetry(_ context.Context, _ string, _ storage.DeliveryChannel, _ int, _ time.Time, _ string) error {
	return nil
}

func (s *atomicCapableStore) MarkDeliverySucceeded(_ context.Context, _ string, _ storage.DeliveryChannel, _ time.Time) error {
	return nil
}

func (s *atomicCapableStore) PutNotificationWithDeliveries(_ context.Context, notification storage.NotificationRecord, deliveries []storage.DeliveryRecord) error {
	s.atomicWriteCalls++
	s.lastAtomicNotification = notification
	s.lastAtomicDeliveries = append(s.lastAtomicDeliveries[:0], deliveries...)
	if s.atomicWriteErr != nil {
		return s.atomicWriteErr
	}
	return nil
}
