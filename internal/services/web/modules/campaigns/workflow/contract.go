// Package workflow defines transport-owned character creation workflow seams.
package workflow

import (
	"net/url"

	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
)

// CharacterCreation combines system-specific domain assembly with transport
// parsing and template view mapping for one game-system workflow.
type CharacterCreation interface {
	BuildView(
		progress Progress,
		catalog Catalog,
		profile Profile,
	) campaignrender.CampaignCharacterCreationView
	ParseStepInput(form url.Values, nextStep int32) (*StepInput, error)
}

// Registry maps canonical game-system identifiers to transport-owned workflow
// implementations.
type Registry = map[GameSystem]CharacterCreation
