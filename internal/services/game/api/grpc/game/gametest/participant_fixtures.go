package gametest

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func ParticipantRecordWithAccess(campaignID, participantID string, access participant.CampaignAccess) storage.ParticipantRecord {
	return storage.ParticipantRecord{
		ID:             participantID,
		CampaignID:     campaignID,
		CampaignAccess: access,
	}
}

func OwnerParticipantRecord(campaignID, participantID string) storage.ParticipantRecord {
	return ParticipantRecordWithAccess(campaignID, participantID, participant.CampaignAccessOwner)
}

func ManagerParticipantRecord(campaignID, participantID string) storage.ParticipantRecord {
	return ParticipantRecordWithAccess(campaignID, participantID, participant.CampaignAccessManager)
}

func MemberParticipantRecord(campaignID, participantID string) storage.ParticipantRecord {
	return ParticipantRecordWithAccess(campaignID, participantID, participant.CampaignAccessMember)
}

func RoleMemberParticipantRecord(campaignID, participantID string, role participant.Role) storage.ParticipantRecord {
	record := MemberParticipantRecord(campaignID, participantID)
	record.Role = role
	return record
}

func NamedRoleMemberParticipantRecord(
	campaignID, participantID, name string,
	role participant.Role,
) storage.ParticipantRecord {
	record := RoleMemberParticipantRecord(campaignID, participantID, role)
	record.Name = name
	return record
}

func UserParticipantRecord(campaignID, participantID, userID, name string) storage.ParticipantRecord {
	return storage.ParticipantRecord{
		ID:         participantID,
		CampaignID: campaignID,
		UserID:     userID,
		Name:       name,
	}
}

func MemberUserParticipantRecord(campaignID, participantID, userID, name string) storage.ParticipantRecord {
	record := UserParticipantRecord(campaignID, participantID, userID, name)
	record.CampaignAccess = participant.CampaignAccessMember
	return record
}
