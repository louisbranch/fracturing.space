package server

import (
	"fmt"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	gamegrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game"
	daggerheartservice "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/gameplaystores"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

type grpcServiceDescriptor struct {
	healthService string
	register      func(*grpc.Server)
}

// registerServices builds service descriptors, registers them on the gRPC
// server, and updates health statuses for each exposed service.
func registerServices(
	grpcServer *grpc.Server,
	healthServer *health.Server,
	stores gamegrpc.Stores,
	bundle *storageBundle,
	authClient authv1.AuthServiceClient,
	aiAgentClient aiv1.AgentServiceClient,
	systemRegistry *bridge.MetadataRegistry,
	sessionGrantConfig aisessiongrant.Config,
) error {
	descriptors, err := buildServiceDescriptors(
		stores,
		bundle,
		authClient,
		aiAgentClient,
		systemRegistry,
		sessionGrantConfig,
	)
	if err != nil {
		return err
	}

	registerGRPCServices(grpcServer, descriptors)
	registerHealthStatuses(grpcServer, healthServer, descriptors)
	return nil
}

// buildServiceDescriptors centralizes service constructor wiring and returns
// registration closures used for both gRPC mount and health status setup.
func buildServiceDescriptors(
	stores gamegrpc.Stores,
	bundle *storageBundle,
	authClient authv1.AuthServiceClient,
	aiAgentClient aiv1.AgentServiceClient,
	systemRegistry *bridge.MetadataRegistry,
	sessionGrantConfig aisessiongrant.Config,
) ([]grpcServiceDescriptor, error) {
	daggerheartStores := gameplaystores.NewFromProjection(gameplaystores.FromProjectionConfig{
		ProjectionStore:  bundle.projections,
		DaggerheartStore: stores.SystemStores.Daggerheart,
		EventStore:       bundle.events,
		Domain:           stores.Write.Executor,
		Events:           stores.Events,
		WriteRuntime:     stores.Write.Runtime,
	})
	daggerheartService, err := daggerheartservice.NewDaggerheartService(daggerheartStores, random.NewSeed)
	if err != nil {
		return nil, fmt.Errorf("create daggerheart service: %w", err)
	}
	contentService, err := daggerheartservice.NewDaggerheartContentService(bundle.content)
	if err != nil {
		return nil, fmt.Errorf("create daggerheart content service: %w", err)
	}
	assetService, err := daggerheartservice.NewDaggerheartAssetService(bundle.content)
	if err != nil {
		return nil, fmt.Errorf("create daggerheart asset service: %w", err)
	}
	campaignService := gamegrpc.NewCampaignService(stores, authClient, aiAgentClient)
	participantService := gamegrpc.NewParticipantService(stores, authClient)
	inviteService := gamegrpc.NewInviteService(stores, authClient)
	characterService := gamegrpc.NewCharacterService(stores)
	snapshotService := gamegrpc.NewSnapshotService(stores)
	sessionService := gamegrpc.NewSessionService(stores)
	sceneService := gamegrpc.NewSceneService(stores)
	forkService := gamegrpc.NewForkService(stores)
	eventService := gamegrpc.NewEventService(stores)
	integrationService := gamegrpc.NewIntegrationService(bundle.events.IntegrationOutboxStore())
	statisticsService := gamegrpc.NewStatisticsService(stores.Statistics)
	systemService := gamegrpc.NewSystemService(systemRegistry)
	authorizationService := gamegrpc.NewAuthorizationService(stores)
	campaignAIService := gamegrpc.NewCampaignAIService(stores, sessionGrantConfig)
	communicationService := gamegrpc.NewCommunicationService(stores)

	descriptors := []grpcServiceDescriptor{
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
		{
			healthService: "game.v1.CommunicationService",
			register: func(server *grpc.Server) {
				statev1.RegisterCommunicationServiceServer(server, communicationService)
			},
		},
	}
	return descriptors, nil
}

func registerGRPCServices(grpcServer *grpc.Server, descriptors []grpcServiceDescriptor) {
	for _, descriptor := range descriptors {
		descriptor.register(grpcServer)
	}
}

func registerHealthStatuses(grpcServer *grpc.Server, healthServer *health.Server, descriptors []grpcServiceDescriptor) {
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	for _, descriptor := range descriptors {
		healthServer.SetServingStatus(descriptor.healthService, grpc_health_v1.HealthCheckResponse_SERVING)
	}
}
