package sqlite

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteconn"
	sqlitemigrate "github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/ai/accessrequest"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage/sqlite/migrations"
	msqlite "modernc.org/sqlite"
	sqlite3lib "modernc.org/sqlite/lib"
)

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
	if path == "" {
		return nil, fmt.Errorf("storage path is required")
	}

	sqlDB, err := sqliteconn.Open(path)
	if err != nil {
		return nil, err
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
	return sqlitemigrate.ApplyMigrations(s.sqlDB, migrations.FS, "", time.Now)
}

func extractUpMigration(content string) string {
	return sqlitemigrate.ExtractUpMigration(content)
}

func isAlreadyExistsError(err error) bool {
	return sqlitemigrate.IsAlreadyExistsError(err)
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	var sqliteErr *msqlite.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code() {
		case sqlite3lib.SQLITE_CONSTRAINT_PRIMARYKEY, sqlite3lib.SQLITE_CONSTRAINT_UNIQUE:
			return true
		}
	}
	return strings.Contains(strings.ToLower(err.Error()), "unique constraint failed")
}

// scanner abstracts *sql.Row and *sql.Rows so scan functions need only one
// implementation per record type.
type scanner interface {
	Scan(dest ...any) error
}

func scanProviderGrant(s scanner) (providergrant.ProviderGrant, error) {
	var (
		id, ownerUserID, providerRaw string
		grantedScopesRaw             string
		tokenCiphertext              string
		refreshSupported             bool
		statusRaw, lastRefreshError  string
		createdAt, updatedAt         int64
		revokedAt, expiresAt         sql.NullInt64
		lastRefreshedAt              sql.NullInt64
	)
	if err := s.Scan(
		&id, &ownerUserID, &providerRaw,
		&grantedScopesRaw, &tokenCiphertext,
		&refreshSupported, &statusRaw, &lastRefreshError,
		&createdAt, &updatedAt,
		&revokedAt, &expiresAt, &lastRefreshedAt,
	); err != nil {
		return providergrant.ProviderGrant{}, err
	}
	scopes, err := decodeScopes(grantedScopesRaw)
	if err != nil {
		return providergrant.ProviderGrant{}, err
	}
	normalizedProvider, _ := provider.Normalize(providerRaw)
	grant := providergrant.ProviderGrant{
		ID:               id,
		OwnerUserID:      ownerUserID,
		Provider:         normalizedProvider,
		GrantedScopes:    scopes,
		TokenCiphertext:  tokenCiphertext,
		RefreshSupported: refreshSupported,
		Status:           providergrant.ParseStatus(statusRaw),
		LastRefreshError: lastRefreshError,
		CreatedAt:        sqliteutil.FromMillis(createdAt),
		UpdatedAt:        sqliteutil.FromMillis(updatedAt),
	}
	if revokedAt.Valid {
		value := sqliteutil.FromMillis(revokedAt.Int64)
		grant.RevokedAt = &value
	}
	if expiresAt.Valid {
		value := sqliteutil.FromMillis(expiresAt.Int64)
		grant.ExpiresAt = &value
	}
	if lastRefreshedAt.Valid {
		value := sqliteutil.FromMillis(lastRefreshedAt.Int64)
		grant.RefreshedAt = &value
	}
	return grant, nil
}

func scanProviderConnectSession(s scanner) (storage.ProviderConnectSessionRecord, error) {
	var (
		rec                storage.ProviderConnectSessionRecord
		requestedScopesRaw string
		createdAt          int64
		updatedAt          int64
		expiresAt          int64
		completedAt        sql.NullInt64
	)
	if err := s.Scan(
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
	rec.CreatedAt = sqliteutil.FromMillis(createdAt)
	rec.UpdatedAt = sqliteutil.FromMillis(updatedAt)
	rec.ExpiresAt = sqliteutil.FromMillis(expiresAt)
	if completedAt.Valid {
		value := sqliteutil.FromMillis(completedAt.Int64)
		rec.CompletedAt = &value
	}
	return rec, nil
}

func scanAccessRequest(s scanner) (accessrequest.AccessRequest, error) {
	var (
		id, requesterUserID, ownerUserID, agentID string
		scopeRaw, requestNote, statusRaw          string
		reviewerUserID, reviewNote                string
		createdAt, updatedAt                      int64
		reviewedAt                                sql.NullInt64
	)
	if err := s.Scan(
		&id, &requesterUserID, &ownerUserID, &agentID,
		&scopeRaw, &requestNote, &statusRaw,
		&reviewerUserID, &reviewNote,
		&createdAt, &updatedAt, &reviewedAt,
	); err != nil {
		return accessrequest.AccessRequest{}, err
	}
	ar := accessrequest.AccessRequest{
		ID:              id,
		RequesterUserID: requesterUserID,
		OwnerUserID:     ownerUserID,
		AgentID:         agentID,
		Scope:           accessrequest.Scope(scopeRaw),
		RequestNote:     requestNote,
		Status:          accessrequest.ParseStatus(statusRaw),
		ReviewerUserID:  reviewerUserID,
		ReviewNote:      reviewNote,
		CreatedAt:       sqliteutil.FromMillis(createdAt),
		UpdatedAt:       sqliteutil.FromMillis(updatedAt),
	}
	if reviewedAt.Valid {
		value := sqliteutil.FromMillis(reviewedAt.Int64)
		ar.ReviewedAt = &value
	}
	return ar, nil
}
