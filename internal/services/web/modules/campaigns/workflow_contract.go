package campaigns

import (
	"net/url"

	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// CharacterCreationWorkflow combines domain assembly with transport parsing and
// view mapping for one game system workflow.
type CharacterCreationWorkflow interface {
	AssembleCatalog(
		progress CampaignCharacterCreationProgress,
		catalog CampaignCharacterCreationCatalog,
		profile CampaignCharacterCreationProfile,
	) CampaignCharacterCreation
	CreationView(CampaignCharacterCreation) webtemplates.CampaignCharacterCreationView
	ParseStepInput(form url.Values, nextStep int32) (*CampaignCharacterCreationStepInput, error)
}
