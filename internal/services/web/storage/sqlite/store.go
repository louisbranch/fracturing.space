package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	sqlitemigrate "github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	webstorage "github.com/louisbranch/fracturing.space/internal/services/web/storage"
	"github.com/louisbranch/fracturing.space/internal/services/web/storage/sqlite/migrations"
	_ "modernc.org/sqlite"
)

// Store provides SQLite-backed persistence for web cache data.
type Store struct {
	sqlDB *sql.DB
}

// Open opens and migrates a web cache SQLite store.
func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("storage path is required")
	}

	cleanPath := filepath.Clean(path)
	dsn := cleanPath + "?_journal_mode=WAL&_foreign_keys=ON&_busy_timeout=5000&_synchronous=NORMAL"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping sqlite db: %w", err)
	}

	store := &Store{sqlDB: sqlDB}
	if err := store.runMigrations(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	return store, nil
}

// Close releases the underlying SQLite connection.
func (s *Store) Close() error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}

// GetCacheEntry loads a cache payload and metadata by key.
func (s *Store) GetCacheEntry(ctx context.Context, cacheKey string) (webstorage.CacheEntry, bool, error) {
	if s == nil || s.sqlDB == nil {
		return webstorage.CacheEntry{}, false, fmt.Errorf("storage is not configured")
	}
	cacheKey = strings.TrimSpace(cacheKey)
	if cacheKey == "" {
		return webstorage.CacheEntry{}, false, fmt.Errorf("cache key is required")
	}

	row := s.sqlDB.QueryRowContext(
		ctx,
		`SELECT cache_key, scope, campaign_id, user_id, payload_json, source_seq, stale, checked_at, refreshed_at, expires_at
		 FROM cache_entries
		 WHERE cache_key = ?`,
		cacheKey,
	)

	var entry webstorage.CacheEntry
	var sourceSeq int64
	var staleInt int64
	var checkedAt int64
	var refreshedAt int64
	var expiresAt int64
	if err := row.Scan(
		&entry.CacheKey,
		&entry.Scope,
		&entry.CampaignID,
		&entry.UserID,
		&entry.PayloadBytes,
		&sourceSeq,
		&staleInt,
		&checkedAt,
		&refreshedAt,
		&expiresAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return webstorage.CacheEntry{}, false, nil
		}
		return webstorage.CacheEntry{}, false, fmt.Errorf("get cache entry: %w", err)
	}

	if sourceSeq > 0 {
		entry.SourceSeq = uint64(sourceSeq)
	}
	entry.Stale = staleInt != 0
	entry.CheckedAt = unixMillisToTime(checkedAt)
	entry.RefreshedAt = unixMillisToTime(refreshedAt)
	entry.ExpiresAt = unixMillisToTime(expiresAt)

	return entry, true, nil
}

// PutCacheEntry upserts a cache payload and metadata by key.
func (s *Store) PutCacheEntry(ctx context.Context, entry webstorage.CacheEntry) error {
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	entry.CacheKey = strings.TrimSpace(entry.CacheKey)
	if entry.CacheKey == "" {
		return fmt.Errorf("cache key is required")
	}
	entry.Scope = strings.TrimSpace(entry.Scope)
	if entry.Scope == "" {
		return fmt.Errorf("cache scope is required")
	}
	if len(entry.PayloadBytes) == 0 {
		return fmt.Errorf("cache payload is required")
	}

	if entry.CheckedAt.IsZero() {
		entry.CheckedAt = time.Now().UTC()
	}
	if entry.RefreshedAt.IsZero() {
		entry.RefreshedAt = entry.CheckedAt
	}

	_, err := s.sqlDB.ExecContext(
		ctx,
		`INSERT INTO cache_entries (
		    cache_key, scope, campaign_id, user_id, payload_json, source_seq, stale, checked_at, refreshed_at, expires_at
		 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(cache_key) DO UPDATE SET
		    scope = excluded.scope,
		    campaign_id = excluded.campaign_id,
		    user_id = excluded.user_id,
		    payload_json = excluded.payload_json,
		    source_seq = excluded.source_seq,
		    stale = excluded.stale,
		    checked_at = excluded.checked_at,
		    refreshed_at = excluded.refreshed_at,
		    expires_at = excluded.expires_at`,
		entry.CacheKey,
		entry.Scope,
		strings.TrimSpace(entry.CampaignID),
		strings.TrimSpace(entry.UserID),
		entry.PayloadBytes,
		int64(entry.SourceSeq),
		boolToInt(entry.Stale),
		timeToUnixMillis(entry.CheckedAt),
		timeToUnixMillis(entry.RefreshedAt),
		timeToUnixMillis(entry.ExpiresAt),
	)
	if err != nil {
		return fmt.Errorf("put cache entry: %w", err)
	}
	return nil
}

// DeleteCacheEntry removes a cache entry by key.
func (s *Store) DeleteCacheEntry(ctx context.Context, cacheKey string) error {
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	cacheKey = strings.TrimSpace(cacheKey)
	if cacheKey == "" {
		return fmt.Errorf("cache key is required")
	}
	if _, err := s.sqlDB.ExecContext(ctx, `DELETE FROM cache_entries WHERE cache_key = ?`, cacheKey); err != nil {
		return fmt.Errorf("delete cache entry: %w", err)
	}
	return nil
}

// ListTrackedCampaignIDs returns campaign IDs that currently participate in
// cache reads or have an existing event cursor.
func (s *Store) ListTrackedCampaignIDs(ctx context.Context) ([]string, error) {
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	rows, err := s.sqlDB.QueryContext(
		ctx,
		`SELECT campaign_id
		 FROM (
		   SELECT DISTINCT campaign_id
		   FROM cache_entries
		   WHERE campaign_id <> ''
		   UNION
		   SELECT campaign_id
		   FROM campaign_event_cursors
		   WHERE campaign_id <> ''
		 )
		 ORDER BY campaign_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list tracked campaign ids: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	campaignIDs := make([]string, 0)
	for rows.Next() {
		var campaignID string
		if err := rows.Scan(&campaignID); err != nil {
			return nil, fmt.Errorf("scan tracked campaign id: %w", err)
		}
		campaignIDs = append(campaignIDs, campaignID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tracked campaign ids: %w", err)
	}
	return campaignIDs, nil
}

// GetCampaignEventCursor loads the latest known event sequence for a campaign.
func (s *Store) GetCampaignEventCursor(ctx context.Context, campaignID string) (webstorage.CampaignEventCursor, bool, error) {
	if s == nil || s.sqlDB == nil {
		return webstorage.CampaignEventCursor{}, false, fmt.Errorf("storage is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return webstorage.CampaignEventCursor{}, false, fmt.Errorf("campaign id is required")
	}

	row := s.sqlDB.QueryRowContext(
		ctx,
		`SELECT campaign_id, latest_seq, checked_at
		 FROM campaign_event_cursors
		 WHERE campaign_id = ?`,
		campaignID,
	)

	var cursor webstorage.CampaignEventCursor
	var latestSeq int64
	var checkedAt int64
	if err := row.Scan(&cursor.CampaignID, &latestSeq, &checkedAt); err != nil {
		if err == sql.ErrNoRows {
			return webstorage.CampaignEventCursor{}, false, nil
		}
		return webstorage.CampaignEventCursor{}, false, fmt.Errorf("get campaign event cursor: %w", err)
	}
	if latestSeq > 0 {
		cursor.LatestSeq = uint64(latestSeq)
	}
	cursor.CheckedAt = unixMillisToTime(checkedAt)
	return cursor, true, nil
}

// PutCampaignEventCursor upserts the latest known event sequence for a campaign.
func (s *Store) PutCampaignEventCursor(ctx context.Context, cursor webstorage.CampaignEventCursor) error {
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	cursor.CampaignID = strings.TrimSpace(cursor.CampaignID)
	if cursor.CampaignID == "" {
		return fmt.Errorf("campaign id is required")
	}
	if cursor.CheckedAt.IsZero() {
		cursor.CheckedAt = time.Now().UTC()
	}

	_, err := s.sqlDB.ExecContext(
		ctx,
		`INSERT INTO campaign_event_cursors (campaign_id, latest_seq, checked_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT(campaign_id) DO UPDATE SET
		   latest_seq = excluded.latest_seq,
		   checked_at = excluded.checked_at`,
		cursor.CampaignID,
		int64(cursor.LatestSeq),
		timeToUnixMillis(cursor.CheckedAt),
	)
	if err != nil {
		return fmt.Errorf("put campaign event cursor: %w", err)
	}
	return nil
}

// MarkCampaignScopeStale marks cache rows stale for one campaign scope.
func (s *Store) MarkCampaignScopeStale(ctx context.Context, campaignID, scope string, headSeq uint64, checkedAt time.Time) error {
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return fmt.Errorf("campaign id is required")
	}
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return fmt.Errorf("cache scope is required")
	}
	if checkedAt.IsZero() {
		checkedAt = time.Now().UTC()
	}

	_, err := s.sqlDB.ExecContext(
		ctx,
		`UPDATE cache_entries
		 SET stale = 1,
		     source_seq = CASE WHEN source_seq < ? THEN ? ELSE source_seq END,
		     checked_at = CASE WHEN checked_at < ? THEN ? ELSE checked_at END
		 WHERE campaign_id = ? AND scope = ?`,
		int64(headSeq),
		int64(headSeq),
		timeToUnixMillis(checkedAt),
		timeToUnixMillis(checkedAt),
		campaignID,
		scope,
	)
	if err != nil {
		return fmt.Errorf("mark campaign scope stale: %w", err)
	}
	return nil
}

// runMigrations applies embedded SQL migrations in filename order.
func (s *Store) runMigrations() error {
	return sqlitemigrate.ApplyMigrations(s.sqlDB, migrations.FS, "")
}

// extractUpMigration isolates the `-- +migrate Up` segment for execution.
func extractUpMigration(content string) string {
	return sqlitemigrate.ExtractUpMigration(content)
}

// isAlreadyExistsError treats idempotent re-creation attempts as no-ops.
func isAlreadyExistsError(err error) bool {
	return sqlitemigrate.IsAlreadyExistsError(err)
}

func boolToInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func timeToUnixMillis(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.UTC().UnixMilli()
}

func unixMillisToTime(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(value).UTC()
}

var _ webstorage.Store = (*Store)(nil)
