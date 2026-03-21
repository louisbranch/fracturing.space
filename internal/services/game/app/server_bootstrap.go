package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/integrity"
	sqlitecoreprojection "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/coreprojection"
	sqlitedaggerheartcontent "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/daggerheartcontent"
	sqliteeventjournal "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/eventjournal"
)

func New(port int) (*Server, error) {
	return NewContext(context.Background(), port)
}

// NewContext creates a configured game server listening on the provided port.
func NewContext(ctx context.Context, port int) (*Server, error) {
	return NewWithAddrContext(ctx, fmt.Sprintf(":%d", port))
}

// NewWithAddr creates a configured game server listening on the provided address.
func NewWithAddr(addr string) (*Server, error) {
	return NewWithAddrContext(context.Background(), addr)
}

// NewWithAddrContext creates a configured game server listening on the provided address.
func NewWithAddrContext(ctx context.Context, addr string) (*Server, error) {
	return newServerBootstrap().NewWithAddr(ctx, addr)
}

func buildProjectionApplyOutboxApply(projectionStore storage.ProjectionApplyExactlyOnceStore, eventRegistry *event.Registry) (func(context.Context, event.Event) error, error) {
	if projectionStore == nil {
		return nil, nil
	}
	// Build a base adapter registry once at closure creation. Per-transaction
	// callbacks rebind it with the transaction-scoped store so each adapter
	// operates within the exactly-once transaction boundary.
	baseAdapters, err := systemmanifest.AdapterRegistry(projectionStore)
	if err != nil {
		return nil, fmt.Errorf("build base adapter registry: %w", err)
	}
	return func(ctx context.Context, evt event.Event) error {
		_, err := projectionStore.ApplyProjectionEventExactlyOnce(
			ctx,
			evt,
			func(applyCtx context.Context, applyEvt event.Event, txStore storage.ProjectionApplyTxStore) error {
				systemAdapters, err := systemmanifest.RebindAdapterRegistry(baseAdapters, txStore)
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
					Scene:            txStore,
					SceneCharacter:   txStore,
					SceneGate:        txStore,
					SceneSpotlight:   txStore,
					Adapters:         systemAdapters,
					Watermarks:       txStore,
				}
				return txApplier.Apply(applyCtx, applyEvt)
			},
		)
		return err
	}, nil
}

func buildSystemRegistry() (*bridge.MetadataRegistry, error) {
	registry := bridge.NewMetadataRegistry()
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
func openStorageBundle(ctx context.Context, srvEnv serverEnv, eventRegistry *event.Registry) (*storageBundle, error) {
	eventStore, err := openEventStore(ctx, srvEnv.EventsDBPath, srvEnv.ProjectionApplyOutboxEnabled, eventRegistry)
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

// openEventStore opens the immutable event store and verifies chain integrity on boot.
func openEventStore(ctx context.Context, path string, projectionApplyOutboxEnabled bool, eventRegistry *event.Registry) (eventBackend, error) {
	if err := ensureDir(path); err != nil {
		return nil, err
	}
	keyring, err := integrity.KeyringFromEnv()
	if err != nil {
		return nil, err
	}
	store, err := sqliteeventjournal.Open(
		path,
		keyring,
		eventRegistry,
		sqliteeventjournal.WithProjectionApplyOutboxEnabled(projectionApplyOutboxEnabled),
	)
	if err != nil {
		return nil, fmt.Errorf("open events store: %w", err)
	}
	if err := store.VerifyEventIntegrity(ctx); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("verify event integrity: %w", err)
	}
	return store, nil
}

// openProjectionStore opens the materialized views database.
func openProjectionStore(path string) (projectionBackend, error) {
	if err := ensureDir(path); err != nil {
		return nil, err
	}
	store, err := sqlitecoreprojection.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open projections store: %w", err)
	}
	return store, nil
}

// openContentStore opens the content reference database.
func openContentStore(path string) (contentBackend, error) {
	if err := ensureDir(path); err != nil {
		return nil, err
	}
	store, err := sqlitedaggerheartcontent.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open content store: %w", err)
	}
	return store, nil
}

// repairProjectionGaps detects and auto-repairs projection gaps on startup by
// replaying missing events from the journal into the projection applier.
//
// Errors are logged but do not prevent startup. This is intentional: gap repair
// is best-effort hardening. A repair failure (e.g., corrupt event) should not
// block the server from serving traffic with stale-but-usable projection state.
// The outbox worker and inline apply handle new events regardless.
func repairProjectionGaps(ctx context.Context, bundle *storageBundle, applier projection.Applier) {
	if bundle == nil || bundle.projections == nil || bundle.events == nil {
		return
	}
	results, err := projection.RepairProjectionGaps(ctx, bundle.projections, bundle.events, applier)
	if err != nil {
		slog.Warn("projection gap repair failed", "error", err)
		return
	}
	for _, r := range results {
		slog.Info("projection gap repaired", "campaign_id", r.CampaignID, "events_replayed", r.EventsReplayed)
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
