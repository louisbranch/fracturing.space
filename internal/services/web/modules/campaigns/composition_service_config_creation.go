package campaigns

import (
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
)

// newCharacterCreationServiceConfig keeps workflow read/mutation wiring local
// to the character-creation capability family.
func newCharacterCreationServiceConfig(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) campaignapp.CharacterCreationServiceConfig {
	return campaignapp.CharacterCreationServiceConfig{
		Read:     campaigngateway.NewCharacterCreationReadGateway(deps.CreationRead, assetBaseURL),
		Mutation: campaigngateway.NewCharacterCreationMutationGateway(deps.CreationMutation),
	}
}
