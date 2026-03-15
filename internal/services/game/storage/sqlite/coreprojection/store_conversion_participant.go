package coreprojection

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

type participantRowData struct {
	CampaignID     string
	ID             string
	UserID         string
	DisplayName    string
	Role           string
	Controller     string
	CampaignAccess string
	AvatarSetID    string
	AvatarAssetID  string
	Pronouns       string
	CreatedAt      int64
	UpdatedAt      int64
}

func participantRowDataToDomain(row participantRowData) (storage.ParticipantRecord, error) {
	return storage.ParticipantRecord{
		ID:             row.ID,
		CampaignID:     row.CampaignID,
		UserID:         row.UserID,
		Name:           row.DisplayName,
		Role:           enumFromStorage(row.Role, participant.NormalizeRole),
		Controller:     enumFromStorage(row.Controller, participant.NormalizeController),
		CampaignAccess: enumFromStorage(row.CampaignAccess, participant.NormalizeCampaignAccess),
		AvatarSetID:    row.AvatarSetID,
		AvatarAssetID:  row.AvatarAssetID,
		Pronouns:       row.Pronouns,
		CreatedAt:      fromMillis(row.CreatedAt),
		UpdatedAt:      fromMillis(row.UpdatedAt),
	}, nil
}

func dbGetParticipantRowToDomain(row db.GetParticipantRow) (storage.ParticipantRecord, error) {
	return participantRowDataToDomain(participantRowData{
		CampaignID:     row.CampaignID,
		ID:             row.ID,
		UserID:         row.UserID,
		DisplayName:    row.DisplayName,
		Role:           row.Role,
		Controller:     row.Controller,
		CampaignAccess: row.CampaignAccess,
		AvatarSetID:    row.AvatarSetID,
		AvatarAssetID:  row.AvatarAssetID,
		Pronouns:       row.Pronouns,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	})
}

func dbListParticipantsByCampaignRowToDomain(row db.ListParticipantsByCampaignRow) (storage.ParticipantRecord, error) {
	return participantRowDataToDomain(participantRowData{
		CampaignID:     row.CampaignID,
		ID:             row.ID,
		UserID:         row.UserID,
		DisplayName:    row.DisplayName,
		Role:           row.Role,
		Controller:     row.Controller,
		CampaignAccess: row.CampaignAccess,
		AvatarSetID:    row.AvatarSetID,
		AvatarAssetID:  row.AvatarAssetID,
		Pronouns:       row.Pronouns,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	})
}

func dbListParticipantsByCampaignPagedFirstRowToDomain(row db.ListParticipantsByCampaignPagedFirstRow) (storage.ParticipantRecord, error) {
	return participantRowDataToDomain(participantRowData{
		CampaignID:     row.CampaignID,
		ID:             row.ID,
		UserID:         row.UserID,
		DisplayName:    row.DisplayName,
		Role:           row.Role,
		Controller:     row.Controller,
		CampaignAccess: row.CampaignAccess,
		AvatarSetID:    row.AvatarSetID,
		AvatarAssetID:  row.AvatarAssetID,
		Pronouns:       row.Pronouns,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	})
}

func dbListParticipantsByCampaignPagedRowToDomain(row db.ListParticipantsByCampaignPagedRow) (storage.ParticipantRecord, error) {
	return participantRowDataToDomain(participantRowData{
		CampaignID:     row.CampaignID,
		ID:             row.ID,
		UserID:         row.UserID,
		DisplayName:    row.DisplayName,
		Role:           row.Role,
		Controller:     row.Controller,
		CampaignAccess: row.CampaignAccess,
		AvatarSetID:    row.AvatarSetID,
		AvatarAssetID:  row.AvatarAssetID,
		Pronouns:       row.Pronouns,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	})
}
