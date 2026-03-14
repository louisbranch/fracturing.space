package templates

import "github.com/a-h/templ"

// CampaignCharacterCreationSummaryBody exposes the shared summary body used by
// campaign detail and dedicated character-creation pages.
func CampaignCharacterCreationSummaryBody(creation CampaignCharacterCreationView, loc Localizer) templ.Component {
	return creationSummaryBody(creation, loc)
}
