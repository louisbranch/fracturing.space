package daggerheart

import (
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/systems"
)

// System implements the Daggerheart game system.
type System struct {
	stateFactory   systems.StateFactory
	outcomeApplier systems.OutcomeApplier
}

// ID returns the system identifier.
func (s *System) ID() commonv1.GameSystem {
	return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
}

// Name returns the human-readable system name.
func (s *System) Name() string {
	return "Daggerheart"
}

// StateFactory returns the factory for creating Daggerheart-specific state.
func (s *System) StateFactory() systems.StateFactory {
	return s.stateFactory
}

// OutcomeApplier returns the handler for applying Daggerheart roll outcomes.
func (s *System) OutcomeApplier() systems.OutcomeApplier {
	return s.outcomeApplier
}

// Ensure System implements GameSystem.
var _ systems.GameSystem = (*System)(nil)

func init() {
	sys := &System{}
	sys.stateFactory = NewStateFactory()
	sys.outcomeApplier = NewOutcomeApplier()
	systems.DefaultRegistry.Register(sys)
}
