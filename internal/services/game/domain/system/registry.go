package system

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
	// ErrProjectorRequired indicates a missing system projector.
	ErrProjectorRequired = errors.New("system projector is required")
)

// Decider handles system-owned commands.
type Decider interface {
	Decide(state any, cmd command.Command, now func() time.Time) command.Decision
}

// Projector applies system-owned events to system state.
type Projector interface {
	Apply(state any, evt event.Event) (any, error)
}

// StateFactory creates initial system-specific state instances.
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
	Decider() Decider
	Projector() Projector
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
		return command.Decision{}, ErrModuleNotFound
	}
	decider := module.Decider()
	if decider == nil {
		return command.Decision{}, ErrDeciderRequired
	}
	return decider.Decide(state, cmd, now), nil
}

// RouteEvent routes a system event to the registered module projector.
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
		return nil, ErrModuleNotFound
	}
	projector := module.Projector()
	if projector == nil {
		return nil, ErrProjectorRequired
	}
	return projector.Apply(state, evt)
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
