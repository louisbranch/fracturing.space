package server

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	gamegrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
)

func TestDefaultSystemsBootstrapperBootstrapsAllChecks(t *testing.T) {
	systemRegistry := bridge.NewMetadataRegistry()
	adapters := bridge.NewAdapterRegistry()
	registries := engine.Registries{Commands: command.NewRegistry()}
	applier := projection.Applier{Adapters: adapters}
	bundle := &storageBundle{}
	order := make([]string, 0, 4)

	bootstrapper := defaultSystemsBootstrapper{
		buildSystemRegistry: func() (*bridge.MetadataRegistry, error) {
			order = append(order, "build")
			return systemRegistry, nil
		},
		validateSystemRegistration: func(modules []module.Module, metadata *bridge.MetadataRegistry, gotAdapters *bridge.AdapterRegistry) error {
			order = append(order, "parity")
			if metadata != systemRegistry {
				t.Fatal("expected built system registry to flow to parity validation")
			}
			if gotAdapters != adapters {
				t.Fatal("expected applier adapters to flow to parity validation")
			}
			return nil
		},
		validateSessionLockPolicy: func(registry *command.Registry) error {
			order = append(order, "lock")
			if registry != registries.Commands {
				t.Fatal("expected command registry to flow to session lock validation")
			}
			return nil
		},
		repairProjectionGaps: func(gotCtx context.Context, gotBundle *storageBundle, gotApplier projection.Applier) {
			order = append(order, "repair")
			if gotCtx == nil {
				t.Fatal("expected bootstrap context")
			}
			if gotBundle != bundle {
				t.Fatal("expected storage bundle to flow to projection repair")
			}
			if gotApplier.Adapters != adapters {
				t.Fatal("expected projection applier to flow to repair")
			}
		},
	}

	state, err := bootstrapper.Bootstrap(context.Background(), bundle, registries, applier)
	if err != nil {
		t.Fatalf("bootstrap systems: %v", err)
	}
	if state.systemRegistry != systemRegistry {
		t.Fatal("expected system registry in bootstrap state")
	}
	if !reflect.DeepEqual(order, []string{"build", "parity", "lock", "repair"}) {
		t.Fatalf("systems bootstrap order = %v", order)
	}
}

func TestDefaultSystemsBootstrapperWrapsFailures(t *testing.T) {
	bootstrapper := defaultSystemsBootstrapper{
		buildSystemRegistry: func() (*bridge.MetadataRegistry, error) {
			return nil, errors.New("boom")
		},
		validateSystemRegistration: func([]module.Module, *bridge.MetadataRegistry, *bridge.AdapterRegistry) error {
			return nil
		},
		validateSessionLockPolicy: func(*command.Registry) error {
			return nil
		},
		repairProjectionGaps: func(context.Context, *storageBundle, projection.Applier) {},
	}

	_, err := bootstrapper.Bootstrap(context.Background(), nil, engine.Registries{}, projection.Applier{})
	if err == nil {
		t.Fatal("expected wrapped systems bootstrap error")
	}
	if got := err.Error(); got != "build system registry: boom" {
		t.Fatalf("systems bootstrap error = %q, want wrapped build failure", got)
	}
}

func TestDefaultTransportBootstrapperBuildsServersAndRegistersServices(t *testing.T) {
	grpcServer := grpc.NewServer()
	healthServer := health.NewServer()
	systemRegistry := bridge.NewMetadataRegistry()
	wantConfig := aisessiongrant.Config{
		Issuer:   "issuer",
		Audience: "audience",
		HMACKey:  []byte("12345678901234567890123456789012"),
		TTL:      5 * time.Minute,
	}
	called := false

	bootstrapper := defaultTransportBootstrapper{
		newGRPCServer: func(gotBundle *storageBundle, gotEnv serverEnv) *grpc.Server {
			if gotBundle == nil {
				t.Fatal("expected storage bundle")
			}
			_ = gotEnv
			return grpcServer
		},
		newHealthServer: func() *health.Server {
			return healthServer
		},
		loadAISessionGrantConfig: func(now func() time.Time) (aisessiongrant.Config, error) {
			if now == nil {
				t.Fatal("expected clock loader to receive a now function")
			}
			return wantConfig, nil
		},
		registerServices: transportServiceRegistrarFunc(func(
			gotGRPC *grpc.Server,
			gotHealth *health.Server,
			_ gamegrpc.Stores,
			gotBundle *storageBundle,
			authClient authv1.AuthServiceClient,
			_ aiv1.AgentServiceClient,
			gotRegistry *bridge.MetadataRegistry,
			gotConfig aisessiongrant.Config,
		) error {
			called = true
			if gotGRPC != grpcServer {
				t.Fatal("expected grpc server from transport bootstrapper")
			}
			if gotHealth != healthServer {
				t.Fatal("expected health server from transport bootstrapper")
			}
			if gotBundle == nil {
				t.Fatal("expected storage bundle to flow to service registration")
			}
			if authClient != nil {
				t.Fatal("expected nil auth client in focused collaborator test")
			}
			if gotRegistry != systemRegistry {
				t.Fatal("expected system registry to flow to service registration")
			}
			if !reflect.DeepEqual(gotConfig, wantConfig) {
				t.Fatalf("session grant config = %#v, want %#v", gotConfig, wantConfig)
			}
			return nil
		}),
	}

	state, err := bootstrapper.Bootstrap(&storageBundle{}, serverEnv{}, gamegrpc.Stores{}, nil, nil, systemRegistry)
	if err != nil {
		t.Fatalf("bootstrap transport: %v", err)
	}
	if !called {
		t.Fatal("expected transport bootstrapper to register services")
	}
	if state.grpcServer != grpcServer {
		t.Fatal("expected grpc server in transport state")
	}
	if state.healthServer != healthServer {
		t.Fatal("expected health server in transport state")
	}
}

func TestDefaultTransportBootstrapperWrapsServiceRegistrationFailure(t *testing.T) {
	bootstrapper := defaultTransportBootstrapper{
		newGRPCServer: func(*storageBundle, serverEnv) *grpc.Server {
			return grpc.NewServer()
		},
		newHealthServer: func() *health.Server {
			return health.NewServer()
		},
		loadAISessionGrantConfig: func(func() time.Time) (aisessiongrant.Config, error) {
			return aisessiongrant.Config{}, nil
		},
		registerServices: transportServiceRegistrarFunc(func(
			*grpc.Server,
			*health.Server,
			gamegrpc.Stores,
			*storageBundle,
			authv1.AuthServiceClient,
			aiv1.AgentServiceClient,
			*bridge.MetadataRegistry,
			aisessiongrant.Config,
		) error {
			return errors.New("register failed")
		}),
	}

	_, err := bootstrapper.Bootstrap(&storageBundle{}, serverEnv{}, gamegrpc.Stores{}, nil, nil, nil)
	if err == nil {
		t.Fatal("expected wrapped transport bootstrap error")
	}
	if got := err.Error(); got != "register gRPC services: register failed" {
		t.Fatalf("transport bootstrap error = %q, want wrapped registration failure", got)
	}
}
