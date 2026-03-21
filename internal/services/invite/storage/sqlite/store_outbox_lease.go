package sqlite

import (
	"context"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/invite/storage"
)

func (s *Store) LeaseOutboxEvents(_ context.Context, consumer string, limit int, leaseTTL time.Duration, now time.Time) ([]storage.LeasedOutboxEvent, error) {
	if limit <= 0 {
		limit = 10
	}
	nowStr := now.Format(timeLayout)
	expiresAt := now.Add(leaseTTL).Format(timeLayout)

	tx, err := s.sqlDB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Select candidates: pending, or leased with expired lease.
	rows, err := tx.Query(`
		SELECT id FROM outbox
		WHERE (status = 'pending' AND (next_attempt_at = '' OR next_attempt_at <= ?))
		   OR (status = 'leased' AND lease_expires_at <= ?)
		ORDER BY id ASC
		LIMIT ?
	`, nowStr, nowStr, limit)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}

	// Lease each candidate.
	result := make([]storage.LeasedOutboxEvent, 0, len(ids))
	for _, id := range ids {
		_, err := tx.Exec(`
			UPDATE outbox SET status = 'leased', lease_owner = ?, lease_expires_at = ?, updated_at = ?
			WHERE id = ? AND (status = 'pending' OR (status = 'leased' AND lease_expires_at <= ?))
		`, consumer, expiresAt, nowStr, id, nowStr)
		if err != nil {
			return nil, err
		}
		row := tx.QueryRow(`
			SELECT id, event_type, payload_json, dedupe_key, status, attempt_count, lease_owner, created_at
			FROM outbox WHERE id = ? AND lease_owner = ?
		`, id, consumer)
		var evt storage.LeasedOutboxEvent
		var createdAt string
		if err := row.Scan(&evt.ID, &evt.EventType, &evt.PayloadJSON, &evt.DedupeKey,
			&evt.Status, &evt.AttemptCount, &evt.LeaseOwner, &createdAt); err != nil {
			continue
		}
		evt.CreatedAt, _ = time.Parse(timeLayout, createdAt)
		result = append(result, evt)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Store) AckOutboxEvent(_ context.Context, eventID, consumer, outcome string, nextAttemptAt time.Time, lastError string, now time.Time) error {
	nowStr := now.Format(timeLayout)
	switch outcome {
	case "succeeded":
		_, err := s.sqlDB.Exec(`
			UPDATE outbox SET status = 'succeeded', processed_at = ?, lease_owner = '', lease_expires_at = '', updated_at = ?
			WHERE id = ? AND lease_owner = ?
		`, nowStr, nowStr, eventID, consumer)
		return err
	case "retry":
		nextStr := nextAttemptAt.Format(timeLayout)
		_, err := s.sqlDB.Exec(`
			UPDATE outbox SET status = 'pending', attempt_count = attempt_count + 1, next_attempt_at = ?,
				lease_owner = '', lease_expires_at = '', last_error = ?, updated_at = ?
			WHERE id = ? AND lease_owner = ?
		`, nextStr, lastError, nowStr, eventID, consumer)
		return err
	case "dead":
		_, err := s.sqlDB.Exec(`
			UPDATE outbox SET status = 'dead', attempt_count = attempt_count + 1, processed_at = ?,
				lease_owner = '', lease_expires_at = '', last_error = ?, updated_at = ?
			WHERE id = ? AND lease_owner = ?
		`, nowStr, lastError, nowStr, eventID, consumer)
		return err
	default:
		return nil
	}
}
