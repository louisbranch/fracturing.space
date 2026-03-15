package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteconn"
	sqlitemigrate "github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite/db"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite/migrations"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
)

const authStatisticsQuery = `
SELECT COUNT(*)
FROM users
WHERE (?1 IS NULL OR created_at >= ?1);
`

// Store implements auth persistence over SQLite.
//
// A single SQLite file backs identity state so every auth subflow can share the
// same transaction and visibility boundaries.
type Store struct {
	sqlDB *sql.DB
	q     *db.Queries
}

type execContexter interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// DB returns the raw database handle for OAuth and legacy callers.
func (s *Store) DB() *sql.DB {
	if s == nil {
		return nil
	}
	return s.sqlDB
}

// Open opens an auth SQLite store and applies bundled migrations.
//
// This keeps startup and schema evolution in one place, instead of requiring
// callers to coordinate migrations independently.
func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("Storage path is required.")
	}

	sqlDB, err := sqliteconn.Open(path)
	if err != nil {
		return nil, err
	}

	store := &Store{
		sqlDB: sqlDB,
		q:     db.New(sqlDB),
	}

	if err := store.runMigrations(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("Run migrations: %w", err)
	}

	return store, nil
}

// Close releases the underlying SQLite database.
func (s *Store) Close() error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}

// runMigrations applies embedded DDL snapshots for known schema versions.
func (s *Store) runMigrations() error {
	return sqlitemigrate.ApplyMigrations(s.sqlDB, migrations.FS, "", time.Now)
}

// extractUpMigration extracts only the upgrade section from a migration file.
func extractUpMigration(content string) string {
	return sqlitemigrate.ExtractUpMigration(content)
}

// isAlreadyExistsError detects SQLite "already exists" conditions during idempotent runs.
func isAlreadyExistsError(err error) bool {
	return sqlitemigrate.IsAlreadyExistsError(err)
}

// dbUserToDomain reconstructs auth-owned identity state from SQLite rows.
func dbUserToDomain(row db.User) user.User {
	parsedLocale := platformi18n.DefaultLocale()
	if parsed, ok := platformi18n.ParseLocale(row.Locale); ok {
		parsedLocale = parsed
	}

	var recoveryReservedUntil *time.Time
	if row.RecoveryReservedUntil.Valid {
		value := sqliteutil.FromMillis(row.RecoveryReservedUntil.Int64)
		recoveryReservedUntil = &value
	}

	return user.User{
		ID:                        row.ID,
		Username:                  row.Username,
		Locale:                    parsedLocale,
		RecoveryCodeHash:          row.RecoveryCodeHash,
		RecoveryReservedSessionID: row.RecoveryReservedSessionID,
		RecoveryReservedUntil:     recoveryReservedUntil,
		RecoveryCodeUpdatedAt:     sqliteutil.FromMillis(row.RecoveryCodeUpdatedAt),
		CreatedAt:                 sqliteutil.FromMillis(row.CreatedAt),
		UpdatedAt:                 sqliteutil.FromMillis(row.UpdatedAt),
	}
}

func scanIntegrationOutboxEvent(scan integrationOutboxScanner) (storage.IntegrationOutboxEvent, error) {
	var event storage.IntegrationOutboxEvent
	var nextAttemptAt int64
	var createdAt int64
	var updatedAt int64
	var leaseExpiresAt sql.NullInt64
	var processedAt sql.NullInt64
	if err := scan(
		&event.ID,
		&event.EventType,
		&event.PayloadJSON,
		&event.DedupeKey,
		&event.Status,
		&event.AttemptCount,
		&nextAttemptAt,
		&event.LeaseOwner,
		&leaseExpiresAt,
		&event.LastError,
		&processedAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return storage.IntegrationOutboxEvent{}, err
	}
	event.NextAttemptAt = sqliteutil.FromMillis(nextAttemptAt)
	event.CreatedAt = sqliteutil.FromMillis(createdAt)
	event.UpdatedAt = sqliteutil.FromMillis(updatedAt)
	if leaseExpiresAt.Valid {
		value := sqliteutil.FromMillis(leaseExpiresAt.Int64)
		event.LeaseExpiresAt = &value
	}
	if processedAt.Valid {
		value := sqliteutil.FromMillis(processedAt.Int64)
		event.ProcessedAt = &value
	}
	return event, nil
}

func normalizeIntegrationOutboxEvent(event storage.IntegrationOutboxEvent) (storage.IntegrationOutboxEvent, error) {
	event.ID = strings.TrimSpace(event.ID)
	event.EventType = strings.TrimSpace(event.EventType)
	event.PayloadJSON = strings.TrimSpace(event.PayloadJSON)
	event.DedupeKey = strings.TrimSpace(event.DedupeKey)
	event.Status = strings.TrimSpace(event.Status)
	event.LeaseOwner = strings.TrimSpace(event.LeaseOwner)
	event.LastError = strings.TrimSpace(event.LastError)
	if event.ID == "" {
		return storage.IntegrationOutboxEvent{}, fmt.Errorf("Event ID is required.")
	}
	if event.EventType == "" {
		return storage.IntegrationOutboxEvent{}, fmt.Errorf("Event type is required.")
	}
	if event.PayloadJSON == "" {
		event.PayloadJSON = "{}"
	}
	if event.Status == "" {
		event.Status = storage.IntegrationOutboxStatusPending
	}
	if event.AttemptCount < 0 {
		return storage.IntegrationOutboxEvent{}, fmt.Errorf("Attempt count must be greater than or equal to zero.")
	}
	now := time.Now().UTC()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	if event.UpdatedAt.IsZero() {
		event.UpdatedAt = event.CreatedAt
	}
	if event.NextAttemptAt.IsZero() {
		event.NextAttemptAt = event.CreatedAt
	}
	return event, nil
}

func enqueueIntegrationOutboxEvent(ctx context.Context, target execContexter, event storage.IntegrationOutboxEvent) error {
	normalized, err := normalizeIntegrationOutboxEvent(event)
	if err != nil {
		return err
	}

	var leaseExpiresAt sql.NullInt64
	if normalized.LeaseExpiresAt != nil {
		leaseExpiresAt = sql.NullInt64{Int64: sqliteutil.ToMillis(normalized.LeaseExpiresAt.UTC()), Valid: true}
	}
	var processedAt sql.NullInt64
	if normalized.ProcessedAt != nil {
		processedAt = sql.NullInt64{Int64: sqliteutil.ToMillis(normalized.ProcessedAt.UTC()), Valid: true}
	}

	_, err = target.ExecContext(ctx, `
INSERT INTO auth_integration_outbox (
	id,
	event_type,
	payload_json,
	dedupe_key,
	status,
	attempt_count,
	next_attempt_at,
	lease_owner,
	lease_expires_at,
	last_error,
	processed_at,
	created_at,
	updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(dedupe_key) WHERE dedupe_key <> '' DO NOTHING
`,
		normalized.ID,
		normalized.EventType,
		normalized.PayloadJSON,
		normalized.DedupeKey,
		normalized.Status,
		normalized.AttemptCount,
		sqliteutil.ToMillis(normalized.NextAttemptAt),
		normalized.LeaseOwner,
		leaseExpiresAt,
		normalized.LastError,
		processedAt,
		sqliteutil.ToMillis(normalized.CreatedAt),
		sqliteutil.ToMillis(normalized.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("Enqueue integration outbox event: %w", err)
	}
	return nil
}

var _ storage.UserStore = (*Store)(nil)
var _ storage.StatisticsStore = (*Store)(nil)
var _ storage.PasskeyStore = (*Store)(nil)
var _ storage.WebSessionStore = (*Store)(nil)
var _ storage.IntegrationOutboxStore = (*Store)(nil)
var _ storage.UserOutboxTransactionalStore = (*Store)(nil)
var _ storage.UserSignupTransactionalStore = (*Store)(nil)
