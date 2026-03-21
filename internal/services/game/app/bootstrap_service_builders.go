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

// daggerheartRegistrationDeps groups the Daggerheart service family
// collaborators needed for both transport registration assembly and service
// descriptor construction.
type daggerheartRegistrationDeps struct {
	projectionStore projectionBackend
	systemStores    gamegrpc.SystemStores
	contentStore    contentBackend
	eventStore      eventBackend
	writePath       gamegrpc.WritePath
	events          *event.Registry
}

// campaignRegistrationDeps groups the campaign service family collaborators.
// Policy deps are derived from the struct's own fields via policyDeps().
type campaignRegistrationDeps struct {
	campaignStore      storage.CampaignStore
	participantStore   storage.ParticipantStore
	characterStore     storage.CharacterStore
	auditStore         storage.AuditEventStore
	sessionStore       storage.SessionStore
	sessionInteraction storage.SessionInteractionStore
	sceneInteraction   storage.SceneInteractionStore
	systemStores       gamegrpc.SystemStores
	claimIndexStore    storage.ClaimIndexStore
	eventStore         storage.EventStore
	contentStore       contentstore.DaggerheartContentReadStore
	socialClient       socialv1.SocialServiceClient
	writePath          gamegrpc.WritePath
	applier            projection.Applier
	authClient         authv1.AuthServiceClient
	aiAgentClient      aiv1.AgentServiceClient
}

// policyDeps derives shared authorization collaborators from the campaign
// registration's own fields.
func (d campaignRegistrationDeps) policyDeps() authz.PolicyDeps {
	return authz.PolicyDeps{
		Participant: d.participantStore,
		Character:   d.characterStore,
		Audit:       d.auditStore,
	}
}

// sessionRegistrationDeps groups the session/scene/interaction service family
// collaborators. Policy deps are derived via policyDeps().
type sessionRegistrationDeps struct {
	campaignStore      storage.CampaignStore
	participantStore   storage.ParticipantStore
	characterStore     storage.CharacterStore
	auditStore         storage.AuditEventStore
	sessionStore       storage.SessionStore
	sessionGateStore   storage.SessionGateStore
	sessionSpotlight   storage.SessionSpotlightStore
	sessionInteraction storage.SessionInteractionStore
	sceneStore         storage.SceneStore
	sceneCharacter     storage.SceneCharacterStore
	sceneInteraction   storage.SceneInteractionStore
	sceneGMInteraction storage.SceneGMInteractionStore
	campaignForkStore  storage.CampaignForkStore
	eventStore         storage.EventStore
	eventRegistry      *event.Registry
	socialClient       socialv1.SocialServiceClient
	writePath          gamegrpc.WritePath
	applier            projection.Applier
}

// policyDeps derives shared authorization collaborators from the session
// registration's own fields.
func (d sessionRegistrationDeps) policyDeps() authz.PolicyDeps {
	return authz.PolicyDeps{
		Participant: d.participantStore,
		Character:   d.characterStore,
		Audit:       d.auditStore,
	}
}

// infrastructureRegistrationDeps groups the operational service collaborators
// that do not belong to a gameplay capability family.
type infrastructureRegistrationDeps struct {
	campaignStore    storage.CampaignStore
	participantStore storage.ParticipantStore
	characterStore   storage.CharacterStore
	auditStore       storage.AuditEventStore
	statisticsStore  storage.StatisticsStore
	bundle           *storageBundle
	systemRegistry   *bridge.MetadataRegistry
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
	policy := deps.policyDeps()
	campaignService := campaigntransport.NewCampaignService(campaigntransport.Deps{
		Auth:               policy,
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
		SceneInteraction:   deps.sceneInteraction,
		Write:              deps.writePath,
		Applier:            deps.applier,
	})
	participantService := participanttransport.NewService(participanttransport.Deps{
		Auth:                   policy,
		Campaign:               deps.campaignStore,
		Participant:            deps.participantStore,
		Character:              deps.characterStore,
		Social:                 deps.socialClient,
		Write:                  deps.writePath,
		Applier:                deps.applier,
		ClaimIndex:             deps.claimIndexStore,
		Event:                  deps.eventStore,
		ClearCampaignAIBinding: campaigntransport.NewClearCampaignAIBindingFunc(deps.campaignStore, deps.writePath, deps.applier),
	}, deps.authClient)
	characterService := charactertransport.NewService(charactertransport.Deps{
		Auth:               policy,
		Campaign:           deps.campaignStore,
		Character:          deps.characterStore,
		Participant:        deps.participantStore,
		Daggerheart:        deps.systemStores.Daggerheart,
		DaggerheartContent: deps.contentStore,
		Write:              deps.writePath,
		Applier:            deps.applier,
	})
	snapshotService := snapshottransport.NewService(snapshottransport.Deps{
		Auth:        policy,
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
	policy := deps.policyDeps()
	sessionService := sessiontransport.NewSessionService(sessiontransport.Deps{
		Auth:               policy,
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
		Auth:               policy,
		Campaign:           deps.campaignStore,
		Participant:        deps.participantStore,
		Character:          deps.characterStore,
		Event:              deps.eventStore,
		Session:            deps.sessionStore,
		SessionInteraction: deps.sessionInteraction,
		Scene:              deps.sceneStore,
		SceneCharacter:     deps.sceneCharacter,
		SceneInteraction:   deps.sceneInteraction,
		SceneGMInteraction: deps.sceneGMInteraction,
		Write:              deps.writePath,
		Applier:            deps.applier,
	})
	forkService := forktransport.NewService(forktransport.Deps{
		Auth:          policy,
		Campaign:      deps.campaignStore,
		Participant:   deps.participantStore,
		Character:     deps.characterStore,
		Session:       deps.sessionStore,
		CampaignFork:  deps.campaignForkStore,
		Event:         deps.eventStore,
		EventRegistry: deps.eventRegistry,
		Social:        deps.socialClient,
		Write:         deps.writePath,
		Applier:       deps.applier,
	})
	eventService := eventtransport.NewService(eventtransport.Deps{
		Auth:        policy,
		Event:       deps.eventStore,
		Campaign:    deps.campaignStore,
		Participant: deps.participantStore,
		Character:   deps.characterStore,
		Session:     deps.sessionStore,
		Write:       deps.writePath,
	})
	interactionService := interactiontransport.NewInteractionService(interactiontransport.Deps{
		Auth:               policy,
		Campaign:           deps.campaignStore,
		Participant:        deps.participantStore,
		Character:          deps.characterStore,
		Event:              deps.eventStore,
		Session:            deps.sessionStore,
		SessionInteraction: deps.sessionInteraction,
		Scene:              deps.sceneStore,
		SceneCharacter:     deps.sceneCharacter,
		SceneInteraction:   deps.sceneInteraction,
		SceneGMInteraction: deps.sceneGMInteraction,
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
