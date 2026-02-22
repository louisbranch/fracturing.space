package aggregate

import (
	"errors"
	"sync"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

// Folder folds events into aggregate state.
//
// The folder is where the domain boundary stays deterministic:
// each event type updates exactly one aggregate slice and is replayed
// identically whether during request execution or historical reconstruction.
// Named "Folder" (not "Applier") to distinguish pure state folds from
// projection.Applier, which performs side-effecting I/O writes to stores.
//
// Core domain dispatch is declarative: coreFoldEntries() defines the mapping
// from event types to fold functions. Adding a new core domain requires only
// adding an entry in fold_registry.go.
type Folder struct {
	// Events provides event definitions so the folder can skip audit-only
	// events that do not affect aggregate state.
	Events *event.Registry
	// SystemRegistry routes system events to their module-specific folder.
	SystemRegistry *module.Registry

	// foldIndex is lazily built on first Fold to avoid dispatch into fold
	// functions that cannot possibly handle the event type.
	foldOnce  sync.Once
	foldIndex map[event.Type]func(*State, event.Event) error
}

// initFoldIndex builds a type-to-handler lookup from the declarative fold entries.
func (a *Folder) initFoldIndex() {
	a.foldOnce.Do(func() {
		entries := coreFoldEntries()
		a.foldIndex = make(map[event.Type]func(*State, event.Event) error)
		for _, entry := range entries {
			fn := entry.fold
			for _, t := range entry.types() {
				a.foldIndex[t] = fn
			}
		}
	})
}

// FoldDispatchedTypes returns the union of all event types wired into the
// applier's fold dispatch index. ValidateAggregateFoldDispatch uses this to
// verify that every type declared in CoreDomains().FoldHandledTypes actually
// reaches a fold function at runtime.
func (a *Folder) FoldDispatchedTypes() []event.Type {
	a.initFoldIndex()
	types := make([]event.Type, 0, len(a.foldIndex))
	for t := range a.foldIndex {
		types = append(types, t)
	}
	return types
}

// Fold applies a single event to aggregate state.
//
// The function only mutates aggregate state through fold functions so state
// transitions remain visible in one place per subdomain and replay behavior matches
// request-time behavior.
func (a *Folder) Fold(state any, evt event.Event) (any, error) {
	// Skip audit-only events: they do not affect aggregate state and should
	// not be passed to fold functions.
	if a.Events != nil {
		if def, ok := a.Events.Definition(evt.Type); ok && def.Intent == event.IntentAuditOnly {
			current, err := AssertState[State](state)
			if err != nil {
				return State{}, err
			}
			return current, nil
		}
	}

	a.initFoldIndex()

	current, err := AssertState[State](state)
	if err != nil {
		return State{}, err
	}

	if fn, ok := a.foldIndex[evt.Type]; ok {
		if err := fn(&current, evt); err != nil {
			return current, err
		}
	}

	if evt.SystemID != "" || evt.SystemVersion != "" {
		if current.Systems == nil {
			current.Systems = make(map[module.Key]any)
		}
		if evt.SystemID == "" || evt.SystemVersion == "" {
			return current, errors.New("system id and version are required")
		}
		registry := a.SystemRegistry
		if registry == nil {
			return current, errors.New("system registry is required")
		}
		key := module.Key{ID: evt.SystemID, Version: evt.SystemVersion}
		systemState := current.Systems[key]
		mod := registry.Get(evt.SystemID, evt.SystemVersion)
		if mod != nil && systemState == nil {
			if factory := mod.StateFactory(); factory != nil {
				seed, err := factory.NewSnapshotState(evt.CampaignID)
				if err != nil {
					return current, err
				}
				systemState = seed
			}
		}
		updated, err := module.RouteEvent(registry, systemState, evt)
		if err != nil {
			return current, err
		}
		current.Systems[key] = updated
	}

	return current, nil
}
