package render

import (
	"github.com/a-h/templ"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// CampaignCharacterCreationSummaryBody exposes the shared summary body used by
// campaign detail and dedicated character-creation pages.
func CampaignCharacterCreationSummaryBody(creation CampaignCharacterCreationView, loc webtemplates.Localizer) templ.Component {
	return creationSummaryBody(creation, loc)
}
