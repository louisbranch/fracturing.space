package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

// Character methods.

// PutCharacter persists a character record.
func (s *Store) PutCharacter(ctx context.Context, c storage.CharacterRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(c.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(c.ID) == "" {
		return fmt.Errorf("character id is required")
	}

	return s.q.PutCharacter(ctx, db.PutCharacterParams{
		CampaignID:              c.CampaignID,
		ID:                      c.ID,
		ControllerParticipantID: toNullString(c.ParticipantID),
		Name:                    c.Name,
		Kind:                    characterKindToString(c.Kind),
		Notes:                   c.Notes,
		AvatarSetID:             c.AvatarSetID,
		AvatarAssetID:           c.AvatarAssetID,
		CreatedAt:               toMillis(c.CreatedAt),
		UpdatedAt:               toMillis(c.UpdatedAt),
	})
}

// DeleteCharacter deletes a character record by IDs.
func (s *Store) DeleteCharacter(ctx context.Context, campaignID, characterID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(characterID) == "" {
		return fmt.Errorf("character id is required")
	}

	return s.q.DeleteCharacter(ctx, db.DeleteCharacterParams{
		CampaignID: campaignID,
		ID:         characterID,
	})
}

// GetCharacter fetches a character record by IDs.
func (s *Store) GetCharacter(ctx context.Context, campaignID, characterID string) (storage.CharacterRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.CharacterRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.CharacterRecord{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.CharacterRecord{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(characterID) == "" {
		return storage.CharacterRecord{}, fmt.Errorf("character id is required")
	}

	row, err := s.q.GetCharacter(ctx, db.GetCharacterParams{
		CampaignID: campaignID,
		ID:         characterID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.CharacterRecord{}, storage.ErrNotFound
		}
		return storage.CharacterRecord{}, fmt.Errorf("get character: %w", err)
	}

	return dbCharacterToDomain(row)
}

// ListCharacters returns a page of character records.
func (s *Store) ListCharacters(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.CharacterPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.CharacterPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.CharacterPage{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.CharacterPage{}, fmt.Errorf("campaign id is required")
	}
	if pageSize <= 0 {
		return storage.CharacterPage{}, fmt.Errorf("page size must be greater than zero")
	}

	var rows []db.Character
	var err error

	if pageToken == "" {
		rows, err = s.q.ListCharactersByCampaignPagedFirst(ctx, db.ListCharactersByCampaignPagedFirstParams{
			CampaignID: campaignID,
			Limit:      int64(pageSize + 1),
		})
	} else {
		rows, err = s.q.ListCharactersByCampaignPaged(ctx, db.ListCharactersByCampaignPagedParams{
			CampaignID: campaignID,
			ID:         pageToken,
			Limit:      int64(pageSize + 1),
		})
	}
	if err != nil {
		return storage.CharacterPage{}, fmt.Errorf("list characters: %w", err)
	}

	page := storage.CharacterPage{
		Characters: make([]storage.CharacterRecord, 0, pageSize),
	}

	for i, row := range rows {
		if i >= pageSize {
			page.NextPageToken = rows[pageSize-1].ID
			break
		}
		c, err := dbCharacterToDomain(row)
		if err != nil {
			return storage.CharacterPage{}, err
		}
		page.Characters = append(page.Characters, c)
	}

	return page, nil
}

// CountCharacters returns the number of characters for a campaign.
func (s *Store) CountCharacters(ctx context.Context, campaignID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if s == nil || s.sqlDB == nil {
		return 0, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return 0, fmt.Errorf("campaign id is required")
	}
	var count int64
	err := s.sqlDB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM characters WHERE campaign_id = ?", campaignID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count characters: %w", err)
	}
	return int(count), nil
}
