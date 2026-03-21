package app

import (
	"fmt"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	gamegrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authorizationtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/campaigntransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/campaigntransport/aitransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/charactertransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/eventtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/forktransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/interactiontransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/invitetransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/participanttransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/scenetransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/sessiontransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/snapshottransport"
	daggerheartservice "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/gameplaystores"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
	"google.golang.org/grpc"
)

// daggerheartRegistrationDeps keeps the Daggerheart registration surface local
// to system-owned read/write paths instead of the full root game store bag.
type daggerheartRegistrationDeps struct {
	projectionStore projectionBackend
	systemStores    gamegrpc.SystemStores
	contentStore    contentBackend
	eventStore      eventBackend
	writePath       gamegrpc.WritePath
	events          *event.Registry
}

// daggerheartRegistrationSources keeps the Daggerheart family builder scoped
// to the exact startup-owned collaborators it needs.
type daggerheartRegistrationSources struct {
	projectionStore projectionBackend
	systemStores    gamegrpc.SystemStores
	contentStore    contentBackend
	eventStore      eventBackend
	writePath       gamegrpc.WritePath
	events          *event.Registry
}

// campaignRegistrationDeps groups the campaign-owned transport dependencies so
// root registration can wire campaign services without passing every store.
type campaignRegistrationDeps struct {
	policy             authz.PolicyDeps
	campaignStore      storage.CampaignStore
	participantStore   storage.ParticipantStore
	characterStore     storage.CharacterStore
	sessionStore       storage.SessionStore
	sessionInteraction storage.SessionInteractionStore
	systemStores       gamegrpc.SystemStores
	inviteStore        storage.InviteStore
	claimIndexStore    storage.ClaimIndexStore
	eventStore         storage.EventStore
	contentStore       contentstore.DaggerheartContentReadStore
	socialClient       socialv1.SocialServiceClient
	writePath          gamegrpc.WritePath
	applier            projection.Applier
	authClient         authv1.AuthServiceClient
	aiAgentClient      aiv1.AgentServiceClient
}

// campaignRegistrationSources keeps the campaign family builder scoped to the
// exact startup-owned collaborators it needs, without passing concern groups.
type campaignRegistrationSources struct {
	campaign           storage.CampaignStore
	participant        storage.ParticipantStore
	character          storage.CharacterStore
	session            storage.SessionStore
	sessionInteraction storage.SessionInteractionStore
	systemStores       gamegrpc.SystemStores
	invite             storage.InviteStore
	claimIndex         storage.ClaimIndexStore
	event              storage.EventStore
	content            contentstore.DaggerheartContentReadStore
	social             socialv1.SocialServiceClient
	writePath          gamegrpc.WritePath
	audit              storage.AuditEventStore
	applier            projection.Applier
	authClient         authv1.AuthServiceClient
	aiAgentClient      aiv1.AgentServiceClient
}

// sessionRegistrationDeps keeps session-owned service registration aligned to
// the session/scene/interaction workflow seams instead of the root store bag.
type sessionRegistrationDeps struct {
	policy             authz.PolicyDeps
	campaignStore      storage.CampaignStore
	participantStore   storage.ParticipantStore
	characterStore     storage.CharacterStore
	sessionStore       storage.SessionStore
	sessionGateStore   storage.SessionGateStore
	sessionSpotlight   storage.SessionSpotlightStore
	sessionInteraction storage.SessionInteractionStore
	sceneStore         storage.SceneStore
	sceneCharacter     storage.SceneCharacterStore
	sceneInteraction   storage.SceneInteractionStore
	campaignForkStore  storage.CampaignForkStore
	eventStore         storage.EventStore
	socialClient       socialv1.SocialServiceClient
	writePath          gamegrpc.WritePath
	applier            projection.Applier
}

// sessionRegistrationSources keeps the session family builder scoped to the
// exact session/scene workflow collaborators it needs from startup.
type sessionRegistrationSources struct {
	campaign           storage.CampaignStore
	participant        storage.ParticipantStore
	character          storage.CharacterStore
	session            storage.SessionStore
	sessionGate        storage.SessionGateStore
	sessionSpotlight   storage.SessionSpotlightStore
	sessionInteraction storage.SessionInteractionStore
	scene              storage.SceneStore
	sceneCharacter     storage.SceneCharacterStore
	sceneInteraction   storage.SceneInteractionStore
	campaignFork       storage.CampaignForkStore
	event              storage.EventStore
	social             socialv1.SocialServiceClient
	writePath          gamegrpc.WritePath
	audit              storage.AuditEventStore
	applier            projection.Applier
}

// infrastructureRegistrationDeps keeps the remaining operational services on a
// minimal infrastructure surface rather than coupling them to gameplay stores.
type infrastructureRegistrationDeps struct {
	campaignStore    storage.CampaignStore
	participantStore storage.ParticipantStore
	characterStore   storage.CharacterStore
	auditStore       storage.AuditEventStore
	statisticsStore  storage.StatisticsStore
	bundle           *storageBundle
	systemRegistry   *bridge.MetadataRegistry
}

// infrastructureRegistrationSources keeps the infrastructure family builder
// scoped to the exact operational collaborators it needs from startup.
type infrastructureRegistrationSources struct {
	campaign       storage.CampaignStore
	participant    storage.ParticipantStore
	character      storage.CharacterStore
	audit          storage.AuditEventStore
	statistics     storage.StatisticsStore
	bundle         *storageBundle
	systemRegistry *bridge.MetadataRegistry
}

// newRegistrationPolicyDeps extracts the shared authorization collaborators
// once so capability helpers can depend on a narrow policy bundle.
func newRegistrationPolicyDeps(
	participant storage.ParticipantStore,
	character storage.CharacterStore,
	audit storage.AuditEventStore,
) authz.PolicyDeps {
	return authz.PolicyDeps{
		Participant: participant,
		Character:   character,
		Audit:       audit,
	}
}

// newDaggerheartRegistrationDeps binds the Daggerheart service family to the
// system-owned read/write paths needed by the root transport.
func newDaggerheartRegistrationDeps(
	sources daggerheartRegistrationSources,
) daggerheartRegistrationDeps {
	return daggerheartRegistrationDeps{
		projectionStore: sources.projectionStore,
		systemStores:    sources.systemStores,
		contentStore:    sources.contentStore,
		eventStore:      sources.eventStore,
		writePath:       sources.writePath,
		events:          sources.events,
	}
}

// newCampaignRegistrationDeps assembles the campaign-owned service family from
// the root stores without introducing a transport-wide registration bag.
func newCampaignRegistrationDeps(
	sources campaignRegistrationSources,
) campaignRegistrationDeps {
	return campaignRegistrationDeps{
		policy:             newRegistrationPolicyDeps(sources.participant, sources.character, sources.audit),
		campaignStore:      sources.campaign,
		participantStore:   sources.participant,
		characterStore:     sources.character,
		sessionStore:       sources.session,
		sessionInteraction: sources.sessionInteraction,
		systemStores:       sources.systemStores,
		inviteStore:        sources.invite,
		claimIndexStore:    sources.claimIndex,
		eventStore:         sources.event,
		contentStore:       sources.content,
		socialClient:       sources.social,
		writePath:          sources.writePath,
		applier:            sources.applier,
		authClient:         sources.authClient,
		aiAgentClient:      sources.aiAgentClient,
	}
}

// newSessionRegistrationDeps assembles the session-owned registration family
// from the stores that back session, scene, and interaction workflows.
func newSessionRegistrationDeps(
	sources sessionRegistrationSources,
) sessionRegistrationDeps {
	return sessionRegistrationDeps{
		policy:             newRegistrationPolicyDeps(sources.participant, sources.character, sources.audit),
		campaignStore:      sources.campaign,
		participantStore:   sources.participant,
		characterStore:     sources.character,
		sessionStore:       sources.session,
		sessionGateStore:   sources.sessionGate,
		sessionSpotlight:   sources.sessionSpotlight,
		sessionInteraction: sources.sessionInteraction,
		sceneStore:         sources.scene,
		sceneCharacter:     sources.sceneCharacter,
		sceneInteraction:   sources.sceneInteraction,
		campaignForkStore:  sources.campaignFork,
		eventStore:         sources.event,
		socialClient:       sources.social,
		writePath:          sources.writePath,
		applier:            sources.applier,
	}
}

// newInfrastructureRegistrationDeps assembles the remaining operational
// services from infrastructure stores and startup-owned collaborators.
func newInfrastructureRegistrationDeps(
	sources infrastructureRegistrationSources,
) infrastructureRegistrationDeps {
	return infrastructureRegistrationDeps{
		campaignStore:    sources.campaign,
		participantStore: sources.participant,
		characterStore:   sources.character,
		auditStore:       sources.audit,
		statisticsStore:  sources.statistics,
		bundle:           sources.bundle,
		systemRegistry:   sources.systemRegistry,
	}
}

// buildDaggerheartServiceDescriptors wires the Daggerheart transport family
// from system-owned read models plus the shared write path.
func buildDaggerheartServiceDescriptors(deps daggerheartRegistrationDeps) ([]grpcServiceDescriptor, error) {
	daggerheartStores := gameplaystores.NewFromProjection(gameplaystores.FromProjectionConfig{
		ProjectionStore:  deps.projectionStore,
		DaggerheartStore: deps.systemStores.Daggerheart,
		ContentStore:     deps.contentStore,
		EventStore:       deps.eventStore,
		Domain:           deps.writePath.Executor,
		WriteRuntime:     deps.writePath.Runtime,
		Events:           deps.events,
	})
	daggerheartService, err := daggerheartservice.NewDaggerheartService(daggerheartStores, random.NewSeed)
	if err != nil {
		return nil, fmt.Errorf("create daggerheart service: %w", err)
	}
	contentService, err := daggerheartservice.NewDaggerheartContentService(deps.contentStore)
	if err != nil {
		return nil, fmt.Errorf("create daggerheart content service: %w", err)
	}
	assetService, err := daggerheartservice.NewDaggerheartAssetService(deps.contentStore)
	if err != nil {
		return nil, fmt.Errorf("create daggerheart asset service: %w", err)
	}
	return []grpcServiceDescriptor{
		{
			healthService: "systems.daggerheart.v1.DaggerheartService",
			register: func(server *grpc.Server) {
				daggerheartv1.RegisterDaggerheartServiceServer(server, daggerheartService)
			},
		},
		{
			healthService: "systems.daggerheart.v1.DaggerheartContentService",
			register: func(server *grpc.Server) {
				daggerheartv1.RegisterDaggerheartContentServiceServer(server, contentService)
			},
		},
		{
			healthService: "systems.daggerheart.v1.DaggerheartAssetService",
			register: func(server *grpc.Server) {
				daggerheartv1.RegisterDaggerheartAssetServiceServer(server, assetService)
			},
		},
	}, nil
}

// buildCampaignServiceDescriptors wires the campaign-owned service family and
// contains the only remaining translation into the legacy orchestration bag.
func buildCampaignServiceDescriptors(
	deps campaignRegistrationDeps,
	sessionGrantConfig aisessiongrant.Config,
) []grpcServiceDescriptor {
	campaignService := campaigntransport.NewCampaignService(campaigntransport.Deps{
		Auth:               deps.policy,
		Campaign:           deps.campaignStore,
		Participant:        deps.participantStore,
		Character:          deps.characterStore,
		Session:            deps.sessionStore,
		SessionInteraction: deps.sessionInteraction,
		Daggerheart:        deps.systemStores.Daggerheart,
		Social:             deps.socialClient,
		Write:              deps.writePath,
		Applier:            deps.applier,
		AuthClient:         deps.authClient,
		AIClient:           deps.aiAgentClient,
	})
	campaignAIService := aitransport.NewService(aitransport.Deps{
		Campaign:           deps.campaignStore,
		Session:            deps.sessionStore,
		Participant:        deps.participantStore,
		SessionInteraction: deps.sessionInteraction,
		SessionGrantConfig: sessionGrantConfig,
	})
	campaignAIOrchestrationService := gamegrpc.NewCampaignAIOrchestrationService(gamegrpc.CampaignAIOrchestrationDeps{
		Campaign:           deps.campaignStore,
		Participant:        deps.participantStore,
		Session:            deps.sessionStore,
		SessionInteraction: deps.sessionInteraction,
		Write:              deps.writePath,
		Applier:            deps.applier,
	})
	participantService := participanttransport.NewService(participanttransport.Deps{
		Auth:                   deps.policy,
		Campaign:               deps.campaignStore,
		Participant:            deps.participantStore,
		Character:              deps.characterStore,
		Social:                 deps.socialClient,
		Write:                  deps.writePath,
		Applier:                deps.applier,
		ClearCampaignAIBinding: campaigntransport.NewClearCampaignAIBindingFunc(deps.campaignStore, deps.writePath, deps.applier),
	}, deps.authClient)
	inviteService := invitetransport.NewService(invitetransport.Deps{
		Auth:        deps.policy,
		Campaign:    deps.campaignStore,
		Participant: deps.participantStore,
		Character:   deps.characterStore,
		Invite:      deps.inviteStore,
		ClaimIndex:  deps.claimIndexStore,
		Event:       deps.eventStore,
		Social:      deps.socialClient,
		Write:       deps.writePath,
		Applier:     deps.applier,
	}, deps.authClient)
	characterService := charactertransport.NewService(charactertransport.Deps{
		Auth:               deps.policy,
		Campaign:           deps.campaignStore,
		Character:          deps.characterStore,
		Participant:        deps.participantStore,
		Daggerheart:        deps.systemStores.Daggerheart,
		DaggerheartContent: deps.contentStore,
		Write:              deps.writePath,
		Applier:            deps.applier,
	})
	snapshotService := snapshottransport.NewService(snapshottransport.Deps{
		Auth:        deps.policy,
		Campaign:    deps.campaignStore,
		Character:   deps.characterStore,
		Daggerheart: deps.systemStores.Daggerheart,
		Write:       deps.writePath,
		Applier:     deps.applier,
	})

	return []grpcServiceDescriptor{
		{
			healthService: "game.v1.CampaignService",
			register: func(server *grpc.Server) {
				statev1.RegisterCampaignServiceServer(server, campaignService)
			},
		},
		{
			healthService: "game.v1.CampaignAIService",
			register: func(server *grpc.Server) {
				statev1.RegisterCampaignAIServiceServer(server, campaignAIService)
			},
		},
		{
			healthService: "game.v1.CampaignAIOrchestrationService",
			register: func(server *grpc.Server) {
				statev1.RegisterCampaignAIOrchestrationServiceServer(server, campaignAIOrchestrationService)
			},
		},
		{
			healthService: "game.v1.ParticipantService",
			register: func(server *grpc.Server) {
				statev1.RegisterParticipantServiceServer(server, participantService)
			},
		},
		{
			healthService: "game.v1.InviteService",
			register: func(server *grpc.Server) {
				statev1.RegisterInviteServiceServer(server, inviteService)
			},
		},
		{
			healthService: "game.v1.CharacterService",
			register: func(server *grpc.Server) {
				statev1.RegisterCharacterServiceServer(server, characterService)
			},
		},
		{
			healthService: "game.v1.SnapshotService",
			register: func(server *grpc.Server) {
				statev1.RegisterSnapshotServiceServer(server, snapshotService)
			},
		},
	}
}

// buildSessionServiceDescriptors wires the session/scene interaction family
// from the session-owned projection and write dependencies.
func buildSessionServiceDescriptors(deps sessionRegistrationDeps) []grpcServiceDescriptor {
	sessionService := sessiontransport.NewSessionService(sessiontransport.Deps{
		Auth:               deps.policy,
		Campaign:           deps.campaignStore,
		Participant:        deps.participantStore,
		Character:          deps.characterStore,
		Session:            deps.sessionStore,
		SessionGate:        deps.sessionGateStore,
		SessionSpotlight:   deps.sessionSpotlight,
		SessionInteraction: deps.sessionInteraction,
		Scene:              deps.sceneStore,
		Write:              deps.writePath,
		Applier:            deps.applier,
	})
	sceneService := scenetransport.NewService(scenetransport.Deps{
		Auth:           deps.policy,
		Campaign:       deps.campaignStore,
		Scene:          deps.sceneStore,
		SceneCharacter: deps.sceneCharacter,
		Write:          deps.writePath,
		Applier:        deps.applier,
	})
	forkService := forktransport.NewService(forktransport.Deps{
		Auth:         deps.policy,
		Campaign:     deps.campaignStore,
		Participant:  deps.participantStore,
		Character:    deps.characterStore,
		Session:      deps.sessionStore,
		CampaignFork: deps.campaignForkStore,
		Event:        deps.eventStore,
		Social:       deps.socialClient,
		Write:        deps.writePath,
		Applier:      deps.applier,
	})
	eventService := eventtransport.NewService(eventtransport.Deps{
		Auth:        deps.policy,
		Event:       deps.eventStore,
		Campaign:    deps.campaignStore,
		Participant: deps.participantStore,
		Character:   deps.characterStore,
		Session:     deps.sessionStore,
		Write:       deps.writePath,
	})
	interactionService := interactiontransport.NewInteractionService(interactiontransport.Deps{
		Auth:               deps.policy,
		Campaign:           deps.campaignStore,
		Participant:        deps.participantStore,
		Character:          deps.characterStore,
		Session:            deps.sessionStore,
		SessionInteraction: deps.sessionInteraction,
		Scene:              deps.sceneStore,
		SceneCharacter:     deps.sceneCharacter,
		SceneInteraction:   deps.sceneInteraction,
		Write:              deps.writePath,
		Applier:            deps.applier,
	})

	return []grpcServiceDescriptor{
		{
			healthService: "game.v1.SessionService",
			register: func(server *grpc.Server) {
				statev1.RegisterSessionServiceServer(server, sessionService)
			},
		},
		{
			healthService: "game.v1.SceneService",
			register: func(server *grpc.Server) {
				statev1.RegisterSceneServiceServer(server, sceneService)
			},
		},
		{
			healthService: "game.v1.ForkService",
			register: func(server *grpc.Server) {
				statev1.RegisterForkServiceServer(server, forkService)
			},
		},
		{
			healthService: "game.v1.EventService",
			register: func(server *grpc.Server) {
				statev1.RegisterEventServiceServer(server, eventService)
			},
		},
		{
			healthService: "game.v1.InteractionService",
			register: func(server *grpc.Server) {
				statev1.RegisterInteractionServiceServer(server, interactionService)
			},
		},
	}
}

// buildInfrastructureServiceDescriptors wires the operational services that do
// not belong to a gameplay capability family.
func buildInfrastructureServiceDescriptors(deps infrastructureRegistrationDeps) []grpcServiceDescriptor {
	integrationService := gamegrpc.NewIntegrationService(deps.bundle.events.IntegrationOutboxStore())
	statisticsService := gamegrpc.NewStatisticsService(deps.statisticsStore)
	systemService := gamegrpc.NewSystemService(deps.systemRegistry)
	authorizationService := authorizationtransport.NewService(authorizationtransport.Deps{
		Campaign:    deps.campaignStore,
		Participant: deps.participantStore,
		Character:   deps.characterStore,
		Audit:       deps.auditStore,
	})

	return []grpcServiceDescriptor{
		{
			healthService: "game.v1.IntegrationService",
			register: func(server *grpc.Server) {
				statev1.RegisterIntegrationServiceServer(server, integrationService)
			},
		},
		{
			healthService: "game.v1.StatisticsService",
			register: func(server *grpc.Server) {
				statev1.RegisterStatisticsServiceServer(server, statisticsService)
			},
		},
		{
			healthService: "game.v1.SystemService",
			register: func(server *grpc.Server) {
				statev1.RegisterSystemServiceServer(server, systemService)
			},
		},
		{
			healthService: "game.v1.AuthorizationService",
			register: func(server *grpc.Server) {
				statev1.RegisterAuthorizationServiceServer(server, authorizationService)
			},
		},
	}
}
