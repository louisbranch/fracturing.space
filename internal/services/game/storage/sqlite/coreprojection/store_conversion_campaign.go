package coreprojection

import (
	"database/sql"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

// campaignRowData holds the common fields from campaign row types.
type campaignRowData struct {
	ID               string
	Name             string
	Locale           string
	GameSystem       string
	Status           string
	GmMode           string
	Intent           string
	AccessPolicy     string
	ParticipantCount int64
	CharacterCount   int64
	ThemePrompt      string
	CoverAssetID     string
	CoverSetID       string
	AIAgentID        string
	AIAuthEpoch      int64
	CreatedAt        int64
	UpdatedAt        int64
	CompletedAt      sql.NullInt64
	ArchivedAt       sql.NullInt64
}

func campaignRowDataToDomain(row campaignRowData) (storage.CampaignRecord, error) {
	c := storage.CampaignRecord{
		ID:               row.ID,
		Name:             row.Name,
		Locale:           row.Locale,
		System:           enumFromStorage(row.GameSystem, bridge.NormalizeSystemID),
		Status:           enumFromStorage(row.Status, campaign.NormalizeStatus),
		GmMode:           enumFromStorage(row.GmMode, campaign.NormalizeGmMode),
		Intent:           campaign.NormalizeIntent(row.Intent),
		AccessPolicy:     campaign.NormalizeAccessPolicy(row.AccessPolicy),
		ParticipantCount: int(row.ParticipantCount),
		CharacterCount:   int(row.CharacterCount),
		ThemePrompt:      row.ThemePrompt,
		CoverAssetID:     row.CoverAssetID,
		CoverSetID:       row.CoverSetID,
		AIAgentID:        row.AIAgentID,
		AIAuthEpoch:      uint64(row.AIAuthEpoch),
		CreatedAt:        sqliteutil.FromMillis(row.CreatedAt),
		UpdatedAt:        sqliteutil.FromMillis(row.UpdatedAt),
	}
	c.CompletedAt = sqliteutil.FromNullMillis(row.CompletedAt)
	c.ArchivedAt = sqliteutil.FromNullMillis(row.ArchivedAt)

	return c, nil
}

func dbGetCampaignRowToDomain(row db.GetCampaignRow) (storage.CampaignRecord, error) {
	return campaignRowDataToDomain(campaignRowData{
		ID:               row.ID,
		Name:             row.Name,
		Locale:           row.Locale,
		GameSystem:       row.GameSystem,
		Status:           row.Status,
		GmMode:           row.GmMode,
		Intent:           row.Intent,
		AccessPolicy:     row.AccessPolicy,
		ParticipantCount: row.ParticipantCount,
		CharacterCount:   row.CharacterCount,
		ThemePrompt:      row.ThemePrompt,
		CoverAssetID:     row.CoverAssetID,
		CoverSetID:       row.CoverSetID,
		AIAgentID:        row.AiAgentID,
		AIAuthEpoch:      row.AiAuthEpoch,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
		CompletedAt:      row.CompletedAt,
		ArchivedAt:       row.ArchivedAt,
	})
}

func dbListCampaignsRowToDomain(row db.ListCampaignsRow) (storage.CampaignRecord, error) {
	return campaignRowDataToDomain(campaignRowData{
		ID:               row.ID,
		Name:             row.Name,
		Locale:           row.Locale,
		GameSystem:       row.GameSystem,
		Status:           row.Status,
		GmMode:           row.GmMode,
		Intent:           row.Intent,
		AccessPolicy:     row.AccessPolicy,
		ParticipantCount: row.ParticipantCount,
		CharacterCount:   row.CharacterCount,
		ThemePrompt:      row.ThemePrompt,
		CoverAssetID:     row.CoverAssetID,
		CoverSetID:       row.CoverSetID,
		AIAgentID:        row.AiAgentID,
		AIAuthEpoch:      row.AiAuthEpoch,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
		CompletedAt:      row.CompletedAt,
		ArchivedAt:       row.ArchivedAt,
	})
}

func dbListAllCampaignsRowToDomain(row db.ListAllCampaignsRow) (storage.CampaignRecord, error) {
	return campaignRowDataToDomain(campaignRowData{
		ID:               row.ID,
		Name:             row.Name,
		Locale:           row.Locale,
		GameSystem:       row.GameSystem,
		Status:           row.Status,
		GmMode:           row.GmMode,
		Intent:           row.Intent,
		AccessPolicy:     row.AccessPolicy,
		ParticipantCount: row.ParticipantCount,
		CharacterCount:   row.CharacterCount,
		ThemePrompt:      row.ThemePrompt,
		CoverAssetID:     row.CoverAssetID,
		CoverSetID:       row.CoverSetID,
		AIAgentID:        row.AiAgentID,
		AIAuthEpoch:      row.AiAuthEpoch,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
		CompletedAt:      row.CompletedAt,
		ArchivedAt:       row.ArchivedAt,
	})
}
