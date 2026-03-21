package gametest

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func CampaignRecordWithStatus(id string, status campaign.Status) storage.CampaignRecord {
	return storage.CampaignRecord{
		ID:     id,
		Status: status,
	}
}

func DraftCampaignRecord(id string) storage.CampaignRecord {
	return CampaignRecordWithStatus(id, campaign.StatusDraft)
}

func ActiveCampaignRecord(id string) storage.CampaignRecord {
	return CampaignRecordWithStatus(id, campaign.StatusActive)
}

func ArchivedCampaignRecord(id string) storage.CampaignRecord {
	return CampaignRecordWithStatus(id, campaign.StatusArchived)
}

func CompletedCampaignRecord(id string) storage.CampaignRecord {
	return CampaignRecordWithStatus(id, campaign.StatusCompleted)
}

func ActiveCampaignRecordWithCharacterCount(id string, count int) storage.CampaignRecord {
	record := ActiveCampaignRecord(id)
	record.CharacterCount = count
	return record
}

func ActiveCampaignRecordWithParticipantCount(id string, count int) storage.CampaignRecord {
	record := ActiveCampaignRecord(id)
	record.ParticipantCount = count
	return record
}

func DaggerheartCampaignRecord(id, name string, status campaign.Status, gmMode campaign.GmMode) storage.CampaignRecord {
	return storage.CampaignRecord{
		ID:     id,
		Name:   name,
		System: bridge.SystemIDDaggerheart,
		Status: status,
		GmMode: gmMode,
	}
}

func DaggerheartCampaignRecordWithCreatedAt(
	id, name string,
	status campaign.Status,
	gmMode campaign.GmMode,
	createdAt time.Time,
) storage.CampaignRecord {
	record := DaggerheartCampaignRecord(id, name, status, gmMode)
	record.CreatedAt = createdAt
	return record
}

func TestCampaignRecordWithStatus(status campaign.Status) storage.CampaignRecord {
	return DaggerheartCampaignRecord("c1", "Test Campaign", status, campaign.GmModeHuman)
}

func TestCampaignRecordWithStatusAndCreatedAt(status campaign.Status, createdAt time.Time) storage.CampaignRecord {
	record := TestCampaignRecordWithStatus(status)
	record.CreatedAt = createdAt
	return record
}

func TestArchivedCampaignRecord(archivedAt time.Time) storage.CampaignRecord {
	record := TestCampaignRecordWithStatus(campaign.StatusArchived)
	record.ArchivedAt = &archivedAt
	return record
}
