package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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
	assertTableExists(t, sqlDB, "web_sessions")
}

func TestSessionPersistenceRoundTrip(t *testing.T) {
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
	expiresAt := time.Now().UTC().Add(time.Hour)
	if err := store.SaveSession(ctx, "sess-1", "token-1", "Alice", expiresAt); err != nil {
		t.Fatalf("save session: %v", err)
	}

	accessToken, displayName, persistedExpiresAt, found, err := store.LoadSession(ctx, "sess-1")
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if !found {
		t.Fatal("expected session row")
	}
	accessTokenHash := hashedAccessToken("token-1")
	if accessToken != accessTokenHash {
		t.Fatalf("access token = %q, want %q", accessToken, accessTokenHash)
	}
	if displayName != "Alice" {
		t.Fatalf("display name = %q, want %q", displayName, "Alice")
	}
	expiresAt = expiresAt.UTC().Truncate(time.Millisecond)
	if !persistedExpiresAt.Equal(expiresAt) {
		t.Fatalf("expiresAt = %s, want %s", persistedExpiresAt, expiresAt)
	}

	if err := store.DeleteSession(ctx, "sess-1"); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	_, _, _, found, err = store.LoadSession(ctx, "sess-1")
	if err != nil {
		t.Fatalf("load session after delete: %v", err)
	}
	if found {
		t.Fatal("expected missing session after delete")
	}
}

func TestSessionPersistencePrunesExpiredSessionsOnSave(t *testing.T) {
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
	if err := store.SaveSession(ctx, "expired-session", "token-1", "Alice", time.Now().Add(-time.Hour)); err != nil {
		t.Fatalf("save expired session: %v", err)
	}
	if err := store.SaveSession(ctx, "fresh-session", "token-2", "Bob", time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("save fresh session: %v", err)
	}

	_, _, _, found, err := store.LoadSession(ctx, "expired-session")
	if err != nil {
		t.Fatalf("load expired session: %v", err)
	}
	if found {
		t.Fatal("expected expired session to be pruned")
	}
}

func TestSessionPersistenceKeepsCreatedAtOnUpdate(t *testing.T) {
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

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	ctx := context.Background()
	expiresAt := time.Now().UTC().Add(time.Hour)
	if err := store.SaveSession(ctx, "sess-1", "token-1", "Alice", expiresAt); err != nil {
		t.Fatalf("save session: %v", err)
	}

	var createdAt int64
	row := db.QueryRowContext(ctx, `SELECT created_at FROM web_sessions WHERE session_id = ?`, "sess-1")
	if err := row.Scan(&createdAt); err != nil {
		t.Fatalf("read created_at: %v", err)
	}
	createdAtFirst := createdAt

	time.Sleep(2 * time.Millisecond)
	if err := store.SaveSession(ctx, "sess-1", "token-2", "Alice", expiresAt.Add(time.Hour)); err != nil {
		t.Fatalf("update session: %v", err)
	}
	row = db.QueryRowContext(ctx, `SELECT created_at FROM web_sessions WHERE session_id = ?`, "sess-1")
	if err := row.Scan(&createdAt); err != nil {
		t.Fatalf("read created_at after update: %v", err)
	}
	createdAtSecond := createdAt

	// Re-reading through the same query confirms created_at was not updated when
	// upserting an existing session.
	if createdAtFirst != createdAtSecond {
		t.Fatalf("created_at changed on update: %d -> %d", createdAtFirst, createdAtSecond)
	}
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

func hashedAccessToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
