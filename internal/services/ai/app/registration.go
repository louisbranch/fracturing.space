package server

import (
	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

type grpcServiceRegistration struct {
	healthName string
	register   func(*grpc.Server)
}

func serviceRegistrations(h serviceHandlers) []grpcServiceRegistration {
	return []grpcServiceRegistration{
		{
			healthName: "ai.v1.CredentialService",
			register: func(server *grpc.Server) {
				aiv1.RegisterCredentialServiceServer(server, h.credentials)
			},
		},
		{
			healthName: "ai.v1.AgentService",
			register: func(server *grpc.Server) {
				aiv1.RegisterAgentServiceServer(server, h.agents)
			},
		},
		{
			healthName: "ai.v1.InvocationService",
			register: func(server *grpc.Server) {
				aiv1.RegisterInvocationServiceServer(server, h.invocations)
			},
		},
		{
			healthName: "ai.v1.CampaignOrchestrationService",
			register: func(server *grpc.Server) {
				aiv1.RegisterCampaignOrchestrationServiceServer(server, h.campaignOrchestration)
			},
		},
		{
			healthName: "ai.v1.CampaignDebugService",
			register: func(server *grpc.Server) {
				aiv1.RegisterCampaignDebugServiceServer(server, h.campaignDebug)
			},
		},
		{
			healthName: "ai.v1.CampaignArtifactService",
			register: func(server *grpc.Server) {
				aiv1.RegisterCampaignArtifactServiceServer(server, h.campaignArtifacts)
			},
		},
		{
			healthName: "ai.v1.SystemReferenceService",
			register: func(server *grpc.Server) {
				aiv1.RegisterSystemReferenceServiceServer(server, h.systemReferences)
			},
		},
		{
			healthName: "ai.v1.ProviderGrantService",
			register: func(server *grpc.Server) {
				aiv1.RegisterProviderGrantServiceServer(server, h.providerGrants)
			},
		},
		{
			healthName: "ai.v1.AccessRequestService",
			register: func(server *grpc.Server) {
				aiv1.RegisterAccessRequestServiceServer(server, h.accessRequests)
			},
		},
	}
}

func registerServices(grpcServer *grpc.Server, healthServer *health.Server, h serviceHandlers) {
	for _, registration := range serviceRegistrations(h) {
		registration.register(grpcServer)
		healthServer.SetServingStatus(registration.healthName, grpc_health_v1.HealthCheckResponse_SERVING)
	}
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
}
