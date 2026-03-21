package app

import (
	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
)

// registrationAssemblies groups the fully constructed transport registration
// deps passed into the transport bootstrap phase.
type registrationAssemblies struct {
	daggerheart    daggerheartRegistrationDeps
	campaign       campaignRegistrationDeps
	session        sessionRegistrationDeps
	infrastructure infrastructureRegistrationDeps
}

// registrationAssemblySources keeps startup-owned registration assembly scoped
// to the exact collaborators needed to build the transport family deps.
type registrationAssemblySources struct {
	bundle          *storageBundle
	projectionStore projectionBackend
	contentStore    contentBackend
	eventStore      eventBackend
	domainState     configuredDomainState
	authClient      authv1.AuthServiceClient
	aiAgentClient   aiv1.AgentServiceClient
	systemRegistry  *bridge.MetadataRegistry
}

// buildRegistrationAssemblies assembles the transport registration families
// from the narrowed domain state plus startup-owned dependency/system context.
func buildRegistrationAssemblies(sources registrationAssemblySources) registrationAssemblies {
	return registrationAssemblies{
		daggerheart: newDaggerheartRegistrationDeps(daggerheartRegistrationSources{
			projectionStore: sources.projectionStore,
			systemStores:    sources.domainState.systemStores,
			contentStore:    sources.contentStore,
			eventStore:      sources.eventStore,
			writePath:       sources.domainState.runtimeStores.Write,
			events:          sources.domainState.applier.Events,
		}),
		campaign: newCampaignRegistrationDeps(campaignRegistrationSources{
			campaign:           sources.domainState.projectionStores.Campaign,
			participant:        sources.domainState.projectionStores.Participant,
			character:          sources.domainState.projectionStores.Character,
			session:            sources.domainState.projectionStores.Session,
			sessionInteraction: sources.domainState.projectionStores.SessionInteraction,
			systemStores:       sources.domainState.systemStores,
			invite:             sources.domainState.projectionStores.Invite,
			claimIndex:         sources.domainState.projectionStores.ClaimIndex,
			event:              sources.domainState.infrastructureStores.Event,
			content:            sources.domainState.contentStores.DaggerheartContent,
			social:             sources.domainState.contentStores.Social,
			writePath:          sources.domainState.runtimeStores.Write,
			audit:              sources.domainState.infrastructureStores.Audit,
			applier:            sources.domainState.applier,
			authClient:         sources.authClient,
			aiAgentClient:      sources.aiAgentClient,
		}),
		session: newSessionRegistrationDeps(sessionRegistrationSources{
			campaign:           sources.domainState.projectionStores.Campaign,
			participant:        sources.domainState.projectionStores.Participant,
			character:          sources.domainState.projectionStores.Character,
			session:            sources.domainState.projectionStores.Session,
			sessionGate:        sources.domainState.projectionStores.SessionGate,
			sessionSpotlight:   sources.domainState.projectionStores.SessionSpotlight,
			sessionInteraction: sources.domainState.projectionStores.SessionInteraction,
			scene:              sources.domainState.projectionStores.Scene,
			sceneCharacter:     sources.domainState.projectionStores.SceneCharacter,
			sceneInteraction:   sources.domainState.projectionStores.SceneInteraction,
			campaignFork:       sources.domainState.projectionStores.CampaignFork,
			event:              sources.domainState.infrastructureStores.Event,
			social:             sources.domainState.contentStores.Social,
			writePath:          sources.domainState.runtimeStores.Write,
			audit:              sources.domainState.infrastructureStores.Audit,
			applier:            sources.domainState.applier,
		}),
		infrastructure: newInfrastructureRegistrationDeps(infrastructureRegistrationSources{
			campaign:       sources.domainState.projectionStores.Campaign,
			participant:    sources.domainState.projectionStores.Participant,
			character:      sources.domainState.projectionStores.Character,
			audit:          sources.domainState.infrastructureStores.Audit,
			statistics:     sources.domainState.infrastructureStores.Statistics,
			bundle:         sources.bundle,
			systemRegistry: sources.systemRegistry,
		}),
	}
}
