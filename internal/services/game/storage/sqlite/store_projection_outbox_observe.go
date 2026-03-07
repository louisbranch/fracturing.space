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
