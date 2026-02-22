package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// Adapter applies system events to system-specific projections.
type Adapter interface {
	ID() string
	Version() string
	Apply(context.Context, event.Event) error
	Snapshot(context.Context, string) (any, error)
	// HandledTypes returns the event types this adapter's Apply handles.
	// Used by startup validation to ensure every emittable event type
	// declared by the system module has a projection handler.
	HandledTypes() []event.Type
}

// ProfileAdapter is an optional interface for adapters that handle
// character profile updates. When a character.profile_updated event arrives,
// the projection applier iterates the system_profile map and delegates each
// key's data to the corresponding adapter if it implements ProfileAdapter.
type ProfileAdapter interface {
	ApplyProfile(ctx context.Context, campaignID, characterID string, profileData json.RawMessage) error
}

// AdapterRegistry routes system adapters by system ID + version.
type AdapterRegistry struct {
	adapters map[systemKey]Adapter
	defaults map[string]string
	mu       sync.RWMutex
}

// systemKey identifies a specific system version.
type systemKey struct {
	ID      string
	Version string
}

var (
	// ErrAdapterRegistryNil indicates registration was attempted on a nil registry.
	ErrAdapterRegistryNil = errors.New("adapter registry is nil")
	// ErrAdapterRequired indicates a nil adapter was provided for registration.
	ErrAdapterRequired = errors.New("adapter is required")
	// ErrAdapterVersionRequired indicates adapter registration omitted a version.
	ErrAdapterVersionRequired = errors.New("adapter version is required")
	// ErrAdapterAlreadyRegistered indicates adapter registration duplicated ID+version.
	ErrAdapterAlreadyRegistered = errors.New("adapter already registered")
	// ErrAdapterNotFound indicates no adapter is registered for the requested system+version.
	ErrAdapterNotFound = errors.New("adapter not found")
)

// NewAdapterRegistry creates a new system adapter registry.
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		adapters: make(map[systemKey]Adapter),
		defaults: make(map[string]string),
	}
}

// Register registers an adapter for a system + version.
func (r *AdapterRegistry) Register(adapter Adapter) error {
	if r == nil {
		return ErrAdapterRegistryNil
	}
	if adapter == nil {
		return ErrAdapterRequired
	}
	version := strings.TrimSpace(adapter.Version())
	if version == "" {
		return fmt.Errorf("%w: system %s", ErrAdapterVersionRequired, adapter.ID())
	}
	key := systemKey{ID: adapter.ID(), Version: version}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.adapters[key]; exists {
		return fmt.Errorf("%w: system %s version %s", ErrAdapterAlreadyRegistered, adapter.ID(), version)
	}
	if _, exists := r.defaults[adapter.ID()]; !exists {
		r.defaults[adapter.ID()] = version
	}
	r.adapters[key] = adapter
	return nil
}

// Adapters returns all registered adapters.
func (r *AdapterRegistry) Adapters() []Adapter {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	adapters := make([]Adapter, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		adapters = append(adapters, adapter)
	}
	return adapters
}

// Get returns the adapter for the system + version, or nil when not found.
// Use GetRequired when the caller cannot proceed without an adapter.
func (r *AdapterRegistry) Get(id string, version string) Adapter {
	if r == nil {
		return nil
	}
	resolved := strings.TrimSpace(version)
	r.mu.RLock()
	defer r.mu.RUnlock()
	if resolved == "" {
		resolved = r.defaults[id]
	}
	if resolved == "" {
		return nil
	}
	return r.adapters[systemKey{ID: id, Version: resolved}]
}

// GetRequired returns the adapter for the system + version, or a typed error
// when no adapter is registered. Use this on paths where a missing adapter
// indicates a configuration bug (e.g., system event projection routing).
func (r *AdapterRegistry) GetRequired(id string, version string) (Adapter, error) {
	if r == nil {
		return nil, ErrAdapterRegistryNil
	}
	adapter := r.Get(id, version)
	if adapter == nil {
		return nil, fmt.Errorf("%w: system %s version %q", ErrAdapterNotFound, id, version)
	}
	return adapter, nil
}
