package engine

import (
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

// CoreDecider is the top-level decider for core and system commands.
//
// It keeps the write-path entrypoint stable while delegating core-owned routing
// and system-owned dispatch to explicit collaborators.
type CoreDecider struct {
	systemCommands systemCommandDispatcher
	coreCommands   coreCommandRouter
}

// NewCoreDecider builds a CoreDecider with validated core routes and explicit
// system-command dispatch wiring.
func NewCoreDecider(systems *module.Registry, definitions []command.Definition) (CoreDecider, error) {
	coreCommands, err := newCoreCommandRouter(systems, definitions)
	if err != nil {
		return CoreDecider{}, err
	}
	return CoreDecider{
		systemCommands: newSystemCommandDispatcher(systems),
		coreCommands:   coreCommands,
	}, nil
}

// Decide routes system envelopes to the system dispatcher and all remaining
// commands through the core router.
func (d CoreDecider) Decide(state any, cmd command.Command, now func() time.Time) command.Decision {
	current := aggregateState(state)
	if isSystemCommand(cmd) {
		return d.systemCommands.Decide(current, cmd, now)
	}
	return d.coreCommands.Decide(current, cmd, now)
}

// aggregateState converts whatever aggregate representation reached this decider
// into a concrete value.
//
// It supports both typed values and pointers for convenience in tests and caller
// boundaries while keeping downstream code safe and side-effect free.
func aggregateState(state any) aggregate.State {
	switch typed := state.(type) {
	case aggregate.State:
		return typed
	case *aggregate.State:
		if typed != nil {
			return *typed
		}
	}
	return aggregate.State{}
}

// isSystemCommand centralizes the write-path distinction between core command
// envelopes and system-owned envelopes keyed by system identity.
func isSystemCommand(cmd command.Command) bool {
	return strings.TrimSpace(cmd.SystemID) != "" || strings.TrimSpace(cmd.SystemVersion) != ""
}
