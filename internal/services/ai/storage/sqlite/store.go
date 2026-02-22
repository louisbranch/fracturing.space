package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	sqlitemigrate "github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage/sqlite/migrations"
	_ "modernc.org/sqlite"
)

func toMillis(value time.Time) int64 {
	return value.UTC().UnixMilli()
}

func fromMillis(value int64) time.Time {
	return time.UnixMilli(value).UTC()
}

func encodeScopes(scopes []string) (string, error) {
	if len(scopes) == 0 {
		return "[]", nil
	}
	encoded, err := json.Marshal(scopes)
	if err != nil {
		return "", fmt.Errorf("marshal scopes: %w", err)
	}
	return string(encoded), nil
}

func decodeScopes(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	var scopes []string
	if err := json.Unmarshal([]byte(value), &scopes); err != nil {
		return nil, fmt.Errorf("unmarshal scopes: %w", err)
	}
	return scopes, nil
}

// Store provides SQLite-backed persistence for AI records.
type Store struct {
	sqlDB *sql.DB
}

// DB returns the underlying sql.DB instance.
func (s *Store) DB() *sql.DB {
	if s == nil {
		return nil
	}
	return s.sqlDB
}

// Open opens a SQLite store at the provided path.
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

// Close closes the underlying SQLite database.
func (s *Store) Close() error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}

func (s *Store) runMigrations() error {
	return sqlitemigrate.ApplyMigrations(s.sqlDB, migrations.FS, "")
}

func extractUpMigration(content string) string {
	return sqlitemigrate.ExtractUpMigration(content)
}

func isAlreadyExistsError(err error) bool {
	return sqlitemigrate.IsAlreadyExistsError(err)
}

// PutAgent persists an agent record.

func scanProviderGrantRow(row *sql.Row) (storage.ProviderGrantRecord, error) {
	var (
		rec              storage.ProviderGrantRecord
		grantedScopesRaw string
		createdAt        int64
		updatedAt        int64
		revokedAt        sql.NullInt64
		expiresAt        sql.NullInt64
		lastRefreshedAt  sql.NullInt64
	)
	if err := row.Scan(
		&rec.ID,
		&rec.OwnerUserID,
		&rec.Provider,
		&grantedScopesRaw,
		&rec.TokenCiphertext,
		&rec.RefreshSupported,
		&rec.Status,
		&rec.LastRefreshError,
		&createdAt,
		&updatedAt,
		&revokedAt,
		&expiresAt,
		&lastRefreshedAt,
	); err != nil {
		return storage.ProviderGrantRecord{}, err
	}
	scopes, err := decodeScopes(grantedScopesRaw)
	if err != nil {
		return storage.ProviderGrantRecord{}, err
	}
	rec.GrantedScopes = scopes
	rec.CreatedAt = fromMillis(createdAt)
	rec.UpdatedAt = fromMillis(updatedAt)
	if revokedAt.Valid {
		value := fromMillis(revokedAt.Int64)
		rec.RevokedAt = &value
	}
	if expiresAt.Valid {
		value := fromMillis(expiresAt.Int64)
		rec.ExpiresAt = &value
	}
	if lastRefreshedAt.Valid {
		value := fromMillis(lastRefreshedAt.Int64)
		rec.LastRefreshedAt = &value
	}
	return rec, nil
}

func scanProviderConnectSessionRow(row *sql.Row) (storage.ProviderConnectSessionRecord, error) {
	var (
		rec                storage.ProviderConnectSessionRecord
		requestedScopesRaw string
		createdAt          int64
		updatedAt          int64
		expiresAt          int64
		completedAt        sql.NullInt64
	)
	if err := row.Scan(
		&rec.ID,
		&rec.OwnerUserID,
		&rec.Provider,
		&rec.Status,
		&requestedScopesRaw,
		&rec.StateHash,
		&rec.CodeVerifierCiphertext,
		&createdAt,
		&updatedAt,
		&expiresAt,
		&completedAt,
	); err != nil {
		return storage.ProviderConnectSessionRecord{}, err
	}
	scopes, err := decodeScopes(requestedScopesRaw)
	if err != nil {
		return storage.ProviderConnectSessionRecord{}, err
	}
	rec.RequestedScopes = scopes
	rec.CreatedAt = fromMillis(createdAt)
	rec.UpdatedAt = fromMillis(updatedAt)
	rec.ExpiresAt = fromMillis(expiresAt)
	if completedAt.Valid {
		value := fromMillis(completedAt.Int64)
		rec.CompletedAt = &value
	}
	return rec, nil
}

func scanProviderGrantRows(rows *sql.Rows) (storage.ProviderGrantRecord, error) {
	var (
		rec              storage.ProviderGrantRecord
		grantedScopesRaw string
		createdAt        int64
		updatedAt        int64
		revokedAt        sql.NullInt64
		expiresAt        sql.NullInt64
		lastRefreshedAt  sql.NullInt64
	)
	if err := rows.Scan(
		&rec.ID,
		&rec.OwnerUserID,
		&rec.Provider,
		&grantedScopesRaw,
		&rec.TokenCiphertext,
		&rec.RefreshSupported,
		&rec.Status,
		&rec.LastRefreshError,
		&createdAt,
		&updatedAt,
		&revokedAt,
		&expiresAt,
		&lastRefreshedAt,
	); err != nil {
		return storage.ProviderGrantRecord{}, err
	}
	scopes, err := decodeScopes(grantedScopesRaw)
	if err != nil {
		return storage.ProviderGrantRecord{}, err
	}
	rec.GrantedScopes = scopes
	rec.CreatedAt = fromMillis(createdAt)
	rec.UpdatedAt = fromMillis(updatedAt)
	if revokedAt.Valid {
		value := fromMillis(revokedAt.Int64)
		rec.RevokedAt = &value
	}
	if expiresAt.Valid {
		value := fromMillis(expiresAt.Int64)
		rec.ExpiresAt = &value
	}
	if lastRefreshedAt.Valid {
		value := fromMillis(lastRefreshedAt.Int64)
		rec.LastRefreshedAt = &value
	}
	return rec, nil
}

func scanAccessRequestRow(row *sql.Row) (storage.AccessRequestRecord, error) {
	var (
		rec        storage.AccessRequestRecord
		createdAt  int64
		updatedAt  int64
		reviewedAt sql.NullInt64
	)
	if err := row.Scan(
		&rec.ID,
		&rec.RequesterUserID,
		&rec.OwnerUserID,
		&rec.AgentID,
		&rec.Scope,
		&rec.RequestNote,
		&rec.Status,
		&rec.ReviewerUserID,
		&rec.ReviewNote,
		&createdAt,
		&updatedAt,
		&reviewedAt,
	); err != nil {
		return storage.AccessRequestRecord{}, err
	}
	rec.CreatedAt = fromMillis(createdAt)
	rec.UpdatedAt = fromMillis(updatedAt)
	if reviewedAt.Valid {
		value := fromMillis(reviewedAt.Int64)
		rec.ReviewedAt = &value
	}
	return rec, nil
}

func scanAccessRequestRows(rows *sql.Rows) (storage.AccessRequestRecord, error) {
	var (
		rec        storage.AccessRequestRecord
		createdAt  int64
		updatedAt  int64
		reviewedAt sql.NullInt64
	)
	if err := rows.Scan(
		&rec.ID,
		&rec.RequesterUserID,
		&rec.OwnerUserID,
		&rec.AgentID,
		&rec.Scope,
		&rec.RequestNote,
		&rec.Status,
		&rec.ReviewerUserID,
		&rec.ReviewNote,
		&createdAt,
		&updatedAt,
		&reviewedAt,
	); err != nil {
		return storage.AccessRequestRecord{}, err
	}
	rec.CreatedAt = fromMillis(createdAt)
	rec.UpdatedAt = fromMillis(updatedAt)
	if reviewedAt.Valid {
		value := fromMillis(reviewedAt.Int64)
		rec.ReviewedAt = &value
	}
	return rec, nil
}
