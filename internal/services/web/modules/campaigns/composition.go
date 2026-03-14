package campaigns

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
)

// CampaignClient keeps composition ownership on the generated campaign client
// while this area composes narrower read and mutation seams internally.
type CampaignClient interface {
	campaigngateway.CampaignReadClient
	campaigngateway.CampaignMutationClient
}

// ParticipantClient keeps composition ownership on the generated participant
// client while this area composes narrower read and mutation seams internally.
type ParticipantClient interface {
	campaigngateway.ParticipantReadClient
	campaigngateway.ParticipantMutationClient
}

// CharacterClient keeps composition ownership on the generated character
// client while this area composes narrower read and mutation seams internally.
type CharacterClient interface {
	campaigngateway.CharacterReadClient
	campaigngateway.CharacterMutationClient
}

// SessionClient keeps composition ownership on the generated session client
// while this area composes narrower read and mutation seams internally.
type SessionClient interface {
	campaigngateway.SessionReadClient
	campaigngateway.SessionMutationClient
}

// InviteClient keeps composition ownership on the generated invite client
// while this area composes narrower read and mutation seams internally.
type InviteClient interface {
	campaigngateway.InviteReadClient
	campaigngateway.InviteMutationClient
}

// CompositionConfig owns the startup wiring required to construct the
// production campaigns module without leaking gateway internals into the
// registry package.
type CompositionConfig struct {
	Base             modulehandler.Base
	ChatFallbackPort string
	DashboardSync    DashboardSync
	AssetBaseURL     string

	CampaignClient           CampaignClient
	CommunicationClient      campaigngateway.CommunicationClient
	AgentClient              campaigngateway.AgentClient
	ParticipantClient        ParticipantClient
	CharacterClient          CharacterClient
	DaggerheartContentClient campaigngateway.DaggerheartContentClient
	DaggerheartAssetClient   campaigngateway.DaggerheartAssetClient
	SessionClient            SessionClient
	InviteClient             InviteClient
	SocialClient             campaigngateway.SocialClient
	AuthClient               campaigngateway.AuthClient
	AuthorizationClient      campaigngateway.AuthorizationClient
}

// Compose builds the production campaigns module from area-owned startup
// dependencies.
func Compose(config CompositionConfig) module.Module {
	workflows := campaignworkflow.Registry{
		campaignapp.GameSystemDaggerheart: daggerheart.New(config.AssetBaseURL),
	}
	gateway := campaigngateway.NewGRPCGateway(campaigngateway.GRPCGatewayDeps{
		Read: campaigngateway.GRPCGatewayReadDeps{
			Campaign:           config.CampaignClient,
			Communication:      config.CommunicationClient,
			Agent:              config.AgentClient,
			Participant:        config.ParticipantClient,
			Character:          config.CharacterClient,
			DaggerheartContent: config.DaggerheartContentClient,
			DaggerheartAsset:   config.DaggerheartAssetClient,
			Session:            config.SessionClient,
			Invite:             config.InviteClient,
			Social:             config.SocialClient,
		},
		Mutation: campaigngateway.GRPCGatewayMutationDeps{
			Campaign:    config.CampaignClient,
			Participant: config.ParticipantClient,
			Character:   config.CharacterClient,
			Session:     config.SessionClient,
			Invite:      config.InviteClient,
			Auth:        config.AuthClient,
		},
		Authorization: campaigngateway.GRPCGatewayAuthorizationDeps{
			Client: config.AuthorizationClient,
		},
		AssetBaseURL: config.AssetBaseURL,
	})
	return New(Config{
		ReadGateway:      gateway,
		MutationGateway:  gateway,
		AuthzGateway:     gateway,
		Base:             config.Base,
		ChatFallbackPort: config.ChatFallbackPort,
		Workflows:        workflows,
		DashboardSync:    config.DashboardSync,
	})
}
