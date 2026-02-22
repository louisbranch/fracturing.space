package maintenance

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/integrity"
	storagesqlite "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite"
)

type fakeOutboxInspector struct {
	summary storagesqlite.ProjectionApplyOutboxSummary
	rows    []storagesqlite.ProjectionApplyOutboxEntry
	err     error
}

func (f *fakeOutboxInspector) GetProjectionApplyOutboxSummary(context.Context) (storagesqlite.ProjectionApplyOutboxSummary, error) {
	if f.err != nil {
		return storagesqlite.ProjectionApplyOutboxSummary{}, f.err
	}
	return f.summary, nil
}

func (f *fakeOutboxInspector) ListProjectionApplyOutboxRows(context.Context, string, int) ([]storagesqlite.ProjectionApplyOutboxEntry, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.rows, nil
}

type fakeOutboxRequeuer struct {
	requeued     bool
	deadRequeued int
	err          error
}

func (f *fakeOutboxRequeuer) RequeueProjectionApplyOutboxRow(context.Context, string, uint64, time.Time) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.requeued, nil
}

func (f *fakeOutboxRequeuer) RequeueProjectionApplyOutboxDeadRows(context.Context, int, time.Time) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.deadRequeued, nil
}

func TestResolveCampaignIDs(t *testing.T) {
	tests := []struct {
		single   string
		list     string
		expected []string
		wantErr  bool
	}{
		{single: "", list: "", wantErr: true},
		{single: "c1", list: "c2", wantErr: true},
		{single: "c1", list: "", expected: []string{"c1"}},
		{single: "", list: "c1, c2", expected: []string{"c1", "c2"}},
		{single: "", list: " , c1 , , c2 ", expected: []string{"c1", "c2"}},
	}

	for _, tc := range tests {
		got, err := resolveCampaignIDs(tc.single, tc.list)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("expected error for %q/%q", tc.single, tc.list)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error for %q/%q: %v", tc.single, tc.list, err)
		}
		if !reflect.DeepEqual(got, tc.expected) {
			t.Fatalf("expected %v, got %v", tc.expected, got)
		}
	}
}

func TestSplitCSV(t *testing.T) {
	if got := splitCSV(" a, b ,, "); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("expected trimmed entries, got %v", got)
	}
}

func TestCapWarnings(t *testing.T) {
	warnings := []string{"a", "b", "c"}
	if got, total := capWarnings(warnings, 0); total != 3 || len(got) != 3 {
		t.Fatalf("expected all warnings, got %v (total=%d)", got, total)
	}
	if got, total := capWarnings(warnings, 2); total != 3 || len(got) != 2 {
		t.Fatalf("expected capped warnings, got %v (total=%d)", got, total)
	}
}

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("maintenance", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.EventsDBPath != "data/game-events.db" {
		t.Fatalf("expected default events db path, got %q", cfg.EventsDBPath)
	}
	if cfg.ProjectionsDBPath != "data/game-projections.db" {
		t.Fatalf("expected default projections db path, got %q", cfg.ProjectionsDBPath)
	}
	if cfg.WarningsCap != 25 {
		t.Fatalf("expected warnings cap 25, got %d", cfg.WarningsCap)
	}
}

func TestParseConfigIgnoresUntaggedEnv(t *testing.T) {
	t.Setenv("CAMPAIGNID", "c1")

	fs := flag.NewFlagSet("maintenance", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.CampaignID != "" {
		t.Fatalf("expected empty campaign id, got %q", cfg.CampaignID)
	}
}

func TestParseConfigOverrides(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_EVENTS_DB_PATH", "env-events")
	t.Setenv("FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH", "env-projections")

	fs := flag.NewFlagSet("maintenance", flag.ContinueOnError)
	args := []string{
		"-events-db-path", "flag-events",
		"-projections-db-path", "flag-projections",
		"-warnings-cap", "5",
	}
	cfg, err := ParseConfig(fs, args)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.EventsDBPath != "flag-events" {
		t.Fatalf("expected flag override for events db, got %q", cfg.EventsDBPath)
	}
	if cfg.ProjectionsDBPath != "flag-projections" {
		t.Fatalf("expected flag override for projections db, got %q", cfg.ProjectionsDBPath)
	}
	if cfg.WarningsCap != 5 {
		t.Fatalf("expected warnings cap 5, got %d", cfg.WarningsCap)
	}
}

func TestParseConfigOutboxFlags(t *testing.T) {
	fs := flag.NewFlagSet("maintenance", flag.ContinueOnError)
	args := []string{
		"-outbox-report",
		"-outbox-status", "failed",
		"-outbox-limit", "7",
	}
	cfg, err := ParseConfig(fs, args)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if !cfg.OutboxReport {
		t.Fatal("expected outbox report mode to be enabled")
	}
	if cfg.OutboxStatus != "failed" {
		t.Fatalf("expected failed status filter, got %q", cfg.OutboxStatus)
	}
	if cfg.OutboxLimit != 7 {
		t.Fatalf("expected outbox limit 7, got %d", cfg.OutboxLimit)
	}
}

func TestParseConfigOutboxRequeueFlags(t *testing.T) {
	fs := flag.NewFlagSet("maintenance", flag.ContinueOnError)
	args := []string{
		"-outbox-requeue",
		"-outbox-requeue-campaign-id", "camp-1",
		"-outbox-requeue-seq", "9",
	}
	cfg, err := ParseConfig(fs, args)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if !cfg.OutboxRequeue {
		t.Fatal("expected outbox requeue mode to be enabled")
	}
	if cfg.OutboxRequeueCampaignID != "camp-1" {
		t.Fatalf("expected outbox requeue campaign id camp-1, got %q", cfg.OutboxRequeueCampaignID)
	}
	if cfg.OutboxRequeueSeq != 9 {
		t.Fatalf("expected outbox requeue seq 9, got %d", cfg.OutboxRequeueSeq)
	}
}

func TestParseConfigOutboxRequeueDeadFlags(t *testing.T) {
	fs := flag.NewFlagSet("maintenance", flag.ContinueOnError)
	args := []string{
		"-outbox-requeue-dead",
		"-outbox-requeue-dead-limit", "25",
	}
	cfg, err := ParseConfig(fs, args)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if !cfg.OutboxRequeueDead {
		t.Fatal("expected outbox requeue dead mode to be enabled")
	}
	if cfg.OutboxRequeueDeadLimit != 25 {
		t.Fatalf("expected outbox requeue dead limit 25, got %d", cfg.OutboxRequeueDeadLimit)
	}
}

func TestRunOutboxReportTextOutput(t *testing.T) {
	inspector := &fakeOutboxInspector{
		summary: storagesqlite.ProjectionApplyOutboxSummary{
			PendingCount:            2,
			ProcessingCount:         1,
			FailedCount:             3,
			DeadCount:               1,
			OldestPendingCampaignID: "camp-oldest",
			OldestPendingSeq:        42,
			OldestPendingAt:         time.Date(2026, 2, 16, 10, 0, 0, 0, time.UTC),
		},
		rows: []storagesqlite.ProjectionApplyOutboxEntry{
			{
				CampaignID:    "camp-oldest",
				Seq:           42,
				EventType:     event.Type("campaign.created"),
				Status:        "failed",
				AttemptCount:  3,
				NextAttemptAt: time.Date(2026, 2, 16, 10, 0, 0, 0, time.UTC),
			},
		},
	}

	var out, errOut bytes.Buffer
	if err := runOutboxReport(t.Context(), inspector, "failed", 10, false, &out, &errOut); err != nil {
		t.Fatalf("run outbox report: %v", err)
	}
	if errOut.Len() != 0 {
		t.Fatalf("unexpected stderr output: %s", errOut.String())
	}
	text := out.String()
	if !strings.Contains(text, "Outbox summary") {
		t.Fatalf("expected summary header, got %q", text)
	}
	if !strings.Contains(text, "pending=2") || !strings.Contains(text, "failed=3") {
		t.Fatalf("expected summary counts, got %q", text)
	}
	if !strings.Contains(text, "camp-oldest/42") {
		t.Fatalf("expected listed row identity, got %q", text)
	}
}

func TestRunOutboxReportJSONOutput(t *testing.T) {
	inspector := &fakeOutboxInspector{
		summary: storagesqlite.ProjectionApplyOutboxSummary{
			PendingCount: 4,
			DeadCount:    2,
		},
		rows: []storagesqlite.ProjectionApplyOutboxEntry{
			{
				CampaignID: "camp-json",
				Seq:        9,
				Status:     "dead",
			},
		},
	}

	var out, errOut bytes.Buffer
	if err := runOutboxReport(t.Context(), inspector, "dead", 5, true, &out, &errOut); err != nil {
		t.Fatalf("run outbox report: %v", err)
	}
	if errOut.Len() != 0 {
		t.Fatalf("unexpected stderr output: %s", errOut.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode JSON output: %v", err)
	}
	if payload["mode"] != "outbox" {
		t.Fatalf("expected mode outbox, got %v", payload["mode"])
	}
}

func TestRunOutboxReportModeNoCampaignIDs(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "test-key")
	eventsPath := filepath.Join(t.TempDir(), "events.db")

	keyring, err := integrity.KeyringFromEnv()
	if err != nil {
		t.Fatalf("build keyring: %v", err)
	}
	eventStore, err := storagesqlite.OpenEvents(
		eventsPath,
		keyring,
		testEventRegistry(t),
		storagesqlite.WithProjectionApplyOutboxEnabled(true),
	)
	if err != nil {
		t.Fatalf("open events store: %v", err)
	}
	if _, err := eventStore.AppendEvent(t.Context(), event.Event{
		CampaignID:  "camp-run-outbox",
		Timestamp:   time.Date(2026, 2, 16, 11, 0, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-run-outbox",
		PayloadJSON: []byte(`{}`),
	}); err != nil {
		_ = eventStore.Close()
		t.Fatalf("append event: %v", err)
	}
	if err := eventStore.Close(); err != nil {
		t.Fatalf("close events store: %v", err)
	}

	cfg := Config{
		OutboxReport: true,
		OutboxLimit:  10,
		EventsDBPath: eventsPath,
	}
	var out, errOut bytes.Buffer
	if err := Run(t.Context(), cfg, &out, &errOut); err != nil {
		t.Fatalf("run maintenance outbox report: %v", err)
	}
	if errOut.Len() != 0 {
		t.Fatalf("unexpected stderr output: %s", errOut.String())
	}
	text := out.String()
	if !strings.Contains(text, "Outbox summary") {
		t.Fatalf("expected outbox summary in output, got %q", text)
	}
	if !strings.Contains(text, "camp-run-outbox") {
		t.Fatalf("expected outbox row in output, got %q", text)
	}
}

func TestRunOutboxReportValidationErrors(t *testing.T) {
	t.Run("campaign id conflict", func(t *testing.T) {
		cfg := Config{
			CampaignID:   "camp-1",
			OutboxReport: true,
			OutboxLimit:  10,
		}
		err := Run(t.Context(), cfg, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "-outbox-report cannot be combined with -campaign-id or -campaign-ids") {
			t.Fatalf("expected campaign conflict error, got %v", err)
		}
	})

	t.Run("invalid outbox limit", func(t *testing.T) {
		cfg := Config{
			OutboxReport: true,
			OutboxLimit:  0,
		}
		err := Run(t.Context(), cfg, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "-outbox-limit must be > 0") {
			t.Fatalf("expected outbox limit error, got %v", err)
		}
	})
}

func TestRunOutboxReportHelperErrors(t *testing.T) {
	t.Run("nil inspector", func(t *testing.T) {
		err := runOutboxReport(t.Context(), nil, "", 10, false, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "outbox inspector is not configured") {
			t.Fatalf("expected nil inspector error, got %v", err)
		}
	})

	t.Run("invalid limit", func(t *testing.T) {
		err := runOutboxReport(t.Context(), &fakeOutboxInspector{}, "", 0, false, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "outbox limit must be > 0") {
			t.Fatalf("expected outbox limit error, got %v", err)
		}
	})

	t.Run("summary read error", func(t *testing.T) {
		err := runOutboxReport(t.Context(), &fakeOutboxInspector{err: fmt.Errorf("boom")}, "", 10, false, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "read outbox summary") {
			t.Fatalf("expected summary read error, got %v", err)
		}
	})
}

func TestRunOutboxReportValidationReplayFlagConflict(t *testing.T) {
	cfg := Config{
		OutboxReport: true,
		OutboxLimit:  10,
		DryRun:       true,
	}
	err := Run(t.Context(), cfg, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "-outbox-report cannot be combined with replay/scan flags") {
		t.Fatalf("expected replay/scan conflict error, got %v", err)
	}
}

func TestRunOutboxRequeueModeRequeuesDeadRow(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "test-key")
	eventsPath := filepath.Join(t.TempDir(), "events.db")

	keyring, err := integrity.KeyringFromEnv()
	if err != nil {
		t.Fatalf("build keyring: %v", err)
	}
	eventStore, err := storagesqlite.OpenEvents(
		eventsPath,
		keyring,
		testEventRegistry(t),
		storagesqlite.WithProjectionApplyOutboxEnabled(true),
	)
	if err != nil {
		t.Fatalf("open events store: %v", err)
	}
	stored, err := eventStore.AppendEvent(t.Context(), event.Event{
		CampaignID:  "camp-run-requeue",
		Timestamp:   time.Date(2026, 2, 16, 12, 0, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-run-requeue",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		_ = eventStore.Close()
		t.Fatalf("append event: %v", err)
	}
	if err := eventStore.Close(); err != nil {
		t.Fatalf("close events store: %v", err)
	}

	dbConn, err := sql.Open("sqlite", eventsPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer dbConn.Close()
	if _, err := dbConn.ExecContext(
		t.Context(),
		`UPDATE projection_apply_outbox
		 SET status = 'dead', attempt_count = 8, next_attempt_at = ?, last_error = 'failed permanently', updated_at = ?
		 WHERE campaign_id = ? AND seq = ?`,
		time.Date(2026, 2, 16, 12, 1, 0, 0, time.UTC).UnixMilli(),
		time.Date(2026, 2, 16, 12, 1, 0, 0, time.UTC).UnixMilli(),
		stored.CampaignID,
		stored.Seq,
	); err != nil {
		t.Fatalf("prepare dead outbox row: %v", err)
	}

	cfg := Config{
		OutboxRequeue:           true,
		OutboxRequeueCampaignID: stored.CampaignID,
		OutboxRequeueSeq:        stored.Seq,
		EventsDBPath:            eventsPath,
	}
	var out, errOut bytes.Buffer
	if err := Run(t.Context(), cfg, &out, &errOut); err != nil {
		t.Fatalf("run maintenance outbox requeue: %v", err)
	}
	if errOut.Len() != 0 {
		t.Fatalf("unexpected stderr output: %s", errOut.String())
	}
	if !strings.Contains(out.String(), "Requeued outbox row") {
		t.Fatalf("expected requeue output, got %q", out.String())
	}
}

func TestRunOutboxRequeueDeadModeRequeuesRows(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "test-key")
	eventsPath := filepath.Join(t.TempDir(), "events.db")

	keyring, err := integrity.KeyringFromEnv()
	if err != nil {
		t.Fatalf("build keyring: %v", err)
	}
	eventStore, err := storagesqlite.OpenEvents(
		eventsPath,
		keyring,
		testEventRegistry(t),
		storagesqlite.WithProjectionApplyOutboxEnabled(true),
	)
	if err != nil {
		t.Fatalf("open events store: %v", err)
	}
	storedA, err := eventStore.AppendEvent(t.Context(), event.Event{
		CampaignID:  "camp-run-requeue-batch-a",
		Timestamp:   time.Date(2026, 2, 16, 13, 0, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-run-requeue-batch-a",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		_ = eventStore.Close()
		t.Fatalf("append event A: %v", err)
	}
	storedB, err := eventStore.AppendEvent(t.Context(), event.Event{
		CampaignID:  "camp-run-requeue-batch-b",
		Timestamp:   time.Date(2026, 2, 16, 13, 0, 1, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-run-requeue-batch-b",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		_ = eventStore.Close()
		t.Fatalf("append event B: %v", err)
	}
	storedC, err := eventStore.AppendEvent(t.Context(), event.Event{
		CampaignID:  "camp-run-requeue-batch-c",
		Timestamp:   time.Date(2026, 2, 16, 13, 0, 2, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-run-requeue-batch-c",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		_ = eventStore.Close()
		t.Fatalf("append event C: %v", err)
	}
	if err := eventStore.Close(); err != nil {
		t.Fatalf("close events store: %v", err)
	}

	dbConn, err := sql.Open("sqlite", eventsPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer dbConn.Close()

	markDead := func(campaignID string, seq uint64, nextAttempt time.Time) {
		t.Helper()
		if _, err := dbConn.ExecContext(
			t.Context(),
			`UPDATE projection_apply_outbox
			 SET status = 'dead', attempt_count = 8, next_attempt_at = ?, last_error = 'failed permanently', updated_at = ?
			 WHERE campaign_id = ? AND seq = ?`,
			nextAttempt.UnixMilli(),
			nextAttempt.UnixMilli(),
			campaignID,
			seq,
		); err != nil {
			t.Fatalf("prepare dead outbox row %s/%d: %v", campaignID, seq, err)
		}
	}
	markDead(storedA.CampaignID, storedA.Seq, time.Date(2026, 2, 16, 13, 1, 0, 0, time.UTC))
	markDead(storedB.CampaignID, storedB.Seq, time.Date(2026, 2, 16, 13, 2, 0, 0, time.UTC))
	markDead(storedC.CampaignID, storedC.Seq, time.Date(2026, 2, 16, 13, 3, 0, 0, time.UTC))

	cfg := Config{
		OutboxRequeueDead:      true,
		OutboxRequeueDeadLimit: 2,
		EventsDBPath:           eventsPath,
	}
	var out, errOut bytes.Buffer
	if err := Run(t.Context(), cfg, &out, &errOut); err != nil {
		t.Fatalf("run maintenance outbox dead requeue: %v", err)
	}
	if errOut.Len() != 0 {
		t.Fatalf("unexpected stderr output: %s", errOut.String())
	}
	if !strings.Contains(out.String(), "Requeued dead outbox rows: 2 (limit=2)") {
		t.Fatalf("expected dead requeue output, got %q", out.String())
	}

	var pendingCount int
	if err := dbConn.QueryRowContext(
		t.Context(),
		`SELECT COUNT(*) FROM projection_apply_outbox WHERE status = 'pending'`,
	).Scan(&pendingCount); err != nil {
		t.Fatalf("count pending rows: %v", err)
	}
	if pendingCount != 2 {
		t.Fatalf("expected two rows pending after requeue, got %d", pendingCount)
	}

	var deadCount int
	if err := dbConn.QueryRowContext(
		t.Context(),
		`SELECT COUNT(*) FROM projection_apply_outbox WHERE status = 'dead'`,
	).Scan(&deadCount); err != nil {
		t.Fatalf("count dead rows: %v", err)
	}
	if deadCount != 1 {
		t.Fatalf("expected one row remaining dead after requeue, got %d", deadCount)
	}
}

func TestRunOutboxRequeueValidationErrors(t *testing.T) {
	t.Run("missing campaign id", func(t *testing.T) {
		cfg := Config{
			OutboxRequeue:    true,
			OutboxRequeueSeq: 1,
		}
		err := Run(t.Context(), cfg, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "-outbox-requeue-campaign-id is required") {
			t.Fatalf("expected missing campaign id error, got %v", err)
		}
	})

	t.Run("missing seq", func(t *testing.T) {
		cfg := Config{
			OutboxRequeue:           true,
			OutboxRequeueCampaignID: "camp-1",
		}
		err := Run(t.Context(), cfg, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "-outbox-requeue-seq must be > 0") {
			t.Fatalf("expected missing seq error, got %v", err)
		}
	})

	t.Run("conflict with outbox report", func(t *testing.T) {
		cfg := Config{
			OutboxRequeue:           true,
			OutboxRequeueCampaignID: "camp-1",
			OutboxRequeueSeq:        1,
			OutboxReport:            true,
		}
		err := Run(t.Context(), cfg, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "-outbox-requeue cannot be combined with -outbox-report") {
			t.Fatalf("expected outbox mode conflict error, got %v", err)
		}
	})

	t.Run("conflict with replay flags", func(t *testing.T) {
		cfg := Config{
			OutboxRequeue:           true,
			OutboxRequeueCampaignID: "camp-1",
			OutboxRequeueSeq:        1,
			DryRun:                  true,
		}
		err := Run(t.Context(), cfg, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "-outbox-requeue cannot be combined with replay/scan flags") {
			t.Fatalf("expected replay conflict error, got %v", err)
		}
	})
}

func TestRunOutboxRequeueDeadValidationErrors(t *testing.T) {
	t.Run("missing limit", func(t *testing.T) {
		cfg := Config{
			OutboxRequeueDead: true,
		}
		err := Run(t.Context(), cfg, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "-outbox-requeue-dead-limit must be > 0") {
			t.Fatalf("expected missing limit error, got %v", err)
		}
	})

	t.Run("conflict with outbox requeue", func(t *testing.T) {
		cfg := Config{
			OutboxRequeueDead:      true,
			OutboxRequeueDeadLimit: 5,
			OutboxRequeue:          true,
		}
		err := Run(t.Context(), cfg, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "-outbox-requeue-dead cannot be combined with -outbox-requeue") {
			t.Fatalf("expected outbox requeue conflict error, got %v", err)
		}
	})

	t.Run("conflict with replay flags", func(t *testing.T) {
		cfg := Config{
			OutboxRequeueDead:      true,
			OutboxRequeueDeadLimit: 5,
			DryRun:                 true,
		}
		err := Run(t.Context(), cfg, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "-outbox-requeue-dead cannot be combined with replay/scan flags") {
			t.Fatalf("expected replay conflict error, got %v", err)
		}
	})
}

func TestRunOutboxRequeueHelper(t *testing.T) {
	t.Run("nil requeuer", func(t *testing.T) {
		err := runOutboxRequeue(t.Context(), nil, "camp-1", 1, time.Now().UTC(), false, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "outbox requeuer is not configured") {
			t.Fatalf("expected nil requeuer error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := runOutboxRequeue(
			t.Context(),
			&fakeOutboxRequeuer{requeued: false},
			"camp-1",
			1,
			time.Now().UTC(),
			false,
			nil,
			nil,
		)
		if err == nil || !strings.Contains(err.Error(), "dead outbox row not found") {
			t.Fatalf("expected dead row not found error, got %v", err)
		}
	})

	t.Run("json output", func(t *testing.T) {
		var out, errOut bytes.Buffer
		err := runOutboxRequeue(
			t.Context(),
			&fakeOutboxRequeuer{requeued: true},
			"camp-1",
			5,
			time.Date(2026, 2, 16, 12, 30, 0, 0, time.UTC),
			true,
			&out,
			&errOut,
		)
		if err != nil {
			t.Fatalf("run outbox requeue helper: %v", err)
		}
		if errOut.Len() != 0 {
			t.Fatalf("unexpected stderr output: %s", errOut.String())
		}
		var payload map[string]any
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("decode JSON output: %v", err)
		}
		if payload["mode"] != "outbox-requeue" {
			t.Fatalf("expected outbox-requeue mode, got %v", payload["mode"])
		}
	})
}

func TestRunOutboxRequeueDeadHelper(t *testing.T) {
	t.Run("nil requeuer", func(t *testing.T) {
		err := runOutboxRequeueDeadRows(t.Context(), nil, 5, time.Now().UTC(), false, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "outbox requeuer is not configured") {
			t.Fatalf("expected nil requeuer error, got %v", err)
		}
	})

	t.Run("invalid limit", func(t *testing.T) {
		err := runOutboxRequeueDeadRows(t.Context(), &fakeOutboxRequeuer{}, 0, time.Now().UTC(), false, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "outbox requeue limit must be > 0") {
			t.Fatalf("expected limit error, got %v", err)
		}
	})

	t.Run("json output", func(t *testing.T) {
		var out, errOut bytes.Buffer
		err := runOutboxRequeueDeadRows(
			t.Context(),
			&fakeOutboxRequeuer{deadRequeued: 4},
			10,
			time.Date(2026, 2, 16, 13, 30, 0, 0, time.UTC),
			true,
			&out,
			&errOut,
		)
		if err != nil {
			t.Fatalf("run outbox dead requeue helper: %v", err)
		}
		if errOut.Len() != 0 {
			t.Fatalf("unexpected stderr output: %s", errOut.String())
		}
		var payload map[string]any
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("decode JSON output: %v", err)
		}
		if payload["mode"] != "outbox-requeue-dead" {
			t.Fatalf("expected outbox-requeue-dead mode, got %v", payload["mode"])
		}
		if payload["requeued"] != float64(4) {
			t.Fatalf("expected requeued=4, got %v", payload["requeued"])
		}
	})
}

func TestOutputJSON(t *testing.T) {
	t.Run("valid result", func(t *testing.T) {
		var out, errOut bytes.Buffer
		result := runResult{
			CampaignID: "c1",
			Mode:       "scan",
		}
		outputJSON(&out, &errOut, result)
		if errOut.Len() != 0 {
			t.Errorf("unexpected error output: %s", errOut.String())
		}
		var decoded runResult
		if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
			t.Fatalf("invalid JSON output: %v", err)
		}
		if decoded.CampaignID != "c1" {
			t.Errorf("campaign_id = %q, want %q", decoded.CampaignID, "c1")
		}
	})

	t.Run("with warnings", func(t *testing.T) {
		var out, errOut bytes.Buffer
		result := runResult{
			CampaignID:    "c2",
			Mode:          "validate",
			Warnings:      []string{"warn1"},
			WarningsTotal: 5,
		}
		outputJSON(&out, &errOut, result)
		if !strings.Contains(out.String(), `"warnings_total":5`) {
			t.Errorf("expected warnings_total in output: %s", out.String())
		}
	})
}

func TestPrintResult(t *testing.T) {
	t.Run("error output", func(t *testing.T) {
		var out, errOut bytes.Buffer
		result := runResult{Error: "something failed"}
		printResult(&out, &errOut, result, "")
		if !strings.Contains(errOut.String(), "Error: something failed") {
			t.Errorf("expected error in errOut: %s", errOut.String())
		}
	})

	t.Run("warnings output", func(t *testing.T) {
		var out, errOut bytes.Buffer
		result := runResult{Warnings: []string{"w1", "w2"}, WarningsTotal: 5}
		printResult(&out, &errOut, result, "")
		if !strings.Contains(errOut.String(), "Warning: w1") {
			t.Errorf("expected warning w1: %s", errOut.String())
		}
		if !strings.Contains(errOut.String(), "3 more warnings suppressed") {
			t.Errorf("expected suppressed warning count: %s", errOut.String())
		}
	})

	t.Run("integrity report", func(t *testing.T) {
		var out, errOut bytes.Buffer
		report := integrityReport{
			LastSeq:             100,
			CharacterMismatches: 2,
			MissingStates:       1,
			GmFearMatch:         true,
			GmFearSource:        5,
			GmFearReplay:        5,
		}
		reportJSON, _ := json.Marshal(report)
		result := runResult{
			CampaignID: "c1",
			Mode:       "integrity",
			Report:     reportJSON,
		}
		printResult(&out, &errOut, result, "")
		if !strings.Contains(out.String(), "Integrity check") {
			t.Errorf("expected integrity output: %s", out.String())
		}
		if !strings.Contains(out.String(), "GM fear match: true") {
			t.Errorf("expected GM fear match: %s", out.String())
		}
		if !strings.Contains(out.String(), "Character state mismatches: 2") {
			t.Errorf("expected character mismatches: %s", out.String())
		}
	})

	t.Run("scan report", func(t *testing.T) {
		var out, errOut bytes.Buffer
		report := snapshotScanReport{
			LastSeq:        50,
			TotalEvents:    100,
			SnapshotEvents: 10,
		}
		reportJSON, _ := json.Marshal(report)
		result := runResult{
			CampaignID: "c1",
			Mode:       "scan",
			Report:     reportJSON,
		}
		printResult(&out, &errOut, result, "")
		if !strings.Contains(out.String(), "Scanned snapshot-related events") {
			t.Errorf("expected scan output: %s", out.String())
		}
	})

	t.Run("validate report", func(t *testing.T) {
		var out, errOut bytes.Buffer
		report := snapshotScanReport{
			LastSeq:        50,
			TotalEvents:    100,
			SnapshotEvents: 10,
			InvalidEvents:  3,
		}
		reportJSON, _ := json.Marshal(report)
		result := runResult{
			CampaignID: "c1",
			Mode:       "validate",
			Report:     reportJSON,
		}
		printResult(&out, &errOut, result, "")
		if !strings.Contains(out.String(), "Validated snapshot-related events") {
			t.Errorf("expected validate output: %s", out.String())
		}
	})

	t.Run("replay report", func(t *testing.T) {
		var out, errOut bytes.Buffer
		report := snapshotScanReport{LastSeq: 50}
		reportJSON, _ := json.Marshal(report)
		result := runResult{
			CampaignID: "c1",
			Mode:       "replay",
			Report:     reportJSON,
		}
		printResult(&out, &errOut, result, "")
		if !strings.Contains(out.String(), "Replayed snapshot-related events") {
			t.Errorf("expected replay output: %s", out.String())
		}
	})

	t.Run("prefix applied", func(t *testing.T) {
		var out, errOut bytes.Buffer
		result := runResult{Error: "oops"}
		printResult(&out, &errOut, result, "[c1] ")
		if !strings.Contains(errOut.String(), "[c1] Error: oops") {
			t.Errorf("expected prefix in output: %s", errOut.String())
		}
	})

	t.Run("empty report returns early", func(t *testing.T) {
		var out, errOut bytes.Buffer
		result := runResult{CampaignID: "c1", Mode: "scan"}
		printResult(&out, &errOut, result, "")
		if out.Len() != 0 {
			t.Errorf("expected no output for empty report: %s", out.String())
		}
	})

	t.Run("invalid integrity JSON", func(t *testing.T) {
		var out, errOut bytes.Buffer
		result := runResult{
			CampaignID: "c1",
			Mode:       "integrity",
			Report:     json.RawMessage(`{invalid`),
		}
		printResult(&out, &errOut, result, "")
		if !strings.Contains(errOut.String(), "decode report") {
			t.Errorf("expected decode error: %s", errOut.String())
		}
	})

	t.Run("invalid scan JSON", func(t *testing.T) {
		var out, errOut bytes.Buffer
		result := runResult{
			CampaignID: "c1",
			Mode:       "scan",
			Report:     json.RawMessage(`{invalid`),
		}
		printResult(&out, &errOut, result, "")
		if !strings.Contains(errOut.String(), "decode report") {
			t.Errorf("expected decode error: %s", errOut.String())
		}
	})
}

func TestRunValidationErrors(t *testing.T) {
	t.Run("integrity with dry-run", func(t *testing.T) {
		cfg := Config{
			CampaignID: "c1",
			Integrity:  true,
			DryRun:     true,
		}
		err := Run(t.Context(), cfg, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "-integrity cannot be combined") {
			t.Fatalf("expected validation error, got %v", err)
		}
	})

	t.Run("integrity with validate", func(t *testing.T) {
		cfg := Config{
			CampaignID: "c1",
			Integrity:  true,
			Validate:   true,
		}
		err := Run(t.Context(), cfg, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "-integrity cannot be combined") {
			t.Fatalf("expected validation error, got %v", err)
		}
	})

	t.Run("integrity with after-seq", func(t *testing.T) {
		cfg := Config{
			CampaignID: "c1",
			Integrity:  true,
			AfterSeq:   10,
		}
		err := Run(t.Context(), cfg, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "-integrity does not support -after-seq") {
			t.Fatalf("expected validation error, got %v", err)
		}
	})

	t.Run("no campaign IDs", func(t *testing.T) {
		cfg := Config{}
		err := Run(t.Context(), cfg, nil, nil)
		if err == nil {
			t.Fatal("expected error for no campaign IDs")
		}
	})

	t.Run("negative warnings cap", func(t *testing.T) {
		cfg := Config{
			CampaignID:  "c1",
			WarningsCap: -1,
		}
		err := Run(t.Context(), cfg, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "warnings-cap") {
			t.Fatalf("expected warnings-cap error, got %v", err)
		}
	})
}

// --- scanSnapshotEvents tests ---

func TestScanSnapshotEventsEmpty(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{}}
	report, warnings, err := scanSnapshotEvents(t.Context(), store, "c1", 0, 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.TotalEvents != 0 {
		t.Fatalf("expected 0 events, got %d", report.TotalEvents)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
}

func TestScanSnapshotEventsCountsSnapshotEvents(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{
		"c1": {
			{Seq: 1, Type: event.Type("campaign.created")},
			{Seq: 2, Type: event.Type("sys.daggerheart.character_state_patched"), CampaignID: "c1", EntityType: "action", EntityID: "entity-1", SystemID: daggerheart.SystemID, SystemVersion: daggerheart.SystemVersion},
			{Seq: 3, Type: event.Type("character.created")},
			{Seq: 4, Type: event.Type("sys.daggerheart.gm_fear_changed"), CampaignID: "c1", EntityType: "action", EntityID: "entity-1", SystemID: daggerheart.SystemID, SystemVersion: daggerheart.SystemVersion},
		},
	}}
	report, _, err := scanSnapshotEvents(t.Context(), store, "c1", 0, 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.TotalEvents != 4 {
		t.Fatalf("expected 4 total events, got %d", report.TotalEvents)
	}
	if report.SnapshotEvents != 2 {
		t.Fatalf("expected 2 snapshot events, got %d", report.SnapshotEvents)
	}
	if report.LastSeq != 4 {
		t.Fatalf("expected last seq 4, got %d", report.LastSeq)
	}
}

func TestScanSnapshotEventsWithValidation(t *testing.T) {
	ts := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	store := &fakeEventStore{events: map[string][]event.Event{
		"c1": {
			{Seq: 1, Type: event.Type("sys.daggerheart.character_state_patched"), CampaignID: "c1", Timestamp: ts, EntityType: "action", EntityID: "entity-1", SystemID: daggerheart.SystemID, SystemVersion: daggerheart.SystemVersion, PayloadJSON: []byte(`{"character_id":"ch1","hp_before":3,"hp_after":2}`)},
			{Seq: 2, Type: event.Type("sys.daggerheart.character_state_patched"), CampaignID: "c1", Timestamp: ts, EntityType: "action", EntityID: "entity-1", SystemID: daggerheart.SystemID, SystemVersion: daggerheart.SystemVersion, PayloadJSON: []byte(`{invalid`)},
		},
	}}
	report, warnings, err := scanSnapshotEvents(t.Context(), store, "c1", 0, 0, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.InvalidEvents != 1 {
		t.Fatalf("expected 1 invalid event, got %d", report.InvalidEvents)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
}

func TestScanSnapshotEventsUntilSeq(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{
		"c1": {
			{Seq: 1, Type: event.Type("campaign.created")},
			{Seq: 2, Type: event.Type("character.created")},
			{Seq: 3, Type: event.Type("character.created")},
		},
	}}
	report, _, err := scanSnapshotEvents(t.Context(), store, "c1", 0, 2, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.TotalEvents != 2 {
		t.Fatalf("expected 2 events, got %d", report.TotalEvents)
	}
}

func TestScanSnapshotEventsAfterSeq(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{
		"c1": {
			{Seq: 1, Type: event.Type("campaign.created")},
			{Seq: 2, Type: event.Type("character.created")},
			{Seq: 3, Type: event.Type("character.created")},
		},
	}}
	report, _, err := scanSnapshotEvents(t.Context(), store, "c1", 2, 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.TotalEvents != 1 {
		t.Fatalf("expected 1 event after seq 2, got %d", report.TotalEvents)
	}
}

func TestScanSnapshotEventsNilStore(t *testing.T) {
	_, _, err := scanSnapshotEvents(t.Context(), nil, "c1", 0, 0, false)
	if err == nil || !strings.Contains(err.Error(), "event store is not configured") {
		t.Fatalf("expected nil store error, got %v", err)
	}
}

func TestScanSnapshotEventsEmptyCampaign(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{}}
	_, _, err := scanSnapshotEvents(t.Context(), store, "", 0, 0, false)
	if err == nil || !strings.Contains(err.Error(), "campaign id is required") {
		t.Fatalf("expected campaign id error, got %v", err)
	}
}

func TestScanSnapshotEventsListError(t *testing.T) {
	store := &fakeEventStore{listErr: fmt.Errorf("db gone")}
	_, _, err := scanSnapshotEvents(t.Context(), store, "c1", 0, 0, false)
	if err == nil || !strings.Contains(err.Error(), "db gone") {
		t.Fatalf("expected list error, got %v", err)
	}
}

// makeDaggerheartEvent builds v2 system events for snapshot validation tests.
func makeDaggerheartEvent(campaignID string, eventType string, payload []byte) event.Event {
	return event.Event{
		CampaignID:    campaignID,
		Type:          event.Type(eventType),
		Timestamp:     time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		EntityType:    "action",
		EntityID:      "entity-1",
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payload,
	}
}

// --- runCampaign tests ---

func TestRunCampaignDryRunScan(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{
		"c1": {
			{Seq: 1, Type: event.Type("campaign.created")},
			{Seq: 2, Type: event.Type("sys.daggerheart.gm_fear_changed"), CampaignID: "c1", EntityType: "action", EntityID: "entity-1", SystemID: daggerheart.SystemID, SystemVersion: daggerheart.SystemVersion},
		},
	}}
	result := runCampaign(t.Context(), store, nil, "c1", runOptions{DryRun: true, WarningsCap: 25}, io.Discard)
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d (error: %s)", result.ExitCode, result.Error)
	}
	if result.Mode != "scan" {
		t.Fatalf("expected mode scan, got %s", result.Mode)
	}
	var report snapshotScanReport
	if err := json.Unmarshal(result.Report, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.TotalEvents != 2 {
		t.Fatalf("expected 2 total events, got %d", report.TotalEvents)
	}
	if report.SnapshotEvents != 1 {
		t.Fatalf("expected 1 snapshot event, got %d", report.SnapshotEvents)
	}
}

func TestRunCampaignValidateMode(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{
		"c1": {
			{Seq: 1, Type: event.Type("sys.daggerheart.character_state_patched"), CampaignID: "c1", EntityType: "action", EntityID: "entity-1", SystemID: daggerheart.SystemID, SystemVersion: daggerheart.SystemVersion, PayloadJSON: []byte(`{invalid`)},
		},
	}}
	result := runCampaign(t.Context(), store, nil, "c1", runOptions{DryRun: true, Validate: true, WarningsCap: 25}, io.Discard)
	if result.Mode != "validate" {
		t.Fatalf("expected mode validate, got %s", result.Mode)
	}
	if result.ExitCode != 1 {
		t.Fatalf("expected exit code 1 for invalid events, got %d", result.ExitCode)
	}
}

func TestRunCampaignReplayNilProjStore(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{}}
	result := runCampaign(t.Context(), store, nil, "c1", runOptions{}, io.Discard)
	if result.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Error, "projection store is not configured") {
		t.Fatalf("expected projection store error, got %s", result.Error)
	}
}

func TestRunCampaignScanError(t *testing.T) {
	store := &fakeEventStore{listErr: fmt.Errorf("db error")}
	result := runCampaign(t.Context(), store, nil, "c1", runOptions{DryRun: true, WarningsCap: 25}, io.Discard)
	if result.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Error, "scan snapshot-related events") {
		t.Fatalf("expected scan error, got %s", result.Error)
	}
}

func TestRunCampaignIntegrityNilStores(t *testing.T) {
	result := runCampaign(t.Context(), nil, nil, "c1", runOptions{Integrity: true, WarningsCap: 25}, io.Discard)
	if result.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Error, "event store is not configured") {
		t.Fatalf("expected event store error, got %s", result.Error)
	}
}

func TestRunCampaignIntegrityNilProjStore(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{}}
	result := runCampaign(t.Context(), store, nil, "c1", runOptions{Integrity: true, WarningsCap: 25}, io.Discard)
	if result.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Error, "projection store is not configured") {
		t.Fatalf("expected projection store error, got %s", result.Error)
	}
}

// checkSnapshotIntegrity empty-campaign requires a non-nil ProjectionStore,
// which is a large composite interface. We test the nil-store branches above
// and leave deeper integrity testing to integration tests.

func TestRunCampaignScanWarningsCapped(t *testing.T) {
	// Build events that each produce a validation warning when validated.
	var events []event.Event
	for i := 1; i <= 5; i++ {
		events = append(events, makeDaggerheartEvent("c1", "sys.daggerheart.character_state_patched", []byte(`{invalid`)))
		events[len(events)-1].Seq = uint64(i)
	}
	store := &fakeEventStore{events: map[string][]event.Event{"c1": events}}
	result := runCampaign(t.Context(), store, nil, "c1", runOptions{DryRun: true, Validate: true, WarningsCap: 3}, io.Discard)
	if len(result.Warnings) != 3 {
		t.Fatalf("expected 3 capped warnings, got %d", len(result.Warnings))
	}
	if result.WarningsTotal != 5 {
		t.Fatalf("expected 5 total warnings, got %d", result.WarningsTotal)
	}
}

func TestRunCampaignValidateNoInvalid(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{
		"c1": {
			{Seq: 1, Type: event.Type("sys.daggerheart.gm_fear_changed"), CampaignID: "c1", Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC), EntityType: "action", EntityID: "entity-1", SystemID: daggerheart.SystemID, SystemVersion: daggerheart.SystemVersion, PayloadJSON: []byte(`{"after":3}`)},
		},
	}}
	result := runCampaign(t.Context(), store, nil, "c1", runOptions{DryRun: true, Validate: true, WarningsCap: 25}, io.Discard)
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d (error: %s)", result.ExitCode, result.Error)
	}
}

func TestRunCampaignDryRunWithJSONOutput(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{
		"c1": {{Seq: 1, Type: event.Type("campaign.created")}},
	}}
	result := runCampaign(t.Context(), store, nil, "c1", runOptions{DryRun: true, WarningsCap: 25, JSONOutput: true}, io.Discard)
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d (error: %s)", result.ExitCode, result.Error)
	}
	// Verify JSON output works
	var out bytes.Buffer
	outputJSON(&out, io.Discard, result)
	if !strings.Contains(out.String(), `"campaign_id":"c1"`) {
		t.Fatalf("expected campaign_id in JSON output: %s", out.String())
	}
}

// --- runWithDeps tests ---

func TestRunWithDeps_MultiCampaign(t *testing.T) {
	evtStore := &fakeClosableEventStore{
		fakeEventStore: fakeEventStore{events: map[string][]event.Event{
			"c1": {{Seq: 1, Type: event.Type("campaign.created")}},
			"c2": {{Seq: 1, Type: event.Type("campaign.created")}},
		}},
	}
	projStore := &fakeClosableProjectionStore{}

	cfg := Config{
		CampaignIDs: "c1, c2",
		DryRun:      true,
		WarningsCap: 25,
	}
	var out, errOut bytes.Buffer
	err := runWithDeps(t.Context(), cfg, evtStore, projStore, &out, &errOut)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "c1") {
		t.Errorf("expected c1 in output: %s", output)
	}
	if !strings.Contains(output, "c2") {
		t.Errorf("expected c2 in output: %s", output)
	}
}

func TestRunWithDeps_OneCampaignFails(t *testing.T) {
	evtStore := &fakeClosableEventStore{
		fakeEventStore: fakeEventStore{events: map[string][]event.Event{
			"c1": {{Seq: 1, Type: event.Type("sys.daggerheart.character_state_patched"), CampaignID: "c1", EntityType: "action", EntityID: "entity-1", SystemID: daggerheart.SystemID, SystemVersion: daggerheart.SystemVersion, PayloadJSON: []byte(`{invalid`)}},
			"c2": {{Seq: 1, Type: event.Type("campaign.created")}},
		}},
	}
	projStore := &fakeClosableProjectionStore{}

	cfg := Config{
		CampaignIDs: "c1, c2",
		DryRun:      true,
		Validate:    true,
		WarningsCap: 25,
	}
	var out, errOut bytes.Buffer
	err := runWithDeps(t.Context(), cfg, evtStore, projStore, &out, &errOut)
	// One campaign has invalid events, so the overall run should fail.
	if err == nil {
		t.Fatal("expected error when one campaign fails")
	}
	// But c2 should still have been processed.
	if !strings.Contains(out.String(), "c2") {
		t.Errorf("expected c2 to be processed despite c1 failure: %s", out.String())
	}
}

func TestRunWithDeps_JSONOutputMode(t *testing.T) {
	evtStore := &fakeClosableEventStore{
		fakeEventStore: fakeEventStore{events: map[string][]event.Event{
			"c1": {{Seq: 1, Type: event.Type("campaign.created")}},
		}},
	}
	projStore := &fakeClosableProjectionStore{}

	cfg := Config{
		CampaignID:  "c1",
		DryRun:      true,
		JSONOutput:  true,
		WarningsCap: 25,
	}
	var out, errOut bytes.Buffer
	err := runWithDeps(t.Context(), cfg, evtStore, projStore, &out, &errOut)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), `"campaign_id":"c1"`) {
		t.Errorf("expected JSON output with campaign_id: %s", out.String())
	}
}

func TestRunWithDeps_StoreCloseError(t *testing.T) {
	evtStore := &fakeClosableEventStore{
		fakeEventStore: fakeEventStore{events: map[string][]event.Event{
			"c1": {{Seq: 1, Type: event.Type("campaign.created")}},
		}},
		closeErr: fmt.Errorf("event close failed"),
	}
	projStore := &fakeClosableProjectionStore{
		closeErr: fmt.Errorf("proj close failed"),
	}

	cfg := Config{
		CampaignID:  "c1",
		DryRun:      true,
		WarningsCap: 25,
	}
	var out, errOut bytes.Buffer
	err := runWithDeps(t.Context(), cfg, evtStore, projStore, &out, &errOut)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Close errors should be written to errOut.
	errOutput := errOut.String()
	if !strings.Contains(errOutput, "event close failed") {
		t.Errorf("expected event close error in errOut: %s", errOutput)
	}
	if !strings.Contains(errOutput, "proj close failed") {
		t.Errorf("expected proj close error in errOut: %s", errOutput)
	}
	// Both stores should be closed.
	if !evtStore.closed {
		t.Error("expected event store to be closed")
	}
	if !projStore.closed {
		t.Error("expected projection store to be closed")
	}
}

// --- runCampaign replay path tests ---

func TestRunCampaignReplayEmptyEvents(t *testing.T) {
	evtStore := &fakeEventStore{events: map[string][]event.Event{}}
	projStore := &fakeProjectionStore{}
	result := runCampaign(t.Context(), evtStore, projStore, "c1", runOptions{WarningsCap: 25}, io.Discard)
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d (error: %s)", result.ExitCode, result.Error)
	}
	if result.Mode != "replay" {
		t.Fatalf("expected mode replay, got %s", result.Mode)
	}
}

func TestRunCampaignReplayAfterSeqEmptyEvents(t *testing.T) {
	evtStore := &fakeEventStore{events: map[string][]event.Event{}}
	projStore := &fakeProjectionStore{}
	result := runCampaign(t.Context(), evtStore, projStore, "c1", runOptions{AfterSeq: 5, WarningsCap: 25}, io.Discard)
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d (error: %s)", result.ExitCode, result.Error)
	}
}

func TestRunCampaignReplayEventStoreError(t *testing.T) {
	evtStore := &fakeEventStore{listErr: fmt.Errorf("disk error")}
	projStore := &fakeProjectionStore{}
	result := runCampaign(t.Context(), evtStore, projStore, "c1", runOptions{WarningsCap: 25}, io.Discard)
	if result.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Error, "replay snapshot") {
		t.Fatalf("expected replay error, got: %s", result.Error)
	}
}

func TestRunCampaignReplayAfterSeqEventStoreError(t *testing.T) {
	evtStore := &fakeEventStore{listErr: fmt.Errorf("disk error")}
	projStore := &fakeProjectionStore{}
	result := runCampaign(t.Context(), evtStore, projStore, "c1", runOptions{AfterSeq: 5, WarningsCap: 25}, io.Discard)
	if result.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Error, "replay snapshot") {
		t.Fatalf("expected replay error, got: %s", result.Error)
	}
}

// --- checkIntegrityWithStores tests ---

func TestCheckIntegrityWithStores_GMFearMismatch(t *testing.T) {
	evtStore := &fakeEventStore{events: map[string][]event.Event{}}
	source := &fakeProjectionStore{
		get: func(_ context.Context, _ string) (storage.CampaignRecord, error) {
			return storage.CampaignRecord{ID: "c1"}, nil
		},
		getDaggerheartSnapshot: func(_ context.Context, _ string) (storage.DaggerheartSnapshot, error) {
			return storage.DaggerheartSnapshot{GMFear: 5}, nil
		},
		listCharacters: func(_ context.Context, _ string, _ int, _ string) (storage.CharacterPage, error) {
			return storage.CharacterPage{}, nil
		},
	}
	scratch := &fakeProjectionStore{
		put: func(_ context.Context, _ storage.CampaignRecord) error { return nil },
		getDaggerheartSnapshot: func(_ context.Context, _ string) (storage.DaggerheartSnapshot, error) {
			return storage.DaggerheartSnapshot{GMFear: 3}, nil
		},
	}
	report, _, err := checkIntegrityWithStores(t.Context(), evtStore, source, scratch, "c1", 0, io.Discard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.GmFearMatch {
		t.Fatal("expected GM fear mismatch")
	}
	if report.GmFearSource != 5 || report.GmFearReplay != 3 {
		t.Fatalf("expected source=5 replay=3, got source=%d replay=%d", report.GmFearSource, report.GmFearReplay)
	}
}

func TestCheckIntegrityWithStores_GMFearMatch(t *testing.T) {
	evtStore := &fakeEventStore{events: map[string][]event.Event{}}
	source := &fakeProjectionStore{
		get: func(_ context.Context, _ string) (storage.CampaignRecord, error) {
			return storage.CampaignRecord{ID: "c1"}, nil
		},
		getDaggerheartSnapshot: func(_ context.Context, _ string) (storage.DaggerheartSnapshot, error) {
			return storage.DaggerheartSnapshot{GMFear: 5}, nil
		},
		listCharacters: func(_ context.Context, _ string, _ int, _ string) (storage.CharacterPage, error) {
			return storage.CharacterPage{}, nil
		},
	}
	scratch := &fakeProjectionStore{
		put: func(_ context.Context, _ storage.CampaignRecord) error { return nil },
		getDaggerheartSnapshot: func(_ context.Context, _ string) (storage.DaggerheartSnapshot, error) {
			return storage.DaggerheartSnapshot{GMFear: 5}, nil
		},
	}
	report, _, err := checkIntegrityWithStores(t.Context(), evtStore, source, scratch, "c1", 0, io.Discard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !report.GmFearMatch {
		t.Fatal("expected GM fear match")
	}
}

func TestCheckIntegrityWithStores_CharacterMismatch(t *testing.T) {
	evtStore := &fakeEventStore{events: map[string][]event.Event{}}
	source := &fakeProjectionStore{
		get: func(_ context.Context, _ string) (storage.CampaignRecord, error) {
			return storage.CampaignRecord{ID: "c1"}, nil
		},
		getDaggerheartSnapshot: func(_ context.Context, _ string) (storage.DaggerheartSnapshot, error) {
			return storage.DaggerheartSnapshot{GMFear: 0}, nil
		},
		listCharacters: func(_ context.Context, _ string, _ int, _ string) (storage.CharacterPage, error) {
			return storage.CharacterPage{
				Characters: []storage.CharacterRecord{{ID: "ch1"}},
			}, nil
		},
		getDaggerheartCharState: func(_ context.Context, _, charID string) (storage.DaggerheartCharacterState, error) {
			return storage.DaggerheartCharacterState{Hp: 10, Hope: 5, Stress: 2}, nil
		},
	}
	scratch := &fakeProjectionStore{
		put: func(_ context.Context, _ storage.CampaignRecord) error { return nil },
		getDaggerheartSnapshot: func(_ context.Context, _ string) (storage.DaggerheartSnapshot, error) {
			return storage.DaggerheartSnapshot{GMFear: 0}, nil
		},
		getDaggerheartCharState: func(_ context.Context, _, charID string) (storage.DaggerheartCharacterState, error) {
			return storage.DaggerheartCharacterState{Hp: 8, Hope: 3, Stress: 2}, nil
		},
	}
	report, warnings, err := checkIntegrityWithStores(t.Context(), evtStore, source, scratch, "c1", 0, io.Discard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.CharacterMismatches != 1 {
		t.Fatalf("expected 1 character mismatch, got %d", report.CharacterMismatches)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
}

func TestCheckIntegrityWithStores_MissingSourceState(t *testing.T) {
	evtStore := &fakeEventStore{events: map[string][]event.Event{}}
	source := &fakeProjectionStore{
		get: func(_ context.Context, _ string) (storage.CampaignRecord, error) {
			return storage.CampaignRecord{ID: "c1"}, nil
		},
		getDaggerheartSnapshot: func(_ context.Context, _ string) (storage.DaggerheartSnapshot, error) {
			return storage.DaggerheartSnapshot{}, nil
		},
		listCharacters: func(_ context.Context, _ string, _ int, _ string) (storage.CharacterPage, error) {
			return storage.CharacterPage{
				Characters: []storage.CharacterRecord{{ID: "ch1"}},
			}, nil
		},
		getDaggerheartCharState: func(_ context.Context, _, _ string) (storage.DaggerheartCharacterState, error) {
			return storage.DaggerheartCharacterState{}, storage.ErrNotFound
		},
	}
	scratch := &fakeProjectionStore{
		put: func(_ context.Context, _ storage.CampaignRecord) error { return nil },
		getDaggerheartSnapshot: func(_ context.Context, _ string) (storage.DaggerheartSnapshot, error) {
			return storage.DaggerheartSnapshot{}, nil
		},
	}
	report, warnings, err := checkIntegrityWithStores(t.Context(), evtStore, source, scratch, "c1", 0, io.Discard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.MissingStates != 1 {
		t.Fatalf("expected 1 missing state, got %d", report.MissingStates)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "missing source state") {
		t.Fatalf("expected missing source state warning, got %v", warnings)
	}
}

func TestCheckIntegrityWithStores_CampaignLoadError(t *testing.T) {
	evtStore := &fakeEventStore{events: map[string][]event.Event{}}
	source := &fakeProjectionStore{
		get: func(_ context.Context, _ string) (storage.CampaignRecord, error) {
			return storage.CampaignRecord{}, fmt.Errorf("load failed")
		},
	}
	scratch := &fakeProjectionStore{}
	_, _, err := checkIntegrityWithStores(t.Context(), evtStore, source, scratch, "c1", 0, io.Discard)
	if err == nil {
		t.Fatal("expected error for campaign load failure")
	}
	if !strings.Contains(err.Error(), "load campaign") {
		t.Fatalf("expected load campaign error, got: %v", err)
	}
}
