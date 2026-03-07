package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/notifications/storage"
)

// PutNotification persists one notification inbox row.
func (s *Store) PutNotification(ctx context.Context, record storage.NotificationRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	normalized, err := normalizeNotificationRecord(record)
	if err != nil {
		return err
	}
	return putNotificationExec(ctx, s.sqlDB, normalized)
}

// PutNotificationWithDeliveries atomically persists one notification with initial deliveries.
func (s *Store) PutNotificationWithDeliveries(ctx context.Context, notification storage.NotificationRecord, deliveries []storage.DeliveryRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}

	normalizedNotification, err := normalizeNotificationRecord(notification)
	if err != nil {
		return err
	}
	normalizedDeliveries := make([]storage.DeliveryRecord, 0, len(deliveries))
	for _, delivery := range deliveries {
		normalizedDelivery, normalizeErr := normalizeDeliveryRecord(delivery)
		if normalizeErr != nil {
			return normalizeErr
		}
		normalizedDeliveries = append(normalizedDeliveries, normalizedDelivery)
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin notification bootstrap write: %w", err)
	}
	rollbackWith := func(cause error) error {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("%w: rollback notification bootstrap write: %v", cause, rollbackErr)
		}
		return cause
	}

	if err := putNotificationExec(ctx, tx, normalizedNotification); err != nil {
		return rollbackWith(err)
	}
	for _, delivery := range normalizedDeliveries {
		if err := putDeliveryExec(ctx, tx, delivery); err != nil {
			return rollbackWith(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit notification bootstrap write: %w", err)
	}
	return nil
}

// GetNotificationByRecipientAndDedupeKey loads one recipient notification by dedupe key.
func (s *Store) GetNotificationByRecipientAndDedupeKey(ctx context.Context, recipientUserID string, dedupeKey string) (storage.NotificationRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.NotificationRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.NotificationRecord{}, fmt.Errorf("storage is not configured")
	}
	recipientUserID = strings.TrimSpace(recipientUserID)
	dedupeKey = strings.TrimSpace(dedupeKey)
	if recipientUserID == "" {
		return storage.NotificationRecord{}, fmt.Errorf("recipient user id is required")
	}
	if dedupeKey == "" {
		return storage.NotificationRecord{}, storage.ErrNotFound
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT id, recipient_user_id, message_type, payload_json, dedupe_key, source, created_at, updated_at, read_at
FROM notifications
WHERE recipient_user_id = ? AND dedupe_key = ?
`, recipientUserID, dedupeKey)
	record, err := scanNotification(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.NotificationRecord{}, storage.ErrNotFound
		}
		return storage.NotificationRecord{}, fmt.Errorf("get notification by dedupe key: %w", err)
	}
	return record, nil
}

// ListNotificationsByRecipient lists one recipient inbox newest-first with cursor pagination.
func (s *Store) ListNotificationsByRecipient(ctx context.Context, recipientUserID string, pageSize int, pageToken string) (storage.NotificationPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.NotificationPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.NotificationPage{}, fmt.Errorf("storage is not configured")
	}
	recipientUserID = strings.TrimSpace(recipientUserID)
	pageToken = strings.TrimSpace(pageToken)
	if recipientUserID == "" {
		return storage.NotificationPage{}, fmt.Errorf("recipient user id is required")
	}
	if pageSize <= 0 {
		return storage.NotificationPage{}, fmt.Errorf("page size must be greater than zero")
	}

	limit := pageSize + 1
	if pageToken == "" {
		rows, err := s.sqlDB.QueryContext(ctx, `
SELECT n.id, n.recipient_user_id, n.message_type, n.payload_json, n.dedupe_key, n.source, n.created_at, n.updated_at, n.read_at
FROM notifications n
JOIN notification_deliveries d ON d.notification_id = n.id
WHERE n.recipient_user_id = ?
  AND d.channel = ?
  AND d.status = ?
ORDER BY n.created_at DESC, n.id DESC
LIMIT ?
`, recipientUserID, storage.DeliveryChannelInApp, storage.DeliveryStatusDelivered, limit)
		if err != nil {
			return storage.NotificationPage{}, fmt.Errorf("list notifications: %w", err)
		}
		defer rows.Close()
		return collectNotificationPage(rows, pageSize)
	}

	tokenCreatedAt, err := s.notificationCreatedAtByID(ctx, recipientUserID, pageToken)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return storage.NotificationPage{}, nil
		}
		return storage.NotificationPage{}, err
	}

	rows, err := s.sqlDB.QueryContext(ctx, `
SELECT n.id, n.recipient_user_id, n.message_type, n.payload_json, n.dedupe_key, n.source, n.created_at, n.updated_at, n.read_at
FROM notifications n
JOIN notification_deliveries d ON d.notification_id = n.id
WHERE n.recipient_user_id = ?
  AND d.channel = ?
  AND d.status = ?
  AND (n.created_at < ? OR (n.created_at = ? AND n.id < ?))
ORDER BY n.created_at DESC, n.id DESC
LIMIT ?
`, recipientUserID, storage.DeliveryChannelInApp, storage.DeliveryStatusDelivered, toMillis(tokenCreatedAt), toMillis(tokenCreatedAt), pageToken, limit)
	if err != nil {
		return storage.NotificationPage{}, fmt.Errorf("list notifications with token: %w", err)
	}
	defer rows.Close()
	return collectNotificationPage(rows, pageSize)
}

// CountUnreadNotificationsByRecipient returns unread inbox count for one recipient.
func (s *Store) CountUnreadNotificationsByRecipient(ctx context.Context, recipientUserID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if s == nil || s.sqlDB == nil {
		return 0, fmt.Errorf("storage is not configured")
	}
	recipientUserID = strings.TrimSpace(recipientUserID)
	if recipientUserID == "" {
		return 0, fmt.Errorf("recipient user id is required")
	}

	var unreadCount int
	if err := s.sqlDB.QueryRowContext(ctx, `
SELECT COUNT(1)
FROM notifications n
JOIN notification_deliveries d ON d.notification_id = n.id
WHERE n.recipient_user_id = ?
  AND n.read_at IS NULL
  AND d.channel = ?
  AND d.status = ?
`, recipientUserID, storage.DeliveryChannelInApp, storage.DeliveryStatusDelivered).Scan(&unreadCount); err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return unreadCount, nil
}

// MarkNotificationRead marks one notification row as read for a recipient.
func (s *Store) MarkNotificationRead(ctx context.Context, recipientUserID string, notificationID string, readAt time.Time) (storage.NotificationRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.NotificationRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.NotificationRecord{}, fmt.Errorf("storage is not configured")
	}
	recipientUserID = strings.TrimSpace(recipientUserID)
	notificationID = strings.TrimSpace(notificationID)
	if recipientUserID == "" {
		return storage.NotificationRecord{}, fmt.Errorf("recipient user id is required")
	}
	if notificationID == "" {
		return storage.NotificationRecord{}, fmt.Errorf("notification id is required")
	}

	now := readAt.UTC()
	result, err := s.sqlDB.ExecContext(ctx, `
UPDATE notifications
SET read_at = ?, updated_at = ?
WHERE recipient_user_id = ?
  AND id = ?
  AND EXISTS (
    SELECT 1
    FROM notification_deliveries d
    WHERE d.notification_id = notifications.id
      AND d.channel = ?
      AND d.status = ?
  )
`, toMillis(now), toMillis(now), recipientUserID, notificationID, storage.DeliveryChannelInApp, storage.DeliveryStatusDelivered)
	if err != nil {
		return storage.NotificationRecord{}, fmt.Errorf("mark notification read: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return storage.NotificationRecord{}, fmt.Errorf("mark notification read rows affected: %w", err)
	}
	if affected == 0 {
		return storage.NotificationRecord{}, storage.ErrNotFound
	}
	return s.GetNotificationByRecipientAndID(ctx, recipientUserID, notificationID)
}

func (s *Store) notificationCreatedAtByID(ctx context.Context, recipientUserID string, notificationID string) (time.Time, error) {
	row := s.sqlDB.QueryRowContext(ctx, `
SELECT n.created_at
FROM notifications n
JOIN notification_deliveries d ON d.notification_id = n.id
WHERE n.recipient_user_id = ?
  AND n.id = ?
  AND d.channel = ?
  AND d.status = ?
`, recipientUserID, notificationID, storage.DeliveryChannelInApp, storage.DeliveryStatusDelivered)
	var createdAtMillis int64
	if err := row.Scan(&createdAtMillis); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return time.Time{}, storage.ErrNotFound
		}
		return time.Time{}, fmt.Errorf("lookup notification cursor: %w", err)
	}
	return fromMillis(createdAtMillis), nil
}

// GetNotificationByRecipientAndID loads one recipient notification by id.
func (s *Store) GetNotificationByRecipientAndID(ctx context.Context, recipientUserID string, notificationID string) (storage.NotificationRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.NotificationRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.NotificationRecord{}, fmt.Errorf("storage is not configured")
	}
	recipientUserID = strings.TrimSpace(recipientUserID)
	notificationID = strings.TrimSpace(notificationID)
	if recipientUserID == "" {
		return storage.NotificationRecord{}, fmt.Errorf("recipient user id is required")
	}
	if notificationID == "" {
		return storage.NotificationRecord{}, fmt.Errorf("notification id is required")
	}
	row := s.sqlDB.QueryRowContext(ctx, `
SELECT n.id, n.recipient_user_id, n.message_type, n.payload_json, n.dedupe_key, n.source, n.created_at, n.updated_at, n.read_at
FROM notifications n
JOIN notification_deliveries d ON d.notification_id = n.id
WHERE n.recipient_user_id = ?
  AND n.id = ?
  AND d.channel = ?
  AND d.status = ?
`, recipientUserID, notificationID, storage.DeliveryChannelInApp, storage.DeliveryStatusDelivered)
	record, err := scanNotification(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.NotificationRecord{}, storage.ErrNotFound
		}
		return storage.NotificationRecord{}, fmt.Errorf("get notification by id: %w", err)
	}
	return record, nil
}
