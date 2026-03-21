package daggerheart

import (
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
)

// RegistrySystem describes Daggerheart in the game-system metadata registry.
type RegistrySystem struct{}

// NewRegistrySystem creates a Daggerheart metadata descriptor for SystemService.
func NewRegistrySystem() *RegistrySystem {
	return &RegistrySystem{}
}

// ID returns the game-system identifier.
func (r *RegistrySystem) ID() bridge.SystemID {
	return bridge.SystemIDDaggerheart
}

// Version returns the published Daggerheart rules version.
func (r *RegistrySystem) Version() string {
	return SystemVersion
}

// Name returns the display name for the system.
func (r *RegistrySystem) Name() string {
	return "Daggerheart"
}

// RegistryMetadata returns rollout and availability metadata.
// ImplementationStage and Notes are derived from the mechanics manifest
// so they stay in sync with actual implementation state.
func (r *RegistrySystem) RegistryMetadata() bridge.RegistryMetadata {
	return bridge.RegistryMetadata{
		ImplementationStage: DeriveImplementationStage(),
		OperationalStatus:   bridge.OperationalStatusOperational,
		AccessLevel:         bridge.AccessLevelBeta,
		Notes:               deriveImplementationNotes(),
	}
}

// StateHandlerFactory intentionally returns nil until the registry metadata is
// backed by typed state APIs instead of placeholder wiring.
func (r *RegistrySystem) StateHandlerFactory() bridge.StateHandlerFactory {
	return nil
}

// OutcomeApplier intentionally returns nil until the registry metadata exposes
// a real outcome-application surface.
func (r *RegistrySystem) OutcomeApplier() bridge.OutcomeApplier {
	return nil
}

var _ bridge.GameSystem = (*RegistrySystem)(nil)
