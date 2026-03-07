package sqlite

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// RequeueProjectionApplyOutboxRow transitions one dead outbox row back to
// pending so workers can retry apply after a fix.
func (s *Store) RequeueProjectionApplyOutboxRow(ctx context.Context, campaignID string, seq uint64, now time.Time) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if s == nil || s.sqlDB == nil {
		return false, fmt.Errorf("storage is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return false, fmt.Errorf("campaign id is required")
	}
	if seq == 0 {
		return false, fmt.Errorf("event sequence must be greater than zero")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	result, err := s.sqlDB.ExecContext(
		ctx,
		`UPDATE projection_apply_outbox
		 SET status = 'pending',
		     attempt_count = 0,
		     next_attempt_at = ?,
		     last_error = '',
		     updated_at = ?
		 WHERE campaign_id = ? AND seq = ? AND status = 'dead'`,
		toMillis(now),
		toMillis(now),
		campaignID,
		int64(seq),
	)
	if err != nil {
		return false, fmt.Errorf("requeue dead outbox row %s/%d: %w", campaignID, seq, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("requeue dead outbox row rows affected %s/%d: %w", campaignID, seq, err)
	}
	if affected == 0 {
		return false, nil
	}
	if affected != 1 {
		return false, fmt.Errorf("requeue dead outbox row %s/%d: expected at most 1 row updated, got %d", campaignID, seq, affected)
	}
	return true, nil
}

// RequeueProjectionApplyOutboxDeadRows transitions up to limit dead outbox rows
// back to pending in deterministic retry order.
func (s *Store) RequeueProjectionApplyOutboxDeadRows(ctx context.Context, limit int, now time.Time) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if s == nil || s.sqlDB == nil {
		return 0, fmt.Errorf("storage is not configured")
	}
	if limit <= 0 {
		return 0, fmt.Errorf("outbox requeue limit must be greater than zero")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	result, err := s.sqlDB.ExecContext(
		ctx,
		`WITH to_requeue AS (
			SELECT campaign_id, seq
			FROM projection_apply_outbox
			WHERE status = 'dead'
			ORDER BY next_attempt_at ASC, seq ASC
			LIMIT ?
		)
		UPDATE projection_apply_outbox
		SET status = 'pending',
		    attempt_count = 0,
		    next_attempt_at = ?,
		    last_error = '',
		    updated_at = ?
		WHERE status = 'dead'
		  AND EXISTS (
			  SELECT 1
			  FROM to_requeue
			  WHERE to_requeue.campaign_id = projection_apply_outbox.campaign_id
			    AND to_requeue.seq = projection_apply_outbox.seq
		  )`,
		limit,
		toMillis(now),
		toMillis(now),
	)
	if err != nil {
		return 0, fmt.Errorf("requeue dead outbox rows: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("requeue dead outbox rows affected: %w", err)
	}
	if affected < 0 {
		return 0, fmt.Errorf("requeue dead outbox rows affected returned negative value: %d", affected)
	}
	return int(affected), nil
}
