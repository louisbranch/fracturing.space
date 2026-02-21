package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	webstorage "github.com/louisbranch/fracturing.space/internal/services/web/storage"
	_ "modernc.org/sqlite"
)

func TestOpenRequiresPath(t *testing.T) {
	_, err := Open("")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestOpenRunsMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "web-cache.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close: %v", err)
		}
	})

	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() {
		_ = sqlDB.Close()
	}()

	assertTableExists(t, sqlDB, "cache_entries")
	assertTableExists(t, sqlDB, "campaign_event_cursors")
}

func TestStoreCampaignCursorAndStaleMarking(t *testing.T) {
	path := filepath.Join(t.TempDir(), "web-cache.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close: %v", err)
		}
	})

	ctx := context.Background()
	checkedAt := time.Unix(1700000000, 0).UTC()
	if err := store.PutCacheEntry(ctx, webstorage.CacheEntry{
		CacheKey:     "campaign_detail:id:camp-1",
		Scope:        "campaign_summary",
		CampaignID:   "camp-1",
		PayloadBytes: []byte(`{"campaign":"one"}`),
		SourceSeq:    1,
		CheckedAt:    checkedAt,
		RefreshedAt:  checkedAt,
	}); err != nil {
		t.Fatalf("put cache entry: %v", err)
	}

	trackedIDs, err := store.ListTrackedCampaignIDs(ctx)
	if err != nil {
		t.Fatalf("list tracked campaign ids: %v", err)
	}
	if len(trackedIDs) != 1 || trackedIDs[0] != "camp-1" {
		t.Fatalf("tracked campaign ids = %v, want [camp-1]", trackedIDs)
	}

	if _, ok, err := store.GetCampaignEventCursor(ctx, "camp-1"); err != nil {
		t.Fatalf("get cursor (pre): %v", err)
	} else if ok {
		t.Fatalf("expected no existing cursor")
	}

	cursorCheckedAt := checkedAt.Add(time.Minute)
	if err := store.PutCampaignEventCursor(ctx, webstorage.CampaignEventCursor{
		CampaignID: "camp-1",
		LatestSeq:  7,
		CheckedAt:  cursorCheckedAt,
	}); err != nil {
		t.Fatalf("put cursor: %v", err)
	}

	cursor, ok, err := store.GetCampaignEventCursor(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get cursor: %v", err)
	}
	if !ok {
		t.Fatalf("expected cursor")
	}
	if cursor.CampaignID != "camp-1" {
		t.Fatalf("cursor campaign id = %q, want %q", cursor.CampaignID, "camp-1")
	}
	if cursor.LatestSeq != 7 {
		t.Fatalf("cursor seq = %d, want %d", cursor.LatestSeq, 7)
	}

	staleCheckedAt := checkedAt.Add(2 * time.Minute)
	if err := store.MarkCampaignScopeStale(ctx, "camp-1", "campaign_summary", 8, staleCheckedAt); err != nil {
		t.Fatalf("mark stale: %v", err)
	}

	entry, found, err := store.GetCacheEntry(ctx, "campaign_detail:id:camp-1")
	if err != nil {
		t.Fatalf("get cache entry: %v", err)
	}
	if !found {
		t.Fatalf("expected cache entry")
	}
	if !entry.Stale {
		t.Fatalf("expected stale entry")
	}
	if entry.SourceSeq != 8 {
		t.Fatalf("source seq = %d, want %d", entry.SourceSeq, 8)
	}
	if entry.CheckedAt.Before(staleCheckedAt) {
		t.Fatalf("checked at = %s, want >= %s", entry.CheckedAt, staleCheckedAt)
	}

	if err := store.DeleteCacheEntry(ctx, "campaign_detail:id:camp-1"); err != nil {
		t.Fatalf("delete cache entry: %v", err)
	}
	if _, found, err := store.GetCacheEntry(ctx, "campaign_detail:id:camp-1"); err != nil {
		t.Fatalf("get cache entry after delete: %v", err)
	} else if found {
		t.Fatalf("expected deleted cache entry")
	}

	trackedIDs, err = store.ListTrackedCampaignIDs(ctx)
	if err != nil {
		t.Fatalf("list tracked campaign ids after delete: %v", err)
	}
	if len(trackedIDs) != 1 || trackedIDs[0] != "camp-1" {
		t.Fatalf("tracked campaign ids after delete = %v, want [camp-1]", trackedIDs)
	}
}

func assertTableExists(t *testing.T, sqlDB *sql.DB, tableName string) {
	t.Helper()

	row := sqlDB.QueryRowContext(context.Background(), `
SELECT COUNT(*)
FROM sqlite_master
WHERE type = 'table' AND name = ?;
`, tableName)
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan sqlite_master for %q: %v", tableName, err)
	}
	if count != 1 {
		t.Fatalf("table %q count = %d, want 1", tableName, count)
	}
}
