package coreprojection

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
		GameSystem:       enumToStorage(c.System),
		Status:           enumToStorage(c.Status),
		GmMode:           enumToStorage(c.GmMode),
		Intent:           enumToStorage(c.Intent),
		AccessPolicy:     enumToStorage(c.AccessPolicy),
		ParticipantCount: int64(c.ParticipantCount),
		CharacterCount:   int64(c.CharacterCount),
		ThemePrompt:      c.ThemePrompt,
		CoverAssetID:     c.CoverAssetID,
		CoverSetID:       c.CoverSetID,
		AiAgentID:        c.AIAgentID,
		AiAuthEpoch:      int64(c.AIAuthEpoch),
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
		campaigns, nextPageToken, err := mapPageRows(rows, pageSize, func(row db.ListAllCampaignsRow) string {
			return row.ID
		}, dbListAllCampaignsRowToDomain)
		if err != nil {
			return storage.CampaignPage{}, err
		}
		page.Campaigns = campaigns
		page.NextPageToken = nextPageToken
	} else {
		rows, err := s.q.ListCampaigns(ctx, db.ListCampaignsParams{
			ID:    pageToken,
			Limit: int64(pageSize + 1),
		})
		if err != nil {
			return storage.CampaignPage{}, fmt.Errorf("list campaigns: %w", err)
		}
		campaigns, nextPageToken, err := mapPageRows(rows, pageSize, func(row db.ListCampaignsRow) string {
			return row.ID
		}, dbListCampaignsRowToDomain)
		if err != nil {
			return storage.CampaignPage{}, err
		}
		page.Campaigns = campaigns
		page.NextPageToken = nextPageToken
	}

	return page, nil
}

// ListCampaignIDsByAIAgent returns campaign IDs bound to one AI agent.
func (s *Store) ListCampaignIDsByAIAgent(ctx context.Context, aiAgentID string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	aiAgentID = strings.TrimSpace(aiAgentID)
	if aiAgentID == "" {
		return nil, fmt.Errorf("ai agent id is required")
	}

	rows, err := s.q.ListCampaignIDsByAIAgent(ctx, aiAgentID)
	if err != nil {
		return nil, fmt.Errorf("list campaign ids by ai agent: %w", err)
	}
	campaignIDs := make([]string, 0, len(rows))
	for _, campaignID := range rows {
		campaignIDs = append(campaignIDs, campaignID)
	}
	return campaignIDs, nil
}
