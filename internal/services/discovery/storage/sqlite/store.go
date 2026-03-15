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
	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/discovery/storage"
	"github.com/louisbranch/fracturing.space/internal/services/discovery/storage/sqlite/migrations"
	msqlite "modernc.org/sqlite"
	sqlite3lib "modernc.org/sqlite/lib"
)

// Store persists discovery state in SQLite.
type Store struct {
	sqlDB *sql.DB
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
	if err := sqlitemigrate.ApplyMigrations(sqlDB, migrations.FS, "", time.Now); err != nil {
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

	normalized, err := normalizeEntry(entry)
	if err != nil {
		return err
	}

	_, err = s.sqlDB.ExecContext(
		ctx,
		`INSERT INTO discovery_entries (
		   entry_id,
		   kind,
		   source_id,
		   title,
		   description,
		   campaign_theme,
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
		   preview_hook,
		   preview_playstyle_label,
		   preview_character_name,
		   preview_character_summary,
		   created_at,
		   updated_at
		 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		normalized.EntryID,
		int32(normalized.Kind),
		normalized.SourceID,
		normalized.Title,
		normalized.Description,
		normalized.CampaignTheme,
		normalized.RecommendedParticipantsMin,
		normalized.RecommendedParticipantsMax,
		int32(normalized.DifficultyTier),
		normalized.ExpectedDurationLabel,
		int32(normalized.System),
		int32(normalized.GmMode),
		int32(normalized.Intent),
		normalized.Level,
		normalized.CharacterCount,
		normalized.Storyline,
		tagsToJSON(normalized.Tags),
		normalized.PreviewHook,
		normalized.PreviewPlaystyleLabel,
		normalized.PreviewCharacterName,
		normalized.PreviewCharacterSummary,
		sqliteutil.ToMillis(normalized.CreatedAt),
		sqliteutil.ToMillis(normalized.UpdatedAt),
	)
	if err != nil {
		if isDiscoveryEntryUniqueViolation(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("create discovery entry: %w", err)
	}
	return nil
}

// UpsertBuiltinDiscoveryEntry inserts or updates a builtin discovery entry.
// When the incoming source_id is empty, an existing reconciled source_id is preserved.
func (s *Store) UpsertBuiltinDiscoveryEntry(ctx context.Context, entry storage.DiscoveryEntry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}

	normalized, err := normalizeEntry(entry)
	if err != nil {
		return err
	}

	if normalized.SourceID == "" {
		current, err := s.GetDiscoveryEntry(ctx, normalized.EntryID)
		switch {
		case err == nil:
			normalized.SourceID = current.SourceID
			if normalized.CreatedAt.IsZero() {
				normalized.CreatedAt = current.CreatedAt
			}
		case errors.Is(err, storage.ErrNotFound):
		default:
			return fmt.Errorf("load existing builtin entry: %w", err)
		}
	}

	_, err = s.sqlDB.ExecContext(
		ctx,
		`INSERT INTO discovery_entries (
		   entry_id,
		   kind,
		   source_id,
		   title,
		   description,
		   campaign_theme,
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
		   preview_hook,
		   preview_playstyle_label,
		   preview_character_name,
		   preview_character_summary,
		   created_at,
		   updated_at
		 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(entry_id) DO UPDATE SET
		   kind = excluded.kind,
		   source_id = CASE
		     WHEN excluded.source_id = '' THEN discovery_entries.source_id
		     ELSE excluded.source_id
		   END,
		   title = excluded.title,
		   description = excluded.description,
		   campaign_theme = excluded.campaign_theme,
		   recommended_participants_min = excluded.recommended_participants_min,
		   recommended_participants_max = excluded.recommended_participants_max,
		   difficulty_tier = excluded.difficulty_tier,
		   expected_duration_label = excluded.expected_duration_label,
		   system = excluded.system,
		   gm_mode = excluded.gm_mode,
		   intent = excluded.intent,
		   level = excluded.level,
		   character_count = excluded.character_count,
		   storyline = excluded.storyline,
		   tags = excluded.tags,
		   preview_hook = excluded.preview_hook,
		   preview_playstyle_label = excluded.preview_playstyle_label,
		   preview_character_name = excluded.preview_character_name,
		   preview_character_summary = excluded.preview_character_summary,
		   updated_at = excluded.updated_at`,
		normalized.EntryID,
		int32(normalized.Kind),
		normalized.SourceID,
		normalized.Title,
		normalized.Description,
		normalized.CampaignTheme,
		normalized.RecommendedParticipantsMin,
		normalized.RecommendedParticipantsMax,
		int32(normalized.DifficultyTier),
		normalized.ExpectedDurationLabel,
		int32(normalized.System),
		int32(normalized.GmMode),
		int32(normalized.Intent),
		normalized.Level,
		normalized.CharacterCount,
		normalized.Storyline,
		tagsToJSON(normalized.Tags),
		normalized.PreviewHook,
		normalized.PreviewPlaystyleLabel,
		normalized.PreviewCharacterName,
		normalized.PreviewCharacterSummary,
		sqliteutil.ToMillis(normalized.CreatedAt),
		sqliteutil.ToMillis(normalized.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("upsert discovery entry: %w", err)
	}
	return nil
}

// UpdateDiscoveryEntrySourceID updates the reconciled template source_id for one builtin entry.
func (s *Store) UpdateDiscoveryEntrySourceID(ctx context.Context, entryID string, sourceID string, updatedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}

	entryID = strings.TrimSpace(entryID)
	sourceID = strings.TrimSpace(sourceID)
	if entryID == "" {
		return fmt.Errorf("entry id is required")
	}
	if sourceID == "" {
		return fmt.Errorf("source id is required")
	}
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}

	result, err := s.sqlDB.ExecContext(
		ctx,
		`UPDATE discovery_entries
		    SET source_id = ?, updated_at = ?
		  WHERE entry_id = ?`,
		sourceID,
		sqliteutil.ToMillis(updatedAt.UTC()),
		entryID,
	)
	if err != nil {
		return fmt.Errorf("update discovery entry source id: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update discovery entry source id rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return storage.ErrNotFound
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
		`SELECT entry_id, kind, source_id, title, description, campaign_theme,
		        recommended_participants_min, recommended_participants_max,
		        difficulty_tier, expected_duration_label, system,
		        gm_mode, intent, level, character_count, storyline, tags,
		        preview_hook, preview_playstyle_label, preview_character_name, preview_character_summary,
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

	const selectCols = `entry_id, kind, source_id, title, description, campaign_theme,
		        recommended_participants_min, recommended_participants_max,
		        difficulty_tier, expected_duration_label, system,
		        gm_mode, intent, level, character_count, storyline, tags,
		        preview_hook, preview_playstyle_label, preview_character_name, preview_character_summary,
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
	var previewHook string
	var previewPlaystyleLabel string
	var previewCharacterName string
	var previewCharacterSummary string
	var createdAt int64
	var updatedAt int64
	if err := scanner.Scan(
		&entry.EntryID,
		&kind,
		&entry.SourceID,
		&entry.Title,
		&entry.Description,
		&entry.CampaignTheme,
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
		&previewHook,
		&previewPlaystyleLabel,
		&previewCharacterName,
		&previewCharacterSummary,
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
	entry.PreviewHook = strings.TrimSpace(previewHook)
	entry.PreviewPlaystyleLabel = strings.TrimSpace(previewPlaystyleLabel)
	entry.PreviewCharacterName = strings.TrimSpace(previewCharacterName)
	entry.PreviewCharacterSummary = strings.TrimSpace(previewCharacterSummary)
	entry.CreatedAt = sqliteutil.FromMillis(createdAt)
	entry.UpdatedAt = sqliteutil.FromMillis(updatedAt)
	return entry, nil
}

func normalizeEntry(entry storage.DiscoveryEntry) (storage.DiscoveryEntry, error) {
	entry.EntryID = strings.TrimSpace(entry.EntryID)
	entry.SourceID = strings.TrimSpace(entry.SourceID)
	entry.Title = strings.TrimSpace(entry.Title)
	entry.Description = strings.TrimSpace(entry.Description)
	entry.CampaignTheme = strings.TrimSpace(entry.CampaignTheme)
	entry.ExpectedDurationLabel = strings.TrimSpace(entry.ExpectedDurationLabel)
	entry.Storyline = strings.TrimSpace(entry.Storyline)
	entry.PreviewHook = strings.TrimSpace(entry.PreviewHook)
	entry.PreviewPlaystyleLabel = strings.TrimSpace(entry.PreviewPlaystyleLabel)
	entry.PreviewCharacterName = strings.TrimSpace(entry.PreviewCharacterName)
	entry.PreviewCharacterSummary = strings.TrimSpace(entry.PreviewCharacterSummary)
	entry.Tags = tagsFromJSON(tagsToJSON(entry.Tags))

	if entry.EntryID == "" {
		return storage.DiscoveryEntry{}, fmt.Errorf("entry id is required")
	}
	if entry.Kind == discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_UNSPECIFIED {
		return storage.DiscoveryEntry{}, fmt.Errorf("entry kind is required")
	}
	if entry.Title == "" {
		return storage.DiscoveryEntry{}, fmt.Errorf("title is required")
	}
	if entry.Description == "" {
		return storage.DiscoveryEntry{}, fmt.Errorf("description is required")
	}
	if entry.ExpectedDurationLabel == "" {
		return storage.DiscoveryEntry{}, fmt.Errorf("expected duration label is required")
	}
	if entry.RecommendedParticipantsMin <= 0 {
		return storage.DiscoveryEntry{}, fmt.Errorf("recommended participants min must be greater than zero")
	}
	if entry.RecommendedParticipantsMax < entry.RecommendedParticipantsMin {
		return storage.DiscoveryEntry{}, fmt.Errorf("recommended participants max must be greater than or equal to min")
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
	entry.CreatedAt = createdAt
	entry.UpdatedAt = updatedAt
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
