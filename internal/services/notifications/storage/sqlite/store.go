package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	sqlitemigrate "github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	"github.com/louisbranch/fracturing.space/internal/services/notifications/storage"
	"github.com/louisbranch/fracturing.space/internal/services/notifications/storage/sqlite/migrations"
	_ "modernc.org/sqlite"
)

// Store provides SQLite-backed persistence for notifications state.
type Store struct {
	sqlDB *sql.DB
}

func toMillis(value time.Time) int64 {
	return value.UTC().UnixMilli()
}

func fromMillis(value int64) time.Time {
	return time.UnixMilli(value).UTC()
}

// Open opens a notifications SQLite store at the provided path.
func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("storage path is required")
	}

	cleanPath := filepath.Clean(path)
	dsn := cleanPath + "?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_pragma=foreign_keys(ON)"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping sqlite db: %w", err)
	}
	if err := ensureForeignKeysEnabled(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	store := &Store{sqlDB: sqlDB}
	if err := store.runMigrations(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	return store, nil
}

// Close closes the underlying SQLite database.
func (s *Store) Close() error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}

func (s *Store) runMigrations() error {
	return sqlitemigrate.ApplyMigrations(s.sqlDB, migrations.FS, "")
}

func ensureForeignKeysEnabled(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("sqlite db is required")
	}
	var enabled int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&enabled); err != nil {
		return fmt.Errorf("check sqlite foreign key pragma: %w", err)
	}
	if enabled != 1 {
		return fmt.Errorf("sqlite foreign keys are disabled")
	}
	return nil
}

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
	return s.getNotificationByRecipientAndID(ctx, recipientUserID, notificationID)
}

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
`, channel, storage.DeliveryStatusPending, storage.DeliveryStatusFailed, toMillis(now.UTC()), limit)
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
`, storage.DeliveryStatusFailed, attemptCount, toMillis(nextAttemptAt.UTC()), lastError, toMillis(now), notificationID, channel)
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
`, storage.DeliveryStatusDelivered, toMillis(now), toMillis(now), notificationID, channel)
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

func (s *Store) getNotificationByRecipientAndID(ctx context.Context, recipientUserID string, notificationID string) (storage.NotificationRecord, error) {
	row := s.sqlDB.QueryRowContext(ctx, `
SELECT id, recipient_user_id, message_type, payload_json, dedupe_key, source, created_at, updated_at, read_at
FROM notifications
WHERE recipient_user_id = ? AND id = ?
`, recipientUserID, notificationID)
	record, err := scanNotification(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.NotificationRecord{}, storage.ErrNotFound
		}
		return storage.NotificationRecord{}, fmt.Errorf("get notification by id: %w", err)
	}
	return record, nil
}

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
		readAt = sql.NullInt64{Int64: toMillis(*record.ReadAt), Valid: true}
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
		toMillis(record.CreatedAt),
		toMillis(record.UpdatedAt),
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
		deliveredAt = sql.NullInt64{Int64: toMillis(*record.DeliveredAt), Valid: true}
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
		toMillis(record.NextAttemptAt),
		record.LastError,
		toMillis(record.CreatedAt),
		toMillis(record.UpdatedAt),
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
	record.CreatedAt = fromMillis(createdAt)
	record.UpdatedAt = fromMillis(updatedAt)
	if readAt.Valid {
		value := fromMillis(readAt.Int64)
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
	record.NextAttemptAt = fromMillis(nextAttemptAt)
	record.CreatedAt = fromMillis(createdAt)
	record.UpdatedAt = fromMillis(updatedAt)
	if deliveredAt.Valid {
		value := fromMillis(deliveredAt.Int64)
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
