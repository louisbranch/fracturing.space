package readiness

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

func TestEvaluateSessionStart_MissingGMRejected(t *testing.T) {
	rejection := EvaluateSessionStart(aggregate.State{
		Participants: map[ids.ParticipantID]participant.State{
			"player-1": {
				ParticipantID: "player-1",
				Joined:        true,
				Role:          participant.RolePlayer,
			},
		},
	}, nil)

	if rejection == nil {
		t.Fatal("expected rejection")
	}
	if rejection.Code != RejectionCodeSessionReadinessGMRequired {
		t.Fatalf("rejection code = %s, want %s", rejection.Code, RejectionCodeSessionReadinessGMRequired)
	}
}

func TestEvaluateSessionStart_MissingPlayerRejected(t *testing.T) {
	rejection := EvaluateSessionStart(aggregate.State{
		Participants: map[ids.ParticipantID]participant.State{
			"gm-1": {
				ParticipantID: "gm-1",
				Joined:        true,
				Role:          participant.RoleGM,
			},
		},
	}, nil)

	if rejection == nil {
		t.Fatal("expected rejection")
	}
	if rejection.Code != RejectionCodeSessionReadinessPlayerRequired {
		t.Fatalf("rejection code = %s, want %s", rejection.Code, RejectionCodeSessionReadinessPlayerRequired)
	}
}

func TestEvaluateSessionStart_UnownedCharacterAcceptedWhenSystemReady(t *testing.T) {
	rejection := EvaluateSessionStart(aggregate.State{
		Participants: map[ids.ParticipantID]participant.State{
			"gm-1": {
				ParticipantID: "gm-1",
				Joined:        true,
				Role:          participant.RoleGM,
			},
			"player-1": {
				ParticipantID: "player-1",
				Joined:        true,
				Role:          participant.RolePlayer,
			},
		},
		Characters: map[ids.CharacterID]character.State{
			"char-1": {
				CharacterID: "char-1",
				Created:     true,
				Name:        "Aria",
			},
		},
	}, nil)

	if rejection != nil {
		t.Fatalf("rejection = %#v, want nil", rejection)
	}
}

func TestEvaluateSessionStart_SystemReadinessRejected(t *testing.T) {
	rejection := EvaluateSessionStart(aggregate.State{
		Participants: map[ids.ParticipantID]participant.State{
			"gm-1": {
				ParticipantID: "gm-1",
				Joined:        true,
				Role:          participant.RoleGM,
			},
			"player-1": {
				ParticipantID: "player-1",
				Joined:        true,
				Role:          participant.RolePlayer,
			},
		},
		Characters: map[ids.CharacterID]character.State{
			"char-1": {
				CharacterID:        "char-1",
				Created:            true,
				Name:               "Aria",
				OwnerParticipantID: "player-1",
			},
		},
	}, func(string) (bool, string) {
		return false, "class is required"
	})

	if rejection == nil {
		t.Fatal("expected rejection")
	}
	if rejection.Code != RejectionCodeSessionReadinessCharacterSystemRequired {
		t.Fatalf("rejection code = %s, want %s", rejection.Code, RejectionCodeSessionReadinessCharacterSystemRequired)
	}
	if !strings.Contains(rejection.Message, "Aria") {
		t.Fatalf("rejection message = %q, want character name", rejection.Message)
	}
	if strings.Contains(rejection.Message, "char-1") {
		t.Fatalf("rejection message = %q, did not expect character id when name is present", rejection.Message)
	}
}

func TestEvaluateSessionStart_ReadyCampaignAccepted(t *testing.T) {
	rejection := EvaluateSessionStart(aggregate.State{
		Participants: map[ids.ParticipantID]participant.State{
			"gm-1": {
				ParticipantID: "gm-1",
				Joined:        true,
				Role:          participant.RoleGM,
			},
			"player-1": {
				ParticipantID: "player-1",
				Joined:        true,
				Role:          participant.RolePlayer,
			},
		},
		Characters: map[ids.CharacterID]character.State{
			"char-1": {
				CharacterID:        "char-1",
				Created:            true,
				OwnerParticipantID: "player-1",
			},
			"char-2": {
				CharacterID:        "char-2",
				Created:            true,
				OwnerParticipantID: "gm-1",
			},
		},
	}, nil)

	if rejection != nil {
		t.Fatalf("rejection = %#v, want nil", rejection)
	}
}

func TestEvaluateSessionStart_AIGMModeRequiresBoundAgent(t *testing.T) {
	rejection := EvaluateSessionStart(aggregate.State{
		Campaign: campaign.State{
			GmMode: campaign.GmModeAI,
		},
	}, nil)
	if rejection == nil {
		t.Fatal("expected rejection")
	}
	if rejection.Code != RejectionCodeSessionReadinessAIAgentRequired {
		t.Fatalf("rejection code = %s, want %s", rejection.Code, RejectionCodeSessionReadinessAIAgentRequired)
	}
}

func TestEvaluateSessionStart_AIGMModeWithBoundAgentAccepted(t *testing.T) {
	rejection := EvaluateSessionStart(aggregate.State{
		Campaign: campaign.State{
			GmMode:    campaign.GmModeHybrid,
			AIAgentID: "agent-1",
		},
		Participants: map[ids.ParticipantID]participant.State{
			"gm-ai-1": {
				ParticipantID: "gm-ai-1",
				Joined:        true,
				Role:          participant.RoleGM,
				Controller:    participant.ControllerAI,
			},
			"player-1": {
				ParticipantID: "player-1",
				Joined:        true,
				Role:          participant.RolePlayer,
			},
		},
		Characters: map[ids.CharacterID]character.State{
			"char-1": {
				CharacterID:        "char-1",
				Created:            true,
				OwnerParticipantID: "player-1",
			},
		},
	}, nil)
	if rejection != nil {
		t.Fatalf("rejection = %#v, want nil", rejection)
	}
}

func TestEvaluateSessionStart_AIGMModeWithoutAIGMParticipantRejected(t *testing.T) {
	rejection := EvaluateSessionStart(aggregate.State{
		Campaign: campaign.State{
			GmMode:    campaign.GmModeAI,
			AIAgentID: "agent-1",
		},
		Participants: map[ids.ParticipantID]participant.State{
			"gm-human-1": {
				ParticipantID: "gm-human-1",
				Joined:        true,
				Role:          participant.RoleGM,
				Controller:    participant.ControllerHuman,
			},
			"player-1": {
				ParticipantID: "player-1",
				Joined:        true,
				Role:          participant.RolePlayer,
			},
		},
		Characters: map[ids.CharacterID]character.State{
			"char-1": {
				CharacterID:        "char-1",
				Created:            true,
				OwnerParticipantID: "player-1",
			},
		},
	}, nil)
	if rejection == nil {
		t.Fatal("expected rejection")
	}
	if rejection.Code != RejectionCodeSessionReadinessAIGMParticipantRequired {
		t.Fatalf("rejection code = %s, want %s", rejection.Code, RejectionCodeSessionReadinessAIGMParticipantRequired)
	}
}

func TestIsAIGMMode(t *testing.T) {
	tests := []struct {
		name string
		mode campaign.GmMode
		want bool
	}{
		{name: "ai", mode: campaign.GmModeAI, want: true},
		{name: "hybrid", mode: campaign.GmModeHybrid, want: true},
		{name: "human", mode: campaign.GmModeHuman, want: false},
		{name: "unspecified", mode: campaign.GmModeUnspecified, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAIGMMode(tt.mode)
			if got != tt.want {
				t.Fatalf("isAIGMMode(%q) = %v, want %v", tt.mode, got, tt.want)
			}
		})
	}
}
