package campaigns

import (
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
)

// Gateway/client aliases keep registry composition callers independent from the
// campaigns gateway package internals.
type (
	CampaignClient           = campaigngateway.CampaignClient
	CommunicationClient      = campaigngateway.CommunicationClient
	AgentClient              = campaigngateway.AgentClient
	ParticipantClient        = campaigngateway.ParticipantClient
	CharacterClient          = campaigngateway.CharacterClient
	DaggerheartContentClient = campaigngateway.DaggerheartContentClient
	DaggerheartAssetClient   = campaigngateway.DaggerheartAssetClient
	SessionClient            = campaigngateway.SessionClient
	InviteClient             = campaigngateway.InviteClient
	AuthClient               = campaigngateway.AuthClient
	AuthorizationClient      = campaigngateway.AuthorizationClient
	GRPCGatewayDeps          = campaigngateway.GRPCGatewayDeps
	grpcGateway              = campaigngateway.GRPCGateway
)

// NewGRPCGateway returns the production campaigns gateway.
func NewGRPCGateway(deps GRPCGatewayDeps) CampaignGateway {
	return campaigngateway.NewGRPCGateway(deps)
}

// mapCampaignCharacterCreationStepToProto maps domain workflow step input into
// the daggerheart transport contract.
func mapCampaignCharacterCreationStepToProto(step *CampaignCharacterCreationStepInput) (*daggerheartv1.DaggerheartCreationStepInput, error) {
	return campaigngateway.MapCampaignCharacterCreationStepToProto(step)
}
