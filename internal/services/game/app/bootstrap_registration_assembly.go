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
		daggerheart: daggerheartRegistrationDeps{
			projectionStore: sources.projectionStore,
			systemStores:    sources.domainState.systemStores,
			contentStore:    sources.contentStore,
			eventStore:      sources.eventStore,
			writePath:       sources.domainState.runtimeStores.Write,
			events:          sources.domainState.applier.Events,
		},
		campaign: campaignRegistrationDeps{
			campaignStore:      sources.domainState.projectionStores.Campaign,
			participantStore:   sources.domainState.projectionStores.Participant,
			characterStore:     sources.domainState.projectionStores.Character,
			auditStore:         sources.domainState.infrastructureStores.Audit,
			sessionStore:       sources.domainState.projectionStores.Session,
			sessionInteraction: sources.domainState.projectionStores.SessionInteraction,
			sceneInteraction:   sources.domainState.projectionStores.SceneInteraction,
			systemStores:       sources.domainState.systemStores,
			claimIndexStore:    sources.domainState.projectionStores.ClaimIndex,
			eventStore:         sources.domainState.infrastructureStores.Event,
			contentStore:       sources.domainState.contentStores.DaggerheartContent,
			socialClient:       sources.domainState.contentStores.Social,
			writePath:          sources.domainState.runtimeStores.Write,
			applier:            sources.domainState.applier,
			authClient:         sources.authClient,
			aiAgentClient:      sources.aiAgentClient,
		},
		session: sessionRegistrationDeps{
			campaignStore:      sources.domainState.projectionStores.Campaign,
			participantStore:   sources.domainState.projectionStores.Participant,
			characterStore:     sources.domainState.projectionStores.Character,
			auditStore:         sources.domainState.infrastructureStores.Audit,
			sessionStore:       sources.domainState.projectionStores.Session,
			sessionGateStore:   sources.domainState.projectionStores.SessionGate,
			sessionSpotlight:   sources.domainState.projectionStores.SessionSpotlight,
			sessionInteraction: sources.domainState.projectionStores.SessionInteraction,
			sceneStore:         sources.domainState.projectionStores.Scene,
			sceneCharacter:     sources.domainState.projectionStores.SceneCharacter,
			sceneInteraction:   sources.domainState.projectionStores.SceneInteraction,
			campaignForkStore:  sources.domainState.projectionStores.CampaignFork,
			eventStore:         sources.domainState.infrastructureStores.Event,
			eventRegistry:      sources.domainState.applier.Events,
			socialClient:       sources.domainState.contentStores.Social,
			writePath:          sources.domainState.runtimeStores.Write,
			applier:            sources.domainState.applier,
		},
		infrastructure: infrastructureRegistrationDeps{
			campaignStore:    sources.domainState.projectionStores.Campaign,
			participantStore: sources.domainState.projectionStores.Participant,
			characterStore:   sources.domainState.projectionStores.Character,
			auditStore:       sources.domainState.infrastructureStores.Audit,
			statisticsStore:  sources.domainState.infrastructureStores.Statistics,
			bundle:           sources.bundle,
			systemRegistry:   sources.systemRegistry,
		},
	}
}
