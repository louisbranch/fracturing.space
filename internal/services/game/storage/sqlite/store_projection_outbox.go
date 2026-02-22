package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func (s *Store) enqueueProjectionApplyOutbox(ctx context.Context, tx *sql.Tx, evt event.Event) error {
	if !s.projectionApplyOutboxEnabled {
		return nil
	}
	enqueuedAt := time.Now().UTC()
	const enqueueOutboxSQL = `
INSERT INTO projection_apply_outbox (
    campaign_id, seq, event_type, status, attempt_count, next_attempt_at, last_error, updated_at
) VALUES (?, ?, ?, 'pending', 0, ?, '', ?)
ON CONFLICT(campaign_id, seq) DO NOTHING
`
	if _, err := tx.ExecContext(
		ctx,
		enqueueOutboxSQL,
		evt.CampaignID,
		int64(evt.Seq),
		string(evt.Type),
		toMillis(enqueuedAt),
		toMillis(enqueuedAt),
	); err != nil {
		return fmt.Errorf("enqueue projection apply outbox: %w", err)
	}
	return nil
}

type projectionApplyOutboxRow struct {
	CampaignID   string
	Seq          uint64
	EventType    string
	AttemptCount int
}

// ProjectionApplyOutboxSummary reports outbox depth and oldest retry-eligible row.
type ProjectionApplyOutboxSummary struct {
	PendingCount            int
	ProcessingCount         int
	FailedCount             int
	DeadCount               int
	OldestPendingCampaignID string
	OldestPendingSeq        uint64
	OldestPendingAt         time.Time
}

// ProjectionApplyOutboxEntry describes one outbox row for inspection tooling.
type ProjectionApplyOutboxEntry struct {
	CampaignID    string
	Seq           uint64
	EventType     event.Type
	Status        string
	AttemptCount  int
	NextAttemptAt time.Time
	LastError     string
	UpdatedAt     time.Time
}

const (
	outboxDeadLetterThreshold = 8
	outboxProcessingLease     = 2 * time.Minute
)

// GetProjectionApplyOutboxSummary returns queue depth by status and the oldest
// pending/failed row metadata.
func (s *Store) GetProjectionApplyOutboxSummary(ctx context.Context) (ProjectionApplyOutboxSummary, error) {
	if err := ctx.Err(); err != nil {
		return ProjectionApplyOutboxSummary{}, err
	}
	if s == nil || s.sqlDB == nil {
		return ProjectionApplyOutboxSummary{}, fmt.Errorf("storage is not configured")
	}

	summary := ProjectionApplyOutboxSummary{}
	rows, err := s.sqlDB.QueryContext(
		ctx,
		`SELECT status, COUNT(*)
		 FROM projection_apply_outbox
		 GROUP BY status`,
	)
	if err != nil {
		return ProjectionApplyOutboxSummary{}, fmt.Errorf("query outbox summary counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			status string
			count  int
		)
		if err := rows.Scan(&status, &count); err != nil {
			return ProjectionApplyOutboxSummary{}, fmt.Errorf("scan outbox summary count: %w", err)
		}
		switch strings.ToLower(strings.TrimSpace(status)) {
		case "pending":
			summary.PendingCount = count
		case "processing":
			summary.ProcessingCount = count
		case "failed":
			summary.FailedCount = count
		case "dead":
			summary.DeadCount = count
		}
	}
	if err := rows.Err(); err != nil {
		return ProjectionApplyOutboxSummary{}, fmt.Errorf("iterate outbox summary counts: %w", err)
	}

	var (
		campaignID  string
		seq         int64
		nextAttempt int64
	)
	err = s.sqlDB.QueryRowContext(
		ctx,
		`SELECT campaign_id, seq, next_attempt_at
		 FROM projection_apply_outbox
		 WHERE status IN ('pending', 'failed')
		 ORDER BY next_attempt_at ASC, seq ASC
		 LIMIT 1`,
	).Scan(&campaignID, &seq, &nextAttempt)
	if err == nil {
		summary.OldestPendingCampaignID = campaignID
		summary.OldestPendingSeq = uint64(seq)
		summary.OldestPendingAt = fromMillis(nextAttempt)
		return summary, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return summary, nil
	}
	return ProjectionApplyOutboxSummary{}, fmt.Errorf("query oldest pending outbox row: %w", err)
}

// ListProjectionApplyOutboxRows lists outbox rows optionally filtered by status.
func (s *Store) ListProjectionApplyOutboxRows(ctx context.Context, status string, limit int) ([]ProjectionApplyOutboxEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if limit <= 0 {
		return []ProjectionApplyOutboxEntry{}, nil
	}

	normalizedStatus, err := normalizeProjectionApplyOutboxStatus(status)
	if err != nil {
		return nil, err
	}

	var rows *sql.Rows
	if normalizedStatus == "" {
		rows, err = s.sqlDB.QueryContext(
			ctx,
			`SELECT campaign_id, seq, event_type, status, attempt_count, next_attempt_at, last_error, updated_at
			 FROM projection_apply_outbox
			 ORDER BY next_attempt_at ASC, seq ASC
			 LIMIT ?`,
			limit,
		)
	} else {
		rows, err = s.sqlDB.QueryContext(
			ctx,
			`SELECT campaign_id, seq, event_type, status, attempt_count, next_attempt_at, last_error, updated_at
			 FROM projection_apply_outbox
			 WHERE status = ?
			 ORDER BY next_attempt_at ASC, seq ASC
			 LIMIT ?`,
			normalizedStatus,
			limit,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("list outbox rows: %w", err)
	}
	defer rows.Close()

	entries := make([]ProjectionApplyOutboxEntry, 0, limit)
	for rows.Next() {
		var (
			entry       ProjectionApplyOutboxEntry
			seq         int64
			nextAttempt int64
			updatedAt   int64
			lastError   sql.NullString
		)
		if err := rows.Scan(
			&entry.CampaignID,
			&seq,
			&entry.EventType,
			&entry.Status,
			&entry.AttemptCount,
			&nextAttempt,
			&lastError,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan outbox row: %w", err)
		}
		entry.Seq = uint64(seq)
		entry.NextAttemptAt = fromMillis(nextAttempt)
		entry.UpdatedAt = fromMillis(updatedAt)
		if lastError.Valid {
			entry.LastError = lastError.String
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outbox rows: %w", err)
	}
	return entries, nil
}

func normalizeProjectionApplyOutboxStatus(status string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(status))
	if normalized == "" {
		return "", nil
	}
	switch normalized {
	case "pending", "processing", "failed", "dead":
		return normalized, nil
	default:
		return "", fmt.Errorf("invalid outbox status %q", status)
	}
}

// ApplyProjectionEventExactlyOnce applies one projection event inside a projection-db
// transaction and records a per-(campaign, seq) checkpoint to dedupe retries.
func (s *Store) ApplyProjectionEventExactlyOnce(
	ctx context.Context,
	evt event.Event,
	apply func(context.Context, event.Event, *Store) error,
) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if s == nil || s.sqlDB == nil {
		return false, fmt.Errorf("storage is not configured")
	}
	if apply == nil {
		return false, fmt.Errorf("projection apply callback is required")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return false, fmt.Errorf("campaign id is required")
	}
	if evt.Seq == 0 {
		return false, fmt.Errorf("event sequence must be greater than zero")
	}

	const (
		maxBusyRetries = 8
		retryBaseDelay = 10 * time.Millisecond
	)

	waitForRetry := func(attempt int) error {
		delay := time.Duration(attempt+1) * retryBaseDelay
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		}
	}

	var lastBusyErr error
	for attempt := 0; ; attempt++ {
		tx, err := s.sqlDB.BeginTx(ctx, nil)
		if err != nil {
			if isSQLiteBusyError(err) && attempt < maxBusyRetries {
				lastBusyErr = err
				if waitErr := waitForRetry(attempt); waitErr != nil {
					return false, waitErr
				}
				continue
			}
			return false, fmt.Errorf("begin projection apply tx: %w", err)
		}

		applied, retry, err := func() (bool, bool, error) {
			defer tx.Rollback()

			checkpointResult, err := tx.ExecContext(
				ctx,
				`INSERT OR IGNORE INTO projection_apply_checkpoints (campaign_id, seq, event_type, applied_at)
				 VALUES (?, ?, ?, ?)`,
				evt.CampaignID,
				int64(evt.Seq),
				string(evt.Type),
				toMillis(time.Now().UTC()),
			)
			if err != nil {
				if isSQLiteBusyError(err) {
					lastBusyErr = err
					return false, true, nil
				}
				return false, false, fmt.Errorf("reserve projection apply checkpoint %s/%d: %w", evt.CampaignID, evt.Seq, err)
			}

			rowsAffected, err := checkpointResult.RowsAffected()
			if err != nil {
				return false, false, fmt.Errorf("inspect projection apply checkpoint reservation %s/%d: %w", evt.CampaignID, evt.Seq, err)
			}
			if rowsAffected == 0 {
				return false, false, nil
			}

			if err := apply(ctx, evt, s.withTx(tx)); err != nil {
				return false, false, err
			}

			if err := tx.Commit(); err != nil {
				if isSQLiteBusyError(err) {
					lastBusyErr = err
					return false, true, nil
				}
				return false, false, fmt.Errorf("commit projection apply tx: %w", err)
			}

			return true, false, nil
		}()
		if retry {
			if attempt < maxBusyRetries {
				if waitErr := waitForRetry(attempt); waitErr != nil {
					return false, waitErr
				}
				continue
			}
			if lastBusyErr != nil {
				return false, fmt.Errorf("projection apply checkpoint %s/%d remained busy: %w", evt.CampaignID, evt.Seq, lastBusyErr)
			}
			return false, fmt.Errorf("projection apply checkpoint %s/%d remained busy", evt.CampaignID, evt.Seq)
		}
		return applied, err
	}
}

// ProcessProjectionApplyOutbox claims due outbox rows and applies projections
// through the provided callback. Successful rows are removed from the outbox.
func (s *Store) ProcessProjectionApplyOutbox(
	ctx context.Context,
	now time.Time,
	limit int,
	apply func(context.Context, event.Event) error,
) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if s == nil || s.sqlDB == nil {
		return 0, fmt.Errorf("storage is not configured")
	}
	if apply == nil {
		return 0, fmt.Errorf("projection apply callback is required")
	}
	if limit <= 0 {
		return 0, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	rows, err := s.claimProjectionApplyOutboxDue(ctx, now, limit)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, row := range rows {
		storedEvent, loadErr := s.GetEventBySeq(ctx, row.CampaignID, row.Seq)
		if loadErr != nil {
			attempt := row.AttemptCount + 1
			nextAttempt := now.Add(outboxRetryBackoff(attempt))
			if err := s.markProjectionApplyOutboxRetry(ctx, row, now, attempt, nextAttempt, fmt.Sprintf("load event: %v", loadErr)); err != nil {
				return processed, err
			}
			processed++
			continue
		}

		if !s.shouldApplyProjectionOutboxEvent(storedEvent) {
			if err := s.completeProjectionApplyOutboxRow(ctx, row); err != nil {
				return processed, err
			}
			processed++
			continue
		}

		if applyErr := apply(ctx, storedEvent); applyErr != nil {
			attempt := row.AttemptCount + 1
			nextAttempt := now.Add(outboxRetryBackoff(attempt))
			if err := s.markProjectionApplyOutboxRetry(ctx, row, now, attempt, nextAttempt, fmt.Sprintf("apply projection: %v", applyErr)); err != nil {
				return processed, err
			}
			processed++
			continue
		}

		if err := s.completeProjectionApplyOutboxRow(ctx, row); err != nil {
			return processed, err
		}
		processed++
	}

	return processed, nil
}

func (s *Store) shouldApplyProjectionOutboxEvent(evt event.Event) bool {
	if s == nil || s.eventRegistry == nil {
		return true
	}
	definition, ok := s.eventRegistry.Definition(evt.Type)
	if !ok {
		return true
	}
	return definition.Intent != event.IntentAuditOnly
}

// ProcessProjectionApplyOutboxShadow claims due outbox rows and requeues them
// as failed retries without applying projections.
func (s *Store) ProcessProjectionApplyOutboxShadow(ctx context.Context, now time.Time, limit int) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if s == nil || s.sqlDB == nil {
		return 0, fmt.Errorf("storage is not configured")
	}
	if limit <= 0 {
		return 0, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	rows, err := s.claimProjectionApplyOutboxDue(ctx, now, limit)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, row := range rows {
		attempt := row.AttemptCount + 1
		nextAttempt := now.Add(outboxRetryBackoff(attempt))
		if err := s.markProjectionApplyOutboxShadowRetry(ctx, row, now, attempt, nextAttempt); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func (s *Store) claimProjectionApplyOutboxDue(ctx context.Context, now time.Time, limit int) ([]projectionApplyOutboxRow, error) {
	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin outbox claim tx: %w", err)
	}
	defer tx.Rollback()

	staleBefore := now.Add(-outboxProcessingLease)
	rows, err := tx.QueryContext(
		ctx,
		`SELECT campaign_id, seq, event_type, attempt_count
		 FROM projection_apply_outbox
		 WHERE (
			 status IN ('pending', 'failed') AND next_attempt_at <= ?
		 ) OR (
			 status = 'processing' AND updated_at <= ?
		 )
		 ORDER BY next_attempt_at, seq
		 LIMIT ?`,
		toMillis(now),
		toMillis(staleBefore),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list due outbox rows: %w", err)
	}
	defer rows.Close()

	candidates := make([]projectionApplyOutboxRow, 0, limit)
	for rows.Next() {
		var row projectionApplyOutboxRow
		var seq int64
		if err := rows.Scan(&row.CampaignID, &seq, &row.EventType, &row.AttemptCount); err != nil {
			return nil, fmt.Errorf("scan due outbox row: %w", err)
		}
		row.Seq = uint64(seq)
		candidates = append(candidates, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due outbox rows: %w", err)
	}

	claimed := make([]projectionApplyOutboxRow, 0, len(candidates))
	for _, candidate := range candidates {
		result, err := tx.ExecContext(
			ctx,
			`UPDATE projection_apply_outbox
			 SET status = 'processing', updated_at = ?
			 WHERE campaign_id = ? AND seq = ?
			   AND (
			   	(status IN ('pending', 'failed') AND next_attempt_at <= ?)
			   	OR (status = 'processing' AND updated_at <= ?)
			   )`,
			toMillis(now),
			candidate.CampaignID,
			int64(candidate.Seq),
			toMillis(now),
			toMillis(staleBefore),
		)
		if err != nil {
			return nil, fmt.Errorf("claim outbox row %s/%d: %w", candidate.CampaignID, candidate.Seq, err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("claim outbox row rows affected %s/%d: %w", candidate.CampaignID, candidate.Seq, err)
		}
		if affected == 1 {
			claimed = append(claimed, candidate)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit outbox claim tx: %w", err)
	}
	return claimed, nil
}

func (s *Store) markProjectionApplyOutboxShadowRetry(ctx context.Context, row projectionApplyOutboxRow, now time.Time, attempt int, nextAttempt time.Time) error {
	const lastError = "shadow worker: projection apply skipped"
	return s.markProjectionApplyOutboxRetry(ctx, row, now, attempt, nextAttempt, lastError)
}

func (s *Store) markProjectionApplyOutboxRetry(ctx context.Context, row projectionApplyOutboxRow, now time.Time, attempt int, nextAttempt time.Time, lastError string) error {
	status := "failed"
	if attempt >= outboxDeadLetterThreshold {
		status = "dead"
	}
	result, err := s.sqlDB.ExecContext(
		ctx,
		`UPDATE projection_apply_outbox
		 SET status = ?,
		     attempt_count = ?,
		     next_attempt_at = ?,
		     last_error = ?,
		     updated_at = ?
		 WHERE campaign_id = ? AND seq = ? AND status = 'processing'`,
		status,
		attempt,
		toMillis(nextAttempt),
		lastError,
		toMillis(now),
		row.CampaignID,
		int64(row.Seq),
	)
	if err != nil {
		return fmt.Errorf("mark outbox retry for row %s/%d: %w", row.CampaignID, row.Seq, err)
	}
	if err := ensureProjectionApplyOutboxSingleRow(result, row, "mark outbox retry for row", "updated"); err != nil {
		return err
	}
	return nil
}

func (s *Store) completeProjectionApplyOutboxRow(ctx context.Context, row projectionApplyOutboxRow) error {
	result, err := s.sqlDB.ExecContext(
		ctx,
		`DELETE FROM projection_apply_outbox
		 WHERE campaign_id = ? AND seq = ? AND status = 'processing'`,
		row.CampaignID,
		int64(row.Seq),
	)
	if err != nil {
		return fmt.Errorf("complete outbox row %s/%d: %w", row.CampaignID, row.Seq, err)
	}
	if err := ensureProjectionApplyOutboxSingleRow(result, row, "complete outbox row", "deleted"); err != nil {
		return err
	}
	return nil
}

func ensureProjectionApplyOutboxSingleRow(result sql.Result, row projectionApplyOutboxRow, operation, verb string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s rows affected %s/%d: %w", operation, row.CampaignID, row.Seq, err)
	}
	if affected != 1 {
		return fmt.Errorf("%s %s/%d: expected 1 row %s, got %d", operation, row.CampaignID, row.Seq, verb, affected)
	}
	return nil
}

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

func outboxRetryBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	backoff := time.Second << (attempt - 1)
	if backoff > 5*time.Minute {
		return 5 * time.Minute
	}
	return backoff
}

// VerifyEventIntegrity validates the event chain and signatures for all campaigns.
