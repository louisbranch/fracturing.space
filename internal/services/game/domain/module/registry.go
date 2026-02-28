package module

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

var (
	// ErrSystemIDRequired indicates a missing system id.
	ErrSystemIDRequired = errors.New("system id is required")
	// ErrSystemVersionRequired indicates a missing system version.
	ErrSystemVersionRequired = errors.New("system version is required")
	// ErrSystemAlreadyRegistered indicates a duplicate module registration.
	ErrSystemAlreadyRegistered = errors.New("system module already registered")
	// ErrRegistryRequired indicates a missing registry.
	ErrRegistryRequired = errors.New("registry is required")
	// ErrModuleNotFound indicates a missing system module.
	ErrModuleNotFound = errors.New("system module is not registered")
	// ErrDeciderRequired indicates a missing system decider.
	ErrDeciderRequired = errors.New("system decider is required")
	// ErrFolderRequired indicates a missing system folder.
	ErrFolderRequired = errors.New("system folder is required")
)

// Decider handles system-owned commands.
type Decider interface {
	Decide(state any, cmd command.Command, now func() time.Time) command.Decision
}

// Folder folds system-owned events into system state.
//
// FoldHandledTypes declares which event types the Fold method handles, enabling
// ValidateSystemFoldCoverage to verify at startup that every emittable event
// type with replay intent has a corresponding fold handler.
// Named "Folder" (not "Applier") because it performs a pure state fold,
// not a side-effecting projection write (projection.Applier.Apply).
type Folder interface {
	Fold(state any, evt event.Event) (any, error)
	FoldHandledTypes() []event.Type
}

// CharacterReadinessChecker is an optional module extension point used by
// session-start readiness evaluation.
//
// Implementations validate system-specific character readiness from the
// character's system profile projection payload. Returning false blocks
// session.start and the reason is surfaced in the command rejection message.
//
// Modules that do not implement this interface fall back to core readiness
// rules only (for example, controller assignment).
type CharacterReadinessChecker interface {
	CharacterReady(systemProfile map[string]any) (ready bool, reason string)
}

// CommandTyper must be implemented by deciders whose modules register system
// commands. ValidateDeciderCommandCoverage verifies at startup that every
// registered system command has a corresponding decider case, failing loudly
// when coverage is incomplete.
type CommandTyper interface {
	DeciderHandledCommands() []command.Type
}

// StateFactory creates initial system-specific state instances.
//
// The aggregate folder calls NewSnapshotState lazily: on the first system
// event for a given (SystemID, SystemVersion) key, the folder looks up the
// module, and if no state exists yet for that key, it calls
// NewSnapshotState to seed the initial value. Subsequent events for the
// same key fold into the already-initialized state.
//
// NewCharacterState is called when a character profile is created or
// updated through the system profile adapter.
//
// Implementations must be deterministic: given the same inputs they must
// return the same state, because replay depends on this guarantee.
//
// NOTE: This is the write-path StateFactory (returns untyped `any`).
// See also bridge.StateHandlerFactory (domain/bridge/registry_bridge.go)
// which returns typed handlers (CharacterStateHandler, SnapshotStateHandler)
// for the API bridge layer. Daggerheart only implements this module variant;
// the bridge variant is used by the API layer to provide resource/damage
// abstractions.
type StateFactory interface {
	NewCharacterState(campaignID, characterID, kind string) (any, error)
	NewSnapshotState(campaignID string) (any, error)
}

// Module defines the interface for a system module.
type Module interface {
	ID() string
	Version() string
	RegisterCommands(registry *command.Registry) error
	RegisterEvents(registry *event.Registry) error
	// EmittableEventTypes returns all event types this module's decider can emit.
	// BuildRegistries validates that every declared type is registered in the
	// event registry, catching missing registrations at startup.
	EmittableEventTypes() []event.Type
	Decider() Decider
	Folder() Folder
	StateFactory() StateFactory
}

// Registry manages registered system modules.
type Registry struct {
	mu       sync.RWMutex
	modules  map[Key]Module
	defaults map[string]string
}

// Key identifies a specific system module version.
type Key struct {
	ID      string
	Version string
}

// NewRegistry creates a new system module registry.
func NewRegistry() *Registry {
	return &Registry{
		modules:  make(map[Key]Module),
		defaults: make(map[string]string),
	}
}

// RouteCommand routes a system command to the registered module decider.
//
// This boundary allows custom game systems to participate in command handling
// without leaking system-specific behavior into core aggregates.
func RouteCommand(registry *Registry, state any, cmd command.Command, now func() time.Time) (command.Decision, error) {
	if registry == nil {
		return command.Decision{}, ErrRegistryRequired
	}
	systemID := strings.TrimSpace(cmd.SystemID)
	if systemID == "" {
		return command.Decision{}, ErrSystemIDRequired
	}
	systemVersion := strings.TrimSpace(cmd.SystemVersion)
	if systemVersion == "" {
		return command.Decision{}, ErrSystemVersionRequired
	}
	module := registry.Get(systemID, systemVersion)
	if module == nil {
		return command.Decision{}, fmt.Errorf("%w: %s@%s", ErrModuleNotFound, systemID, systemVersion)
	}
	decider := module.Decider()
	if decider == nil {
		return command.Decision{}, ErrDeciderRequired
	}
	return decider.Decide(state, cmd, now), nil
}

// RouteEvent routes a system event to the registered module folder.
//
// Folders keep system-owned aggregate state slices aligned with event semantics
// defined by the same module that emitted them.
func RouteEvent(registry *Registry, state any, evt event.Event) (any, error) {
	if registry == nil {
		return nil, ErrRegistryRequired
	}
	systemID := strings.TrimSpace(evt.SystemID)
	if systemID == "" {
		return nil, ErrSystemIDRequired
	}
	systemVersion := strings.TrimSpace(evt.SystemVersion)
	if systemVersion == "" {
		return nil, ErrSystemVersionRequired
	}
	module := registry.Get(systemID, systemVersion)
	if module == nil {
		return nil, fmt.Errorf("%w: %s@%s", ErrModuleNotFound, systemID, systemVersion)
	}
	folder := module.Folder()
	if folder == nil {
		return nil, ErrFolderRequired
	}
	return folder.Fold(state, evt)
}

// Register adds a system module to the registry.
func (r *Registry) Register(module Module) error {
	if r == nil {
		return ErrRegistryRequired
	}
	if module == nil {
		return errors.New("system module is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	id := strings.TrimSpace(module.ID())
	if id == "" {
		return ErrSystemIDRequired
	}
	version := strings.TrimSpace(module.Version())
	if version == "" {
		return ErrSystemVersionRequired
	}
	if r.modules == nil {
		r.modules = make(map[Key]Module)
	}
	key := Key{ID: id, Version: version}
	if _, exists := r.modules[key]; exists {
		return fmt.Errorf("%w: %s@%s", ErrSystemAlreadyRegistered, id, version)
	}
	if r.defaults == nil {
		r.defaults = make(map[string]string)
	}
	if _, exists := r.defaults[id]; !exists {
		r.defaults[id] = version
	}
	r.modules[key] = module
	return nil
}

// Get returns the system module for the given id and version.
// If version is empty, the default registered version is returned.
func (r *Registry) Get(id, version string) Module {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	resolvedID := strings.TrimSpace(id)
	resolvedVersion := strings.TrimSpace(version)
	if resolvedID == "" {
		return nil
	}
	if resolvedVersion == "" {
		resolvedVersion = r.defaults[resolvedID]
	}
	if resolvedVersion == "" {
		return nil
	}
	return r.modules[Key{ID: resolvedID, Version: resolvedVersion}]
}

// DefaultVersion returns the default registered version for the given system id.
func (r *Registry) DefaultVersion(id string) string {
	if r == nil {
		return ""
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.defaults[strings.TrimSpace(id)]
}

// List returns all registered system modules.
func (r *Registry) List() []Module {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	modules := make([]Module, 0, len(r.modules))
	for _, module := range r.modules {
		modules = append(modules, module)
	}
	return modules
}
