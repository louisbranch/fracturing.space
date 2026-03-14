package render

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestFragmentRendersCharacterDetailState(t *testing.T) {
	t.Parallel()

	view := DetailView{
		Marker:        "campaign-character-detail",
		CampaignID:    "camp-1",
		CharacterID:   "char-1",
		Name:          "Skyline",
		ActionsLocked: true,
		Participants: []ParticipantView{{
			ID:             "p-1",
			Name:           "Rook",
			Role:           "player",
			CampaignAccess: "member",
			Controller:     "human",
		}},
		ParticipantEditor: ParticipantEditorView{
			ID:            "p-1",
			Name:          "Rook",
			AccessOptions: []ParticipantAccessOptionView{{Value: "member", Allowed: true}},
		},
		AIBindingEditor: AIBindingEditorView{
			Visible: true,
			Options: []AIAgentOptionView{{ID: "agent-1", Name: "Guide", Enabled: true, Selected: true}},
		},
		Characters: []CharacterView{{
			ID:                      "char-1",
			Name:                    "Mira",
			Controller:              "human",
			ControllerParticipantID: "p-1",
			Aliases:                 []string{"Starling"},
			Daggerheart: &CharacterDaggerheartSummaryView{
				Level:         2,
				ClassName:     "Rogue",
				SubclassName:  "Night",
				AncestryName:  "Human",
				CommunityName: "Warden",
			},
		}},
		CharacterControl: CharacterControlView{
			CurrentParticipantName: "Rook",
			CanManageControl:       true,
			Options:                []CharacterControlOptionView{{ParticipantID: "p-1", Label: "Rook", Selected: true}},
		},
		Sessions:         []SessionView{{ID: "s-1", Name: "Session One", Status: "active", UpdatedAt: "2026-03-09T10:00:00Z"}},
		SessionReadiness: SessionReadinessView{Ready: true, Blockers: []SessionReadinessBlockerView{{Code: "ok", Message: "ready"}}},
		Invites: []InviteView{{
			ID:              "i-1",
			ParticipantID:   "p-2",
			ParticipantName: "Scout",
			HasRecipient:    true,
			PublicURL:       "/invite/i-1",
			Status:          "pending",
		}},
		InviteSeatOptions: []InviteSeatOptionView{{
			ParticipantID: "p-2",
			Label:         "Scout",
		}},
		CharacterCreationEnabled: true,
		CharacterCreation: CampaignCharacterCreationView{
			Ready:    true,
			NextStep: 9,
		},
	}

	var buf bytes.Buffer
	if err := Fragment(view, nil).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render fragment: %v", err)
	}
	body := buf.String()
	if !strings.Contains(body, `data-campaign-character-detail-id="char-1">Mira</h2>`) {
		t.Fatalf("missing character heading: %s", body)
	}
	if !strings.Contains(body, `data-campaign-character-controller-submit-disabled="true"`) {
		t.Fatalf("missing locked control submit state: %s", body)
	}
	if !strings.Contains(body, `data-character-creation-workflow="true"`) {
		t.Fatalf("missing character creation panel: %s", body)
	}
	if !strings.Contains(body, `data-campaign-character-control-manager-card="true"`) {
		t.Fatalf("missing control manager card: %s", body)
	}
}
