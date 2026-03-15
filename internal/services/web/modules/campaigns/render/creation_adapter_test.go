package render

import (
	"testing"

	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
)

func TestNewCharacterCreationPageViewAdaptsWorkflowPageData(t *testing.T) {
	t.Parallel()

	page := campaignworkflow.PageData{
		CharacterName: "Aria",
		Creation: campaignworkflow.CharacterCreationView{
			NextStep: 2,
			ClassID:  "class-1",
			Classes: []campaignworkflow.CreationClassView{
				{ID: "class-1", Name: "Bard"},
			},
		},
	}

	view := NewCharacterCreationPageView("campaign-1", "character-1", page)
	if view.CampaignID != "campaign-1" || view.CharacterID != "character-1" {
		t.Fatalf("page ids = %#v, want campaign-1/character-1", view)
	}
	if view.Creation.NextStep != 2 || view.Creation.ClassID != "class-1" {
		t.Fatalf("creation = %#v, want adapted workflow view", view.Creation)
	}
	if len(view.Creation.Classes) != 1 || view.Creation.Classes[0].Name != "Bard" {
		t.Fatalf("creation classes = %#v, want Bard option", view.Creation.Classes)
	}
}
