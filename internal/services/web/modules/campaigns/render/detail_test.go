package render

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestFragmentRendersCharacterDetailState(t *testing.T) {
	t.Parallel()

	view := CharacterDetailPageView{
		CampaignDetailBaseView: CampaignDetailBaseView{
			CampaignID:       "camp-1",
			Name:             "Skyline",
			ActionsLocked:    true,
			CanManageInvites: true,
		},
		CharacterID: "char-1",
		Character: CharacterView{
			ID:                      "char-1",
			Name:                    "Mira",
			Controller:              "human",
			ControllerParticipantID: "p-1",
			Aliases:                 []string{"Starling"},
			CanEdit:                 true,
			Daggerheart: &CharacterDaggerheartSummaryView{
				Level:         2,
				ClassName:     "Rogue",
				SubclassName:  "Night",
				AncestryName:  "Human",
				CommunityName: "Warden",
			},
		},
		CharacterControl: CharacterControlView{
			CurrentParticipantName: "Rook",
			CanManageControl:       true,
			Options:                []CharacterControlOptionView{{ParticipantID: "p-1", Label: "Rook", Selected: true}},
		},
		CharacterCreationEnabled: true,
		CharacterCreation: CampaignCharacterCreationView{
			Ready:    true,
			NextStep: 9,
		},
	}

	var buf bytes.Buffer
	if err := CharacterDetailFragment(view, nil).Render(context.Background(), &buf); err != nil {
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
