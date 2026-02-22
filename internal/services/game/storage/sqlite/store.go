package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/integrity"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/migrations"
)

func toMillis(value time.Time) int64 {
	return value.UTC().UnixMilli()
}

// fromMillis reverses toMillis for persisted millisecond timestamps.
func fromMillis(value int64) time.Time {
	return time.UnixMilli(value).UTC()
}

// toNullMillis maps optional domain times to sql.NullInt64 for nullable DB columns.
func toNullMillis(value *time.Time) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: toMillis(*value), Valid: true}
}

// fromNullMillis maps nullable SQL timestamps back into optional domain time values.
func fromNullMillis(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}
	t := fromMillis(value.Int64)
	return &t
}

// Store provides a SQLite-backed store implementing all storage interfaces.
type Store struct {
	sqlDB                        *sql.DB
	q                            *db.Queries
	keyring                      *integrity.Keyring
	eventRegistry                *event.Registry
	projectionApplyOutboxEnabled bool
}

func (s *Store) withTx(tx *sql.Tx) *Store {
	if s == nil || tx == nil {
		return s
	}
	cloned := *s
	cloned.q = s.q.WithTx(tx)
	return &cloned
}

// OpenEventsOption configures event-store behavior.
type OpenEventsOption func(*Store)

// WithProjectionApplyOutboxEnabled toggles enqueueing projection-apply work for appended events.
func WithProjectionApplyOutboxEnabled(enabled bool) OpenEventsOption {
	return func(s *Store) {
		s.projectionApplyOutboxEnabled = enabled
	}
}

// Open opens a SQLite projections store at the provided path.
//
// This is the historic convenience constructor used by some startup codepaths that
// only need projection storage.
func Open(path string) (*Store, error) {
	return OpenProjections(path)
}

// OpenEvents opens a SQLite event journal store at the provided path.
//
// This path wires integrity key material and the event registry so every appended
// event can be consistently hashed and validated in one place.
func OpenEvents(path string, keyring *integrity.Keyring, registry *event.Registry, opts ...OpenEventsOption) (*Store, error) {
	store, err := openStore(path, migrations.EventsFS, "events", keyring)
	if err != nil {
		return nil, err
	}
	store.eventRegistry = registry
	for _, opt := range opts {
		if opt != nil {
			opt(store)
		}
	}
	return store, nil
}

// OpenProjections opens a SQLite projections store at the provided path.
func OpenProjections(path string) (*Store, error) {
	return openStore(path, migrations.ProjectionsFS, "projections", nil)
}

// OpenContent opens a SQLite content catalog store at the provided path.
func OpenContent(path string) (*Store, error) {
	return openStore(path, migrations.ContentFS, "content", nil)
}

// Close closes the underlying SQLite database.
//
// Close is intentionally nil-safe so callers can defer it in all startup paths.
func (s *Store) Close() error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}

// openStore boots a SQLite bundle for a domain purpose (events/projections/content)
// and applies embedded migrations before the store is handed to higher layers.
func openStore(path string, migrationFS fs.FS, migrationRoot string, keyring *integrity.Keyring) (*Store, error) {
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

	store := &Store{
		sqlDB:   sqlDB,
		q:       db.New(sqlDB),
		keyring: keyring,
	}

	if err := runMigrations(sqlDB, migrationFS, migrationRoot); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	if migrationRoot == "projections" {
		if err := ensureInviteRecipientColumn(sqlDB); err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("ensure invite schema: %w", err)
		}
	}

	return store, nil
}

// ensureInviteRecipientColumn backfills invite schema when older databases omit recipient_user_id.
func ensureInviteRecipientColumn(sqlDB *sql.DB) error {
	rows, err := sqlDB.Query("PRAGMA table_info(invites)")
	if err != nil {
		return fmt.Errorf("inspect invites table: %w", err)
	}
	defer rows.Close()

	var hasRecipient bool
	for rows.Next() {
		var cid int
		var name string
		var colType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("scan invites table info: %w", err)
		}
		if name == "recipient_user_id" {
			hasRecipient = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read invites table info: %w", err)
	}
	if hasRecipient {
		return nil
	}

	const inviteRebuildSQL = `
DROP INDEX IF EXISTS idx_invites_recipient_status;
DROP INDEX IF EXISTS idx_invites_participant;
DROP INDEX IF EXISTS idx_invites_campaign;
DROP TABLE IF EXISTS invites;

CREATE TABLE invites (
    id TEXT PRIMARY KEY,
    campaign_id TEXT NOT NULL,
    participant_id TEXT NOT NULL,
    recipient_user_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    created_by_participant_id TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE,
    FOREIGN KEY (campaign_id, participant_id) REFERENCES participants(campaign_id, id) ON DELETE CASCADE
);

CREATE INDEX idx_invites_campaign ON invites(campaign_id);
CREATE INDEX idx_invites_participant ON invites(participant_id);
CREATE INDEX idx_invites_recipient_status ON invites(recipient_user_id, status);
`

	if _, err := sqlDB.Exec(inviteRebuildSQL); err != nil {
		return fmt.Errorf("rebuild invites table: %w", err)
	}

	return nil
}

// runMigrations executes embedded SQL migrations from the provided migration set.
// Files are sorted lexicographically to make startup behavior deterministic.
func runMigrations(sqlDB *sql.DB, migrationFS fs.FS, migrationRoot string) error {
	return sqlitemigrate.ApplyMigrations(sqlDB, migrationFS, migrationRoot)
}

// extractUpMigration extracts the Up migration portion from a migration file.
// Down sections are intentionally ignored during startup execution.
func extractUpMigration(content string) string {
	return sqlitemigrate.ExtractUpMigration(content)
}

// isAlreadyExistsError checks if the error is a table/index already exists error.
func isAlreadyExistsError(err error) bool {
	return sqlitemigrate.IsAlreadyExistsError(err)
}

// Campaign methods

// Put persists a campaign record.
func appendEventTx(ctx context.Context, qtx *db.Queries, keyring *integrity.Keyring, registry *event.Registry, evt event.Event) (event.Event, error) {
	if qtx == nil {
		return event.Event{}, fmt.Errorf("event store is not configured")
	}
	if registry == nil {
		return event.Event{}, fmt.Errorf("event registry is required")
	}

	normalized, err := registry.ValidateForAppend(evt)
	if err != nil {
		return event.Event{}, err
	}
	evt = normalized
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}
	evt.Timestamp = evt.Timestamp.UTC().Truncate(time.Millisecond)

	if err := qtx.InitEventSeq(ctx, evt.CampaignID); err != nil {
		return event.Event{}, fmt.Errorf("init event seq: %w", err)
	}

	seq, err := qtx.GetEventSeq(ctx, evt.CampaignID)
	if err != nil {
		return event.Event{}, fmt.Errorf("get event seq: %w", err)
	}
	if err := qtx.IncrementEventSeq(ctx, evt.CampaignID); err != nil {
		return event.Event{}, fmt.Errorf("increment event seq: %w", err)
	}
	if seq <= 0 {
		return event.Event{}, fmt.Errorf("event seq is required")
	}
	evt.Seq = uint64(seq)

	if keyring == nil {
		return event.Event{}, fmt.Errorf("event integrity keyring is required")
	}

	hash, err := integrity.EventHash(evt)
	if err != nil {
		return event.Event{}, fmt.Errorf("compute event hash: %w", err)
	}
	if strings.TrimSpace(hash) == "" {
		return event.Event{}, fmt.Errorf("event hash is required")
	}
	evt.Hash = hash

	prevHash := ""
	if evt.Seq > 1 {
		prevRow, err := qtx.GetEventBySeq(ctx, db.GetEventBySeqParams{
			CampaignID: evt.CampaignID,
			Seq:        int64(evt.Seq - 1),
		})
		if err != nil {
			return event.Event{}, fmt.Errorf("load previous event: %w", err)
		}
		prevHash = prevRow.ChainHash
	}

	chainHash, err := integrity.ChainHash(evt, prevHash)
	if err != nil {
		return event.Event{}, fmt.Errorf("compute chain hash: %w", err)
	}
	if strings.TrimSpace(chainHash) == "" {
		return event.Event{}, fmt.Errorf("chain hash is required")
	}

	signature, keyID, err := keyring.SignChainHash(evt.CampaignID, chainHash)
	if err != nil {
		return event.Event{}, fmt.Errorf("sign chain hash: %w", err)
	}

	evt.PrevHash = prevHash
	evt.ChainHash = chainHash
	evt.Signature = signature
	evt.SignatureKeyID = keyID
	if err := qtx.AppendEvent(ctx, db.AppendEventParams{
		CampaignID:     evt.CampaignID,
		Seq:            seq,
		EventHash:      hash,
		PrevEventHash:  prevHash,
		ChainHash:      chainHash,
		SignatureKeyID: keyID,
		EventSignature: signature,
		Timestamp:      toMillis(evt.Timestamp),
		EventType:      string(evt.Type),
		SessionID:      evt.SessionID,
		RequestID:      evt.RequestID,
		InvocationID:   evt.InvocationID,
		ActorType:      string(evt.ActorType),
		ActorID:        evt.ActorID,
		EntityType:     evt.EntityType,
		EntityID:       evt.EntityID,
		SystemID:       evt.SystemID,
		SystemVersion:  evt.SystemVersion,
		CorrelationID:  evt.CorrelationID,
		CausationID:    evt.CausationID,
		PayloadJson:    evt.PayloadJSON,
	}); err != nil {
		return event.Event{}, fmt.Errorf("append event: %w", err)
	}

	return evt, nil
}
