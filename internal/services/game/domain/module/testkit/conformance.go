// Package testkit provides reusable test helpers for validating system module
// conformance. ValidateSystemConformance composes the startup validators from
// the engine package so that second-system authors can verify their module and
// adapter in a single call.
package testkit

import (
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
}
