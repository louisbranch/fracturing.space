package engine

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func TestSessionLifecycleSystemReadinessGuardRails(t *testing.T) {
	t.Run("nil registry", func(t *testing.T) {
		checker := sessionLifecycle{}.systemReadiness(aggregate.State{})
		if checker != nil {
			t.Fatal("expected nil checker when systems registry is missing")
		}
	})

	t.Run("blank system id", func(t *testing.T) {
		checker := sessionLifecycle{systems: module.NewRegistry()}.systemReadiness(aggregate.State{})
		if checker != nil {
			t.Fatal("expected nil checker when campaign system id is blank")
		}
	})

	t.Run("module without readiness checker", func(t *testing.T) {
		systems := module.NewRegistry()
		if err := systems.Register(stubModuleWithoutReadiness{}); err != nil {
			t.Fatalf("register module: %v", err)
		}
		checker := sessionLifecycle{systems: systems}.systemReadiness(aggregate.State{
			Campaign: aggregateCampaignState("stub"),
		})
		if checker != nil {
			t.Fatal("expected nil checker when module does not implement readiness")
		}
	})

	t.Run("missing character", func(t *testing.T) {
		systems := module.NewRegistry()
		if err := systems.Register(stubReadinessModule{ready: true, reason: "ready"}); err != nil {
			t.Fatalf("register module: %v", err)
		}
		checker := sessionLifecycle{systems: systems}.systemReadiness(aggregate.State{
			Campaign: aggregateCampaignState("stub"),
			Systems: map[module.Key]any{
				{ID: "stub", Version: "1.0.0"}: struct{}{},
			},
		})
		if checker == nil {
			t.Fatal("expected readiness checker")
		}
		ready, reason := checker("missing")
		if ready || reason != "character is missing" {
			t.Fatalf("checker result = (%t, %q), want (false, %q)", ready, reason, "character is missing")
		}
	})

	t.Run("delegates to module checker", func(t *testing.T) {
		systems := module.NewRegistry()
		if err := systems.Register(stubReadinessModule{ready: false, reason: "class is required"}); err != nil {
			t.Fatalf("register module: %v", err)
		}
		checker := sessionLifecycle{systems: systems}.systemReadiness(aggregate.State{
			Campaign: aggregateCampaignState("stub"),
			Characters: map[ids.CharacterID]character.State{
				"char-1": {CharacterID: "char-1", Created: true},
			},
			Systems: map[module.Key]any{
				{ID: "stub", Version: "1.0.0"}: struct{}{},
			},
		})
		if checker == nil {
			t.Fatal("expected readiness checker")
		}
		ready, reason := checker("char-1")
		if ready || reason != "class is required" {
			t.Fatalf("checker result = (%t, %q), want (false, %q)", ready, reason, "class is required")
		}
	})
}

func aggregateCampaignState(systemID string) campaign.State {
	return campaign.State{GameSystem: campaign.GameSystem(systemID)}
}

type stubModuleWithoutReadiness struct{}

func (stubModuleWithoutReadiness) ID() string                               { return "stub" }
func (stubModuleWithoutReadiness) Version() string                          { return "1.0.0" }
func (stubModuleWithoutReadiness) RegisterCommands(*command.Registry) error { return nil }
func (stubModuleWithoutReadiness) RegisterEvents(*event.Registry) error     { return nil }
func (stubModuleWithoutReadiness) EmittableEventTypes() []event.Type        { return nil }
func (stubModuleWithoutReadiness) Decider() module.Decider                  { return nil }
func (stubModuleWithoutReadiness) Folder() module.Folder                    { return nil }
func (stubModuleWithoutReadiness) StateFactory() module.StateFactory        { return nil }
