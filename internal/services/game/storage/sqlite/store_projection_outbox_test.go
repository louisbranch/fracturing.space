package sqlite

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestProcessProjectionApplyOutboxShadowMarksDueRowsFailed(t *testing.T) {
	store := openTestEventsStoreWithOutbox(t, true)

	stored, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-shadow",
		Timestamp:   time.Date(2026, 2, 16, 2, 0, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-shadow",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	now := time.Now().UTC().Add(time.Minute)
	processed, err := store.ProcessProjectionApplyOutboxShadow(context.Background(), now, 10)
	if err != nil {
		t.Fatalf("process projection apply outbox shadow: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected one processed outbox row, got %d", processed)
	}

	var (
		status      string
		attempts    int
		nextAttempt int64
		lastError   string
	)
	if err := store.sqlDB.QueryRowContext(
		context.Background(),
		`SELECT status, attempt_count, next_attempt_at, last_error
		 FROM projection_apply_outbox
		 WHERE campaign_id = ? AND seq = ?`,
		stored.CampaignID,
		stored.Seq,
	).Scan(&status, &attempts, &nextAttempt, &lastError); err != nil {
		t.Fatalf("query outbox row: %v", err)
	}
	if status != "failed" {
		t.Fatalf("expected status failed, got %q", status)
	}
	if attempts != 1 {
		t.Fatalf("expected attempt count 1, got %d", attempts)
	}
	if nextAttempt <= now.UnixMilli() {
		t.Fatalf("expected next attempt after now, got %d", nextAttempt)
	}
	if !strings.Contains(lastError, "shadow worker") {
		t.Fatalf("expected shadow worker error marker, got %q", lastError)
	}
}

func TestProcessProjectionApplyOutboxShadowSkipsNotDueRows(t *testing.T) {
	store := openTestEventsStoreWithOutbox(t, true)

	stored, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-shadow-future",
		Timestamp:   time.Date(2026, 2, 16, 3, 0, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-shadow-future",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	now := time.Date(2026, 2, 16, 3, 1, 0, 0, time.UTC)
	nextAttempt := now.Add(30 * time.Minute)
	if _, err := store.sqlDB.ExecContext(
		context.Background(),
		`UPDATE projection_apply_outbox SET next_attempt_at = ? WHERE campaign_id = ? AND seq = ?`,
		nextAttempt.UnixMilli(),
		stored.CampaignID,
		stored.Seq,
	); err != nil {
		t.Fatalf("prepare future outbox row: %v", err)
	}

	processed, err := store.ProcessProjectionApplyOutboxShadow(context.Background(), now, 10)
	if err != nil {
		t.Fatalf("process projection apply outbox shadow: %v", err)
	}
	if processed != 0 {
		t.Fatalf("expected zero processed outbox rows, got %d", processed)
	}
}

func TestProcessProjectionApplyOutboxShadowZeroLimitNoop(t *testing.T) {
	store := openTestEventsStoreWithOutbox(t, true)

	processed, err := store.ProcessProjectionApplyOutboxShadow(context.Background(), time.Now().UTC(), 0)
	if err != nil {
		t.Fatalf("process projection apply outbox shadow: %v", err)
	}
	if processed != 0 {
		t.Fatalf("expected zero processed outbox rows, got %d", processed)
	}
}

func TestMarkProjectionApplyOutboxShadowRetryRequiresProcessingStatus(t *testing.T) {
	store := openTestEventsStoreWithOutbox(t, true)

	stored, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-shadow-mark",
		Timestamp:   time.Date(2026, 2, 16, 4, 0, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-shadow-mark",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	err = store.markProjectionApplyOutboxShadowRetry(
		context.Background(),
		projectionApplyOutboxRow{CampaignID: stored.CampaignID, Seq: stored.Seq},
		time.Now().UTC(),
		1,
		time.Now().UTC().Add(time.Second),
	)
	if err == nil {
		t.Fatal("expected mark retry to fail when row is not in processing status")
	}
	if !strings.Contains(err.Error(), "expected 1 row updated") {
		t.Fatalf("expected rows-updated error, got %v", err)
	}
}

func TestOutboxRetryBackoffBounds(t *testing.T) {
	if got := outboxRetryBackoff(0); got != time.Second {
		t.Fatalf("expected attempt zero backoff of 1s, got %s", got)
	}
	if got := outboxRetryBackoff(1); got != time.Second {
		t.Fatalf("expected attempt one backoff of 1s, got %s", got)
	}
	if got := outboxRetryBackoff(2); got != 2*time.Second {
		t.Fatalf("expected attempt two backoff of 2s, got %s", got)
	}
	if got := outboxRetryBackoff(20); got != 5*time.Minute {
		t.Fatalf("expected capped backoff of 5m, got %s", got)
	}
}

func TestProcessProjectionApplyOutboxAppliesAndDeletesOnSuccess(t *testing.T) {
	store := openTestEventsStoreWithOutbox(t, true)

	stored, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-apply-success",
		Timestamp:   time.Date(2026, 2, 16, 6, 0, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-apply-success",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	calls := 0
	now := time.Now().UTC().Add(time.Minute)
	processed, err := store.ProcessProjectionApplyOutbox(
		context.Background(),
		now,
		10,
		func(_ context.Context, evt event.Event) error {
			calls++
			if evt.CampaignID != stored.CampaignID || evt.Seq != stored.Seq {
				t.Fatalf("unexpected event in apply callback: %+v", evt)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("process projection apply outbox: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected one processed row, got %d", processed)
	}
	if calls != 1 {
		t.Fatalf("expected one apply callback invocation, got %d", calls)
	}

	var count int
	if err := store.sqlDB.QueryRowContext(
		context.Background(),
		`SELECT COUNT(*) FROM projection_apply_outbox WHERE campaign_id = ? AND seq = ?`,
		stored.CampaignID,
		stored.Seq,
	).Scan(&count); err != nil {
		t.Fatalf("query outbox row count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected outbox row to be deleted after success, got %d", count)
	}
}

func TestProcessProjectionApplyOutboxReclaimsStaleProcessingRows(t *testing.T) {
	store := openTestEventsStoreWithOutbox(t, true)

	stored, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-apply-stale-processing",
		Timestamp:   time.Date(2026, 2, 16, 6, 5, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-apply-stale-processing",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	now := time.Date(2026, 2, 16, 7, 0, 0, 0, time.UTC)
	if _, err := store.sqlDB.ExecContext(
		context.Background(),
		`UPDATE projection_apply_outbox
		 SET status = 'processing', attempt_count = 1, next_attempt_at = ?, updated_at = ?
		 WHERE campaign_id = ? AND seq = ?`,
		now.Add(-10*time.Minute).UnixMilli(),
		now.Add(-10*time.Minute).UnixMilli(),
		stored.CampaignID,
		stored.Seq,
	); err != nil {
		t.Fatalf("prepare stale processing row: %v", err)
	}

	calls := 0
	processed, err := store.ProcessProjectionApplyOutbox(
		context.Background(),
		now,
		10,
		func(_ context.Context, evt event.Event) error {
			calls++
			if evt.CampaignID != stored.CampaignID || evt.Seq != stored.Seq {
				t.Fatalf("unexpected event in apply callback: %+v", evt)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("process projection apply outbox: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected one processed stale processing row, got %d", processed)
	}
	if calls != 1 {
		t.Fatalf("expected one apply callback invocation, got %d", calls)
	}

	var count int
	if err := store.sqlDB.QueryRowContext(
		context.Background(),
		`SELECT COUNT(*) FROM projection_apply_outbox WHERE campaign_id = ? AND seq = ?`,
		stored.CampaignID,
		stored.Seq,
	).Scan(&count); err != nil {
		t.Fatalf("query outbox row count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected stale processing outbox row to be deleted after success, got %d", count)
	}
}

func TestProcessProjectionApplyOutboxApplyFailureMarksRetry(t *testing.T) {
	store := openTestEventsStoreWithOutbox(t, true)

	stored, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-apply-failure",
		Timestamp:   time.Date(2026, 2, 16, 7, 0, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-apply-failure",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	now := time.Now().UTC().Add(time.Minute)
	processed, err := store.ProcessProjectionApplyOutbox(
		context.Background(),
		now,
		10,
		func(context.Context, event.Event) error {
			return fmt.Errorf("apply failed")
		},
	)
	if err != nil {
		t.Fatalf("process projection apply outbox: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected one processed row, got %d", processed)
	}

	var (
		status      string
		attempts    int
		nextAttempt int64
		lastError   string
	)
	if err := store.sqlDB.QueryRowContext(
		context.Background(),
		`SELECT status, attempt_count, next_attempt_at, last_error
		 FROM projection_apply_outbox
		 WHERE campaign_id = ? AND seq = ?`,
		stored.CampaignID,
		stored.Seq,
	).Scan(&status, &attempts, &nextAttempt, &lastError); err != nil {
		t.Fatalf("query outbox row: %v", err)
	}
	if status != "failed" {
		t.Fatalf("expected status failed, got %q", status)
	}
	if attempts != 1 {
		t.Fatalf("expected attempt count 1, got %d", attempts)
	}
	if nextAttempt <= now.UnixMilli() {
		t.Fatalf("expected next attempt after now, got %d", nextAttempt)
	}
	if !strings.Contains(lastError, "apply failed") {
		t.Fatalf("expected apply error details, got %q", lastError)
	}
}

func TestGetProjectionApplyOutboxSummaryCountsAndOldestPending(t *testing.T) {
	store := openTestEventsStoreWithOutbox(t, true)

	pending, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-summary-pending",
		Timestamp:   time.Date(2026, 2, 16, 8, 0, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-summary-pending",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append pending event: %v", err)
	}
	failed, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-summary-failed",
		Timestamp:   time.Date(2026, 2, 16, 8, 0, 1, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-summary-failed",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append failed event: %v", err)
	}
	processing, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-summary-processing",
		Timestamp:   time.Date(2026, 2, 16, 8, 0, 2, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-summary-processing",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append processing event: %v", err)
	}
	dead, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-summary-dead",
		Timestamp:   time.Date(2026, 2, 16, 8, 0, 3, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-summary-dead",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append dead event: %v", err)
	}

	now := time.Date(2026, 2, 16, 8, 1, 0, 0, time.UTC)
	if _, err := store.sqlDB.ExecContext(
		context.Background(),
		`UPDATE projection_apply_outbox
		 SET status = 'failed', attempt_count = 2, next_attempt_at = ?, updated_at = ?
		 WHERE campaign_id = ? AND seq = ?`,
		now.Add(-3*time.Minute).UnixMilli(),
		now.UnixMilli(),
		failed.CampaignID,
		failed.Seq,
	); err != nil {
		t.Fatalf("prepare failed row: %v", err)
	}
	if _, err := store.sqlDB.ExecContext(
		context.Background(),
		`UPDATE projection_apply_outbox
		 SET status = 'processing', next_attempt_at = ?, updated_at = ?
		 WHERE campaign_id = ? AND seq = ?`,
		now.Add(-2*time.Minute).UnixMilli(),
		now.UnixMilli(),
		processing.CampaignID,
		processing.Seq,
	); err != nil {
		t.Fatalf("prepare processing row: %v", err)
	}
	if _, err := store.sqlDB.ExecContext(
		context.Background(),
		`UPDATE projection_apply_outbox
		 SET status = 'dead', attempt_count = 8, next_attempt_at = ?, updated_at = ?
		 WHERE campaign_id = ? AND seq = ?`,
		now.Add(-10*time.Minute).UnixMilli(),
		now.UnixMilli(),
		dead.CampaignID,
		dead.Seq,
	); err != nil {
		t.Fatalf("prepare dead row: %v", err)
	}
	if _, err := store.sqlDB.ExecContext(
		context.Background(),
		`UPDATE projection_apply_outbox
		 SET next_attempt_at = ?, updated_at = ?
		 WHERE campaign_id = ? AND seq = ?`,
		now.Add(-time.Minute).UnixMilli(),
		now.UnixMilli(),
		pending.CampaignID,
		pending.Seq,
	); err != nil {
		t.Fatalf("prepare pending row: %v", err)
	}

	summary, err := store.GetProjectionApplyOutboxSummary(context.Background())
	if err != nil {
		t.Fatalf("get outbox summary: %v", err)
	}
	if summary.PendingCount != 1 || summary.FailedCount != 1 || summary.ProcessingCount != 1 || summary.DeadCount != 1 {
		t.Fatalf("unexpected summary counts: %+v", summary)
	}
	if summary.OldestPendingCampaignID != failed.CampaignID || summary.OldestPendingSeq != failed.Seq {
		t.Fatalf("unexpected oldest pending row: %+v", summary)
	}
	if summary.OldestPendingAt.IsZero() || !summary.OldestPendingAt.Equal(now.Add(-3*time.Minute)) {
		t.Fatalf("unexpected oldest pending timestamp: %s", summary.OldestPendingAt)
	}
}

func TestListProjectionApplyOutboxRowsFiltersOrdersAndLimits(t *testing.T) {
	store := openTestEventsStoreWithOutbox(t, true)

	failedFirst, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-list-failed-1",
		Timestamp:   time.Date(2026, 2, 16, 9, 0, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-list-failed-1",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append failed first event: %v", err)
	}
	failedSecond, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-list-failed-2",
		Timestamp:   time.Date(2026, 2, 16, 9, 0, 1, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-list-failed-2",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append failed second event: %v", err)
	}
	pending, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-list-pending",
		Timestamp:   time.Date(2026, 2, 16, 9, 0, 2, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-list-pending",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append pending event: %v", err)
	}

	now := time.Date(2026, 2, 16, 9, 1, 0, 0, time.UTC)
	if _, err := store.sqlDB.ExecContext(
		context.Background(),
		`UPDATE projection_apply_outbox
		 SET status = 'failed', attempt_count = 1, next_attempt_at = ?, updated_at = ?
		 WHERE campaign_id = ? AND seq = ?`,
		now.Add(-5*time.Minute).UnixMilli(),
		now.UnixMilli(),
		failedFirst.CampaignID,
		failedFirst.Seq,
	); err != nil {
		t.Fatalf("prepare failed first row: %v", err)
	}
	if _, err := store.sqlDB.ExecContext(
		context.Background(),
		`UPDATE projection_apply_outbox
		 SET status = 'failed', attempt_count = 2, next_attempt_at = ?, updated_at = ?
		 WHERE campaign_id = ? AND seq = ?`,
		now.Add(-2*time.Minute).UnixMilli(),
		now.UnixMilli(),
		failedSecond.CampaignID,
		failedSecond.Seq,
	); err != nil {
		t.Fatalf("prepare failed second row: %v", err)
	}
	if _, err := store.sqlDB.ExecContext(
		context.Background(),
		`UPDATE projection_apply_outbox
		 SET next_attempt_at = ?, updated_at = ?
		 WHERE campaign_id = ? AND seq = ?`,
		now.Add(-time.Minute).UnixMilli(),
		now.UnixMilli(),
		pending.CampaignID,
		pending.Seq,
	); err != nil {
		t.Fatalf("prepare pending row: %v", err)
	}

	failedRows, err := store.ListProjectionApplyOutboxRows(context.Background(), "failed", 10)
	if err != nil {
		t.Fatalf("list failed outbox rows: %v", err)
	}
	if len(failedRows) != 2 {
		t.Fatalf("expected two failed rows, got %d", len(failedRows))
	}
	if failedRows[0].CampaignID != failedFirst.CampaignID || failedRows[1].CampaignID != failedSecond.CampaignID {
		t.Fatalf("expected failed rows ordered by next_attempt_at asc, got %+v", failedRows)
	}
	if failedRows[0].Status != "failed" || failedRows[1].Status != "failed" {
		t.Fatalf("expected failed status rows, got %+v", failedRows)
	}
	if failedRows[0].AttemptCount != 1 || failedRows[1].AttemptCount != 2 {
		t.Fatalf("expected attempt counts preserved, got %+v", failedRows)
	}
	if failedRows[0].Seq != failedFirst.Seq || failedRows[1].Seq != failedSecond.Seq {
		t.Fatalf("expected sequences preserved, got %+v", failedRows)
	}

	limitedRows, err := store.ListProjectionApplyOutboxRows(context.Background(), "failed", 1)
	if err != nil {
		t.Fatalf("list failed outbox rows with limit: %v", err)
	}
	if len(limitedRows) != 1 || limitedRows[0].CampaignID != failedFirst.CampaignID {
		t.Fatalf("expected one oldest failed row, got %+v", limitedRows)
	}

	if _, err := store.ListProjectionApplyOutboxRows(context.Background(), "invalid-status", 5); err == nil {
		t.Fatal("expected invalid status error")
	}
}

func TestGetProjectionApplyOutboxSummaryNoRows(t *testing.T) {
	store := openTestEventsStoreWithOutbox(t, true)

	summary, err := store.GetProjectionApplyOutboxSummary(context.Background())
	if err != nil {
		t.Fatalf("get outbox summary: %v", err)
	}
	if summary.PendingCount != 0 || summary.ProcessingCount != 0 || summary.FailedCount != 0 || summary.DeadCount != 0 {
		t.Fatalf("expected zero counts, got %+v", summary)
	}
	if summary.OldestPendingCampaignID != "" || summary.OldestPendingSeq != 0 || !summary.OldestPendingAt.IsZero() {
		t.Fatalf("expected no oldest pending metadata, got %+v", summary)
	}
}

func TestListProjectionApplyOutboxRowsAllStatusesAndZeroLimit(t *testing.T) {
	store := openTestEventsStoreWithOutbox(t, true)

	first, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-list-all-1",
		Timestamp:   time.Date(2026, 2, 16, 9, 30, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-list-all-1",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append first event: %v", err)
	}
	second, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-list-all-2",
		Timestamp:   time.Date(2026, 2, 16, 9, 30, 1, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-list-all-2",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append second event: %v", err)
	}

	if _, err := store.sqlDB.ExecContext(
		context.Background(),
		`UPDATE projection_apply_outbox SET status = 'dead', attempt_count = 8 WHERE campaign_id = ? AND seq = ?`,
		second.CampaignID,
		second.Seq,
	); err != nil {
		t.Fatalf("prepare dead row: %v", err)
	}

	rows, err := store.ListProjectionApplyOutboxRows(context.Background(), "", 10)
	if err != nil {
		t.Fatalf("list all outbox rows: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected two rows, got %d", len(rows))
	}
	if rows[0].CampaignID != first.CampaignID && rows[1].CampaignID != first.CampaignID {
		t.Fatalf("expected first row campaign to be present, got %+v", rows)
	}
	if rows[0].CampaignID != second.CampaignID && rows[1].CampaignID != second.CampaignID {
		t.Fatalf("expected second row campaign to be present, got %+v", rows)
	}

	none, err := store.ListProjectionApplyOutboxRows(context.Background(), "", 0)
	if err != nil {
		t.Fatalf("list outbox rows zero limit: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("expected no rows for zero limit, got %d", len(none))
	}
}

func TestProcessProjectionApplyOutboxMarksDeadAfterThreshold(t *testing.T) {
	store := openTestEventsStoreWithOutbox(t, true)

	stored, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-apply-dead",
		Timestamp:   time.Date(2026, 2, 16, 9, 45, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-apply-dead",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	now := time.Date(2026, 2, 16, 9, 46, 0, 0, time.UTC)
	if _, err := store.sqlDB.ExecContext(
		context.Background(),
		`UPDATE projection_apply_outbox
		 SET status = 'failed', attempt_count = 7, next_attempt_at = ?, updated_at = ?
		 WHERE campaign_id = ? AND seq = ?`,
		now.Add(-time.Minute).UnixMilli(),
		now.UnixMilli(),
		stored.CampaignID,
		stored.Seq,
	); err != nil {
		t.Fatalf("prepare failed row near dead threshold: %v", err)
	}

	processed, err := store.ProcessProjectionApplyOutbox(
		context.Background(),
		now,
		10,
		func(context.Context, event.Event) error {
			return fmt.Errorf("still failing")
		},
	)
	if err != nil {
		t.Fatalf("process projection apply outbox: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected one processed row, got %d", processed)
	}

	var (
		status   string
		attempts int
	)
	if err := store.sqlDB.QueryRowContext(
		context.Background(),
		`SELECT status, attempt_count
		 FROM projection_apply_outbox
		 WHERE campaign_id = ? AND seq = ?`,
		stored.CampaignID,
		stored.Seq,
	).Scan(&status, &attempts); err != nil {
		t.Fatalf("query outbox row: %v", err)
	}
	if status != "dead" {
		t.Fatalf("expected status dead, got %q", status)
	}
	if attempts != 8 {
		t.Fatalf("expected attempt count 8 at dead threshold, got %d", attempts)
	}
}

func TestProcessProjectionApplyOutboxLoadFailureMarksRetry(t *testing.T) {
	store := openTestEventsStoreWithOutbox(t, true)

	now := time.Date(2026, 2, 16, 10, 0, 0, 0, time.UTC)
	if _, err := store.sqlDB.ExecContext(
		context.Background(),
		`INSERT INTO projection_apply_outbox (
			campaign_id, seq, event_type, status, attempt_count, next_attempt_at, updated_at
		) VALUES (?, ?, ?, 'pending', 0, ?, ?)`,
		"camp-outbox-missing-event",
		999,
		"campaign.created",
		now.Add(-time.Minute).UnixMilli(),
		now.UnixMilli(),
	); err != nil {
		t.Fatalf("insert outbox row: %v", err)
	}

	processed, err := store.ProcessProjectionApplyOutbox(
		context.Background(),
		now,
		10,
		func(context.Context, event.Event) error { return nil },
	)
	if err != nil {
		t.Fatalf("process projection apply outbox: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected one processed row, got %d", processed)
	}

	var (
		status    string
		attempts  int
		lastError string
	)
	if err := store.sqlDB.QueryRowContext(
		context.Background(),
		`SELECT status, attempt_count, last_error
		 FROM projection_apply_outbox
		 WHERE campaign_id = ? AND seq = ?`,
		"camp-outbox-missing-event",
		999,
	).Scan(&status, &attempts, &lastError); err != nil {
		t.Fatalf("query outbox row: %v", err)
	}
	if status != "failed" {
		t.Fatalf("expected failed status for missing event, got %q", status)
	}
	if attempts != 1 {
		t.Fatalf("expected attempt count 1, got %d", attempts)
	}
	if !strings.Contains(lastError, "load event") {
		t.Fatalf("expected load event error marker, got %q", lastError)
	}
}

func TestProcessProjectionApplyOutboxRequiresApplyCallback(t *testing.T) {
	store := openTestEventsStoreWithOutbox(t, true)

	if _, err := store.ProcessProjectionApplyOutbox(
		context.Background(),
		time.Now().UTC(),
		10,
		nil,
	); err == nil {
		t.Fatal("expected missing apply callback error")
	}
}

func TestCompleteProjectionApplyOutboxRowRequiresProcessingStatus(t *testing.T) {
	store := openTestEventsStoreWithOutbox(t, true)

	stored, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-complete-status",
		Timestamp:   time.Date(2026, 2, 16, 10, 15, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-complete-status",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	err = store.completeProjectionApplyOutboxRow(
		context.Background(),
		projectionApplyOutboxRow{
			CampaignID: stored.CampaignID,
			Seq:        stored.Seq,
		},
	)
	if err == nil {
		t.Fatal("expected processing-status guard error")
	}
}

func TestProjectionApplyOutboxSummaryAndListRespectCanceledContext(t *testing.T) {
	store := openTestEventsStoreWithOutbox(t, true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := store.GetProjectionApplyOutboxSummary(ctx); err == nil {
		t.Fatal("expected context cancellation error for summary")
	}
	if _, err := store.ListProjectionApplyOutboxRows(ctx, "", 10); err == nil {
		t.Fatal("expected context cancellation error for listing")
	}
}

func TestRequeueProjectionApplyOutboxRowTransitionsDeadToPending(t *testing.T) {
	store := openTestEventsStoreWithOutbox(t, true)

	stored, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-requeue-dead",
		Timestamp:   time.Date(2026, 2, 16, 10, 30, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-requeue-dead",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	now := time.Date(2026, 2, 16, 10, 31, 0, 0, time.UTC)
	if _, err := store.sqlDB.ExecContext(
		context.Background(),
		`UPDATE projection_apply_outbox
		 SET status = 'dead', attempt_count = 8, next_attempt_at = ?, last_error = 'failed permanently', updated_at = ?
		 WHERE campaign_id = ? AND seq = ?`,
		now.Add(-2*time.Minute).UnixMilli(),
		now.Add(-time.Minute).UnixMilli(),
		stored.CampaignID,
		stored.Seq,
	); err != nil {
		t.Fatalf("prepare dead row: %v", err)
	}

	requeued, err := store.RequeueProjectionApplyOutboxRow(context.Background(), stored.CampaignID, stored.Seq, now)
	if err != nil {
		t.Fatalf("requeue dead outbox row: %v", err)
	}
	if !requeued {
		t.Fatal("expected dead outbox row to be requeued")
	}

	var (
		status      string
		attempts    int
		nextAttempt int64
		lastError   string
	)
	if err := store.sqlDB.QueryRowContext(
		context.Background(),
		`SELECT status, attempt_count, next_attempt_at, last_error
		 FROM projection_apply_outbox
		 WHERE campaign_id = ? AND seq = ?`,
		stored.CampaignID,
		stored.Seq,
	).Scan(&status, &attempts, &nextAttempt, &lastError); err != nil {
		t.Fatalf("query outbox row: %v", err)
	}
	if status != "pending" {
		t.Fatalf("expected status pending after requeue, got %q", status)
	}
	if attempts != 0 {
		t.Fatalf("expected attempt count reset to 0, got %d", attempts)
	}
	if nextAttempt != now.UnixMilli() {
		t.Fatalf("expected next attempt set to now, got %d", nextAttempt)
	}
	if lastError != "" {
		t.Fatalf("expected last error cleared after requeue, got %q", lastError)
	}
}

func TestRequeueProjectionApplyOutboxRowReturnsFalseWhenNotDead(t *testing.T) {
	store := openTestEventsStoreWithOutbox(t, true)

	stored, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-requeue-pending",
		Timestamp:   time.Date(2026, 2, 16, 10, 35, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-requeue-pending",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	requeued, err := store.RequeueProjectionApplyOutboxRow(
		context.Background(),
		stored.CampaignID,
		stored.Seq,
		time.Date(2026, 2, 16, 10, 36, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("requeue non-dead outbox row: %v", err)
	}
	if requeued {
		t.Fatal("expected non-dead outbox row to remain unchanged")
	}
}

func TestRequeueProjectionApplyOutboxRowValidationErrors(t *testing.T) {
	store := openTestEventsStoreWithOutbox(t, true)

	if _, err := store.RequeueProjectionApplyOutboxRow(
		context.Background(),
		"",
		1,
		time.Now().UTC(),
	); err == nil {
		t.Fatal("expected missing campaign id error")
	}
	if _, err := store.RequeueProjectionApplyOutboxRow(
		context.Background(),
		"camp-1",
		0,
		time.Now().UTC(),
	); err == nil {
		t.Fatal("expected invalid sequence error")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.RequeueProjectionApplyOutboxRow(
		ctx,
		"camp-1",
		1,
		time.Now().UTC(),
	); err == nil {
		t.Fatal("expected canceled context error")
	}
}

func TestRequeueProjectionApplyOutboxDeadRowsRequeuesByLimitAndOrder(t *testing.T) {
	store := openTestEventsStoreWithOutbox(t, true)

	evtA, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-requeue-batch-a",
		Timestamp:   time.Date(2026, 2, 16, 11, 0, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-requeue-batch-a",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append event A: %v", err)
	}
	evtB, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-requeue-batch-b",
		Timestamp:   time.Date(2026, 2, 16, 11, 0, 1, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-requeue-batch-b",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append event B: %v", err)
	}
	evtC, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-requeue-batch-c",
		Timestamp:   time.Date(2026, 2, 16, 11, 0, 2, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-requeue-batch-c",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append event C: %v", err)
	}

	now := time.Date(2026, 2, 16, 11, 5, 0, 0, time.UTC)
	if _, err := store.sqlDB.ExecContext(
		context.Background(),
		`UPDATE projection_apply_outbox
		 SET status = 'dead', attempt_count = 8, next_attempt_at = ?, last_error = 'dead-A', updated_at = ?
		 WHERE campaign_id = ? AND seq = ?`,
		now.Add(-10*time.Minute).UnixMilli(),
		now.Add(-10*time.Minute).UnixMilli(),
		evtA.CampaignID,
		evtA.Seq,
	); err != nil {
		t.Fatalf("prepare dead row A: %v", err)
	}
	if _, err := store.sqlDB.ExecContext(
		context.Background(),
		`UPDATE projection_apply_outbox
		 SET status = 'dead', attempt_count = 8, next_attempt_at = ?, last_error = 'dead-B', updated_at = ?
		 WHERE campaign_id = ? AND seq = ?`,
		now.Add(-8*time.Minute).UnixMilli(),
		now.Add(-8*time.Minute).UnixMilli(),
		evtB.CampaignID,
		evtB.Seq,
	); err != nil {
		t.Fatalf("prepare dead row B: %v", err)
	}
	if _, err := store.sqlDB.ExecContext(
		context.Background(),
		`UPDATE projection_apply_outbox
		 SET status = 'dead', attempt_count = 8, next_attempt_at = ?, last_error = 'dead-C', updated_at = ?
		 WHERE campaign_id = ? AND seq = ?`,
		now.Add(-6*time.Minute).UnixMilli(),
		now.Add(-6*time.Minute).UnixMilli(),
		evtC.CampaignID,
		evtC.Seq,
	); err != nil {
		t.Fatalf("prepare dead row C: %v", err)
	}

	requeued, err := store.RequeueProjectionApplyOutboxDeadRows(context.Background(), 2, now)
	if err != nil {
		t.Fatalf("bulk requeue dead outbox rows: %v", err)
	}
	if requeued != 2 {
		t.Fatalf("expected two rows requeued, got %d", requeued)
	}

	type rowState struct {
		status      string
		attempts    int
		nextAttempt int64
		lastError   string
	}
	fetch := func(campaignID string, seq uint64) rowState {
		t.Helper()
		var state rowState
		if err := store.sqlDB.QueryRowContext(
			context.Background(),
			`SELECT status, attempt_count, next_attempt_at, last_error
			 FROM projection_apply_outbox
			 WHERE campaign_id = ? AND seq = ?`,
			campaignID,
			seq,
		).Scan(&state.status, &state.attempts, &state.nextAttempt, &state.lastError); err != nil {
			t.Fatalf("query row state %s/%d: %v", campaignID, seq, err)
		}
		return state
	}

	stateA := fetch(evtA.CampaignID, evtA.Seq)
	stateB := fetch(evtB.CampaignID, evtB.Seq)
	stateC := fetch(evtC.CampaignID, evtC.Seq)

	if stateA.status != "pending" || stateB.status != "pending" {
		t.Fatalf("expected oldest two rows requeued to pending, got A=%q B=%q", stateA.status, stateB.status)
	}
	if stateA.attempts != 0 || stateB.attempts != 0 {
		t.Fatalf("expected attempts reset for requeued rows, got A=%d B=%d", stateA.attempts, stateB.attempts)
	}
	if stateA.nextAttempt != now.UnixMilli() || stateB.nextAttempt != now.UnixMilli() {
		t.Fatalf("expected next attempt reset to now for requeued rows, got A=%d B=%d", stateA.nextAttempt, stateB.nextAttempt)
	}
	if stateA.lastError != "" || stateB.lastError != "" {
		t.Fatalf("expected last_error cleared for requeued rows, got A=%q B=%q", stateA.lastError, stateB.lastError)
	}
	if stateC.status != "dead" || stateC.attempts != 8 || stateC.lastError != "dead-C" {
		t.Fatalf("expected newest dead row unchanged, got %+v", stateC)
	}
}

func TestRequeueProjectionApplyOutboxDeadRowsValidationAndNoRows(t *testing.T) {
	store := openTestEventsStoreWithOutbox(t, true)

	if _, err := store.RequeueProjectionApplyOutboxDeadRows(
		context.Background(),
		0,
		time.Now().UTC(),
	); err == nil {
		t.Fatal("expected invalid limit error")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.RequeueProjectionApplyOutboxDeadRows(
		ctx,
		10,
		time.Now().UTC(),
	); err == nil {
		t.Fatal("expected canceled context error")
	}

	requeued, err := store.RequeueProjectionApplyOutboxDeadRows(
		context.Background(),
		10,
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("bulk requeue with no dead rows: %v", err)
	}
	if requeued != 0 {
		t.Fatalf("expected zero rows requeued, got %d", requeued)
	}
}
