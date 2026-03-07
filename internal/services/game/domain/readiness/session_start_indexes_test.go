package readiness

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

func TestActiveParticipantsByID_IsDeterministicAndFiltersInactive(t *testing.T) {
	indexed := activeParticipantsByID(aggregate.State{
		Participants: map[string]participant.State{
			"player-z": {
				ParticipantID: "player-z",
				Joined:        true,
				Role:          string(participant.RolePlayer),
			},
			"gm-a": {
				ParticipantID: "gm-a",
				Joined:        true,
				Role:          string(participant.RoleGM),
			},
			"gm-ai": {
				ParticipantID: "gm-ai",
				Joined:        true,
				Role:          string(participant.RoleGM),
				Controller:    string(participant.ControllerAI),
			},
			"player-a": {
				ParticipantID: "player-a",
				Joined:        true,
				Role:          string(participant.RolePlayer),
			},
			"ignored-left": {
				ParticipantID: "ignored-left",
				Joined:        true,
				Left:          true,
				Role:          string(participant.RolePlayer),
			},
			"ignored-invalid-role": {
				ParticipantID: "ignored-invalid-role",
				Joined:        true,
				Role:          "invalid",
			},
		},
	})

	if len(indexed.byID) != 5 {
		t.Fatalf("active participants byID len = %d, want 5", len(indexed.byID))
	}
	if strings.Join(indexed.gmIDs, ",") != "gm-a,gm-ai" {
		t.Fatalf("gm ids = %v, want [gm-a gm-ai]", indexed.gmIDs)
	}
	if strings.Join(indexed.aiGMIDs, ",") != "gm-ai" {
		t.Fatalf("ai gm ids = %v, want [gm-ai]", indexed.aiGMIDs)
	}
	if strings.Join(indexed.playerIDs, ",") != "player-a,player-z" {
		t.Fatalf("player ids = %v, want [player-a player-z]", indexed.playerIDs)
	}
}

func TestActiveCharactersByID_IsDeterministicAndFiltersInactive(t *testing.T) {
	indexed := activeCharactersByID(aggregate.State{
		Characters: map[string]character.State{
			"char-z": {
				CharacterID:   "char-z",
				Created:       true,
				ParticipantID: "player-z",
			},
			"char-a": {
				CharacterID:   "char-a",
				Created:       true,
				ParticipantID: "player-a",
			},
			"char-deleted": {
				CharacterID: "char-deleted",
				Created:     true,
				Deleted:     true,
			},
			"char-not-created": {
				CharacterID: "char-not-created",
			},
		},
	})

	if strings.Join(indexed.ids, ",") != "char-a,char-z" {
		t.Fatalf("character ids = %v, want [char-a char-z]", indexed.ids)
	}
	if indexed.byID["char-a"].ParticipantID != "player-a" {
		t.Fatalf("char-a participant = %q, want %q", indexed.byID["char-a"].ParticipantID, "player-a")
	}
	if _, ok := indexed.byID["char-deleted"]; ok {
		t.Fatal("expected deleted character to be excluded")
	}
}

func TestEvaluateSessionStart_SystemReadinessMessageFormatting(t *testing.T) {
	base := aggregate.State{
		Participants: map[string]participant.State{
			"gm-1": {
				ParticipantID: "gm-1",
				Joined:        true,
				Role:          string(participant.RoleGM),
			},
			"player-1": {
				ParticipantID: "player-1",
				Joined:        true,
				Role:          string(participant.RolePlayer),
			},
		},
		Characters: map[string]character.State{
			"char-1": {
				CharacterID:   "char-1",
				Created:       true,
				ParticipantID: "player-1",
				SystemProfile: map[string]any{"ready": false},
			},
		},
	}

	withReason := EvaluateSessionStart(base, func(map[string]any) (bool, string) {
		return false, "profile missing class"
	})
	if withReason == nil {
		t.Fatal("expected rejection")
	}
	if withReason.Code != RejectionCodeSessionReadinessCharacterSystemRequired {
		t.Fatalf("rejection code = %q, want %q", withReason.Code, RejectionCodeSessionReadinessCharacterSystemRequired)
	}
	if withReason.Message != "campaign readiness requires character char-1 to satisfy system readiness: profile missing class" {
		t.Fatalf("message = %q, want reason-suffixed message", withReason.Message)
	}

	withoutReason := EvaluateSessionStart(base, func(map[string]any) (bool, string) {
		return false, "   "
	})
	if withoutReason == nil {
		t.Fatal("expected rejection")
	}
	if withoutReason.Message != "campaign readiness requires character char-1 to satisfy system readiness" {
		t.Fatalf("message = %q, want base message without suffix", withoutReason.Message)
	}
}
