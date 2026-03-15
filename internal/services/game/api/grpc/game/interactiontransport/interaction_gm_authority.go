package interactiontransport

import (
	"fmt"
	"sort"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func defaultGMAuthorityParticipant(campaignRecord storage.CampaignRecord, participants []storage.ParticipantRecord) (storage.ParticipantRecord, error) {
	desiredController := participant.ControllerHuman
	if campaignRecord.GmMode == campaign.GmModeAI {
		desiredController = participant.ControllerAI
	}
	candidates := make([]storage.ParticipantRecord, 0, len(participants))
	for _, record := range participants {
		if record.Role != participant.RoleGM || record.Controller != desiredController {
			continue
		}
		candidates = append(candidates, record)
	}
	if len(candidates) == 0 {
		return storage.ParticipantRecord{}, fmt.Errorf("no matching gm participant found for controller %s", desiredController)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if desiredController == participant.ControllerHuman {
			iOwner := candidates[i].CampaignAccess == participant.CampaignAccessOwner
			jOwner := candidates[j].CampaignAccess == participant.CampaignAccessOwner
			if iOwner != jOwner {
				return iOwner
			}
		}
		return strings.TrimSpace(candidates[i].ID) < strings.TrimSpace(candidates[j].ID)
	})
	return candidates[0], nil
}

func findCampaignParticipant(participants []storage.ParticipantRecord, participantID string) (storage.ParticipantRecord, bool) {
	participantID = strings.TrimSpace(participantID)
	for _, record := range participants {
		if strings.TrimSpace(record.ID) == participantID {
			return record, true
		}
	}
	return storage.ParticipantRecord{}, false
}

func aiTurnToken(sessionID, ownerParticipantID, sourceEventType, sourceSceneID, sourcePhaseID string) string {
	return strings.Join([]string{
		strings.TrimSpace(sessionID),
		strings.TrimSpace(ownerParticipantID),
		strings.TrimSpace(sourceEventType),
		strings.TrimSpace(sourceSceneID),
		strings.TrimSpace(sourcePhaseID),
	}, "|")
}
