package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/integrity"
	storagesqlite "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite"
)

func New(port int) (*Server, error) {
	return NewWithAddr(fmt.Sprintf(":%d", port))
}

// NewWithAddr creates a configured game server listening on the provided address.
func NewWithAddr(addr string) (*Server, error) {
	return newServerBootstrap().NewWithAddr(addr)
}

func buildProjectionApplyOutboxApply(projectionStore *storagesqlite.Store, eventRegistry *event.Registry) func(context.Context, event.Event) error {
	if projectionStore == nil {
		return nil
	}
	// Build a base adapter registry once at closure creation. Per-transaction
	// callbacks rebind it with the transaction-scoped store so each adapter
	// operates within the exactly-once transaction boundary.
	baseAdapters, err := systemmanifest.AdapterRegistry(systemmanifest.ProjectionStores{Daggerheart: projectionStore})
	if err != nil {
		log.Printf("build base adapter registry: %v", err)
		return nil
	}
	return func(ctx context.Context, evt event.Event) error {
		_, err := projectionStore.ApplyProjectionEventExactlyOnce(
			ctx,
			evt,
			func(applyCtx context.Context, applyEvt event.Event, txStore *storagesqlite.Store) error {
				systemAdapters, err := systemmanifest.RebindAdapterRegistry(baseAdapters, systemmanifest.ProjectionStores{Daggerheart: txStore})
				if err != nil {
					return fmt.Errorf("rebind projection system adapter registry: %w", err)
				}
				txApplier := projection.Applier{
					Events:           eventRegistry,
					Campaign:         txStore,
					Character:        txStore,
					CampaignFork:     txStore,
					ClaimIndex:       txStore,
					Invite:           txStore,
					Participant:      txStore,
					Session:          txStore,
					SessionGate:      txStore,
					SessionSpotlight: txStore,
					Adapters:         systemAdapters,
				}
				return txApplier.Apply(applyCtx, applyEvt)
			},
		)
		return err
	}
}

func buildSystemRegistry() (*bridge.Registry, error) {
	registry := bridge.NewRegistry()
	for _, gameSystem := range registeredMetadataSystems() {
		if err := registry.Register(gameSystem); err != nil {
			return nil, fmt.Errorf("register system %s@%s: %w", gameSystem.ID(), gameSystem.Version(), err)
		}
	}
	return registry, nil
}

// buildProjectionRegistries validates projection coverage using pre-built
// registries. This is a startup-time safety check that ensures every core
// projection-and-replay event has a handler, no dead projection handlers
// exist, and every system-emittable event has an adapter handler.
func buildProjectionRegistries(registries engine.Registries, adapters *bridge.AdapterRegistry) (*event.Registry, error) {
	handledTypes := projection.ProjectionHandledTypes()
	if err := engine.ValidateProjectionRegistries(registries.Events, registries.Systems, adapters, handledTypes); err != nil {
		return nil, err
	}
	return registries.Events, nil
}

// openStorageBundle opens event, projection, and content databases.
func openStorageBundle(srvEnv serverEnv, eventRegistry *event.Registry) (*storageBundle, error) {
	eventStore, err := openEventStore(srvEnv.EventsDBPath, srvEnv.ProjectionApplyOutboxEnabled, eventRegistry)
	if err != nil {
		return nil, err
	}
	projStore, err := openProjectionStore(srvEnv.ProjectionsDBPath)
	if err != nil {
		_ = eventStore.Close()
		return nil, err
	}
	contentStore, err := openContentStore(srvEnv.ContentDBPath)
	if err != nil {
		_ = eventStore.Close()
		_ = projStore.Close()
		return nil, err
	}
	return &storageBundle{
		events:      eventStore,
		projections: projStore,
		content:     contentStore,
	}, nil
}

// dialAuthGRPC opens an authenticated gRPC client to auth service.
func dialAuthGRPC(ctx context.Context, authAddr string) (authGRPCClients, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	logf := func(format string, args ...any) {
		log.Printf("auth %s", fmt.Sprintf(format, args...))
	}
	conn, err := platformgrpc.DialWithHealth(
		ctx,
		nil,
		authAddr,
		timeouts.GRPCDial,
		logf,
		platformgrpc.DefaultClientDialOptions()...,
	)
	if err != nil {
		var dialErr *platformgrpc.DialError
		if errors.As(err, &dialErr) {
			if dialErr.Stage == platformgrpc.DialStageHealth {
				return authGRPCClients{}, fmt.Errorf("auth gRPC health check failed for %s: %w", authAddr, dialErr.Err)
			}
			return authGRPCClients{}, fmt.Errorf("dial auth gRPC %s: %w", authAddr, dialErr.Err)
		}
		return authGRPCClients{}, fmt.Errorf("dial auth gRPC %s: %w", authAddr, err)
	}
	return authGRPCClients{
		conn:       conn,
		authClient: authv1.NewAuthServiceClient(conn),
	}, nil
}

// openEventStore opens the immutable event store and verifies chain integrity on boot.
func openEventStore(path string, projectionApplyOutboxEnabled bool, eventRegistry *event.Registry) (*storagesqlite.Store, error) {
	if err := ensureDir(path); err != nil {
		return nil, err
	}
	keyring, err := integrity.KeyringFromEnv()
	if err != nil {
		return nil, err
	}
	store, err := storagesqlite.OpenEvents(
		path,
		keyring,
		eventRegistry,
		storagesqlite.WithProjectionApplyOutboxEnabled(projectionApplyOutboxEnabled),
	)
	if err != nil {
		return nil, fmt.Errorf("open events store: %w", err)
	}
	if err := store.VerifyEventIntegrity(context.Background()); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("verify event integrity: %w", err)
	}
	return store, nil
}

// openProjectionStore opens the materialized views database.
func openProjectionStore(path string) (*storagesqlite.Store, error) {
	if err := ensureDir(path); err != nil {
		return nil, err
	}
	store, err := storagesqlite.OpenProjections(path)
	if err != nil {
		return nil, fmt.Errorf("open projections store: %w", err)
	}
	return store, nil
}

// openContentStore opens the content reference database.
func openContentStore(path string) (*storagesqlite.Store, error) {
	if err := ensureDir(path); err != nil {
		return nil, err
	}
	store, err := storagesqlite.OpenContent(path)
	if err != nil {
		return nil, fmt.Errorf("open content store: %w", err)
	}
	return store, nil
}

// repairProjectionGaps detects and auto-repairs projection gaps on startup by
// replaying missing events from the journal into the projection applier.
func repairProjectionGaps(bundle *storageBundle, applier projection.Applier) {
	if bundle == nil || bundle.projections == nil || bundle.events == nil {
		return
	}
	results, err := projection.RepairProjectionGaps(context.Background(), bundle.projections, bundle.events, applier)
	if err != nil {
		log.Printf("projection gap repair: %v", err)
		return
	}
	for _, r := range results {
		log.Printf("projection gap repaired: campaign %s replayed %d events", r.CampaignID, r.EventsReplayed)
	}
}

// ensureDir creates parent paths for sqlite files so startup can create DB files.
func ensureDir(path string) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create storage dir: %w", err)
		}
	}
	return nil
}
