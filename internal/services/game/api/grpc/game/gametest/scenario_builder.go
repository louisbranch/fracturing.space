package gametest

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// CampaignScenario is the built result of a CampaignScenarioBuilder.
// It holds populated fake stores ready for injection into service constructors.
type CampaignScenario struct {
	CampaignID   string
	Campaigns    *FakeCampaignStore
	Participants *FakeParticipantStore
	Characters   *FakeCharacterStore
	Sessions     *FakeSessionStore
	Events       *FakeEventStore
	Daggerheart  *FakeDaggerheartStore
}

// CampaignScenarioBuilder is a fluent test fixture builder that replaces
// imperative store setup boilerplate. It constructs a consistent set of
// fake stores with campaign, participant, character, and session state.
type CampaignScenarioBuilder struct {
	t            *testing.T
	campaignID   string
	campaignName string
	system       systems.SystemID
	gmMode       campaign.GmMode
	status       campaign.Status
	createdAt    time.Time
	participants []scenarioParticipant
	characters   []scenarioCharacter
	session      *scenarioSession
}

type scenarioParticipant struct {
	id         string
	name       string
	userID     string
	role       participant.Role
	controller participant.Controller
	access     participant.CampaignAccess
}

type scenarioCharacter struct {
	id                      string
	name                    string
	ownerParticipantID      string
	controllerParticipantID string
}

type scenarioSession struct {
	id     string
	name   string
	status session.Status
}

// NewCampaignScenario starts building a test campaign scenario with sensible defaults.
// The campaign is created with Daggerheart system, human GM mode, and active status.
func NewCampaignScenario(t *testing.T) *CampaignScenarioBuilder {
	t.Helper()
	return &CampaignScenarioBuilder{
		t:            t,
		campaignID:   "c1",
		campaignName: "Test Campaign",
		system:       systems.SystemIDDaggerheart,
		gmMode:       campaign.GmModeHuman,
		status:       campaign.StatusActive,
		createdAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

// WithCampaignID sets the campaign ID. Default is "c1".
func (b *CampaignScenarioBuilder) WithCampaignID(id string) *CampaignScenarioBuilder {
	b.campaignID = id
	return b
}

// WithCampaignName sets the campaign display name.
func (b *CampaignScenarioBuilder) WithCampaignName(name string) *CampaignScenarioBuilder {
	b.campaignName = name
	return b
}

// WithStatus sets the campaign lifecycle status.
func (b *CampaignScenarioBuilder) WithStatus(status campaign.Status) *CampaignScenarioBuilder {
	b.status = status
	return b
}

// WithGmMode sets the campaign GM mode.
func (b *CampaignScenarioBuilder) WithGmMode(mode campaign.GmMode) *CampaignScenarioBuilder {
	b.gmMode = mode
	return b
}

// WithParticipant adds a participant with the given ID and role.
func (b *CampaignScenarioBuilder) WithParticipant(id string, role participant.Role) *CampaignScenarioBuilder {
	access := participant.CampaignAccessMember
	if role == participant.RoleGM {
		access = participant.CampaignAccessManager
	}
	b.participants = append(b.participants, scenarioParticipant{
		id:         id,
		name:       "Participant " + id,
		role:       role,
		controller: participant.ControllerHuman,
		access:     access,
	})
	return b
}

// WithUserParticipant adds a participant with an associated user ID.
func (b *CampaignScenarioBuilder) WithUserParticipant(id, userID string, role participant.Role) *CampaignScenarioBuilder {
	access := participant.CampaignAccessMember
	if role == participant.RoleGM {
		access = participant.CampaignAccessManager
	}
	b.participants = append(b.participants, scenarioParticipant{
		id:         id,
		name:       "Participant " + id,
		userID:     userID,
		role:       role,
		controller: participant.ControllerHuman,
		access:     access,
	})
	return b
}

// WithCharacter adds a character with the given ID owned and controlled by ownerParticipantID.
func (b *CampaignScenarioBuilder) WithCharacter(id string, ownerParticipantID string) *CampaignScenarioBuilder {
	b.characters = append(b.characters, scenarioCharacter{
		id:                      id,
		name:                    "Character " + id,
		ownerParticipantID:      ownerParticipantID,
		controllerParticipantID: ownerParticipantID,
	})
	return b
}

// WithActiveSession adds an active session to the scenario.
func (b *CampaignScenarioBuilder) WithActiveSession(id string) *CampaignScenarioBuilder {
	b.session = &scenarioSession{
		id:     id,
		name:   "Session " + id,
		status: session.StatusActive,
	}
	return b
}

// Build constructs the CampaignScenario with populated fake stores.
func (b *CampaignScenarioBuilder) Build() CampaignScenario {
	b.t.Helper()

	campaigns := NewFakeCampaignStore()
	participants := NewFakeParticipantStore()
	characters := NewFakeCharacterStore()
	sessions := NewFakeSessionStore()
	events := NewFakeEventStore()
	daggerheart := NewFakeDaggerheartStore()

	// Seed campaign.
	campaigns.Campaigns[b.campaignID] = storage.CampaignRecord{
		ID:               b.campaignID,
		Name:             b.campaignName,
		System:           b.system,
		Status:           b.status,
		GmMode:           b.gmMode,
		CreatedAt:        b.createdAt,
		ParticipantCount: len(b.participants),
		CharacterCount:   len(b.characters),
	}

	// Seed participants.
	for _, p := range b.participants {
		if participants.Participants[b.campaignID] == nil {
			participants.Participants[b.campaignID] = make(map[string]storage.ParticipantRecord)
		}
		participants.Participants[b.campaignID][p.id] = storage.ParticipantRecord{
			ID:             p.id,
			CampaignID:     b.campaignID,
			Name:           p.name,
			UserID:         p.userID,
			Role:           p.role,
			Controller:     p.controller,
			CampaignAccess: p.access,
		}
	}

	// Seed characters.
	for _, ch := range b.characters {
		if characters.Characters[b.campaignID] == nil {
			characters.Characters[b.campaignID] = make(map[string]storage.CharacterRecord)
		}
		characters.Characters[b.campaignID][ch.id] = storage.CharacterRecord{
			ID:                 ch.id,
			CampaignID:         b.campaignID,
			Name:               ch.name,
			OwnerParticipantID: ch.ownerParticipantID,
			ParticipantID:      ch.controllerParticipantID,
		}
	}

	// Seed session.
	if b.session != nil {
		if sessions.Sessions[b.campaignID] == nil {
			sessions.Sessions[b.campaignID] = make(map[string]storage.SessionRecord)
		}
		sessions.Sessions[b.campaignID][b.session.id] = storage.SessionRecord{
			ID:         b.session.id,
			CampaignID: b.campaignID,
			Name:       b.session.name,
			Status:     b.session.status,
			StartedAt:  b.createdAt,
		}
		if b.session.status == session.StatusActive {
			sessions.ActiveSession[b.campaignID] = b.session.id
		}
	}

	return CampaignScenario{
		CampaignID:   b.campaignID,
		Campaigns:    campaigns,
		Participants: participants,
		Characters:   characters,
		Sessions:     sessions,
		Events:       events,
		Daggerheart:  daggerheart,
	}
}
