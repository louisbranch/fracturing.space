package render

import campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"

// NewCharacterCreationView adapts the workflow-owned creation model to the
// render contract used by templates and detail views.
func NewCharacterCreationView(view campaignworkflow.CharacterCreationView) CampaignCharacterCreationView {
	return view
}

// NewCharacterCreationPageView adapts one workflow-owned page result to the
// dedicated creation-page render contract.
func NewCharacterCreationPageView(campaignID string, characterID string, page campaignworkflow.PageData) CharacterCreationPageView {
	return CharacterCreationPageView{
		CampaignID:  campaignID,
		CharacterID: characterID,
		Creation:    NewCharacterCreationView(page.Creation),
	}
}
