package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

type coverageDecider struct {
	handled []command.Type
}

func (d coverageDecider) Decide(_ any, _ command.Command, _ func() time.Time) command.Decision {
	return command.Decision{}
}

func (d coverageDecider) DeciderHandledCommands() []command.Type {
	return d.handled
}

type coverageModule struct {
	id      string
	version string
	decider module.Decider
}

func (m coverageModule) ID() string                                 { return m.id }
func (m coverageModule) Version() string                            { return m.version }
func (m coverageModule) RegisterCommands(_ *command.Registry) error { return nil }
func (m coverageModule) RegisterEvents(_ *event.Registry) error     { return nil }
func (m coverageModule) EmittableEventTypes() []event.Type          { return nil }
func (m coverageModule) Decider() module.Decider                    { return m.decider }
func (m coverageModule) Folder() module.Folder                      { return nil }
func (m coverageModule) StateFactory() module.StateFactory          { return nil }

func TestValidateDeciderCommandCoverage_ReportsMissingCommandsSorted(t *testing.T) {
	modules := module.NewRegistry()
	if err := modules.Register(coverageModule{
		id:      "sorter",
		version: "v1",
		decider: coverageDecider{handled: []command.Type{"sys.sorter.cmd3"}},
	}); err != nil {
		t.Fatalf("register module: %v", err)
	}

	commands := command.NewRegistry()
	for _, commandType := range []command.Type{
		"sys.sorter.cmd3",
		"sys.sorter.cmd2",
		"sys.sorter.cmd1",
	} {
		if err := commands.Register(command.Definition{Type: commandType, Owner: command.OwnerSystem}); err != nil {
			t.Fatalf("register command %s: %v", commandType, err)
		}
	}

	err := ValidateDeciderCommandCoverage(modules, commands)
	if err == nil {
		t.Fatal("expected missing-command coverage error")
	}
	want := "system commands missing decider handlers: sys.sorter.cmd1, sys.sorter.cmd2"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("ValidateDeciderCommandCoverage() error = %v, want sorted missing commands %q", err, want)
	}
}
