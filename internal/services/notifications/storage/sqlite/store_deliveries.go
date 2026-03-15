package sqlite

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/notifications/storage"
)

// PutDelivery upserts one delivery attempt state row.
func (s *Store) PutDelivery(ctx context.Context, record storage.DeliveryRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	normalized, err := normalizeDeliveryRecord(record)
	if err != nil {
		return err
	}
	return putDeliveryExec(ctx, s.sqlDB, normalized)
}

// ListPendingDeliveries lists due channel deliveries ordered by next-attempt time.
func (s *Store) ListPendingDeliveries(ctx context.Context, channel storage.DeliveryChannel, limit int, now time.Time) ([]storage.DeliveryRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	channel = storage.DeliveryChannel(strings.TrimSpace(string(channel)))
	if channel == "" {
		return nil, fmt.Errorf("delivery channel is required")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}
	if now.IsZero() {
		return nil, fmt.Errorf("now is required")
	}

	rows, err := s.sqlDB.QueryContext(ctx, `
SELECT notification_id, channel, status, attempt_count, next_attempt_at, last_error, created_at, updated_at, delivered_at
FROM notification_deliveries
WHERE channel = ?
  AND status IN (?, ?)
  AND next_attempt_at <= ?
ORDER BY next_attempt_at ASC, notification_id ASC
LIMIT ?
`, channel, storage.DeliveryStatusPending, storage.DeliveryStatusFailed, sqliteutil.ToMillis(now.UTC()), limit)
	if err != nil {
		return nil, fmt.Errorf("list pending deliveries: %w", err)
	}
	defer rows.Close()

	results := make([]storage.DeliveryRecord, 0, limit)
	for rows.Next() {
		record, scanErr := scanDelivery(rows.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("scan pending delivery row: %w", scanErr)
		}
		results = append(results, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending delivery rows: %w", err)
	}
	return results, nil
}

// MarkDeliveryRetry records one failed delivery attempt and schedules next retry.
func (s *Store) MarkDeliveryRetry(ctx context.Context, notificationID string, channel storage.DeliveryChannel, attemptCount int, nextAttemptAt time.Time, lastError string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	notificationID = strings.TrimSpace(notificationID)
	channel = storage.DeliveryChannel(strings.TrimSpace(string(channel)))
	lastError = strings.TrimSpace(lastError)
	if notificationID == "" {
		return fmt.Errorf("notification id is required")
	}
	if channel == "" {
		return fmt.Errorf("delivery channel is required")
	}
	if attemptCount < 0 {
		return fmt.Errorf("attempt count must be non-negative")
	}
	if nextAttemptAt.IsZero() {
		return fmt.Errorf("next attempt at is required")
	}

	now := time.Now().UTC()
	result, err := s.sqlDB.ExecContext(ctx, `
UPDATE notification_deliveries
SET status = ?, attempt_count = ?, next_attempt_at = ?, last_error = ?, updated_at = ?, delivered_at = NULL
WHERE notification_id = ? AND channel = ?
`, storage.DeliveryStatusFailed, attemptCount, sqliteutil.ToMillis(nextAttemptAt.UTC()), lastError, sqliteutil.ToMillis(now), notificationID, channel)
	if err != nil {
		return fmt.Errorf("mark delivery retry: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark delivery retry rows affected: %w", err)
	}
	if affected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// MarkDeliverySucceeded records successful channel delivery.
func (s *Store) MarkDeliverySucceeded(ctx context.Context, notificationID string, channel storage.DeliveryChannel, deliveredAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	notificationID = strings.TrimSpace(notificationID)
	channel = storage.DeliveryChannel(strings.TrimSpace(string(channel)))
	if notificationID == "" {
		return fmt.Errorf("notification id is required")
	}
	if channel == "" {
		return fmt.Errorf("delivery channel is required")
	}
	if deliveredAt.IsZero() {
		return fmt.Errorf("delivered at is required")
	}

	now := deliveredAt.UTC()
	result, err := s.sqlDB.ExecContext(ctx, `
UPDATE notification_deliveries
SET status = ?, updated_at = ?, delivered_at = ?, last_error = ''
WHERE notification_id = ? AND channel = ?
`, storage.DeliveryStatusDelivered, sqliteutil.ToMillis(now), sqliteutil.ToMillis(now), notificationID, channel)
	if err != nil {
		return fmt.Errorf("mark delivery succeeded: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark delivery succeeded rows affected: %w", err)
	}
	if affected == 0 {
		return storage.ErrNotFound
	}
	return nil
}
