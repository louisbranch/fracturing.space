// Package sqlite provides a SQLite-backed discovery storage implementation.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteconn"
	sqlitemigrate "github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	"github.com/louisbranch/fracturing.space/internal/services/discovery/storage"
	"github.com/louisbranch/fracturing.space/internal/services/discovery/storage/sqlite/migrations"
	msqlite "modernc.org/sqlite"
	sqlite3lib "modernc.org/sqlite/lib"
)

// Store persists discovery state in SQLite.
type Store struct {
	sqlDB *sql.DB
}

func toMillis(value time.Time) int64 {
	return value.UTC().UnixMilli()
}

func fromMillis(value int64) time.Time {
	return time.UnixMilli(value).UTC()
}

// Open opens a SQLite discovery store and applies embedded migrations.
func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("storage path is required")
	}
	sqlDB, err := sqliteconn.Open(path)
	if err != nil {
		return nil, err
	}
	if err := sqlitemigrate.ApplyMigrations(sqlDB, migrations.FS, ""); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	return &Store{sqlDB: sqlDB}, nil
}

// Close closes the SQLite handle.
func (s *Store) Close() error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}

// CreateDiscoveryEntry inserts one discovery entry record.
func (s *Store) CreateDiscoveryEntry(ctx context.Context, entry storage.DiscoveryEntry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}

	entryID := strings.TrimSpace(entry.EntryID)
	sourceID := strings.TrimSpace(entry.SourceID)
	title := strings.TrimSpace(entry.Title)
	description := strings.TrimSpace(entry.Description)
	expectedDuration := strings.TrimSpace(entry.ExpectedDurationLabel)
	storyline := strings.TrimSpace(entry.Storyline)
	if entryID == "" {
		return fmt.Errorf("entry id is required")
	}
	if sourceID == "" {
		return fmt.Errorf("source id is required")
	}
	if entry.Kind == discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_UNSPECIFIED {
		return fmt.Errorf("entry kind is required")
	}
	if title == "" {
		return fmt.Errorf("title is required")
	}
	if description == "" {
		return fmt.Errorf("description is required")
	}
	if expectedDuration == "" {
		return fmt.Errorf("expected duration label is required")
	}
	if entry.RecommendedParticipantsMin <= 0 {
		return fmt.Errorf("recommended participants min must be greater than zero")
	}
	if entry.RecommendedParticipantsMax < entry.RecommendedParticipantsMin {
		return fmt.Errorf("recommended participants max must be greater than or equal to min")
	}

	createdAt := entry.CreatedAt.UTC()
	updatedAt := entry.UpdatedAt.UTC()
	if createdAt.IsZero() && updatedAt.IsZero() {
		createdAt = time.Now().UTC()
		updatedAt = createdAt
	} else {
		if createdAt.IsZero() {
			createdAt = updatedAt
		}
		if updatedAt.IsZero() {
			updatedAt = createdAt
		}
	}

	_, err := s.sqlDB.ExecContext(
		ctx,
		`INSERT INTO discovery_entries (
		   entry_id,
		   kind,
		   source_id,
		   title,
		   description,
		   recommended_participants_min,
		   recommended_participants_max,
		   difficulty_tier,
		   expected_duration_label,
		   system,
		   gm_mode,
		   intent,
		   level,
		   character_count,
		   storyline,
		   tags,
		   created_at,
		   updated_at
		 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entryID,
		int32(entry.Kind),
		sourceID,
		title,
		description,
		entry.RecommendedParticipantsMin,
		entry.RecommendedParticipantsMax,
		int32(entry.DifficultyTier),
		expectedDuration,
		int32(entry.System),
		int32(entry.GmMode),
		int32(entry.Intent),
		entry.Level,
		entry.CharacterCount,
		storyline,
		tagsToJSON(entry.Tags),
		toMillis(createdAt),
		toMillis(updatedAt),
	)
	if err != nil {
		if isDiscoveryEntryUniqueViolation(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("create discovery entry: %w", err)
	}
	return nil
}

// GetDiscoveryEntry returns one discovery entry by entry ID.
func (s *Store) GetDiscoveryEntry(ctx context.Context, entryID string) (storage.DiscoveryEntry, error) {
	if err := ctx.Err(); err != nil {
		return storage.DiscoveryEntry{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DiscoveryEntry{}, fmt.Errorf("storage is not configured")
	}
	entryID = strings.TrimSpace(entryID)
	if entryID == "" {
		return storage.DiscoveryEntry{}, fmt.Errorf("entry id is required")
	}

	row := s.sqlDB.QueryRowContext(
		ctx,
		`SELECT entry_id, kind, source_id, title, description,
		        recommended_participants_min, recommended_participants_max,
		        difficulty_tier, expected_duration_label, system,
		        gm_mode, intent, level, character_count, storyline, tags,
		        created_at, updated_at
		   FROM discovery_entries
		  WHERE entry_id = ?`,
		entryID,
	)

	entry, err := scanEntry(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DiscoveryEntry{}, storage.ErrNotFound
		}
		return storage.DiscoveryEntry{}, fmt.Errorf("get discovery entry: %w", err)
	}
	return entry, nil
}

// ListDiscoveryEntries returns one page of discovery records.
func (s *Store) ListDiscoveryEntries(
	ctx context.Context,
	pageSize int,
	pageToken string,
	kind discoveryv1.DiscoveryEntryKind,
) (storage.DiscoveryEntryPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.DiscoveryEntryPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DiscoveryEntryPage{}, fmt.Errorf("storage is not configured")
	}
	if pageSize <= 0 {
		return storage.DiscoveryEntryPage{}, fmt.Errorf("page size must be greater than zero")
	}
	pageToken = strings.TrimSpace(pageToken)

	page := storage.DiscoveryEntryPage{
		Entries: make([]storage.DiscoveryEntry, 0, pageSize),
	}

	const selectCols = `entry_id, kind, source_id, title, description,
		        recommended_participants_min, recommended_participants_max,
		        difficulty_tier, expected_duration_label, system,
		        gm_mode, intent, level, character_count, storyline, tags,
		        created_at, updated_at`

	var (
		rows *sql.Rows
		err  error
	)
	switch {
	case kind == discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_UNSPECIFIED && pageToken == "":
		rows, err = s.sqlDB.QueryContext(
			ctx,
			`SELECT `+selectCols+`
			   FROM discovery_entries
			  ORDER BY entry_id ASC
			  LIMIT ?`,
			pageSize+1,
		)
	case kind == discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_UNSPECIFIED:
		rows, err = s.sqlDB.QueryContext(
			ctx,
			`SELECT `+selectCols+`
			   FROM discovery_entries
			  WHERE entry_id > ?
			  ORDER BY entry_id ASC
			  LIMIT ?`,
			pageToken,
			pageSize+1,
		)
	case pageToken == "":
		rows, err = s.sqlDB.QueryContext(
			ctx,
			`SELECT `+selectCols+`
			   FROM discovery_entries
			  WHERE kind = ?
			  ORDER BY entry_id ASC
			  LIMIT ?`,
			int32(kind),
			pageSize+1,
		)
	default:
		rows, err = s.sqlDB.QueryContext(
			ctx,
			`SELECT `+selectCols+`
			   FROM discovery_entries
			  WHERE kind = ? AND entry_id > ?
			  ORDER BY entry_id ASC
			  LIMIT ?`,
			int32(kind),
			pageToken,
			pageSize+1,
		)
	}
	if err != nil {
		return storage.DiscoveryEntryPage{}, fmt.Errorf("list discovery entries: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return storage.DiscoveryEntryPage{}, fmt.Errorf("list discovery entries: %w", err)
		}
		page.Entries = append(page.Entries, entry)
	}
	if err := rows.Err(); err != nil {
		return storage.DiscoveryEntryPage{}, fmt.Errorf("list discovery entries: %w", err)
	}
	if len(page.Entries) > pageSize {
		page.NextPageToken = page.Entries[pageSize-1].EntryID
		page.Entries = page.Entries[:pageSize]
	}

	return page, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEntry(scanner rowScanner) (storage.DiscoveryEntry, error) {
	var entry storage.DiscoveryEntry
	var kind int32
	var difficultyTier int32
	var system int32
	var gmMode int32
	var intent int32
	var tagsJSON string
	var createdAt int64
	var updatedAt int64
	if err := scanner.Scan(
		&entry.EntryID,
		&kind,
		&entry.SourceID,
		&entry.Title,
		&entry.Description,
		&entry.RecommendedParticipantsMin,
		&entry.RecommendedParticipantsMax,
		&difficultyTier,
		&entry.ExpectedDurationLabel,
		&system,
		&gmMode,
		&intent,
		&entry.Level,
		&entry.CharacterCount,
		&entry.Storyline,
		&tagsJSON,
		&createdAt,
		&updatedAt,
	); err != nil {
		return storage.DiscoveryEntry{}, err
	}

	entry.Kind = discoveryv1.DiscoveryEntryKind(kind)
	entry.DifficultyTier = discoveryv1.DiscoveryDifficultyTier(difficultyTier)
	entry.System = commonv1.GameSystem(system)
	entry.GmMode = discoveryv1.DiscoveryGmMode(gmMode)
	entry.Intent = discoveryv1.DiscoveryIntent(intent)
	entry.Tags = tagsFromJSON(tagsJSON)
	entry.CreatedAt = fromMillis(createdAt)
	entry.UpdatedAt = fromMillis(updatedAt)
	return entry, nil
}

func tagsToJSON(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	b, err := json.Marshal(tags)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func tagsFromJSON(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || value == "[]" {
		return nil
	}
	var tags []string
	if err := json.Unmarshal([]byte(value), &tags); err != nil {
		return nil
	}
	return tags
}

func isDiscoveryEntryUniqueViolation(err error) bool {
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
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint failed") &&
		strings.Contains(message, "discovery_entries.entry_id")
}

var _ storage.DiscoveryEntryStore = (*Store)(nil)
