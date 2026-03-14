// Package workflow defines transport-owned character creation workflow seams.
package workflow

import (
	"net/url"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
)

// CharacterCreation combines system-specific domain assembly with transport
// parsing and template view mapping for one game-system workflow.
type CharacterCreation interface {
	AssembleCatalog(
		progress campaignapp.CampaignCharacterCreationProgress,
		catalog campaignapp.CampaignCharacterCreationCatalog,
		profile campaignapp.CampaignCharacterCreationProfile,
	) campaignapp.CampaignCharacterCreation
	CreationView(campaignapp.CampaignCharacterCreation) campaignrender.CampaignCharacterCreationView
	ParseStepInput(form url.Values, nextStep int32) (*campaignapp.CampaignCharacterCreationStepInput, error)
}

// Registry maps canonical game-system identifiers to transport-owned workflow
// implementations.
type Registry = map[campaignapp.GameSystem]CharacterCreation
