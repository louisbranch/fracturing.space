package campaigns

import (
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
)

// newCharacterCreationServiceConfig keeps workflow read/mutation wiring local
// to the character-creation capability family.
func newCharacterCreationServiceConfig(config CompositionConfig) campaignapp.CharacterCreationServiceConfig {
	return campaignapp.CharacterCreationServiceConfig{
		Read:     campaigngateway.NewCharacterCreationReadGateway(config.Gateway.Characters.CreationRead, config.Options.AssetBaseURL),
		Mutation: campaigngateway.NewCharacterCreationMutationGateway(config.Gateway.Characters.CreationMutation),
	}
}
