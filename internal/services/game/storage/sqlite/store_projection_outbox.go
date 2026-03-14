package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// enqueueProjectionApplyOutbox inserts a pending outbox row for the given event
// inside the caller's transaction. This must be called within the same tx that
// appends the event so the journal entry and its outbox work item are committed
// atomically — see AppendEvent and BatchAppendEvents.
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

const (
	outboxDeadLetterThreshold = 8
	outboxProcessingLease     = 2 * time.Minute
)

// ApplyProjectionEventExactlyOnce applies one projection event inside a projection-db
// transaction and records a per-(campaign, seq) checkpoint to dedupe retries.
//
// Transaction boundary semantics:
//  1. BEGIN — a new transaction is opened per event.
//  2. INSERT OR IGNORE into projection_apply_checkpoints — the idempotency key
//     is (campaign_id, seq). If the row already exists (duplicate), zero rows
//     are affected and the function returns (false, nil) without invoking apply.
//  3. apply(ctx, evt, txStore) — the caller's projection apply function runs
//     inside the same transaction. Any error rolls back the entire transaction,
//     including the checkpoint reservation. This ensures the event can be retried.
//  4. COMMIT — makes both the checkpoint and all projection writes durable atomically.
//
// SQLITE_BUSY errors at any stage trigger a retry with linear backoff (up to 8
// attempts) since SQLite's single-writer model can cause contention under
// concurrent outbox processing.
func (s *Store) ApplyProjectionEventExactlyOnce(
	ctx context.Context,
	evt event.Event,
	apply func(context.Context, event.Event, storage.ProjectionApplyTxStore) error,
) (bool, error) {
	if err := validateProjectionApplyExactlyOnceRequest(ctx, s, evt, apply); err != nil {
		return false, err
	}

	const (
		maxBusyRetries = 8
		retryBaseDelay = 10 * time.Millisecond
	)

	var lastBusyErr error
	for attempt := 0; ; attempt++ {
		applied, retry, busyErr, err := s.tryApplyProjectionEventExactlyOnce(ctx, evt, apply)
		if retry {
			lastBusyErr = busyErr
			if attempt < maxBusyRetries {
				if waitErr := waitProjectionApplyRetry(ctx, attempt, retryBaseDelay); waitErr != nil {
					return false, waitErr
				}
				continue
			}
			slog.Warn("projection apply BUSY retries exhausted",
				"campaign_id", evt.CampaignID,
				"seq", evt.Seq,
				"retries", attempt,
			)
			if lastBusyErr != nil {
				return false, fmt.Errorf("projection apply checkpoint %s/%d remained busy: %w", evt.CampaignID, evt.Seq, lastBusyErr)
			}
			return false, fmt.Errorf("projection apply checkpoint %s/%d remained busy", evt.CampaignID, evt.Seq)
		}
		return applied, err
	}
}

func validateProjectionApplyExactlyOnceRequest(
	ctx context.Context,
	store *Store,
	evt event.Event,
	apply func(context.Context, event.Event, storage.ProjectionApplyTxStore) error,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if store == nil || store.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if apply == nil {
		return fmt.Errorf("projection apply callback is required")
	}
	if strings.TrimSpace(string(evt.CampaignID)) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if evt.Seq == 0 {
		return fmt.Errorf("event sequence must be greater than zero")
	}
	return nil
}

func waitProjectionApplyRetry(ctx context.Context, attempt int, baseDelay time.Duration) error {
	delay := time.Duration(attempt+1) * baseDelay
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (s *Store) tryApplyProjectionEventExactlyOnce(
	ctx context.Context,
	evt event.Event,
	apply func(context.Context, event.Event, storage.ProjectionApplyTxStore) error,
) (applied bool, retry bool, busyErr error, err error) {
	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		if isSQLiteBusyError(err) {
			return false, true, err, nil
		}
		return false, false, nil, fmt.Errorf("begin projection apply tx: %w", err)
	}

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
			return false, true, err, nil
		}
		return false, false, nil, fmt.Errorf("reserve projection apply checkpoint %s/%d: %w", evt.CampaignID, evt.Seq, err)
	}

	rowsAffected, err := checkpointResult.RowsAffected()
	if err != nil {
		return false, false, nil, fmt.Errorf("inspect projection apply checkpoint reservation %s/%d: %w", evt.CampaignID, evt.Seq, err)
	}
	if rowsAffected == 0 {
		return false, false, nil, nil
	}

	if err := apply(ctx, evt, s.txStore(tx)); err != nil {
		return false, false, nil, err
	}

	if err := tx.Commit(); err != nil {
		if isSQLiteBusyError(err) {
			return false, true, err, nil
		}
		return false, false, nil, fmt.Errorf("commit projection apply tx: %w", err)
	}

	return true, false, nil, nil
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

	return processProjectionApplyOutboxRows(rows, func(row projectionApplyOutboxRow) error {
		return s.processProjectionApplyOutboxRow(ctx, row, now, apply)
	})
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
	return processProjectionApplyOutboxRows(rows, func(row projectionApplyOutboxRow) error {
		return s.processProjectionApplyOutboxShadowRow(ctx, row, now)
	})
}

func processProjectionApplyOutboxRows(
	rows []projectionApplyOutboxRow,
	handle func(projectionApplyOutboxRow) error,
) (int, error) {
	processed := 0
	for _, row := range rows {
		if err := handle(row); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func (s *Store) processProjectionApplyOutboxRow(
	ctx context.Context,
	row projectionApplyOutboxRow,
	now time.Time,
	apply func(context.Context, event.Event) error,
) error {
	storedEvent, loadErr := s.GetEventBySeq(ctx, row.CampaignID, row.Seq)
	if loadErr != nil {
		return s.scheduleProjectionApplyOutboxRetry(ctx, row, now, fmt.Sprintf("load event: %v", loadErr))
	}

	if !s.shouldApplyProjectionOutboxEvent(storedEvent) {
		return s.completeProjectionApplyOutboxRow(ctx, row)
	}

	if applyErr := apply(ctx, storedEvent); applyErr != nil {
		return s.scheduleProjectionApplyOutboxRetry(ctx, row, now, fmt.Sprintf("apply projection: %v", applyErr))
	}

	return s.completeProjectionApplyOutboxRow(ctx, row)
}

func (s *Store) processProjectionApplyOutboxShadowRow(
	ctx context.Context,
	row projectionApplyOutboxRow,
	now time.Time,
) error {
	attempt := row.AttemptCount + 1
	nextAttempt := now.Add(outboxRetryBackoff(attempt))
	return s.markProjectionApplyOutboxShadowRetry(ctx, row, now, attempt, nextAttempt)
}

func (s *Store) scheduleProjectionApplyOutboxRetry(
	ctx context.Context,
	row projectionApplyOutboxRow,
	now time.Time,
	lastError string,
) error {
	attempt := row.AttemptCount + 1
	nextAttempt := now.Add(outboxRetryBackoff(attempt))
	return s.markProjectionApplyOutboxRetry(ctx, row, now, attempt, nextAttempt, lastError)
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
		log.Printf("projection outbox dead letter campaign_id=%s seq=%d attempts=%d last_error=%s",
			row.CampaignID, row.Seq, attempt, lastError)
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
