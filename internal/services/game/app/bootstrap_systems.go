package app

import (
	"context"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/interceptors"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
)

// systemsBootstrapper owns the startup phase that validates system registration
// parity and runs best-effort projection repair before transport is exposed.
type systemsBootstrapper interface {
	Bootstrap(context.Context, *storageBundle, systemRegistrationSnapshot, engine.Registries, projection.Applier) (systemsRuntimeState, error)
}

type systemsRuntimeState struct {
	systemRegistry *bridge.MetadataRegistry
}

type defaultSystemsBootstrapper struct {
	buildSystemRegistry        func(systemRegistrationSnapshot) (*bridge.MetadataRegistry, error)
	validateSystemRegistration func([]module.Module, *bridge.MetadataRegistry, *bridge.AdapterRegistry) error
	validateSessionLockPolicy  func(*command.Registry) error
	repairProjectionGaps       func(context.Context, *storageBundle, projection.Applier)
}

func (b defaultSystemsBootstrapper) Bootstrap(
	ctx context.Context,
	bundle *storageBundle,
	systemRegistration systemRegistrationSnapshot,
	registries engine.Registries,
	applier projection.Applier,
) (systemsRuntimeState, error) {
	systemRegistry, err := b.buildSystemRegistry(systemRegistration)
	if err != nil {
		return systemsRuntimeState{}, fmt.Errorf("build system registry: %w", err)
	}
	if err := b.validateSystemRegistration(systemRegistration.modulesCopy(), systemRegistry, applier.Adapters); err != nil {
		return systemsRuntimeState{}, fmt.Errorf("validate system parity: %w", err)
	}
	if err := b.validateSessionLockPolicy(registries.Commands); err != nil {
		return systemsRuntimeState{}, fmt.Errorf("validate session lock policy: %w", err)
	}
	b.repairProjectionGaps(ctx, bundle, applier)
	return systemsRuntimeState{systemRegistry: systemRegistry}, nil
}

var _ systemsBootstrapper = defaultSystemsBootstrapper{}

func validateSessionLockPolicy(registry *command.Registry) error {
	return interceptors.ValidateSessionLockPolicyCoverage(registry)
}
