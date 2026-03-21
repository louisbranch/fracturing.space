package app

import (
	gamegrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/observability/audit"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// storesConstructionSources keeps root store construction scoped to the exact
// startup-owned collaborators needed for phase 4.
type storesConstructionSources struct {
	projectionStore storage.ProjectionStore
	systemStores    gamegrpc.SystemStores
	eventStore      storage.EventStore
	auditStore      storage.AuditEventStore
	contentStore    contentstore.DaggerheartContentReadStore
	runtimeConfig   gamegrpc.StoresRuntimeConfig
}

type constructedStoreGroups struct {
	projection     gamegrpc.ProjectionStores
	system         gamegrpc.SystemStores
	infrastructure gamegrpc.InfrastructureStores
	content        gamegrpc.ContentStores
	runtime        gamegrpc.RuntimeStores
}

// buildStoreGroupsFromSources assembles the root game transport concerns from
// exact startup-owned sources instead of one root store bag.
func buildStoreGroupsFromSources(sources storesConstructionSources) constructedStoreGroups {
	return constructedStoreGroups{
		projection: gamegrpc.NewProjectionStores(gamegrpc.StoresProjectionConfig{
			ProjectionStore: sources.projectionStore,
			SystemStores:    sources.systemStores,
		}),
		system: sources.systemStores,
		infrastructure: gamegrpc.NewInfrastructureStores(
			sources.projectionStore,
			gamegrpc.StoresInfrastructureConfig{
				EventStore: sources.eventStore,
				AuditStore: sources.auditStore,
			},
		),
		content: gamegrpc.NewContentStores(gamegrpc.StoresContentConfig{
			ContentStore: sources.contentStore,
		}),
		runtime: gamegrpc.NewRuntimeStores(sources.runtimeConfig, sources.auditStore),
	}
}

// applierConstructionSources keeps root applier construction scoped to the
// exact projection-owned collaborators needed for phase 4.
type applierConstructionSources struct {
	projectionStore storage.ProjectionStore
	systemStores    gamegrpc.SystemStores
	auditStore      storage.AuditEventStore
	events          *event.Registry
}

// buildApplierFromSources assembles the root projection applier from exact
// startup-owned sources rather than an inline config literal inside bootstrap.
func buildApplierFromSources(sources applierConstructionSources) (projection.Applier, error) {
	return projection.NewApplier(projection.ApplierConfig{
		Stores:       projection.StoreGroupsFromBundle(sources.projectionStore),
		SystemStores: sources.systemStores,
		AuditPolicy:  newAuditPolicy(sources.auditStore),
		Events:       sources.events,
	})
}

func newAuditPolicy(store storage.AuditEventStore) audit.Policy {
	if store == nil {
		return audit.DisabledPolicy()
	}
	return audit.EnabledPolicy(store)
}
