package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

// Campaign projection methods.
func (s *Store) Put(ctx context.Context, c storage.CampaignRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(c.ID) == "" {
		return fmt.Errorf("campaign id is required")
	}

	completedAt := toNullMillis(c.CompletedAt)
	archivedAt := toNullMillis(c.ArchivedAt)

	return s.q.PutCampaign(ctx, db.PutCampaignParams{
		ID:               c.ID,
		Name:             c.Name,
		Locale:           platformi18n.LocaleString(c.Locale),
		GameSystem:       gameSystemToString(c.System),
		Status:           campaignStatusToString(c.Status),
		GmMode:           gmModeToString(c.GmMode),
		Intent:           campaignIntentToString(c.Intent),
		AccessPolicy:     campaignAccessPolicyToString(c.AccessPolicy),
		ParticipantCount: int64(c.ParticipantCount),
		CharacterCount:   int64(c.CharacterCount),
		ThemePrompt:      c.ThemePrompt,
		CoverAssetID:     c.CoverAssetID,
		CoverSetID:       c.CoverSetID,
		CreatedAt:        toMillis(c.CreatedAt),
		UpdatedAt:        toMillis(c.UpdatedAt),
		CompletedAt:      completedAt,
		ArchivedAt:       archivedAt,
	})
}

// Get fetches a campaign record by ID.
func (s *Store) Get(ctx context.Context, id string) (storage.CampaignRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.CampaignRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.CampaignRecord{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.CampaignRecord{}, fmt.Errorf("campaign id is required")
	}

	row, err := s.q.GetCampaign(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.CampaignRecord{}, storage.ErrNotFound
		}
		return storage.CampaignRecord{}, fmt.Errorf("get campaign: %w", err)
	}

	return dbGetCampaignRowToDomain(row)
}

// List returns a page of campaign records ordered by storage key.
func (s *Store) List(ctx context.Context, pageSize int, pageToken string) (storage.CampaignPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.CampaignPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.CampaignPage{}, fmt.Errorf("storage is not configured")
	}
	if pageSize <= 0 {
		return storage.CampaignPage{}, fmt.Errorf("page size must be greater than zero")
	}

	page := storage.CampaignPage{
		Campaigns: make([]storage.CampaignRecord, 0, pageSize),
	}

	if pageToken == "" {
		rows, err := s.q.ListAllCampaigns(ctx, int64(pageSize+1))
		if err != nil {
			return storage.CampaignPage{}, fmt.Errorf("list campaigns: %w", err)
		}
		for i, row := range rows {
			if i >= pageSize {
				page.NextPageToken = rows[pageSize-1].ID
				break
			}
			c, err := dbListAllCampaignsRowToDomain(row)
			if err != nil {
				return storage.CampaignPage{}, err
			}
			page.Campaigns = append(page.Campaigns, c)
		}
	} else {
		rows, err := s.q.ListCampaigns(ctx, db.ListCampaignsParams{
			ID:    pageToken,
			Limit: int64(pageSize + 1),
		})
		if err != nil {
			return storage.CampaignPage{}, fmt.Errorf("list campaigns: %w", err)
		}
		for i, row := range rows {
			if i >= pageSize {
				page.NextPageToken = rows[pageSize-1].ID
				break
			}
			c, err := dbListCampaignsRowToDomain(row)
			if err != nil {
				return storage.CampaignPage{}, err
			}
			page.Campaigns = append(page.Campaigns, c)
		}
	}

	return page, nil
}
