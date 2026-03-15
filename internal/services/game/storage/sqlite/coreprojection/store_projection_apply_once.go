package coreprojection

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// ApplyProjectionEventExactlyOnce applies one projection event inside a projection-db
// transaction and records a per-(campaign, seq) checkpoint to dedupe retries.
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
