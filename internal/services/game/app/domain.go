package server

import (
	"errors"
	"fmt"

	gamegrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/checkpoint"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// configureDomain wires the write-path domain engine into gRPC stores when enabled.
//
// This is intentionally guarded by config so deployments can run without domain
// execution in specific environments (for example, projection-only workflows).
func configureDomain(srvEnv serverEnv, stores *gamegrpc.Stores, registries engine.Registries) error {
	if !srvEnv.DomainEnabled {
		return nil
	}
	if stores == nil {
		return errors.New("stores are required")
	}
	domainEngine, err := buildDomainEngine(stores.Event, registries)
	if err != nil {
		return fmt.Errorf("build domain engine: %w", err)
	}
	stores.Domain = domainEngine
	return nil
}

// buildDomainEngine builds the replay-capable domain handler used by write paths.
//
// It composes registries, replay-based state loading, gate evaluation, and
// decider routing once, so command execution stays consistent for every request.
func buildDomainEngine(eventStore storage.EventStore, registries engine.Registries) (gamegrpc.Domain, error) {
	if eventStore == nil {
		return nil, errors.New("event store is required")
	}
	decider, err := engine.NewCoreDecider(registries.Systems, registries.Commands.ListDefinitions())
	if err != nil {
		return nil, fmt.Errorf("build core decider: %w", err)
	}

	checkpoints := checkpoint.NewMemory()
	folder := &aggregate.Folder{
		Events:         registries.Events,
		SystemRegistry: registries.Systems,
	}
	stateLoader := engine.ReplayStateLoader{
		Events:       gamegrpc.NewEventStoreAdapter(eventStore),
		Checkpoints:  checkpoints,
		Snapshots:    checkpoints,
		Folder:       folder,
		StateFactory: func() any { return aggregate.State{} },
	}
	return engine.NewHandler(engine.HandlerConfig{
		Commands:        registries.Commands,
		Events:          registries.Events,
		Journal:         gamegrpc.NewJournalAdapter(eventStore),
		Checkpoints:     checkpoints,
		Snapshots:       checkpoints,
		Gate:            engine.DecisionGate{Registry: registries.Commands},
		GateStateLoader: engine.ReplayGateStateLoader{StateLoader: stateLoader},
		StateLoader:     stateLoader,
		Decider:         decider,
		Folder:          folder,
	})
}
