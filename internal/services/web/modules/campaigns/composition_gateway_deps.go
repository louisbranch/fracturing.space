package campaigns

import campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"

// newGatewayDeps keeps generated-client grouping in one place for production composition.
func newGatewayDeps(config CompositionConfig) campaigngateway.GRPCGatewayDeps {
	return campaigngateway.GRPCGatewayDeps{
		CatalogRead: campaigngateway.CatalogReadDeps{
			Campaign: config.CampaignClient,
		},
		Starter: campaigngateway.StarterDeps{
			Discovery:        config.DiscoveryClient,
			Agent:            config.AgentClient,
			CampaignArtifact: config.CampaignArtifactClient,
			Campaign:         config.CampaignClient,
			Fork:             config.ForkClient,
		},
		CatalogMutation: campaigngateway.CatalogMutationDeps{
			Campaign: config.CampaignClient,
		},
		WorkspaceRead: campaigngateway.WorkspaceReadDeps{
			Campaign: config.CampaignClient,
		},
		ParticipantRead: campaigngateway.ParticipantReadDeps{
			Participant: config.ParticipantClient,
		},
		ParticipantMutate: campaigngateway.ParticipantMutationDeps{
			Participant: config.ParticipantClient,
		},
		CharacterRead: campaigngateway.CharacterReadDeps{
			Character:          config.CharacterClient,
			Participant:        config.ParticipantClient,
			DaggerheartContent: config.DaggerheartContentClient,
		},
		CharacterControl: campaigngateway.CharacterControlMutationDeps{
			Character: config.CharacterClient,
		},
		CharacterMutate: campaigngateway.CharacterMutationDeps{
			Character: config.CharacterClient,
		},
		SessionRead: campaigngateway.SessionReadDeps{
			Session:  config.SessionClient,
			Campaign: config.CampaignClient,
		},
		SessionMutate: campaigngateway.SessionMutationDeps{
			Session: config.SessionClient,
		},
		InviteRead: campaigngateway.InviteReadDeps{
			Invite:      config.InviteClient,
			Participant: config.ParticipantClient,
			Social:      config.SocialClient,
			Auth:        config.AuthClient,
		},
		InviteMutate: campaigngateway.InviteMutationDeps{
			Invite: config.InviteClient,
			Auth:   config.AuthClient,
		},
		ConfigMutate: campaigngateway.ConfigurationMutationDeps{
			Campaign: config.CampaignClient,
		},
		AutomationRead: campaigngateway.AutomationReadDeps{
			Agent: config.AgentClient,
		},
		AutomationMutate: campaigngateway.AutomationMutationDeps{
			Campaign: config.CampaignClient,
		},
		CreationRead: campaigngateway.CharacterCreationReadDeps{
			Character:          config.CharacterClient,
			DaggerheartContent: config.DaggerheartContentClient,
			DaggerheartAsset:   config.DaggerheartAssetClient,
		},
		CreationMutation: campaigngateway.CharacterCreationMutationDeps{
			Character: config.CharacterClient,
		},
		Authorization: campaigngateway.AuthorizationDeps{
			Client: config.AuthorizationClient,
		},
		AssetBaseURL: config.AssetBaseURL,
	}
}
