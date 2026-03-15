package campaigns

import (
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow/daggerheart"
)

// buildCampaignSystems installs the production game-system manifest for the
// campaigns area so transport parsing and workflow registration share one owner.
func buildCampaignSystems(config CompositionConfig) campaignSystemRegistry {
	return newCampaignSystemRegistry(campaignSystemInstall{
		ID:                campaignapp.GameSystemDaggerheart,
		Aliases:           []string{"Daggerheart", "game_system_daggerheart"},
		CreateLabelKey:    "game.create.field_system_value_daggerheart",
		DefaultCreate:     true,
		CharacterCreation: daggerheart.New(config.Options.AssetBaseURL),
	})
}
