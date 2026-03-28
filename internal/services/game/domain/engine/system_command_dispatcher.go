package engine

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

// systemCommandDispatcher routes system-owned command envelopes through the
// registered module registry.
type systemCommandDispatcher struct {
	systems *module.Registry
}

// newSystemCommandDispatcher captures the system registry once so the public
// decider can stay focused on top-level branching.
func newSystemCommandDispatcher(systems *module.Registry) systemCommandDispatcher {
	return systemCommandDispatcher{systems: systems}
}

// Decide dispatches one system-owned command against the current aggregate's
// system state snapshot.
func (d systemCommandDispatcher) Decide(current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	key := module.Key{ID: cmd.SystemID, Version: cmd.SystemVersion}
	_, systemState, err := module.ResolveSnapshotState(
		d.systems,
		cmd.CampaignID,
		cmd.SystemID,
		cmd.SystemVersion,
		current.Systems[key],
	)
	if err != nil {
		return command.Reject(command.Rejection{Code: "SYSTEM_COMMAND_STATE_RESOLVE_FAILED", Message: err.Error()})
	}
	decision, err := module.RouteCommand(d.systems, systemState, cmd, now)
	if err != nil {
		return command.Reject(command.Rejection{Code: "SYSTEM_COMMAND_ROUTE_FAILED", Message: err.Error()})
	}
	return decision
}
