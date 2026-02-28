package server

import (
	"context"
	"errors"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/notifications/domain"
	"github.com/louisbranch/fracturing.space/internal/services/notifications/storage"
)

type domainStoreAdapter struct {
	notificationStore    storage.NotificationStore
	deliveryStore        storage.DeliveryStore
	emailDeliveryEnabled bool
}

func newDomainStoreAdapter(notificationStore storage.NotificationStore, deliveryStore storage.DeliveryStore, emailDeliveryEnabled bool) *domainStoreAdapter {
	return &domainStoreAdapter{
		notificationStore:    notificationStore,
		deliveryStore:        deliveryStore,
		emailDeliveryEnabled: emailDeliveryEnabled,
	}
}

func (a *domainStoreAdapter) GetNotificationByRecipientAndDedupeKey(ctx context.Context, recipientUserID string, dedupeKey string) (domain.Notification, error) {
	if a == nil || a.notificationStore == nil {
		return domain.Notification{}, domain.ErrStoreNotConfigured
	}
	record, err := a.notificationStore.GetNotificationByRecipientAndDedupeKey(ctx, recipientUserID, dedupeKey)
	if err != nil {
		return domain.Notification{}, mapStorageError(err)
	}
	return toDomainNotification(record), nil
}

func (a *domainStoreAdapter) GetNotificationByRecipientAndID(ctx context.Context, recipientUserID string, notificationID string) (domain.Notification, error) {
	if a == nil || a.notificationStore == nil {
		return domain.Notification{}, domain.ErrStoreNotConfigured
	}
	record, err := a.notificationStore.GetNotificationByRecipientAndID(ctx, recipientUserID, notificationID)
	if err != nil {
		return domain.Notification{}, mapStorageError(err)
	}
	return toDomainNotification(record), nil
}

func (a *domainStoreAdapter) PutNotification(ctx context.Context, notification domain.Notification) error {
	if a == nil || a.notificationStore == nil {
		return domain.ErrStoreNotConfigured
	}
	record := toStorageNotification(notification)
	if a.deliveryStore == nil {
		return mapStorageError(a.notificationStore.PutNotification(ctx, record))
	}

	baseTime := notification.CreatedAt.UTC()
	if baseTime.IsZero() {
		baseTime = time.Now().UTC()
	}

	policy := domain.ResolveDeliveryPolicy(notification.MessageType)
	deliveries := make([]storage.DeliveryRecord, 0, 2)
	if policy.InApp {
		deliveries = append(deliveries, storage.DeliveryRecord{
			NotificationID: notification.ID,
			Channel:        storage.DeliveryChannelInApp,
			Status:         storage.DeliveryStatusDelivered,
			AttemptCount:   1,
			NextAttemptAt:  baseTime,
			LastError:      "",
			CreatedAt:      baseTime,
			UpdatedAt:      baseTime,
			DeliveredAt:    &baseTime,
		})
	}
	if policy.Email {
		emailStatus := storage.DeliveryStatusPending
		emailDeliveredAt := (*time.Time)(nil)
		emailLastError := ""
		if !a.emailDeliveryEnabled {
			emailStatus = storage.DeliveryStatusSkipped
			emailDeliveredAt = &baseTime
			emailLastError = "email delivery disabled"
		}
		deliveries = append(deliveries, storage.DeliveryRecord{
			NotificationID: notification.ID,
			Channel:        storage.DeliveryChannelEmail,
			Status:         emailStatus,
			AttemptCount:   0,
			NextAttemptAt:  baseTime,
			LastError:      emailLastError,
			CreatedAt:      baseTime,
			UpdatedAt:      baseTime,
			DeliveredAt:    emailDeliveredAt,
		})
	}

	if bootstrapStore, ok := a.notificationStore.(storage.NotificationBootstrapStore); ok {
		if err := bootstrapStore.PutNotificationWithDeliveries(ctx, record, deliveries); err != nil {
			return mapStorageError(err)
		}
		return nil
	}

	if err := a.notificationStore.PutNotification(ctx, record); err != nil {
		return mapStorageError(err)
	}

	for _, delivery := range deliveries {
		if err := a.deliveryStore.PutDelivery(ctx, delivery); err != nil && !errors.Is(err, storage.ErrConflict) {
			return mapStorageError(err)
		}
	}
	return nil
}

func (a *domainStoreAdapter) ListNotificationsByRecipient(ctx context.Context, recipientUserID string, pageSize int, pageToken string) (domain.NotificationPage, error) {
	if a == nil || a.notificationStore == nil {
		return domain.NotificationPage{}, domain.ErrStoreNotConfigured
	}
	page, err := a.notificationStore.ListNotificationsByRecipient(ctx, recipientUserID, pageSize, pageToken)
	if err != nil {
		return domain.NotificationPage{}, mapStorageError(err)
	}
	result := domain.NotificationPage{
		Notifications: make([]domain.Notification, 0, len(page.Notifications)),
		NextPageToken: page.NextPageToken,
	}
	for _, record := range page.Notifications {
		result.Notifications = append(result.Notifications, toDomainNotification(record))
	}
	return result, nil
}

func (a *domainStoreAdapter) CountUnreadNotificationsByRecipient(ctx context.Context, recipientUserID string) (int, error) {
	if a == nil || a.notificationStore == nil {
		return 0, domain.ErrStoreNotConfigured
	}
	unreadCount, err := a.notificationStore.CountUnreadNotificationsByRecipient(ctx, recipientUserID)
	if err != nil {
		return 0, mapStorageError(err)
	}
	return unreadCount, nil
}

func (a *domainStoreAdapter) MarkNotificationRead(ctx context.Context, recipientUserID string, notificationID string, readAt time.Time) (domain.Notification, error) {
	if a == nil || a.notificationStore == nil {
		return domain.Notification{}, domain.ErrStoreNotConfigured
	}
	record, err := a.notificationStore.MarkNotificationRead(ctx, recipientUserID, notificationID, readAt)
	if err != nil {
		return domain.Notification{}, mapStorageError(err)
	}
	return toDomainNotification(record), nil
}

func toStorageNotification(notification domain.Notification) storage.NotificationRecord {
	return storage.NotificationRecord{
		ID:              notification.ID,
		RecipientUserID: notification.RecipientUserID,
		MessageType:     notification.MessageType,
		PayloadJSON:     notification.PayloadJSON,
		DedupeKey:       notification.DedupeKey,
		Source:          notification.Source,
		CreatedAt:       notification.CreatedAt,
		UpdatedAt:       notification.UpdatedAt,
		ReadAt:          notification.ReadAt,
	}
}

func toDomainNotification(record storage.NotificationRecord) domain.Notification {
	return domain.Notification{
		ID:              record.ID,
		RecipientUserID: record.RecipientUserID,
		MessageType:     record.MessageType,
		PayloadJSON:     record.PayloadJSON,
		DedupeKey:       record.DedupeKey,
		Source:          record.Source,
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
		ReadAt:          record.ReadAt,
	}
}

func mapStorageError(err error) error {
	switch {
	case errors.Is(err, storage.ErrNotFound):
		return domain.ErrNotFound
	case errors.Is(err, storage.ErrConflict):
		return domain.ErrConflict
	default:
		return err
	}
}
