package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/notifications/storage"
)

type scanner func(dest ...any) error

type sqlExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func normalizeNotificationRecord(record storage.NotificationRecord) (storage.NotificationRecord, error) {
	record.ID = strings.TrimSpace(record.ID)
	record.RecipientUserID = strings.TrimSpace(record.RecipientUserID)
	record.MessageType = strings.TrimSpace(record.MessageType)
	record.DedupeKey = strings.TrimSpace(record.DedupeKey)
	record.Source = strings.TrimSpace(record.Source)
	record.PayloadJSON = strings.TrimSpace(record.PayloadJSON)
	if record.PayloadJSON == "" {
		record.PayloadJSON = "{}"
	}
	if record.ID == "" {
		return storage.NotificationRecord{}, fmt.Errorf("notification id is required")
	}
	if record.RecipientUserID == "" {
		return storage.NotificationRecord{}, fmt.Errorf("recipient user id is required")
	}
	if record.MessageType == "" {
		return storage.NotificationRecord{}, fmt.Errorf("message type is required")
	}
	if record.CreatedAt.IsZero() {
		return storage.NotificationRecord{}, fmt.Errorf("created_at is required")
	}
	if record.UpdatedAt.IsZero() {
		return storage.NotificationRecord{}, fmt.Errorf("updated_at is required")
	}
	record.CreatedAt = record.CreatedAt.UTC()
	record.UpdatedAt = record.UpdatedAt.UTC()
	if record.ReadAt != nil {
		readAt := record.ReadAt.UTC()
		record.ReadAt = &readAt
	}
	return record, nil
}

func normalizeDeliveryRecord(record storage.DeliveryRecord) (storage.DeliveryRecord, error) {
	record.NotificationID = strings.TrimSpace(record.NotificationID)
	record.Channel = storage.DeliveryChannel(strings.TrimSpace(string(record.Channel)))
	record.Status = storage.DeliveryStatus(strings.TrimSpace(string(record.Status)))
	record.LastError = strings.TrimSpace(record.LastError)
	if record.NotificationID == "" {
		return storage.DeliveryRecord{}, fmt.Errorf("notification id is required")
	}
	if record.Channel == "" {
		return storage.DeliveryRecord{}, fmt.Errorf("delivery channel is required")
	}
	if record.Status == "" {
		return storage.DeliveryRecord{}, fmt.Errorf("delivery status is required")
	}
	if record.NextAttemptAt.IsZero() {
		return storage.DeliveryRecord{}, fmt.Errorf("next attempt at is required")
	}
	if record.CreatedAt.IsZero() {
		return storage.DeliveryRecord{}, fmt.Errorf("created_at is required")
	}
	if record.UpdatedAt.IsZero() {
		return storage.DeliveryRecord{}, fmt.Errorf("updated_at is required")
	}
	record.NextAttemptAt = record.NextAttemptAt.UTC()
	record.CreatedAt = record.CreatedAt.UTC()
	record.UpdatedAt = record.UpdatedAt.UTC()
	if record.DeliveredAt != nil {
		deliveredAt := record.DeliveredAt.UTC()
		record.DeliveredAt = &deliveredAt
	}
	return record, nil
}

func putNotificationExec(ctx context.Context, execer sqlExecer, record storage.NotificationRecord) error {
	var readAt sql.NullInt64
	if record.ReadAt != nil {
		readAt = sql.NullInt64{Int64: sqliteutil.ToMillis(*record.ReadAt), Valid: true}
	}

	_, err := execer.ExecContext(ctx, `
	INSERT INTO notifications (
		id, recipient_user_id, message_type, payload_json, dedupe_key, source, created_at, updated_at, read_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		recipient_user_id = excluded.recipient_user_id,
		message_type = excluded.message_type,
		payload_json = excluded.payload_json,
		dedupe_key = excluded.dedupe_key,
		source = excluded.source,
		created_at = excluded.created_at,
		updated_at = excluded.updated_at,
		read_at = excluded.read_at
	`,
		record.ID,
		record.RecipientUserID,
		record.MessageType,
		record.PayloadJSON,
		record.DedupeKey,
		record.Source,
		sqliteutil.ToMillis(record.CreatedAt),
		sqliteutil.ToMillis(record.UpdatedAt),
		readAt,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return storage.ErrConflict
		}
		return fmt.Errorf("put notification: %w", err)
	}
	return nil
}

func putDeliveryExec(ctx context.Context, execer sqlExecer, record storage.DeliveryRecord) error {
	var deliveredAt sql.NullInt64
	if record.DeliveredAt != nil {
		deliveredAt = sql.NullInt64{Int64: sqliteutil.ToMillis(*record.DeliveredAt), Valid: true}
	}

	_, err := execer.ExecContext(ctx, `
	INSERT INTO notification_deliveries (
		notification_id, channel, status, attempt_count, next_attempt_at, last_error, created_at, updated_at, delivered_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(notification_id, channel) DO UPDATE SET
		status = excluded.status,
		attempt_count = excluded.attempt_count,
		next_attempt_at = excluded.next_attempt_at,
		last_error = excluded.last_error,
		updated_at = excluded.updated_at,
		delivered_at = excluded.delivered_at
	`,
		record.NotificationID,
		record.Channel,
		record.Status,
		record.AttemptCount,
		sqliteutil.ToMillis(record.NextAttemptAt),
		record.LastError,
		sqliteutil.ToMillis(record.CreatedAt),
		sqliteutil.ToMillis(record.UpdatedAt),
		deliveredAt,
	)
	if err != nil {
		if isUniqueConstraintError(err) || isForeignKeyConstraintError(err) {
			return storage.ErrConflict
		}
		return fmt.Errorf("put delivery: %w", err)
	}
	return nil
}

func scanNotification(scan scanner) (storage.NotificationRecord, error) {
	var record storage.NotificationRecord
	var createdAt int64
	var updatedAt int64
	var readAt sql.NullInt64
	if err := scan(
		&record.ID,
		&record.RecipientUserID,
		&record.MessageType,
		&record.PayloadJSON,
		&record.DedupeKey,
		&record.Source,
		&createdAt,
		&updatedAt,
		&readAt,
	); err != nil {
		return storage.NotificationRecord{}, err
	}
	record.CreatedAt = sqliteutil.FromMillis(createdAt)
	record.UpdatedAt = sqliteutil.FromMillis(updatedAt)
	if readAt.Valid {
		value := sqliteutil.FromMillis(readAt.Int64)
		record.ReadAt = &value
	}
	return record, nil
}

func collectNotificationPage(rows *sql.Rows, pageSize int) (storage.NotificationPage, error) {
	page := storage.NotificationPage{
		Notifications: make([]storage.NotificationRecord, 0, pageSize),
	}
	for rows.Next() {
		record, err := scanNotification(rows.Scan)
		if err != nil {
			return storage.NotificationPage{}, fmt.Errorf("scan notification row: %w", err)
		}
		page.Notifications = append(page.Notifications, record)
	}
	if err := rows.Err(); err != nil {
		return storage.NotificationPage{}, fmt.Errorf("iterate notification rows: %w", err)
	}
	if len(page.Notifications) > pageSize {
		page.NextPageToken = page.Notifications[pageSize-1].ID
		page.Notifications = page.Notifications[:pageSize]
	}
	return page, nil
}

func scanDelivery(scan scanner) (storage.DeliveryRecord, error) {
	var record storage.DeliveryRecord
	var nextAttemptAt int64
	var createdAt int64
	var updatedAt int64
	var deliveredAt sql.NullInt64
	if err := scan(
		&record.NotificationID,
		&record.Channel,
		&record.Status,
		&record.AttemptCount,
		&nextAttemptAt,
		&record.LastError,
		&createdAt,
		&updatedAt,
		&deliveredAt,
	); err != nil {
		return storage.DeliveryRecord{}, err
	}
	record.NextAttemptAt = sqliteutil.FromMillis(nextAttemptAt)
	record.CreatedAt = sqliteutil.FromMillis(createdAt)
	record.UpdatedAt = sqliteutil.FromMillis(updatedAt)
	if deliveredAt.Valid {
		value := sqliteutil.FromMillis(deliveredAt.Int64)
		record.DeliveredAt = &value
	}
	return record, nil
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	value := strings.ToLower(err.Error())
	return strings.Contains(value, "unique constraint failed")
}

func isForeignKeyConstraintError(err error) bool {
	if err == nil {
		return false
	}
	value := strings.ToLower(err.Error())
	return strings.Contains(value, "foreign key constraint failed")
}
