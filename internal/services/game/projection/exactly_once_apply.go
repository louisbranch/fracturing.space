package projection

import (
	"context"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
)

// BuildExactlyOnceApply constructs the projection apply callback used by the
// projection outbox worker. The concrete store source is responsible for both
// exactly-once transaction application and system adapter extraction.
func BuildExactlyOnceApply(
	storeSource ExactlyOnceStore,
	eventRegistry *event.Registry,
) (func(context.Context, event.Event) error, error) {
	if storeSource == nil {
		return nil, nil
	}

	baseAdapters, err := systemmanifest.AdapterRegistry(systemmanifest.ProjectionStoresFromSource(storeSource))
	if err != nil {
		return nil, fmt.Errorf("build base adapter registry: %w", err)
	}

	return func(ctx context.Context, evt event.Event) error {
		_, err := storeSource.ApplyProjectionEventExactlyOnce(
			ctx,
			evt,
			func(applyCtx context.Context, applyEvt event.Event, txStore TxStoreBundle) error {
				systemAdapters, err := systemmanifest.RebindAdapterRegistry(baseAdapters, systemmanifest.ProjectionStoresFromSource(txStore))
				if err != nil {
					return fmt.Errorf("rebind projection system adapter registry: %w", err)
				}
				return NewBoundApplier(BoundApplierConfig{
					Stores:   StoreGroupsFromBundle(txStore),
					Events:   eventRegistry,
					Adapters: systemAdapters,
				}).Apply(applyCtx, applyEvt)
			},
		)
		return err
	}, nil
}
