package server

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

func TestRegisterGRPCServices_InvokesAllDescriptors(t *testing.T) {
	grpcServer := grpc.NewServer()
	calls := 0

	descriptors := []grpcServiceDescriptor{
		{
			healthService: "game.v1.ServiceA",
			register: func(*grpc.Server) {
				calls++
			},
		},
		{
			healthService: "game.v1.ServiceB",
			register: func(*grpc.Server) {
				calls++
			},
		},
	}

	registerGRPCServices(grpcServer, descriptors)

	if calls != 2 {
		t.Fatalf("register calls = %d, want 2", calls)
	}
}

func TestRegisterHealthStatuses_SetsServingForDescriptors(t *testing.T) {
	grpcServer := grpc.NewServer()
	healthServer := health.NewServer()

	descriptors := []grpcServiceDescriptor{
		{healthService: "game.v1.ServiceA"},
		{healthService: "game.v1.ServiceB"},
	}

	registerHealthStatuses(grpcServer, healthServer, descriptors)

	services := []string{"", "game.v1.ServiceA", "game.v1.ServiceB"}
	for _, service := range services {
		response, err := healthServer.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{Service: service})
		if err != nil {
			t.Fatalf("health check %q: %v", service, err)
		}
		if response.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
			t.Fatalf("health status %q = %v, want SERVING", service, response.GetStatus())
		}
	}
}
