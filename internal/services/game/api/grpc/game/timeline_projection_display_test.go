package game

import (
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestCampaignProjectionDisplay(t *testing.T) {
	tests := []struct {
		name         string
		record       storage.CampaignRecord
		wantSubtitle string
		wantStatus   string
	}{
		{
			name:         "daggerheart active",
			record:       storage.CampaignRecord{Name: "Riverfall", System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, Status: campaign.StatusActive},
			wantSubtitle: "DAGGERHEART",
			wantStatus:   "ACTIVE",
		},
		{
			name:         "unknown system and status",
			record:       storage.CampaignRecord{Name: "Riverfall"},
			wantSubtitle: "",
			wantStatus:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			display := campaignProjectionDisplay(tc.record)
			if display.GetTitle() != tc.record.Name {
				t.Fatalf("title = %q, want %q", display.GetTitle(), tc.record.Name)
			}
			if display.GetSubtitle() != tc.wantSubtitle {
				t.Fatalf("subtitle = %q, want %q", display.GetSubtitle(), tc.wantSubtitle)
			}
			if display.GetStatus() != tc.wantStatus {
				t.Fatalf("status = %q, want %q", display.GetStatus(), tc.wantStatus)
			}
		})
	}
}

func TestParticipantProjectionDisplay(t *testing.T) {
	display := participantProjectionDisplay(storage.ParticipantRecord{
		Name:       "Ada",
		Role:       participant.RoleGM,
		Controller: participant.ControllerAI,
	})
	if display.GetTitle() != "Ada" {
		t.Fatalf("title = %q, want %q", display.GetTitle(), "Ada")
	}
	if display.GetSubtitle() != "GM" {
		t.Fatalf("subtitle = %q, want %q", display.GetSubtitle(), "GM")
	}
	if display.GetStatus() != "AI" {
		t.Fatalf("status = %q, want %q", display.GetStatus(), "AI")
	}

	display = participantProjectionDisplay(storage.ParticipantRecord{
		Name:       "Bob",
		Role:       participant.RolePlayer,
		Controller: participant.ControllerHuman,
	})
	if display.GetSubtitle() != "PLAYER" {
		t.Fatalf("subtitle = %q, want %q", display.GetSubtitle(), "PLAYER")
	}
	if display.GetStatus() != "HUMAN" {
		t.Fatalf("status = %q, want %q", display.GetStatus(), "HUMAN")
	}
}

func TestCharacterProjectionDisplay(t *testing.T) {
	display := characterProjectionDisplay(storage.CharacterRecord{Name: "Frodo", Kind: character.KindPC})
	if display.GetTitle() != "Frodo" {
		t.Fatalf("title = %q, want %q", display.GetTitle(), "Frodo")
	}
	if display.GetSubtitle() != "PC" {
		t.Fatalf("subtitle = %q, want %q", display.GetSubtitle(), "PC")
	}

	display = characterProjectionDisplay(storage.CharacterRecord{Name: "Goblin", Kind: character.KindNPC})
	if display.GetSubtitle() != "NPC" {
		t.Fatalf("subtitle = %q, want %q", display.GetSubtitle(), "NPC")
	}
}

func TestSessionProjectionDisplay(t *testing.T) {
	display := sessionProjectionDisplay(storage.SessionRecord{Name: "Session 1", Status: session.StatusActive})
	if display.GetTitle() != "Session 1" {
		t.Fatalf("title = %q, want %q", display.GetTitle(), "Session 1")
	}
	if display.GetStatus() != "ACTIVE" {
		t.Fatalf("status = %q, want %q", display.GetStatus(), "ACTIVE")
	}

	display = sessionProjectionDisplay(storage.SessionRecord{Name: "Session 2", Status: session.StatusEnded})
	if display.GetStatus() != "ENDED" {
		t.Fatalf("status = %q, want %q", display.GetStatus(), "ENDED")
	}
}
