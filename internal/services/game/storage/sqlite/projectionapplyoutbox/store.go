package projectionapplyoutbox

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type eventLoader interface {
	GetEventBySeq(context.Context, string, uint64) (event.Event, error)
}

// Store binds projection-apply outbox persistence to the event-store SQLite DB.
type Store struct {
	sqlDB         *sql.DB
	eventLoader   eventLoader
	eventRegistry *event.Registry
}

var (
	_ storage.ProjectionApplyOutboxProcessor       = (*Store)(nil)
	_ storage.ProjectionApplyOutboxShadowProcessor = (*Store)(nil)
	_ storage.ProjectionApplyOutboxInspector       = (*Store)(nil)
	_ storage.ProjectionApplyOutboxRequeuer        = (*Store)(nil)
)

type row struct {
	CampaignID   string
	Seq          uint64
	EventType    string
	AttemptCount int
}

const (
	deadLetterThreshold = 8
	processingLease     = 2 * time.Minute
)

// Bind creates a projection-apply outbox backend bound to the provided event DB.
func Bind(sqlDB *sql.DB, loader eventLoader, registry *event.Registry) *Store {
	if sqlDB == nil {
		return nil
	}
	return &Store{
		sqlDB:         sqlDB,
		eventLoader:   loader,
		eventRegistry: registry,
	}
}

// EnqueueForEvent inserts a pending outbox row for one event inside the
// caller's append transaction when the queue is enabled.
func EnqueueForEvent(ctx context.Context, tx *sql.Tx, evt event.Event, enabled bool) error {
	if !enabled {
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

// ProcessProjectionApplyOutbox claims due outbox rows and applies projections
// through the provided callback. Successful rows are removed from the queue.
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

	rows, err := s.claimDue(ctx, now, limit)
	if err != nil {
		return 0, err
	}

	return processRows(rows, func(row row) error {
		return s.processRow(ctx, row, now, apply)
	})
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

	rows, err := s.claimDue(ctx, now, limit)
	if err != nil {
		return 0, err
	}
	return processRows(rows, func(row row) error {
		return s.processShadowRow(ctx, row, now)
	})
}

func processRows(rows []row, handle func(row) error) (int, error) {
	processed := 0
	for _, row := range rows {
		if err := handle(row); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func (s *Store) processRow(
	ctx context.Context,
	row row,
	now time.Time,
	apply func(context.Context, event.Event) error,
) error {
	if s.eventLoader == nil {
		return s.scheduleRetry(ctx, row, now, "load event: event loader is not configured")
	}
	storedEvent, loadErr := s.eventLoader.GetEventBySeq(ctx, row.CampaignID, row.Seq)
	if loadErr != nil {
		return s.scheduleRetry(ctx, row, now, fmt.Sprintf("load event: %v", loadErr))
	}

	if !s.shouldApplyEvent(storedEvent) {
		return s.completeRow(ctx, row)
	}

	if applyErr := apply(ctx, storedEvent); applyErr != nil {
		return s.scheduleRetry(ctx, row, now, fmt.Sprintf("apply projection: %v", applyErr))
	}

	return s.completeRow(ctx, row)
}

func (s *Store) shouldApplyEvent(evt event.Event) bool {
	if s == nil || s.eventRegistry == nil {
		return true
	}
	definition, ok := s.eventRegistry.Definition(evt.Type)
	if !ok {
		return true
	}
	return definition.Intent != event.IntentAuditOnly
}

func (s *Store) processShadowRow(ctx context.Context, row row, now time.Time) error {
	attempt := row.AttemptCount + 1
	nextAttempt := now.Add(retryBackoff(attempt))
	return s.markShadowRetry(ctx, row, now, attempt, nextAttempt)
}

func (s *Store) scheduleRetry(
	ctx context.Context,
	row row,
	now time.Time,
	lastError string,
) error {
	attempt := row.AttemptCount + 1
	nextAttempt := now.Add(retryBackoff(attempt))
	return s.markRetry(ctx, row, now, attempt, nextAttempt, lastError)
}

func (s *Store) claimDue(ctx context.Context, now time.Time, limit int) ([]row, error) {
	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin outbox claim tx: %w", err)
	}
	defer tx.Rollback()

	staleBefore := now.Add(-processingLease)
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

	candidates := make([]row, 0, limit)
	for rows.Next() {
		var candidate row
		var seq int64
		if err := rows.Scan(&candidate.CampaignID, &seq, &candidate.EventType, &candidate.AttemptCount); err != nil {
			return nil, fmt.Errorf("scan due outbox row: %w", err)
		}
		candidate.Seq = uint64(seq)
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due outbox rows: %w", err)
	}

	claimed := make([]row, 0, len(candidates))
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

func (s *Store) markShadowRetry(ctx context.Context, row row, now time.Time, attempt int, nextAttempt time.Time) error {
	const lastError = "shadow worker: projection apply skipped"
	return s.markRetry(ctx, row, now, attempt, nextAttempt, lastError)
}

func (s *Store) markRetry(ctx context.Context, row row, now time.Time, attempt int, nextAttempt time.Time, lastError string) error {
	status := "failed"
	if attempt >= deadLetterThreshold {
		status = "dead"
		slog.Warn("projection apply dead letter",
			"campaign_id", row.CampaignID,
			"seq", row.Seq,
			"attempts", attempt,
			"last_error", lastError,
		)
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
	if err := ensureSingleRow(result, row, "mark outbox retry for row", "updated"); err != nil {
		return err
	}
	return nil
}

func (s *Store) completeRow(ctx context.Context, row row) error {
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
	if err := ensureSingleRow(result, row, "complete outbox row", "deleted"); err != nil {
		return err
	}
	return nil
}

func ensureSingleRow(result sql.Result, row row, operation, verb string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s rows affected %s/%d: %w", operation, row.CampaignID, row.Seq, err)
	}
	if affected != 1 {
		return fmt.Errorf("%s %s/%d: expected 1 row %s, got %d", operation, row.CampaignID, row.Seq, verb, affected)
	}
	return nil
}

func retryBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	backoff := time.Second << (attempt - 1)
	if backoff > 5*time.Minute {
		return 5 * time.Minute
	}
	return backoff
}

// GetProjectionApplyOutboxSummary returns queue depth by status and the oldest
// pending/failed row metadata.
func (s *Store) GetProjectionApplyOutboxSummary(ctx context.Context) (storage.ProjectionApplyOutboxSummary, error) {
	if err := ctx.Err(); err != nil {
		return storage.ProjectionApplyOutboxSummary{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.ProjectionApplyOutboxSummary{}, fmt.Errorf("storage is not configured")
	}

	summary := storage.ProjectionApplyOutboxSummary{}
	rows, err := s.sqlDB.QueryContext(
		ctx,
		`SELECT status, COUNT(*)
		 FROM projection_apply_outbox
		 GROUP BY status`,
	)
	if err != nil {
		return storage.ProjectionApplyOutboxSummary{}, fmt.Errorf("query outbox summary counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			status string
			count  int
		)
		if err := rows.Scan(&status, &count); err != nil {
			return storage.ProjectionApplyOutboxSummary{}, fmt.Errorf("scan outbox summary count: %w", err)
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
		return storage.ProjectionApplyOutboxSummary{}, fmt.Errorf("iterate outbox summary counts: %w", err)
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
	if err == sql.ErrNoRows {
		return summary, nil
	}
	return storage.ProjectionApplyOutboxSummary{}, fmt.Errorf("query oldest pending outbox row: %w", err)
}

// ListProjectionApplyOutboxRows lists outbox rows optionally filtered by status.
func (s *Store) ListProjectionApplyOutboxRows(ctx context.Context, status string, limit int) ([]storage.ProjectionApplyOutboxEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if limit <= 0 {
		return []storage.ProjectionApplyOutboxEntry{}, nil
	}

	normalizedStatus, err := normalizeStatus(status)
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

	entries := make([]storage.ProjectionApplyOutboxEntry, 0, limit)
	for rows.Next() {
		var (
			entry       storage.ProjectionApplyOutboxEntry
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

func normalizeStatus(status string) (string, error) {
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

func toMillis(value time.Time) int64 {
	return value.UTC().UnixMilli()
}

func fromMillis(value int64) time.Time {
	return time.UnixMilli(value).UTC()
}
