package campaigns

import (
	"net/http"

	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// CharacterCreationWorkflow abstracts system-specific character creation.
type CharacterCreationWorkflow interface {
	// AssembleCatalog builds the system-specific catalog view from generic
	// gateway data (progress, catalog, profile).
	AssembleCatalog(
		progress CampaignCharacterCreationProgress,
		catalog CampaignCharacterCreationCatalog,
		profile CampaignCharacterCreationProfile,
	) CampaignCharacterCreation

	// CreationView maps the creation model to the template view.
	CreationView(CampaignCharacterCreation) webtemplates.CampaignCharacterCreationView

	// ParseStepInput extracts a step from an HTTP form for the given step number.
	ParseStepInput(r *http.Request, nextStep int32) (*CampaignCharacterCreationStepInput, error)
}
