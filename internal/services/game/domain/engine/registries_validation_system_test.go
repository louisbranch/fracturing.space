package engine

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

// paramModule is a test module with configurable ID and version.
type paramModule struct {
	id      string
	version string
}

func (m paramModule) ID() string                                 { return m.id }
func (m paramModule) Version() string                            { return m.version }
func (m paramModule) RegisterCommands(_ *command.Registry) error { return nil }
func (m paramModule) RegisterEvents(_ *event.Registry) error     { return nil }
func (m paramModule) EmittableEventTypes() []event.Type          { return nil }
func (m paramModule) Decider() module.Decider                    { return nil }
func (m paramModule) Folder() module.Folder                      { return nil }
func (m paramModule) StateFactory() module.StateFactory          { return nil }

func TestValidateSystemMetadataConsistency_PassesForSystemEventsWithModules(t *testing.T) {
	events := event.NewRegistry()
	if err := events.Register(event.Definition{
		Type:  "sys.alpha.action.tested",
		Owner: event.OwnerSystem,
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	modules := module.NewRegistry()
	if err := modules.Register(paramModule{id: "alpha", version: "v1"}); err != nil {
		t.Fatalf("register module: %v", err)
	}

	if err := ValidateSystemMetadataConsistency(events, modules); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSystemMetadataConsistency_FailsForOrphanedSystemEvent(t *testing.T) {
	events := event.NewRegistry()
	if err := events.Register(event.Definition{
		Type:  "sys.orphan.action.tested",
		Owner: event.OwnerSystem,
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	// No modules registered — the system event has no matching module.
	modules := module.NewRegistry()

	err := ValidateSystemMetadataConsistency(events, modules)
	if err == nil {
		t.Fatal("expected error for orphaned system event")
	}
	if !strings.Contains(err.Error(), "sys.orphan.action.tested") {
		t.Fatalf("expected error to mention event type, got: %v", err)
	}
}

func TestValidateSystemMetadataConsistency_SkipsCoreEvents(t *testing.T) {
	events := event.NewRegistry()
	// Core events should be ignored.
	if err := events.Register(event.Definition{
		Type:  "campaign.created",
		Owner: event.OwnerCore,
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	modules := module.NewRegistry()

	// Should pass even with no modules, because core events are skipped.
	if err := ValidateSystemMetadataConsistency(events, modules); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
