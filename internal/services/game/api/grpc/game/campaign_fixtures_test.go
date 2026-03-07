package game

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func campaignRecordWithStatus(id string, status campaign.Status) storage.CampaignRecord {
	return storage.CampaignRecord{
		ID:     id,
		Status: status,
	}
}

func draftCampaignRecord(id string) storage.CampaignRecord {
	return campaignRecordWithStatus(id, campaign.StatusDraft)
}

func activeCampaignRecord(id string) storage.CampaignRecord {
	return campaignRecordWithStatus(id, campaign.StatusActive)
}

func archivedCampaignRecord(id string) storage.CampaignRecord {
	return campaignRecordWithStatus(id, campaign.StatusArchived)
}

func completedCampaignRecord(id string) storage.CampaignRecord {
	return campaignRecordWithStatus(id, campaign.StatusCompleted)
}

func activeCampaignRecordWithCharacterCount(id string, count int) storage.CampaignRecord {
	record := activeCampaignRecord(id)
	record.CharacterCount = count
	return record
}

func activeCampaignRecordWithParticipantCount(id string, count int) storage.CampaignRecord {
	record := activeCampaignRecord(id)
	record.ParticipantCount = count
	return record
}

func daggerheartCampaignRecord(id, name string, status campaign.Status, gmMode campaign.GmMode) storage.CampaignRecord {
	return storage.CampaignRecord{
		ID:     id,
		Name:   name,
		System: bridge.SystemIDDaggerheart,
		Status: status,
		GmMode: gmMode,
	}
}

func daggerheartCampaignRecordWithCreatedAt(
	id, name string,
	status campaign.Status,
	gmMode campaign.GmMode,
	createdAt time.Time,
) storage.CampaignRecord {
	record := daggerheartCampaignRecord(id, name, status, gmMode)
	record.CreatedAt = createdAt
	return record
}

func testCampaignRecordWithStatus(status campaign.Status) storage.CampaignRecord {
	return daggerheartCampaignRecord("c1", "Test Campaign", status, campaign.GmModeHuman)
}

func testCampaignRecordWithStatusAndCreatedAt(status campaign.Status, createdAt time.Time) storage.CampaignRecord {
	record := testCampaignRecordWithStatus(status)
	record.CreatedAt = createdAt
	return record
}

func testArchivedCampaignRecord(archivedAt time.Time) storage.CampaignRecord {
	record := testCampaignRecordWithStatus(campaign.StatusArchived)
	record.ArchivedAt = &archivedAt
	return record
}
