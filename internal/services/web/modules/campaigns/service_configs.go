package campaigns

import campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"

// pageServiceConfig groups shared workspace-shell app config for detail
// surfaces.
type pageServiceConfig struct {
	Workspace     campaignapp.WorkspaceServiceConfig
	SessionRead   campaignapp.SessionReadServiceConfig
	Authorization campaignapp.AuthorizationGateway
}

// catalogServiceConfig groups campaign catalog app config.
type catalogServiceConfig struct {
	Catalog campaignapp.CatalogServiceConfig
}

// starterServiceConfig groups protected starter app config.
type starterServiceConfig struct {
	Starter campaignapp.StarterServiceConfig
}

// overviewServiceConfig groups overview, AI binding, and campaign settings app
// config.
type overviewServiceConfig struct {
	AutomationRead     campaignapp.AutomationReadServiceConfig
	AutomationMutation campaignapp.AutomationMutationServiceConfig
	Configuration      campaignapp.ConfigurationServiceConfig
	Authorization      campaignapp.AuthorizationGateway
}

// participantServiceConfig groups participant read and mutation app config.
type participantServiceConfig struct {
	Read          campaignapp.ParticipantReadServiceConfig
	Mutation      campaignapp.ParticipantMutationServiceConfig
	Authorization campaignapp.AuthorizationGateway
}

// characterServiceConfig groups character read, control, mutation, and
// creation app config.
type characterServiceConfig struct {
	Read          campaignapp.CharacterReadServiceConfig
	Control       campaignapp.CharacterControlServiceConfig
	Mutation      campaignapp.CharacterMutationServiceConfig
	Creation      campaignapp.CharacterCreationServiceConfig
	Authorization campaignapp.AuthorizationGateway
}

// sessionServiceConfig groups session mutation app config.
type sessionServiceConfig struct {
	Mutation campaignapp.SessionMutationServiceConfig
}

// inviteServiceConfig groups invite read, mutation, and search-adjacent app
// config.
type inviteServiceConfig struct {
	Read            campaignapp.InviteReadServiceConfig
	Mutation        campaignapp.InviteMutationServiceConfig
	ParticipantRead campaignapp.ParticipantReadServiceConfig
	Authorization   campaignapp.AuthorizationGateway
}

// serviceConfigs groups campaigns app config by owned route surface instead of
// one root constructor bag.
type serviceConfigs struct {
	Page         pageServiceConfig
	Catalog      catalogServiceConfig
	Starter      starterServiceConfig
	Overview     overviewServiceConfig
	Participants participantServiceConfig
	Characters   characterServiceConfig
	Sessions     sessionServiceConfig
	Invites      inviteServiceConfig
}

// newServiceConfigs assembles the production app-service graph from explicit
// surface-local composition builders instead of routing every capability through
// one root wiring sink.
func newServiceConfigs(config CompositionConfig) serviceConfigs {
	return serviceConfigs{
		Page:         newPageServiceConfig(config),
		Catalog:      newCatalogSurfaceConfig(config),
		Starter:      newStarterSurfaceConfig(config),
		Overview:     newOverviewSurfaceConfig(config),
		Participants: newParticipantSurfaceConfig(config),
		Characters:   newCharacterSurfaceConfig(config),
		Sessions:     newSessionSurfaceConfig(config),
		Invites:      newInviteSurfaceConfig(config),
	}
}
