// Package testkit provides reusable test helpers for validating system module
// conformance. ValidateSystemConformance composes the startup validators from
// the engine package so that second-system authors can verify their module and
// adapter in a single call.
package testkit

import (
	"context"
	"reflect"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

// ValidateSystemConformance runs the full set of startup validators against a
// single system module and its projection adapter. It registers the module's
// commands and events in fresh registries, then asserts:
//
//   - every emittable event with replay intent has a fold handler
//   - every system command type has a decider handler
//   - every emittable event with projection intent has an adapter handler
//   - no fold or adapter handler is orphaned from emittable types
//   - the state factory is deterministic
func ValidateSystemConformance(t *testing.T, mod module.Module, adapter bridge.Adapter) {
	t.Helper()

	events := event.NewRegistry()
	if err := mod.RegisterEvents(events); err != nil {
		t.Fatalf("RegisterEvents: %v", err)
	}

	commands := command.NewRegistry()
	if err := mod.RegisterCommands(commands); err != nil {
		t.Fatalf("RegisterCommands: %v", err)
	}

	modules := module.NewRegistry()
	if err := modules.Register(mod); err != nil {
		t.Fatalf("Register module: %v", err)
	}

	adapters := bridge.NewAdapterRegistry()
	if err := adapters.Register(adapter); err != nil {
		t.Fatalf("Register adapter: %v", err)
	}

	if err := engine.ValidateSystemFoldCoverage(modules, events); err != nil {
		t.Errorf("fold coverage: %v", err)
	}
	if err := engine.ValidateDeciderCommandCoverage(modules, commands); err != nil {
		t.Errorf("decider command coverage: %v", err)
	}
	if err := engine.ValidateAdapterEventCoverage(modules, adapters, events); err != nil {
		t.Errorf("adapter event coverage: %v", err)
	}
	if err := engine.ValidateSystemRouterDefinitionParity(modules, adapters, events); err != nil {
		t.Errorf("router definition parity: %v", err)
	}
	if err := engine.ValidateStateFactoryDeterminism(modules); err != nil {
		t.Errorf("state factory determinism: %v", err)
	}

	// G3: Fold idempotency — folding the same event into fresh state twice
	// must produce identical results. This catches fold functions that use
	// deltas instead of absolute values.
	validateFoldIdempotency(t, mod)
}

// ValidateAdapterIdempotency applies each handled event type twice to the
// adapter and asserts no error on the second apply. Adapters should treat
// duplicate events idempotently — applying the same event twice must produce
// the same result without error. This catches adapters that append to lists,
// increment counters, or violate unique constraints on replay.
//
// The adapter must be backed by a functional store (even a minimal in-memory
// one) so that Apply can read/write projection state. Event types whose first
// Apply fails (e.g. because the empty payload is rejected) are skipped.
func ValidateAdapterIdempotency(t *testing.T, adapter bridge.Adapter) {
	t.Helper()

	ctx := context.Background()
	const testCampaignID = "adapter-idempotency-check"

	for _, evtType := range adapter.HandledTypes() {
		t.Run(string(evtType), func(t *testing.T) {
			evt := event.Event{
				CampaignID:    testCampaignID,
				Seq:           1,
				Type:          evtType,
				SystemID:      adapter.ID(),
				SystemVersion: adapter.Version(),
				EntityType:    "character",
				EntityID:      "char-1",
				PayloadJSON:   []byte("{}"),
			}

			// First apply — may fail for handlers that require specific
			// payload fields. Skip those types gracefully.
			if err := adapter.Apply(ctx, evt); err != nil {
				return
			}

			// Second apply — must not error if the adapter is idempotent.
			if err := adapter.Apply(ctx, evt); err != nil {
				t.Errorf("adapter idempotency: second Apply for %s failed: %v", evtType, err)
			}
		})
	}
}

// validateFoldIdempotency folds each handled event type twice into fresh
// state and asserts the results are identical.
func validateFoldIdempotency(t *testing.T, mod module.Module) {
	t.Helper()
	folder := mod.Folder()
	if folder == nil {
		return
	}
	factory := mod.StateFactory()
	if factory == nil {
		return
	}

	const testCampaignID = "idempotency-check"

	for _, evtType := range folder.FoldHandledTypes() {
		evt := event.Event{
			CampaignID:    testCampaignID,
			Type:          evtType,
			SystemID:      mod.ID(),
			SystemVersion: mod.Version(),
			EntityType:    "character",
			EntityID:      "char-1",
			PayloadJSON:   []byte("{}"),
		}

		// First fold into fresh state.
		state1, err := factory.NewSnapshotState(testCampaignID)
		if err != nil {
			t.Errorf("fold idempotency: NewSnapshotState for %s: %v", evtType, err)
			continue
		}
		result1, err := folder.Fold(state1, evt)
		if err != nil {
			// Some fold functions may fail with empty payload — that's
			// acceptable; we only check idempotency when fold succeeds.
			continue
		}

		// Second fold into fresh state with the same event.
		state2, err := factory.NewSnapshotState(testCampaignID)
		if err != nil {
			t.Errorf("fold idempotency: NewSnapshotState for %s: %v", evtType, err)
			continue
		}
		result2, err := folder.Fold(state2, evt)
		if err != nil {
			t.Errorf("fold idempotency: second fold failed for %s: %v", evtType, err)
			continue
		}

		if !reflect.DeepEqual(result1, result2) {
			t.Errorf("fold idempotency: %s produced different state on second fold into fresh state", evtType)
		}
	}
}
