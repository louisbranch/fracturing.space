package game

import (
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// campaignProjectionDisplay builds a display summary for campaign projections.
func campaignProjectionDisplay(entry storage.CampaignRecord) *campaignv1.ProjectionDisplay {
	display := &campaignv1.ProjectionDisplay{
		Title: entry.Name,
	}
	switch systemIDFromCampaignRecord(entry) {
	case bridge.SystemIDDaggerheart:
		display.Subtitle = "DAGGERHEART"
	}
	switch entry.Status {
	case campaign.StatusDraft:
		display.Status = "DRAFT"
	case campaign.StatusActive:
		display.Status = "ACTIVE"
	case campaign.StatusCompleted:
		display.Status = "COMPLETED"
	case campaign.StatusArchived:
		display.Status = "ARCHIVED"
	}
	return display
}

// participantProjectionDisplay builds a display summary for participant projections.
func participantProjectionDisplay(entry storage.ParticipantRecord) *campaignv1.ProjectionDisplay {
	display := &campaignv1.ProjectionDisplay{
		Title: entry.Name,
	}
	switch entry.Role {
	case participant.RoleGM:
		display.Subtitle = "GM"
	case participant.RolePlayer:
		display.Subtitle = "PLAYER"
	}
	switch entry.Controller {
	case participant.ControllerHuman:
		display.Status = "HUMAN"
	case participant.ControllerAI:
		display.Status = "AI"
	}
	return display
}

// characterProjectionDisplay builds a display summary for character projections.
func characterProjectionDisplay(entry storage.CharacterRecord) *campaignv1.ProjectionDisplay {
	display := &campaignv1.ProjectionDisplay{
		Title: entry.Name,
	}
	switch entry.Kind {
	case character.KindPC:
		display.Subtitle = "PC"
	case character.KindNPC:
		display.Subtitle = "NPC"
	}
	return display
}

// sessionProjectionDisplay builds a display summary for session projections.
func sessionProjectionDisplay(entry storage.SessionRecord) *campaignv1.ProjectionDisplay {
	display := &campaignv1.ProjectionDisplay{
		Title: entry.Name,
	}
	switch entry.Status {
	case session.StatusActive:
		display.Status = "ACTIVE"
	case session.StatusEnded:
		display.Status = "ENDED"
	}
	return display
}
