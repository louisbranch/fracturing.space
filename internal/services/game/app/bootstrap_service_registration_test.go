package app

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestBuildServiceDescriptors_ExcludesLegacyInteractionTransport(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(filename), "bootstrap_service_registration.go")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	source := string(content)
	for _, legacyMarker := range []string{
		"Register" + "Commun" + "icationServiceServer",
		"New" + "Commun" + "icationService",
	} {
		if strings.Contains(source, legacyMarker) {
			t.Fatalf("%s still references removed communication registration", filepath.Base(path))
		}
	}
}

func TestBuildServiceDescriptors_DoesNotReintroduceTransportRegistrationBag(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(filename), "bootstrap_service_registration.go")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	source := string(content)
	for _, marker := range []string{
		"type transportRegistrationDeps struct",
		"newTransportRegistrationDeps(",
	} {
		if strings.Contains(source, marker) {
			t.Fatalf("%s unexpectedly contains removed transport-wide registration marker %q", filepath.Base(path), marker)
		}
	}
}

func TestBuildServiceDescriptors_RegistersCampaignAIOrchestrationService(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(filename), "bootstrap_service_builders.go")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	source := string(content)
	for _, marker := range []string{
		`healthService: "game.v1.CampaignAIOrchestrationService"`,
		"RegisterCampaignAIOrchestrationServiceServer",
		"NewCampaignAIOrchestrationService(gamegrpc.CampaignAIOrchestrationDeps{",
		"SessionInteraction: deps.sessionInteraction",
	} {
		if !strings.Contains(source, marker) {
			t.Fatalf("%s missing %q", filepath.Base(path), marker)
		}
	}
}
