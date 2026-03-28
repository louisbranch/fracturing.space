package projectionapplyoutbox

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
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
	ownerID       string
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
	Status       string
	AttemptCount int
	NextAttempt  time.Time
	UpdatedAt    time.Time
}

type campaignLease struct {
	CampaignID string
}

const (
	deadLetterThreshold = 8
	processingLease     = 2 * time.Minute
)

// Bind creates a projection-apply outbox backend bound to the provided event DB.
func Bind(sqlDB *sql.DB, loader eventLoader, registry *event.Registry) *Store {
	return bindWithOwnerID(sqlDB, loader, registry, newOwnerID())
}

func bindWithOwnerID(sqlDB *sql.DB, loader eventLoader, registry *event.Registry, ownerID string) *Store {
	if sqlDB == nil {
		return nil
	}
	if strings.TrimSpace(ownerID) == "" {
		ownerID = newOwnerID()
	}
	return &Store{
		sqlDB:         sqlDB,
		eventLoader:   loader,
		eventRegistry: registry,
		ownerID:       ownerID,
	}
}

func newOwnerID() string {
	value, err := id.NewID()
	if err == nil {
		return value
	}
	return fmt.Sprintf("projection-apply-%d", time.Now().UTC().UnixNano())
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
		sqliteutil.ToMillis(enqueuedAt),
		sqliteutil.ToMillis(enqueuedAt),
	); err != nil {
		return fmt.Errorf("enqueue projection apply outbox: %w", err)
	}
	return nil
}

// ProcessProjectionApplyOutbox claims due campaigns, processes their rows in
// sequence order, and applies projections through the provided callback.
// Successful rows are removed from the queue.
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

	campaigns, err := s.claimDueCampaigns(ctx, now, limit)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, campaign := range campaigns {
		if processed >= limit {
			break
		}
		count, err := s.processCampaign(ctx, campaign, now, limit-processed, apply)
		if err != nil {
			return processed, err
		}
		processed += count
	}
	return processed, nil
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

	rows, err := s.claimDueRows(ctx, now, limit)
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

func (s *Store) processCampaign(
	ctx context.Context,
	campaign campaignLease,
	now time.Time,
	limit int,
	apply func(context.Context, event.Event) error,
) (processed int, err error) {
	defer func() {
		releaseErr := s.releaseCampaignLease(ctx, campaign.CampaignID)
		if err == nil && releaseErr != nil {
			err = releaseErr
		}
	}()

	for processed < limit {
		leaseNow := time.Now().UTC()
		if leaseNow.Before(now) {
			leaseNow = now
		}
		if err := s.renewCampaignLease(ctx, campaign.CampaignID, leaseNow); err != nil {
			return processed, err
		}

		row, ready, err := s.claimNextRowForCampaign(ctx, campaign.CampaignID, now)
		if err != nil {
			return processed, err
		}
		if !ready {
			return processed, nil
		}
		if err := s.processRow(ctx, row, now, apply); err != nil {
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

func (s *Store) claimDueCampaigns(ctx context.Context, now time.Time, limit int) ([]campaignLease, error) {
	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin campaign claim tx: %w", err)
	}
	defer tx.Rollback()

	staleBefore := now.Add(-processingLease)
	rows, err := tx.QueryContext(
		ctx,
		`SELECT o.campaign_id
		 FROM projection_apply_outbox o
		 WHERE o.seq = (
			 SELECT MIN(seq)
			 FROM projection_apply_outbox
			 WHERE campaign_id = o.campaign_id
		 )
		 AND (
			 (o.status IN ('pending', 'failed') AND o.next_attempt_at <= ?)
			 OR (o.status = 'processing' AND o.updated_at <= ?)
		 )
		 ORDER BY o.next_attempt_at, o.seq
		 LIMIT ?`,
		sqliteutil.ToMillis(now),
		sqliteutil.ToMillis(staleBefore),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list due outbox campaigns: %w", err)
	}
	defer rows.Close()

	candidates := make([]campaignLease, 0, limit)
	for rows.Next() {
		var candidate campaignLease
		if err := rows.Scan(&candidate.CampaignID); err != nil {
			return nil, fmt.Errorf("scan due outbox campaign: %w", err)
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due outbox campaigns: %w", err)
	}

	claimed := make([]campaignLease, 0, len(candidates))
	for _, candidate := range candidates {
		result, err := tx.ExecContext(
			ctx,
			`INSERT INTO projection_apply_campaign_leases (campaign_id, owner_id, lease_expires_at, updated_at)
			 VALUES (?, ?, ?, ?)
			 ON CONFLICT(campaign_id) DO UPDATE
			 SET owner_id = excluded.owner_id,
			     lease_expires_at = excluded.lease_expires_at,
			     updated_at = excluded.updated_at
			 WHERE projection_apply_campaign_leases.lease_expires_at <= ?`,
			candidate.CampaignID,
			s.ownerID,
			sqliteutil.ToMillis(now.Add(processingLease)),
			sqliteutil.ToMillis(now),
			sqliteutil.ToMillis(staleBefore),
		)
		if err != nil {
			return nil, fmt.Errorf("claim campaign lease %s: %w", candidate.CampaignID, err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("claim campaign lease rows affected %s: %w", candidate.CampaignID, err)
		}
		if affected == 1 {
			claimed = append(claimed, candidate)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit campaign claim tx: %w", err)
	}
	return claimed, nil
}

func (s *Store) claimDueRows(ctx context.Context, now time.Time, limit int) ([]row, error) {
	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin outbox claim tx: %w", err)
	}
	defer tx.Rollback()

	staleBefore := now.Add(-processingLease)
	rows, err := tx.QueryContext(
		ctx,
		`SELECT campaign_id, seq, event_type, status, attempt_count, next_attempt_at, updated_at
		 FROM projection_apply_outbox
		 WHERE (
			 status IN ('pending', 'failed') AND next_attempt_at <= ?
		 ) OR (
			 status = 'processing' AND updated_at <= ?
		 )
		 ORDER BY next_attempt_at, seq
		 LIMIT ?`,
		sqliteutil.ToMillis(now),
		sqliteutil.ToMillis(staleBefore),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list due outbox rows: %w", err)
	}
	defer rows.Close()

	candidates := make([]row, 0, limit)
	for rows.Next() {
		candidate, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan due outbox row: %w", err)
		}
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
			sqliteutil.ToMillis(now),
			candidate.CampaignID,
			int64(candidate.Seq),
			sqliteutil.ToMillis(now),
			sqliteutil.ToMillis(staleBefore),
		)
		if err != nil {
			return nil, fmt.Errorf("claim outbox row %s/%d: %w", candidate.CampaignID, candidate.Seq, err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("claim outbox row rows affected %s/%d: %w", candidate.CampaignID, candidate.Seq, err)
		}
		if affected == 1 {
			candidate.Status = "processing"
			candidate.UpdatedAt = now
			claimed = append(claimed, candidate)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit outbox claim tx: %w", err)
	}
	return claimed, nil
}

func scanRow(scanner interface{ Scan(...any) error }) (row, error) {
	var (
		result      row
		seq         int64
		nextAttempt int64
		updatedAt   int64
	)
	if err := scanner.Scan(
		&result.CampaignID,
		&seq,
		&result.EventType,
		&result.Status,
		&result.AttemptCount,
		&nextAttempt,
		&updatedAt,
	); err != nil {
		return row{}, err
	}
	result.Seq = uint64(seq)
	result.NextAttempt = sqliteutil.FromMillis(nextAttempt)
	result.UpdatedAt = sqliteutil.FromMillis(updatedAt)
	return result, nil
}

func (s *Store) claimNextRowForCampaign(ctx context.Context, campaignID string, now time.Time) (row, bool, error) {
	staleBefore := now.Add(-processingLease)
	var (
		next        row
		seq         int64
		nextAttempt int64
		updatedAt   int64
	)
	err := s.sqlDB.QueryRowContext(
		ctx,
		`SELECT campaign_id, seq, event_type, status, attempt_count, next_attempt_at, updated_at
		 FROM projection_apply_outbox
		 WHERE campaign_id = ?
		 ORDER BY seq
		 LIMIT 1`,
		campaignID,
	).Scan(
		&next.CampaignID,
		&seq,
		&next.EventType,
		&next.Status,
		&next.AttemptCount,
		&nextAttempt,
		&updatedAt,
	)
	if err == sql.ErrNoRows {
		return row{}, false, nil
	}
	if err != nil {
		return row{}, false, fmt.Errorf("load next outbox row for campaign %s: %w", campaignID, err)
	}
	next.Seq = uint64(seq)
	next.NextAttempt = sqliteutil.FromMillis(nextAttempt)
	next.UpdatedAt = sqliteutil.FromMillis(updatedAt)

	switch next.Status {
	case "dead":
		return row{}, false, nil
	case "pending", "failed":
		if next.NextAttempt.After(now) {
			return row{}, false, nil
		}
	case "processing":
		if next.UpdatedAt.After(staleBefore) {
			return row{}, false, nil
		}
	default:
		return row{}, false, fmt.Errorf("invalid projection apply outbox status %q for %s/%d", next.Status, next.CampaignID, next.Seq)
	}

	result, err := s.sqlDB.ExecContext(
		ctx,
		`UPDATE projection_apply_outbox
		 SET status = 'processing', updated_at = ?
		 WHERE campaign_id = ? AND seq = ?
		   AND (
		   	(status IN ('pending', 'failed') AND next_attempt_at <= ?)
		   	OR (status = 'processing' AND updated_at <= ?)
		   )`,
		sqliteutil.ToMillis(now),
		next.CampaignID,
		int64(next.Seq),
		sqliteutil.ToMillis(now),
		sqliteutil.ToMillis(staleBefore),
	)
	if err != nil {
		return row{}, false, fmt.Errorf("claim next outbox row for campaign %s/%d: %w", next.CampaignID, next.Seq, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return row{}, false, fmt.Errorf("claim next outbox row rows affected %s/%d: %w", next.CampaignID, next.Seq, err)
	}
	if affected != 1 {
		return row{}, false, nil
	}
	next.Status = "processing"
	next.UpdatedAt = now
	return next, true, nil
}

func (s *Store) renewCampaignLease(ctx context.Context, campaignID string, now time.Time) error {
	result, err := s.sqlDB.ExecContext(
		ctx,
		`UPDATE projection_apply_campaign_leases
		 SET lease_expires_at = ?, updated_at = ?
		 WHERE campaign_id = ? AND owner_id = ?`,
		sqliteutil.ToMillis(now.Add(processingLease)),
		sqliteutil.ToMillis(now),
		campaignID,
		s.ownerID,
	)
	if err != nil {
		return fmt.Errorf("renew campaign lease %s: %w", campaignID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("renew campaign lease rows affected %s: %w", campaignID, err)
	}
	if affected != 1 {
		return fmt.Errorf("renew campaign lease %s: expected 1 row updated, got %d", campaignID, affected)
	}
	return nil
}

func (s *Store) releaseCampaignLease(ctx context.Context, campaignID string) error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	_, err := s.sqlDB.ExecContext(
		ctx,
		`DELETE FROM projection_apply_campaign_leases
		 WHERE campaign_id = ? AND owner_id = ?`,
		campaignID,
		s.ownerID,
	)
	if err != nil {
		return fmt.Errorf("release campaign lease %s: %w", campaignID, err)
	}
	return nil
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
		sqliteutil.ToMillis(nextAttempt),
		lastError,
		sqliteutil.ToMillis(now),
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
		summary.OldestPendingAt = sqliteutil.FromMillis(nextAttempt)
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
		entry.NextAttemptAt = sqliteutil.FromMillis(nextAttempt)
		entry.UpdatedAt = sqliteutil.FromMillis(updatedAt)
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
		sqliteutil.ToMillis(now),
		sqliteutil.ToMillis(now),
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
		sqliteutil.ToMillis(now),
		sqliteutil.ToMillis(now),
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
