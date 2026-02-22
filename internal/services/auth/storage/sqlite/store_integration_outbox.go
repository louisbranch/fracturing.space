package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
)

func (s *Store) EnqueueIntegrationOutboxEvent(ctx context.Context, event storage.IntegrationOutboxEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	return enqueueIntegrationOutboxEvent(ctx, s.sqlDB, event)
}

// GetIntegrationOutboxEvent returns one integration outbox event by ID.
func (s *Store) GetIntegrationOutboxEvent(ctx context.Context, id string) (storage.IntegrationOutboxEvent, error) {
	if err := ctx.Err(); err != nil {
		return storage.IntegrationOutboxEvent{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.IntegrationOutboxEvent{}, fmt.Errorf("storage is not configured")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return storage.IntegrationOutboxEvent{}, fmt.Errorf("event id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT
	id,
	event_type,
	payload_json,
	dedupe_key,
	status,
	attempt_count,
	next_attempt_at,
	lease_owner,
	lease_expires_at,
	last_error,
	processed_at,
	created_at,
	updated_at
FROM auth_integration_outbox
WHERE id = ?
`, id)
	event, err := scanIntegrationOutboxEvent(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.IntegrationOutboxEvent{}, storage.ErrNotFound
		}
		return storage.IntegrationOutboxEvent{}, fmt.Errorf("get integration outbox event: %w", err)
	}
	return event, nil
}

// LeaseIntegrationOutboxEvents leases due integration outbox events for one worker.
func (s *Store) LeaseIntegrationOutboxEvents(ctx context.Context, consumer string, limit int, now time.Time, leaseTTL time.Duration) ([]storage.IntegrationOutboxEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	consumer = strings.TrimSpace(consumer)
	if consumer == "" {
		return nil, fmt.Errorf("consumer is required")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}
	if leaseTTL <= 0 {
		return nil, fmt.Errorf("lease ttl must be greater than zero")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	leaseExpiresAt := now.Add(leaseTTL)

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("start lease transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	rows, err := tx.QueryContext(ctx, `
SELECT id
FROM auth_integration_outbox
WHERE (
	(status = ? AND next_attempt_at <= ?)
	OR
	(status = ? AND lease_expires_at IS NOT NULL AND lease_expires_at <= ?)
)
ORDER BY next_attempt_at ASC, created_at ASC, id ASC
LIMIT ?
`,
		storage.IntegrationOutboxStatusPending,
		toMillis(now),
		storage.IntegrationOutboxStatusLeased,
		toMillis(now),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("select lease candidates: %w", err)
	}
	candidateIDs := make([]string, 0, limit)
	for rows.Next() {
		var id string
		if scanErr := rows.Scan(&id); scanErr != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan lease candidate: %w", scanErr)
		}
		candidateIDs = append(candidateIDs, id)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate lease candidates: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close lease candidates: %w", err)
	}
	if len(candidateIDs) == 0 {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit empty lease transaction: %w", err)
		}
		return []storage.IntegrationOutboxEvent{}, nil
	}

	leased := make([]storage.IntegrationOutboxEvent, 0, len(candidateIDs))
	for _, id := range candidateIDs {
		result, updateErr := tx.ExecContext(ctx, `
UPDATE auth_integration_outbox
SET
	status = ?,
	lease_owner = ?,
	lease_expires_at = ?,
	updated_at = ?
WHERE id = ?
AND (
	(status = ? AND next_attempt_at <= ?)
	OR
	(status = ? AND lease_expires_at IS NOT NULL AND lease_expires_at <= ?)
)
`,
			storage.IntegrationOutboxStatusLeased,
			consumer,
			toMillis(leaseExpiresAt),
			toMillis(now),
			id,
			storage.IntegrationOutboxStatusPending,
			toMillis(now),
			storage.IntegrationOutboxStatusLeased,
			toMillis(now),
		)
		if updateErr != nil {
			return nil, fmt.Errorf("lease integration outbox event %s: %w", id, updateErr)
		}
		rowsAffected, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			return nil, fmt.Errorf("lease rows affected for %s: %w", id, rowsErr)
		}
		if rowsAffected == 0 {
			continue
		}

		row := tx.QueryRowContext(ctx, `
SELECT
	id,
	event_type,
	payload_json,
	dedupe_key,
	status,
	attempt_count,
	next_attempt_at,
	lease_owner,
	lease_expires_at,
	last_error,
	processed_at,
	created_at,
	updated_at
FROM auth_integration_outbox
WHERE id = ?
`, id)
		event, scanErr := scanIntegrationOutboxEvent(row.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("scan leased integration outbox event %s: %w", id, scanErr)
		}
		leased = append(leased, event)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit lease transaction: %w", err)
	}
	return leased, nil
}

// MarkIntegrationOutboxSucceeded marks one leased integration outbox event as succeeded.
func (s *Store) MarkIntegrationOutboxSucceeded(ctx context.Context, id string, consumer string, processedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}

	id = strings.TrimSpace(id)
	consumer = strings.TrimSpace(consumer)
	if id == "" {
		return fmt.Errorf("event id is required")
	}
	if consumer == "" {
		return fmt.Errorf("consumer is required")
	}
	if processedAt.IsZero() {
		processedAt = time.Now().UTC()
	}
	processedAt = processedAt.UTC()

	result, err := s.sqlDB.ExecContext(ctx, `
UPDATE auth_integration_outbox
SET
	status = ?,
	lease_owner = '',
	lease_expires_at = NULL,
	last_error = '',
	processed_at = ?,
	updated_at = ?
WHERE id = ?
AND status = ?
AND lease_owner = ?
`,
		storage.IntegrationOutboxStatusSucceeded,
		toMillis(processedAt),
		toMillis(processedAt),
		id,
		storage.IntegrationOutboxStatusLeased,
		consumer,
	)
	if err != nil {
		return fmt.Errorf("mark integration outbox succeeded: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark integration outbox succeeded rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// MarkIntegrationOutboxRetry marks one leased integration outbox event for retry.
func (s *Store) MarkIntegrationOutboxRetry(ctx context.Context, id string, consumer string, nextAttemptAt time.Time, lastError string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}

	id = strings.TrimSpace(id)
	consumer = strings.TrimSpace(consumer)
	lastError = strings.TrimSpace(lastError)
	if id == "" {
		return fmt.Errorf("event id is required")
	}
	if consumer == "" {
		return fmt.Errorf("consumer is required")
	}
	if nextAttemptAt.IsZero() {
		return fmt.Errorf("next attempt at is required")
	}
	now := time.Now().UTC()
	nextAttemptAt = nextAttemptAt.UTC()

	result, err := s.sqlDB.ExecContext(ctx, `
UPDATE auth_integration_outbox
SET
	status = ?,
	attempt_count = attempt_count + 1,
	next_attempt_at = ?,
	lease_owner = '',
	lease_expires_at = NULL,
	last_error = ?,
	processed_at = NULL,
	updated_at = ?
WHERE id = ?
AND status = ?
AND lease_owner = ?
`,
		storage.IntegrationOutboxStatusPending,
		toMillis(nextAttemptAt),
		lastError,
		toMillis(now),
		id,
		storage.IntegrationOutboxStatusLeased,
		consumer,
	)
	if err != nil {
		return fmt.Errorf("mark integration outbox retry: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark integration outbox retry rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// MarkIntegrationOutboxDead marks one leased integration outbox event as dead.
func (s *Store) MarkIntegrationOutboxDead(ctx context.Context, id string, consumer string, lastError string, processedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}

	id = strings.TrimSpace(id)
	consumer = strings.TrimSpace(consumer)
	lastError = strings.TrimSpace(lastError)
	if id == "" {
		return fmt.Errorf("event id is required")
	}
	if consumer == "" {
		return fmt.Errorf("consumer is required")
	}
	if processedAt.IsZero() {
		processedAt = time.Now().UTC()
	}
	processedAt = processedAt.UTC()

	result, err := s.sqlDB.ExecContext(ctx, `
UPDATE auth_integration_outbox
SET
	status = ?,
	attempt_count = attempt_count + 1,
	lease_owner = '',
	lease_expires_at = NULL,
	last_error = ?,
	processed_at = ?,
	updated_at = ?
WHERE id = ?
AND status = ?
AND lease_owner = ?
`,
		storage.IntegrationOutboxStatusDead,
		lastError,
		toMillis(processedAt),
		toMillis(processedAt),
		id,
		storage.IntegrationOutboxStatusLeased,
		consumer,
	)
	if err != nil {
		return fmt.Errorf("mark integration outbox dead: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark integration outbox dead rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

type integrationOutboxScanner func(dest ...any) error
