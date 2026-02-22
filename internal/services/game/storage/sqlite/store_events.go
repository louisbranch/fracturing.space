package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/integrity"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// EventStore methods (unified event journal)

// AppendEvent atomically appends an event and returns it with sequence and hash set.
func (s *Store) AppendEvent(ctx context.Context, evt event.Event) (event.Event, error) {
	if err := ctx.Err(); err != nil {
		return event.Event{}, err
	}
	if s == nil || s.sqlDB == nil {
		return event.Event{}, fmt.Errorf("storage is not configured")
	}
	if s.eventRegistry == nil {
		return event.Event{}, fmt.Errorf("event registry is required")
	}

	validated, err := s.eventRegistry.ValidateForAppend(evt)
	if err != nil {
		return event.Event{}, err
	}
	evt = validated

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return event.Event{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)

	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}
	evt.Timestamp = evt.Timestamp.UTC().Truncate(time.Millisecond)

	if err := qtx.InitEventSeq(ctx, evt.CampaignID); err != nil {
		return event.Event{}, fmt.Errorf("init event seq: %w", err)
	}

	seq, err := qtx.GetEventSeq(ctx, evt.CampaignID)
	if err != nil {
		return event.Event{}, fmt.Errorf("get event seq: %w", err)
	}
	evt.Seq = uint64(seq)

	if err := qtx.IncrementEventSeq(ctx, evt.CampaignID); err != nil {
		return event.Event{}, fmt.Errorf("increment event seq: %w", err)
	}

	if s.keyring == nil {
		return event.Event{}, fmt.Errorf("event integrity keyring is required")
	}

	hash, err := integrity.EventHash(evt)
	if err != nil {
		return event.Event{}, fmt.Errorf("compute event hash: %w", err)
	}
	if strings.TrimSpace(hash) == "" {
		return event.Event{}, fmt.Errorf("event hash is required")
	}
	evt.Hash = hash

	prevHash := ""
	if evt.Seq > 1 {
		prevRow, err := qtx.GetEventBySeq(ctx, db.GetEventBySeqParams{
			CampaignID: evt.CampaignID,
			Seq:        int64(evt.Seq - 1),
		})
		if err != nil {
			return event.Event{}, fmt.Errorf("load previous event: %w", err)
		}
		prevHash = prevRow.ChainHash
	}

	chainHash, err := integrity.ChainHash(evt, prevHash)
	if err != nil {
		return event.Event{}, fmt.Errorf("compute chain hash: %w", err)
	}
	if strings.TrimSpace(chainHash) == "" {
		return event.Event{}, fmt.Errorf("chain hash is required")
	}

	signature, keyID, err := s.keyring.SignChainHash(evt.CampaignID, chainHash)
	if err != nil {
		return event.Event{}, fmt.Errorf("sign chain hash: %w", err)
	}

	evt.PrevHash = prevHash
	evt.ChainHash = chainHash
	evt.Signature = signature
	evt.SignatureKeyID = keyID

	if err := qtx.AppendEvent(ctx, db.AppendEventParams{
		CampaignID:     evt.CampaignID,
		Seq:            int64(evt.Seq),
		EventHash:      evt.Hash,
		PrevEventHash:  prevHash,
		ChainHash:      chainHash,
		SignatureKeyID: keyID,
		EventSignature: signature,
		Timestamp:      toMillis(evt.Timestamp),
		EventType:      string(evt.Type),
		SessionID:      evt.SessionID,
		RequestID:      evt.RequestID,
		InvocationID:   evt.InvocationID,
		ActorType:      string(evt.ActorType),
		ActorID:        evt.ActorID,
		EntityType:     evt.EntityType,
		EntityID:       evt.EntityID,
		SystemID:       evt.SystemID,
		SystemVersion:  evt.SystemVersion,
		CorrelationID:  evt.CorrelationID,
		CausationID:    evt.CausationID,
		PayloadJson:    evt.PayloadJSON,
	}); err != nil {
		if isConstraintError(err) {
			stored, lookupErr := s.GetEventByHash(ctx, evt.Hash)
			if lookupErr == nil {
				return stored, nil
			}
		}
		return event.Event{}, fmt.Errorf("append event: %w", err)
	}
	if err := s.enqueueProjectionApplyOutbox(ctx, tx, evt); err != nil {
		return event.Event{}, err
	}

	if err := tx.Commit(); err != nil {
		return event.Event{}, fmt.Errorf("commit: %w", err)
	}

	return evt, nil
}

// BatchAppendEvents atomically appends multiple events in a single transaction.
//
// All events must belong to the same campaign. Sequence numbers are allocated
// contiguously, and chain hashes link each event to its predecessor — including
// the last previously stored event for the first item in the batch.
func (s *Store) BatchAppendEvents(ctx context.Context, events []event.Event) ([]event.Event, error) {
	if len(events) == 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if s.eventRegistry == nil {
		return nil, fmt.Errorf("event registry is required")
	}
	if s.keyring == nil {
		return nil, fmt.Errorf("event integrity keyring is required")
	}

	// Validate all events before opening a transaction.
	validated := make([]event.Event, len(events))
	for i, evt := range events {
		v, err := s.eventRegistry.ValidateForAppend(evt)
		if err != nil {
			return nil, fmt.Errorf("event %d: %w", i, err)
		}
		if v.Timestamp.IsZero() {
			v.Timestamp = time.Now().UTC()
		}
		v.Timestamp = v.Timestamp.UTC().Truncate(time.Millisecond)
		validated[i] = v
	}

	campaignID := validated[0].CampaignID

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)

	if err := qtx.InitEventSeq(ctx, campaignID); err != nil {
		return nil, fmt.Errorf("init event seq: %w", err)
	}

	baseSeq, err := qtx.GetEventSeq(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("get event seq: %w", err)
	}

	// Load previous chain hash for linking the first event in the batch.
	prevChainHash := ""
	if baseSeq > 1 {
		prevRow, err := qtx.GetEventBySeq(ctx, db.GetEventBySeqParams{
			CampaignID: campaignID,
			Seq:        baseSeq - 1,
		})
		if err != nil {
			return nil, fmt.Errorf("load previous event: %w", err)
		}
		prevChainHash = prevRow.ChainHash
	}

	stored := make([]event.Event, len(validated))
	for i, evt := range validated {
		evt.Seq = uint64(baseSeq) + uint64(i)

		hash, err := integrity.EventHash(evt)
		if err != nil {
			return nil, fmt.Errorf("event %d hash: %w", i, err)
		}
		if strings.TrimSpace(hash) == "" {
			return nil, fmt.Errorf("event %d: hash is empty", i)
		}
		evt.Hash = hash

		chainHash, err := integrity.ChainHash(evt, prevChainHash)
		if err != nil {
			return nil, fmt.Errorf("event %d chain hash: %w", i, err)
		}
		if strings.TrimSpace(chainHash) == "" {
			return nil, fmt.Errorf("event %d: chain hash is empty", i)
		}

		signature, keyID, err := s.keyring.SignChainHash(evt.CampaignID, chainHash)
		if err != nil {
			return nil, fmt.Errorf("event %d sign: %w", i, err)
		}

		evt.PrevHash = prevChainHash
		evt.ChainHash = chainHash
		evt.Signature = signature
		evt.SignatureKeyID = keyID

		if err := qtx.AppendEvent(ctx, db.AppendEventParams{
			CampaignID:     evt.CampaignID,
			Seq:            int64(evt.Seq),
			EventHash:      evt.Hash,
			PrevEventHash:  prevChainHash,
			ChainHash:      chainHash,
			SignatureKeyID: keyID,
			EventSignature: signature,
			Timestamp:      toMillis(evt.Timestamp),
			EventType:      string(evt.Type),
			SessionID:      evt.SessionID,
			RequestID:      evt.RequestID,
			InvocationID:   evt.InvocationID,
			ActorType:      string(evt.ActorType),
			ActorID:        evt.ActorID,
			EntityType:     evt.EntityType,
			EntityID:       evt.EntityID,
			SystemID:       evt.SystemID,
			SystemVersion:  evt.SystemVersion,
			CorrelationID:  evt.CorrelationID,
			CausationID:    evt.CausationID,
			PayloadJson:    evt.PayloadJSON,
		}); err != nil {
			return nil, fmt.Errorf("append event %d: %w", i, err)
		}

		if err := s.enqueueProjectionApplyOutbox(ctx, tx, evt); err != nil {
			return nil, err
		}

		prevChainHash = chainHash
		stored[i] = evt
	}

	// Advance the sequence counter to account for all appended events.
	nextSeq := int64(baseSeq) + int64(len(events))
	if _, err := tx.ExecContext(ctx,
		"UPDATE event_seq SET next_seq = ? WHERE campaign_id = ?",
		nextSeq, campaignID,
	); err != nil {
		return nil, fmt.Errorf("update event seq: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return stored, nil
}

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
func (s *Store) VerifyEventIntegrity(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if s.keyring == nil {
		return fmt.Errorf("event integrity keyring is required")
	}

	campaignIDs, err := s.listEventCampaignIDs(ctx)
	if err != nil {
		return err
	}
	for _, campaignID := range campaignIDs {
		if err := s.verifyCampaignEvents(ctx, campaignID); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) listEventCampaignIDs(ctx context.Context) ([]string, error) {
	rows, err := s.sqlDB.QueryContext(ctx, "SELECT DISTINCT campaign_id FROM events ORDER BY campaign_id")
	if err != nil {
		return nil, fmt.Errorf("list campaign ids: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan campaign id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate campaign ids: %w", err)
	}
	return ids, nil
}

func (s *Store) verifyCampaignEvents(ctx context.Context, campaignID string) error {
	var lastSeq uint64
	prevChainHash := ""
	for {
		events, err := s.ListEvents(ctx, campaignID, lastSeq, 200)
		if err != nil {
			return fmt.Errorf("list events campaign_id=%s: %w", campaignID, err)
		}
		if len(events) == 0 {
			return nil
		}
		for _, evt := range events {
			if evt.Seq != lastSeq+1 {
				return fmt.Errorf("event sequence gap campaign_id=%s expected=%d got=%d", campaignID, lastSeq+1, evt.Seq)
			}
			if evt.Seq == 1 && evt.PrevHash != "" {
				return fmt.Errorf("first event prev hash must be empty campaign_id=%s", campaignID)
			}
			if evt.Seq > 1 && evt.PrevHash != prevChainHash {
				return fmt.Errorf("prev hash mismatch campaign_id=%s seq=%d", campaignID, evt.Seq)
			}

			hash, err := integrity.EventHash(evt)
			if err != nil {
				return fmt.Errorf("compute event hash campaign_id=%s seq=%d: %w", campaignID, evt.Seq, err)
			}
			if hash != evt.Hash {
				return fmt.Errorf("event hash mismatch campaign_id=%s seq=%d", campaignID, evt.Seq)
			}

			chainHash, err := integrity.ChainHash(evt, prevChainHash)
			if err != nil {
				return fmt.Errorf("compute chain hash campaign_id=%s seq=%d: %w", campaignID, evt.Seq, err)
			}
			if chainHash != evt.ChainHash {
				return fmt.Errorf("chain hash mismatch campaign_id=%s seq=%d", campaignID, evt.Seq)
			}

			if err := s.keyring.VerifyChainHash(campaignID, chainHash, evt.Signature, evt.SignatureKeyID); err != nil {
				return fmt.Errorf("signature mismatch campaign_id=%s seq=%d: %w", campaignID, evt.Seq, err)
			}

			prevChainHash = evt.ChainHash
			lastSeq = evt.Seq
		}
	}
}

func isConstraintError(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	code := sqliteErr.Code()
	return code == sqlite3.SQLITE_CONSTRAINT || code == sqlite3.SQLITE_CONSTRAINT_UNIQUE || code == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY
}

func isSQLiteBusyError(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	code := sqliteErr.Code()
	return code == sqlite3.SQLITE_BUSY || code == sqlite3.SQLITE_LOCKED
}

func isParticipantUserConflict(err error) bool {
	if !isConstraintError(err) {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "idx_participants_campaign_user") ||
		(strings.Contains(message, "participant") && strings.Contains(message, "user_id"))
}

func isParticipantClaimConflict(err error) bool {
	if !isConstraintError(err) {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "participant_claims") ||
		strings.Contains(message, "idx_participant_claims")
}

// AppendTelemetryEvent records an operational telemetry event.
func (s *Store) AppendTelemetryEvent(ctx context.Context, evt storage.TelemetryEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(evt.EventName) == "" {
		return fmt.Errorf("event name is required")
	}
	if strings.TrimSpace(evt.Severity) == "" {
		return fmt.Errorf("severity is required")
	}
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}
	if len(evt.AttributesJSON) == 0 && len(evt.Attributes) > 0 {
		payload, err := json.Marshal(evt.Attributes)
		if err != nil {
			return fmt.Errorf("marshal telemetry attributes: %w", err)
		}
		evt.AttributesJSON = payload
	}

	return s.q.AppendTelemetryEvent(ctx, db.AppendTelemetryEventParams{
		Timestamp:      toMillis(evt.Timestamp),
		EventName:      evt.EventName,
		Severity:       evt.Severity,
		CampaignID:     toNullString(evt.CampaignID),
		SessionID:      toNullString(evt.SessionID),
		ActorType:      toNullString(evt.ActorType),
		ActorID:        toNullString(evt.ActorID),
		RequestID:      toNullString(evt.RequestID),
		InvocationID:   toNullString(evt.InvocationID),
		TraceID:        toNullString(evt.TraceID),
		SpanID:         toNullString(evt.SpanID),
		AttributesJson: evt.AttributesJSON,
	})
}

// GetGameStatistics returns aggregate counts across the game data set.
func (s *Store) GetGameStatistics(ctx context.Context, since *time.Time) (storage.GameStatistics, error) {
	if err := ctx.Err(); err != nil {
		return storage.GameStatistics{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.GameStatistics{}, fmt.Errorf("storage is not configured")
	}

	sinceValue := toNullMillis(since)

	row, err := s.q.GetGameStatistics(ctx, sinceValue)
	if err != nil {
		return storage.GameStatistics{}, fmt.Errorf("get game statistics: %w", err)
	}

	return storage.GameStatistics{
		CampaignCount:    row.CampaignCount,
		SessionCount:     row.SessionCount,
		CharacterCount:   row.CharacterCount,
		ParticipantCount: row.ParticipantCount,
	}, nil
}

func toNullString(value string) sql.NullString {
	if strings.TrimSpace(value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

// GetEventByHash retrieves an event by its content hash.
func (s *Store) GetEventByHash(ctx context.Context, hash string) (event.Event, error) {
	if err := ctx.Err(); err != nil {
		return event.Event{}, err
	}
	if s == nil || s.sqlDB == nil {
		return event.Event{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(hash) == "" {
		return event.Event{}, fmt.Errorf("event hash is required")
	}

	row, err := s.q.GetEventByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return event.Event{}, storage.ErrNotFound
		}
		return event.Event{}, fmt.Errorf("get event by hash: %w", err)
	}

	return eventRowDataToDomain(eventRowDataFromGetEventByHashRow(row))
}

// GetEventBySeq retrieves a specific event by sequence number.
func (s *Store) GetEventBySeq(ctx context.Context, campaignID string, seq uint64) (event.Event, error) {
	if err := ctx.Err(); err != nil {
		return event.Event{}, err
	}
	if s == nil || s.sqlDB == nil {
		return event.Event{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return event.Event{}, fmt.Errorf("campaign id is required")
	}

	row, err := s.q.GetEventBySeq(ctx, db.GetEventBySeqParams{
		CampaignID: campaignID,
		Seq:        int64(seq),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return event.Event{}, storage.ErrNotFound
		}
		return event.Event{}, fmt.Errorf("get event by seq: %w", err)
	}

	return eventRowDataToDomain(eventRowDataFromGetEventBySeqRow(row))
}

// ListEvents returns events ordered by sequence ascending.
func (s *Store) ListEvents(ctx context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}

	rows, err := s.q.ListEvents(ctx, db.ListEventsParams{
		CampaignID: campaignID,
		Seq:        int64(afterSeq),
		Limit:      int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	return eventRowsToDomain(rows)
}

// ListEventsBySession returns events for a specific session.
func (s *Store) ListEventsBySession(ctx context.Context, campaignID, sessionID string, afterSeq uint64, limit int) ([]event.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("session id is required")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}

	rows, err := s.q.ListEventsBySession(ctx, db.ListEventsBySessionParams{
		CampaignID: campaignID,
		SessionID:  sessionID,
		Seq:        int64(afterSeq),
		Limit:      int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list events by session: %w", err)
	}

	return eventRowsBySessionToDomain(rows)
}

// GetLatestEventSeq returns the latest event sequence number for a campaign.
func (s *Store) GetLatestEventSeq(ctx context.Context, campaignID string) (uint64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if s == nil || s.sqlDB == nil {
		return 0, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return 0, fmt.Errorf("campaign id is required")
	}

	seq, err := s.q.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return 0, fmt.Errorf("get latest event seq: %w", err)
	}

	return uint64(seq), nil
}

// ListEventsPage returns a paginated, filtered, and sorted list of events.
func (s *Store) ListEventsPage(ctx context.Context, req storage.ListEventsPageRequest) (storage.ListEventsPageResult, error) {
	if err := ctx.Err(); err != nil {
		return storage.ListEventsPageResult{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.ListEventsPageResult{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(req.CampaignID) == "" {
		return storage.ListEventsPageResult{}, fmt.Errorf("campaign id is required")
	}
	if req.PageSize <= 0 {
		req.PageSize = 50
	}
	if req.PageSize > 200 {
		req.PageSize = 200
	}

	plan := buildListEventsPageSQLPlan(req)

	// Build and execute the query
	query := fmt.Sprintf(
		"SELECT campaign_id, seq, event_hash, prev_event_hash, chain_hash, signature_key_id, event_signature, timestamp, event_type, session_id, request_id, invocation_id, actor_type, actor_id, entity_type, entity_id, system_id, system_version, correlation_id, causation_id, payload_json FROM events WHERE %s %s %s",
		plan.whereClause,
		plan.orderClause,
		plan.limitClause,
	)

	rows, err := s.sqlDB.QueryContext(ctx, query, plan.params...)
	if err != nil {
		return storage.ListEventsPageResult{}, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	events := make([]event.Event, 0, req.PageSize)
	for rows.Next() {
		var row db.Event
		if err := rows.Scan(
			&row.CampaignID,
			&row.Seq,
			&row.EventHash,
			&row.PrevEventHash,
			&row.ChainHash,
			&row.SignatureKeyID,
			&row.EventSignature,
			&row.Timestamp,
			&row.EventType,
			&row.SessionID,
			&row.RequestID,
			&row.InvocationID,
			&row.ActorType,
			&row.ActorID,
			&row.EntityType,
			&row.EntityID,
			&row.SystemID,
			&row.SystemVersion,
			&row.CorrelationID,
			&row.CausationID,
			&row.PayloadJson,
		); err != nil {
			return storage.ListEventsPageResult{}, fmt.Errorf("scan event: %w", err)
		}

		evt, err := eventRowDataToDomain(eventRowDataFromEvent(row))
		if err != nil {
			return storage.ListEventsPageResult{}, err
		}
		events = append(events, evt)
	}
	if err := rows.Err(); err != nil {
		return storage.ListEventsPageResult{}, fmt.Errorf("iterate events: %w", err)
	}

	// Determine if there are more pages
	hasMore := len(events) > req.PageSize
	if hasMore {
		events = events[:req.PageSize]
	}

	// For "previous page" navigation, reverse the results to maintain consistent order
	if req.CursorReverse {
		for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
			events[i], events[j] = events[j], events[i]
		}
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM events WHERE %s", plan.countWhereClause)
	var totalCount int
	if err := s.sqlDB.QueryRowContext(ctx, countQuery, plan.countParams...).Scan(&totalCount); err != nil {
		return storage.ListEventsPageResult{}, fmt.Errorf("count events: %w", err)
	}

	// Determine hasPrev/hasNext based on pagination direction
	result := storage.ListEventsPageResult{
		Events:     events,
		TotalCount: totalCount,
	}

	if req.CursorReverse {
		result.HasNextPage = true // We came from next, so there is a next
		result.HasPrevPage = hasMore
	} else {
		result.HasNextPage = hasMore
		result.HasPrevPage = req.CursorSeq > 0
	}

	return result, nil
}

// Snapshot Store methods

// PutSnapshot stores a snapshot.
func (s *Store) PutSnapshot(ctx context.Context, snapshot storage.Snapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(snapshot.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(snapshot.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}

	return s.q.PutSnapshot(ctx, db.PutSnapshotParams{
		CampaignID:          snapshot.CampaignID,
		SessionID:           snapshot.SessionID,
		EventSeq:            int64(snapshot.EventSeq),
		CharacterStatesJson: snapshot.CharacterStatesJSON,
		GmStateJson:         snapshot.GMStateJSON,
		SystemStateJson:     snapshot.SystemStateJSON,
		CreatedAt:           toMillis(snapshot.CreatedAt),
	})
}

// GetSnapshot retrieves a snapshot by campaign and session ID.
func (s *Store) GetSnapshot(ctx context.Context, campaignID, sessionID string) (storage.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return storage.Snapshot{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.Snapshot{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.Snapshot{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return storage.Snapshot{}, fmt.Errorf("session id is required")
	}

	row, err := s.q.GetSnapshot(ctx, db.GetSnapshotParams{
		CampaignID: campaignID,
		SessionID:  sessionID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.Snapshot{}, storage.ErrNotFound
		}
		return storage.Snapshot{}, fmt.Errorf("get snapshot: %w", err)
	}

	return dbSnapshotToDomain(row)
}

// GetLatestSnapshot retrieves the most recent snapshot for a campaign.
func (s *Store) GetLatestSnapshot(ctx context.Context, campaignID string) (storage.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return storage.Snapshot{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.Snapshot{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.Snapshot{}, fmt.Errorf("campaign id is required")
	}

	row, err := s.q.GetLatestSnapshot(ctx, campaignID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.Snapshot{}, storage.ErrNotFound
		}
		return storage.Snapshot{}, fmt.Errorf("get latest snapshot: %w", err)
	}

	return dbSnapshotToDomain(row)
}

// ListSnapshots returns snapshots ordered by event sequence descending.
func (s *Store) ListSnapshots(ctx context.Context, campaignID string, limit int) ([]storage.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}

	rows, err := s.q.ListSnapshots(ctx, db.ListSnapshotsParams{
		CampaignID: campaignID,
		Limit:      int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}

	snapshots := make([]storage.Snapshot, 0, len(rows))
	for _, row := range rows {
		snapshot, err := dbSnapshotToDomain(row)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// Campaign Fork Store methods

// GetCampaignForkMetadata retrieves fork metadata for a campaign.
func (s *Store) GetCampaignForkMetadata(ctx context.Context, campaignID string) (storage.ForkMetadata, error) {
	if err := ctx.Err(); err != nil {
		return storage.ForkMetadata{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.ForkMetadata{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.ForkMetadata{}, fmt.Errorf("campaign id is required")
	}

	row, err := s.q.GetCampaignForkMetadata(ctx, campaignID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ForkMetadata{}, storage.ErrNotFound
		}
		return storage.ForkMetadata{}, fmt.Errorf("get campaign fork metadata: %w", err)
	}

	metadata := storage.ForkMetadata{}
	if row.ParentCampaignID.Valid {
		metadata.ParentCampaignID = row.ParentCampaignID.String
	}
	if row.ForkEventSeq.Valid {
		metadata.ForkEventSeq = uint64(row.ForkEventSeq.Int64)
	}
	if row.OriginCampaignID.Valid {
		metadata.OriginCampaignID = row.OriginCampaignID.String
	}

	return metadata, nil
}

// SetCampaignForkMetadata sets fork metadata for a campaign.
func (s *Store) SetCampaignForkMetadata(ctx context.Context, campaignID string, metadata storage.ForkMetadata) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}

	var parentCampaignID sql.NullString
	if metadata.ParentCampaignID != "" {
		parentCampaignID = sql.NullString{String: metadata.ParentCampaignID, Valid: true}
	}

	var forkEventSeq sql.NullInt64
	if metadata.ForkEventSeq > 0 {
		forkEventSeq = sql.NullInt64{Int64: int64(metadata.ForkEventSeq), Valid: true}
	}

	var originCampaignID sql.NullString
	if metadata.OriginCampaignID != "" {
		originCampaignID = sql.NullString{String: metadata.OriginCampaignID, Valid: true}
	}

	return s.q.SetCampaignForkMetadata(ctx, db.SetCampaignForkMetadataParams{
		ParentCampaignID: parentCampaignID,
		ForkEventSeq:     forkEventSeq,
		OriginCampaignID: originCampaignID,
		ID:               campaignID,
	})
}

// Domain conversion helpers for events

type eventRowData struct {
	CampaignID     string
	Seq            int64
	EventHash      string
	PrevEventHash  string
	ChainHash      string
	SignatureKeyID string
	EventSignature string
	Timestamp      int64
	EventType      string
	SessionID      string
	RequestID      string
	InvocationID   string
	ActorType      string
	ActorID        string
	EntityType     string
	EntityID       string
	SystemID       string
	SystemVersion  string
	CorrelationID  string
	CausationID    string
	PayloadJSON    []byte
}

func eventRowDataToDomain(row eventRowData) (event.Event, error) {
	return event.Event{
		CampaignID:     row.CampaignID,
		Seq:            uint64(row.Seq),
		Hash:           row.EventHash,
		PrevHash:       row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		Signature:      row.EventSignature,
		Timestamp:      fromMillis(row.Timestamp),
		Type:           event.Type(row.EventType),
		SessionID:      row.SessionID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      event.ActorType(row.ActorType),
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		CorrelationID:  row.CorrelationID,
		CausationID:    row.CausationID,
		PayloadJSON:    row.PayloadJSON,
	}, nil
}

func eventRowDataFromEvent(row db.Event) eventRowData {
	return eventRowData{
		CampaignID:     row.CampaignID,
		Seq:            row.Seq,
		EventHash:      row.EventHash,
		PrevEventHash:  row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		EventSignature: row.EventSignature,
		Timestamp:      row.Timestamp,
		EventType:      row.EventType,
		SessionID:      row.SessionID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      row.ActorType,
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		CorrelationID:  row.CorrelationID,
		CausationID:    row.CausationID,
		PayloadJSON:    row.PayloadJson,
	}
}

func eventRowDataFromGetEventByHashRow(row db.GetEventByHashRow) eventRowData {
	return eventRowData{
		CampaignID:     row.CampaignID,
		Seq:            row.Seq,
		EventHash:      row.EventHash,
		PrevEventHash:  row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		EventSignature: row.EventSignature,
		Timestamp:      row.Timestamp,
		EventType:      row.EventType,
		SessionID:      row.SessionID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      row.ActorType,
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		CorrelationID:  row.CorrelationID,
		CausationID:    row.CausationID,
		PayloadJSON:    row.PayloadJson,
	}
}

func eventRowDataFromGetEventBySeqRow(row db.GetEventBySeqRow) eventRowData {
	return eventRowData{
		CampaignID:     row.CampaignID,
		Seq:            row.Seq,
		EventHash:      row.EventHash,
		PrevEventHash:  row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		EventSignature: row.EventSignature,
		Timestamp:      row.Timestamp,
		EventType:      row.EventType,
		SessionID:      row.SessionID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      row.ActorType,
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		CorrelationID:  row.CorrelationID,
		CausationID:    row.CausationID,
		PayloadJSON:    row.PayloadJson,
	}
}

func eventRowDataFromListEventsRow(row db.ListEventsRow) eventRowData {
	return eventRowData{
		CampaignID:     row.CampaignID,
		Seq:            row.Seq,
		EventHash:      row.EventHash,
		PrevEventHash:  row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		EventSignature: row.EventSignature,
		Timestamp:      row.Timestamp,
		EventType:      row.EventType,
		SessionID:      row.SessionID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      row.ActorType,
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		CorrelationID:  row.CorrelationID,
		CausationID:    row.CausationID,
		PayloadJSON:    row.PayloadJson,
	}
}

func eventRowDataFromListEventsBySessionRow(row db.ListEventsBySessionRow) eventRowData {
	return eventRowData{
		CampaignID:     row.CampaignID,
		Seq:            row.Seq,
		EventHash:      row.EventHash,
		PrevEventHash:  row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		EventSignature: row.EventSignature,
		Timestamp:      row.Timestamp,
		EventType:      row.EventType,
		SessionID:      row.SessionID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      row.ActorType,
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		CorrelationID:  row.CorrelationID,
		CausationID:    row.CausationID,
		PayloadJSON:    row.PayloadJson,
	}
}

func eventRowsToDomain(rows []db.ListEventsRow) ([]event.Event, error) {
	events := make([]event.Event, 0, len(rows))
	for _, row := range rows {
		evt, err := eventRowDataToDomain(eventRowDataFromListEventsRow(row))
		if err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	return events, nil
}

func eventRowsBySessionToDomain(rows []db.ListEventsBySessionRow) ([]event.Event, error) {
	events := make([]event.Event, 0, len(rows))
	for _, row := range rows {
		evt, err := eventRowDataToDomain(eventRowDataFromListEventsBySessionRow(row))
		if err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	return events, nil
}

func dbSnapshotToDomain(row db.Snapshot) (storage.Snapshot, error) {
	return storage.Snapshot{
		CampaignID:          row.CampaignID,
		SessionID:           row.SessionID,
		EventSeq:            uint64(row.EventSeq),
		CharacterStatesJSON: row.CharacterStatesJson,
		GMStateJSON:         row.GmStateJson,
		SystemStateJSON:     row.SystemStateJson,
		CreatedAt:           fromMillis(row.CreatedAt),
	}, nil
}
