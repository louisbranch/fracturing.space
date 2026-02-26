package daggerheart

import (
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
)

// RegistrySystem describes Daggerheart in the game-system metadata registry.
type RegistrySystem struct{}

// NewRegistrySystem creates a Daggerheart metadata descriptor for SystemService.
func NewRegistrySystem() *RegistrySystem {
	return &RegistrySystem{}
}

// ID returns the game-system enum identifier.
func (r *RegistrySystem) ID() commonv1.GameSystem {
	return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
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
func (r *RegistrySystem) RegistryMetadata() bridge.RegistryMetadata {
	return bridge.RegistryMetadata{
		ImplementationStage: commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PARTIAL,
		OperationalStatus:   commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL,
		AccessLevel:         commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_BETA,
		Notes:               "partial support",
	}
}

// StateHandlerFactory returns nil until SystemService metadata is backed by state APIs.
func (r *RegistrySystem) StateHandlerFactory() bridge.StateHandlerFactory {
	return nil
}

// OutcomeApplier returns nil until metadata wiring includes outcome application.
func (r *RegistrySystem) OutcomeApplier() bridge.OutcomeApplier {
	return nil
}

var _ bridge.GameSystem = (*RegistrySystem)(nil)
