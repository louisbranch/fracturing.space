package readiness

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

func TestEvaluateSessionStart_MissingGMRejected(t *testing.T) {
	rejection := EvaluateSessionStart(aggregate.State{
		Participants: map[string]participant.State{
			"player-1": {
				ParticipantID: "player-1",
				Joined:        true,
				Role:          string(participant.RolePlayer),
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
		Participants: map[string]participant.State{
			"gm-1": {
				ParticipantID: "gm-1",
				Joined:        true,
				Role:          string(participant.RoleGM),
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

func TestEvaluateSessionStart_PlayerWithoutCharacterRejected(t *testing.T) {
	rejection := EvaluateSessionStart(aggregate.State{
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
		Characters: map[string]character.State{},
	}, nil)

	if rejection == nil {
		t.Fatal("expected rejection")
	}
	if rejection.Code != RejectionCodeSessionReadinessPlayerCharacterRequired {
		t.Fatalf("rejection code = %s, want %s", rejection.Code, RejectionCodeSessionReadinessPlayerCharacterRequired)
	}
}

func TestEvaluateSessionStart_CharacterWithoutControllerRejected(t *testing.T) {
	rejection := EvaluateSessionStart(aggregate.State{
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
				CharacterID: "char-1",
				Created:     true,
			},
		},
	}, nil)

	if rejection == nil {
		t.Fatal("expected rejection")
	}
	if rejection.Code != RejectionCodeSessionReadinessCharacterControllerRequired {
		t.Fatalf("rejection code = %s, want %s", rejection.Code, RejectionCodeSessionReadinessCharacterControllerRequired)
	}
}

func TestEvaluateSessionStart_SystemReadinessRejected(t *testing.T) {
	rejection := EvaluateSessionStart(aggregate.State{
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
				SystemProfile: map[string]any{
					"daggerheart": map[string]any{"class": "warrior"},
				},
			},
		},
	}, func(map[string]any) (bool, string) {
		return false, "class is required"
	})

	if rejection == nil {
		t.Fatal("expected rejection")
	}
	if rejection.Code != RejectionCodeSessionReadinessCharacterSystemRequired {
		t.Fatalf("rejection code = %s, want %s", rejection.Code, RejectionCodeSessionReadinessCharacterSystemRequired)
	}
}

func TestEvaluateSessionStart_ReadyCampaignAccepted(t *testing.T) {
	rejection := EvaluateSessionStart(aggregate.State{
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
			},
			"char-2": {
				CharacterID:   "char-2",
				Created:       true,
				ParticipantID: "gm-1",
			},
		},
	}, nil)

	if rejection != nil {
		t.Fatalf("rejection = %#v, want nil", rejection)
	}
}
