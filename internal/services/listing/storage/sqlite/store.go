// Package sqlite provides a SQLite-backed listing storage implementation.
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
	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteconn"
	sqlitemigrate "github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	"github.com/louisbranch/fracturing.space/internal/services/listing/storage"
	"github.com/louisbranch/fracturing.space/internal/services/listing/storage/sqlite/migrations"
	msqlite "modernc.org/sqlite"
	sqlite3lib "modernc.org/sqlite/lib"
)

// Store persists listing state in SQLite.
type Store struct {
	sqlDB *sql.DB
}

func toMillis(value time.Time) int64 {
	return value.UTC().UnixMilli()
}

func fromMillis(value int64) time.Time {
	return time.UnixMilli(value).UTC()
}

// Open opens a SQLite listing store and applies embedded migrations.
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

// CreateCampaignListing inserts one listing record.
func (s *Store) CreateCampaignListing(ctx context.Context, listing storage.CampaignListing) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	campaignID := strings.TrimSpace(listing.CampaignID)
	title := strings.TrimSpace(listing.Title)
	description := strings.TrimSpace(listing.Description)
	expectedDuration := strings.TrimSpace(listing.ExpectedDurationLabel)
	if campaignID == "" {
		return fmt.Errorf("campaign id is required")
	}
	if title == "" {
		return fmt.Errorf("title is required")
	}
	if listing.RecommendedParticipantsMin <= 0 {
		return fmt.Errorf("recommended participants min must be greater than zero")
	}
	if listing.RecommendedParticipantsMax < listing.RecommendedParticipantsMin {
		return fmt.Errorf("recommended participants max must be greater than or equal to min")
	}
	createdAt := listing.CreatedAt.UTC()
	updatedAt := listing.UpdatedAt.UTC()
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
		`INSERT INTO campaign_listings (
		   campaign_id,
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
		 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		campaignID,
		title,
		description,
		listing.RecommendedParticipantsMin,
		listing.RecommendedParticipantsMax,
		int32(listing.DifficultyTier),
		expectedDuration,
		int32(listing.System),
		int32(listing.GmMode),
		int32(listing.Intent),
		listing.Level,
		listing.CharacterCount,
		listing.Storyline,
		tagsToJSON(listing.Tags),
		toMillis(createdAt),
		toMillis(updatedAt),
	)
	if err != nil {
		if isCampaignListingUniqueViolation(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("create campaign listing: %w", err)
	}
	return nil
}

// GetCampaignListing returns one listing by campaign ID.
func (s *Store) GetCampaignListing(ctx context.Context, campaignID string) (storage.CampaignListing, error) {
	if err := ctx.Err(); err != nil {
		return storage.CampaignListing{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.CampaignListing{}, fmt.Errorf("storage is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return storage.CampaignListing{}, fmt.Errorf("campaign id is required")
	}

	row := s.sqlDB.QueryRowContext(
		ctx,
		`SELECT campaign_id, title, description,
		        recommended_participants_min, recommended_participants_max,
		        difficulty_tier, expected_duration_label, system,
		        gm_mode, intent, level, character_count, storyline, tags,
		        created_at, updated_at
		   FROM campaign_listings
		  WHERE campaign_id = ?`,
		campaignID,
	)

	var listing storage.CampaignListing
	var difficultyTier int32
	var system int32
	var gmMode int32
	var intent int32
	var tagsJSON string
	var createdAt int64
	var updatedAt int64
	err := row.Scan(
		&listing.CampaignID,
		&listing.Title,
		&listing.Description,
		&listing.RecommendedParticipantsMin,
		&listing.RecommendedParticipantsMax,
		&difficultyTier,
		&listing.ExpectedDurationLabel,
		&system,
		&gmMode,
		&intent,
		&listing.Level,
		&listing.CharacterCount,
		&listing.Storyline,
		&tagsJSON,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.CampaignListing{}, storage.ErrNotFound
		}
		return storage.CampaignListing{}, fmt.Errorf("get campaign listing: %w", err)
	}

	listing.DifficultyTier = listingv1.CampaignDifficultyTier(difficultyTier)
	listing.System = commonv1.GameSystem(system)
	listing.GmMode = listingv1.CampaignListingGmMode(gmMode)
	listing.Intent = listingv1.CampaignListingIntent(intent)
	listing.Tags = tagsFromJSON(tagsJSON)
	listing.CreatedAt = fromMillis(createdAt)
	listing.UpdatedAt = fromMillis(updatedAt)
	return listing, nil
}

// ListCampaignListings returns one page of listing records.
func (s *Store) ListCampaignListings(ctx context.Context, pageSize int, pageToken string) (storage.CampaignListingPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.CampaignListingPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.CampaignListingPage{}, fmt.Errorf("storage is not configured")
	}
	if pageSize <= 0 {
		return storage.CampaignListingPage{}, fmt.Errorf("page size must be greater than zero")
	}
	pageToken = strings.TrimSpace(pageToken)

	page := storage.CampaignListingPage{
		Listings: make([]storage.CampaignListing, 0, pageSize),
	}

	var (
		rows *sql.Rows
		err  error
	)
	const selectCols = `campaign_id, title, description,
			        recommended_participants_min, recommended_participants_max,
			        difficulty_tier, expected_duration_label, system,
			        gm_mode, intent, level, character_count, storyline, tags,
			        created_at, updated_at`
	if pageToken == "" {
		rows, err = s.sqlDB.QueryContext(
			ctx,
			`SELECT `+selectCols+`
			   FROM campaign_listings
			  ORDER BY campaign_id ASC
			  LIMIT ?`,
			pageSize+1,
		)
	} else {
		rows, err = s.sqlDB.QueryContext(
			ctx,
			`SELECT `+selectCols+`
			   FROM campaign_listings
			  WHERE campaign_id > ?
			  ORDER BY campaign_id ASC
			  LIMIT ?`,
			pageToken,
			pageSize+1,
		)
	}
	if err != nil {
		return storage.CampaignListingPage{}, fmt.Errorf("list campaign listings: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var listing storage.CampaignListing
		var difficultyTier int32
		var system int32
		var gmMode int32
		var intent int32
		var tagsJSON string
		var createdAt int64
		var updatedAt int64
		if err := rows.Scan(
			&listing.CampaignID,
			&listing.Title,
			&listing.Description,
			&listing.RecommendedParticipantsMin,
			&listing.RecommendedParticipantsMax,
			&difficultyTier,
			&listing.ExpectedDurationLabel,
			&system,
			&gmMode,
			&intent,
			&listing.Level,
			&listing.CharacterCount,
			&listing.Storyline,
			&tagsJSON,
			&createdAt,
			&updatedAt,
		); err != nil {
			return storage.CampaignListingPage{}, fmt.Errorf("list campaign listings: %w", err)
		}
		listing.DifficultyTier = listingv1.CampaignDifficultyTier(difficultyTier)
		listing.System = commonv1.GameSystem(system)
		listing.GmMode = listingv1.CampaignListingGmMode(gmMode)
		listing.Intent = listingv1.CampaignListingIntent(intent)
		listing.Tags = tagsFromJSON(tagsJSON)
		listing.CreatedAt = fromMillis(createdAt)
		listing.UpdatedAt = fromMillis(updatedAt)
		page.Listings = append(page.Listings, listing)
	}
	if err := rows.Err(); err != nil {
		return storage.CampaignListingPage{}, fmt.Errorf("list campaign listings: %w", err)
	}
	if len(page.Listings) > pageSize {
		page.NextPageToken = page.Listings[pageSize-1].CampaignID
		page.Listings = page.Listings[:pageSize]
	}

	return page, nil
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

func isCampaignListingUniqueViolation(err error) bool {
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
		strings.Contains(message, "campaign_listings.campaign_id")
}

var _ storage.CampaignListingStore = (*Store)(nil)
