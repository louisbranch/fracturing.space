package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func participantRecordWithAccess(campaignID, participantID string, access participant.CampaignAccess) storage.ParticipantRecord {
	return storage.ParticipantRecord{
		ID:             participantID,
		CampaignID:     campaignID,
		CampaignAccess: access,
	}
}

func ownerParticipantRecord(campaignID, participantID string) storage.ParticipantRecord {
	return participantRecordWithAccess(campaignID, participantID, participant.CampaignAccessOwner)
}

func managerParticipantRecord(campaignID, participantID string) storage.ParticipantRecord {
	return participantRecordWithAccess(campaignID, participantID, participant.CampaignAccessManager)
}

func memberParticipantRecord(campaignID, participantID string) storage.ParticipantRecord {
	return participantRecordWithAccess(campaignID, participantID, participant.CampaignAccessMember)
}

func roleMemberParticipantRecord(campaignID, participantID string, role participant.Role) storage.ParticipantRecord {
	record := memberParticipantRecord(campaignID, participantID)
	record.Role = role
	return record
}

func namedRoleMemberParticipantRecord(
	campaignID, participantID, name string,
	role participant.Role,
) storage.ParticipantRecord {
	record := roleMemberParticipantRecord(campaignID, participantID, role)
	record.Name = name
	return record
}

func userParticipantRecord(campaignID, participantID, userID, name string) storage.ParticipantRecord {
	return storage.ParticipantRecord{
		ID:         participantID,
		CampaignID: campaignID,
		UserID:     userID,
		Name:       name,
	}
}

func memberUserParticipantRecord(campaignID, participantID, userID, name string) storage.ParticipantRecord {
	record := userParticipantRecord(campaignID, participantID, userID, name)
	record.CampaignAccess = participant.CampaignAccessMember
	return record
}
