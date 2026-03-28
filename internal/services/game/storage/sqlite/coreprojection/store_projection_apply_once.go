package coreprojection

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
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

	return s.applyProjectionEventExactlyOnce(ctx, evt, apply)
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

func (s *Store) applyProjectionEventExactlyOnce(
	ctx context.Context,
	evt event.Event,
	apply func(context.Context, event.Event, storage.ProjectionApplyTxStore) error,
) (applied bool, err error) {
	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		if sqliteutil.IsSQLiteBusyError(err) {
			return false, fmt.Errorf("begin projection apply tx %s/%d: %w", evt.CampaignID, evt.Seq, err)
		}
		return false, fmt.Errorf("begin projection apply tx: %w", err)
	}

	defer tx.Rollback()
	checkpointResult, err := tx.ExecContext(
		ctx,
		`INSERT OR IGNORE INTO projection_apply_checkpoints (campaign_id, seq, event_type, applied_at)
		 VALUES (?, ?, ?, ?)`,
		evt.CampaignID,
		int64(evt.Seq),
		string(evt.Type),
		sqliteutil.ToMillis(time.Now().UTC()),
	)
	if err != nil {
		if sqliteutil.IsSQLiteBusyError(err) {
			return false, fmt.Errorf("reserve projection apply checkpoint %s/%d: %w", evt.CampaignID, evt.Seq, err)
		}
		return false, fmt.Errorf("reserve projection apply checkpoint %s/%d: %w", evt.CampaignID, evt.Seq, err)
	}

	rowsAffected, err := checkpointResult.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("inspect projection apply checkpoint reservation %s/%d: %w", evt.CampaignID, evt.Seq, err)
	}
	if rowsAffected == 0 {
		return false, nil
	}

	if err := apply(ctx, evt, s.txStore(tx)); err != nil {
		return false, err
	}

	if err := tx.Commit(); err != nil {
		if sqliteutil.IsSQLiteBusyError(err) {
			return false, fmt.Errorf("commit projection apply tx %s/%d: %w", evt.CampaignID, evt.Seq, err)
		}
		return false, fmt.Errorf("commit projection apply tx: %w", err)
	}

	return true, nil
}
