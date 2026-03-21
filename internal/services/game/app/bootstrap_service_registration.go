package app

import (
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
	daggerheartDeps daggerheartRegistrationDeps,
	campaignDeps campaignRegistrationDeps,
	sessionDeps sessionRegistrationDeps,
	infrastructureDeps infrastructureRegistrationDeps,
	sessionGrantConfig aisessiongrant.Config,
) error {
	descriptors, err := buildServiceDescriptors(
		daggerheartDeps,
		campaignDeps,
		sessionDeps,
		infrastructureDeps,
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
	daggerheartDeps daggerheartRegistrationDeps,
	campaignDeps campaignRegistrationDeps,
	sessionDeps sessionRegistrationDeps,
	infrastructureDeps infrastructureRegistrationDeps,
	sessionGrantConfig aisessiongrant.Config,
) ([]grpcServiceDescriptor, error) {
	descriptors := make([]grpcServiceDescriptor, 0, 19)

	daggerheartDescriptors, err := buildDaggerheartServiceDescriptors(daggerheartDeps)
	if err != nil {
		return nil, err
	}
	descriptors = append(descriptors, daggerheartDescriptors...)
	descriptors = append(descriptors, buildCampaignServiceDescriptors(campaignDeps, sessionGrantConfig)...)
	descriptors = append(descriptors, buildSessionServiceDescriptors(sessionDeps)...)
	descriptors = append(descriptors, buildInfrastructureServiceDescriptors(infrastructureDeps)...)
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
