package app

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	gamegrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
)

func TestDefaultSystemsBootstrapperBootstrapsAllChecks(t *testing.T) {
	systemRegistry := bridge.NewMetadataRegistry()
	systemRegistration := loadSystemRegistrationSnapshot()
	adapters := bridge.NewAdapterRegistry()
	registries := engine.Registries{Commands: command.NewRegistry()}
	applier := projection.Applier{Adapters: adapters}
	bundle := &storageBundle{}
	order := make([]string, 0, 4)

	bootstrapper := defaultSystemsBootstrapper{
		buildSystemRegistry: func(got systemRegistrationSnapshot) (*bridge.MetadataRegistry, error) {
			order = append(order, "build")
			if !reflect.DeepEqual(got.modulesCopy(), systemRegistration.modulesCopy()) {
				t.Fatal("expected system registration snapshot to flow to system registry builder")
			}
			return systemRegistry, nil
		},
		validateSystemRegistration: func(modules []module.Module, metadata *bridge.MetadataRegistry, gotAdapters *bridge.AdapterRegistry) error {
			order = append(order, "parity")
			if !reflect.DeepEqual(modules, systemRegistration.modulesCopy()) {
				t.Fatal("expected system registration modules to flow to parity validation")
			}
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

	state, err := bootstrapper.Bootstrap(context.Background(), bundle, systemRegistration, registries, applier)
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
		buildSystemRegistry: func(systemRegistrationSnapshot) (*bridge.MetadataRegistry, error) {
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

	_, err := bootstrapper.Bootstrap(context.Background(), nil, loadSystemRegistrationSnapshot(), engine.Registries{}, projection.Applier{})
	if err == nil {
		t.Fatal("expected wrapped systems bootstrap error")
	}
	if got := err.Error(); got != "build system registry: boom" {
		t.Fatalf("systems bootstrap error = %q, want wrapped build failure", got)
	}
}

func TestDefaultTransportBootstrapperBuildsServersAndRegistersServices(t *testing.T) {
	bundle := &storageBundle{}
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
			_ daggerheartRegistrationDeps,
			gotCampaign campaignRegistrationDeps,
			_ sessionRegistrationDeps,
			gotInfrastructure infrastructureRegistrationDeps,
			gotConfig aisessiongrant.Config,
		) error {
			called = true
			if gotGRPC != grpcServer {
				t.Fatal("expected grpc server from transport bootstrapper")
			}
			if gotHealth != healthServer {
				t.Fatal("expected health server from transport bootstrapper")
			}
			if gotCampaign.authClient != nil {
				t.Fatal("expected nil auth client in focused collaborator test")
			}
			if gotInfrastructure.bundle != bundle {
				t.Fatal("expected infrastructure deps to keep the startup storage bundle")
			}
			if gotInfrastructure.systemRegistry != systemRegistry {
				t.Fatal("expected system registry to flow through infrastructure deps")
			}
			if !reflect.DeepEqual(gotConfig, wantConfig) {
				t.Fatalf("session grant config = %#v, want %#v", gotConfig, wantConfig)
			}
			return nil
		}),
	}

	state, err := bootstrapper.Bootstrap(
		bundle,
		serverEnv{},
		daggerheartRegistrationDeps{},
		campaignRegistrationDeps{},
		sessionRegistrationDeps{},
		infrastructureRegistrationDeps{bundle: bundle, systemRegistry: systemRegistry},
	)
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
			daggerheartRegistrationDeps,
			campaignRegistrationDeps,
			sessionRegistrationDeps,
			infrastructureRegistrationDeps,
			aisessiongrant.Config,
		) error {
			return errors.New("register failed")
		}),
	}

	_, err := bootstrapper.Bootstrap(
		&storageBundle{},
		serverEnv{},
		daggerheartRegistrationDeps{},
		campaignRegistrationDeps{},
		sessionRegistrationDeps{},
		infrastructureRegistrationDeps{},
	)
	if err == nil {
		t.Fatal("expected wrapped transport bootstrap error")
	}
	if got := err.Error(); got != "register gRPC services: register failed" {
		t.Fatalf("transport bootstrap error = %q, want wrapped registration failure", got)
	}
}

func TestAttachDependencyClientsSetsSocialOnContentStores(t *testing.T) {
	mc, err := platformgrpc.NewManagedConn(context.Background(), platformgrpc.ManagedConnConfig{
		Name: "social-test",
		Addr: "dns:///127.0.0.1:1",
		Mode: platformgrpc.ModeOptional,
	})
	if err != nil {
		t.Fatalf("new managed conn: %v", err)
	}
	defer func() { _ = mc.Close() }()

	contentStores := &gamegrpc.ContentStores{}

	attachDependencyClients(contentStores, dependencyConns{social: mc})

	if contentStores.Social == nil {
		t.Fatal("expected social client to be attached")
	}
}

func TestAssertWatermarkStoreConfigured(t *testing.T) {
	t.Run("passes when outbox disabled", func(t *testing.T) {
		err := assertWatermarkStoreConfigured(
			serverEnv{ProjectionApplyOutboxEnabled: false},
			gamegrpc.InfrastructureStores{Watermarks: nil},
		)
		if err != nil {
			t.Fatalf("expected nil error when outbox disabled, got %v", err)
		}
	})

	t.Run("passes when outbox enabled and watermarks configured", func(t *testing.T) {
		err := assertWatermarkStoreConfigured(
			serverEnv{ProjectionApplyOutboxEnabled: true},
			gamegrpc.InfrastructureStores{Watermarks: stubWatermarkStore{}},
		)
		if err != nil {
			t.Fatalf("expected nil error with watermarks configured, got %v", err)
		}
	})

	t.Run("fails when outbox enabled and watermarks nil", func(t *testing.T) {
		err := assertWatermarkStoreConfigured(
			serverEnv{ProjectionApplyOutboxEnabled: true},
			gamegrpc.InfrastructureStores{Watermarks: nil},
		)
		if err == nil {
			t.Fatal("expected error when outbox enabled but watermarks nil")
		}
	})
}

type stubWatermarkStore struct {
	storage.ProjectionWatermarkStore
}
