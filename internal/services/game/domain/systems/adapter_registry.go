package systems

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// Adapter applies system events to system-specific projections.
type Adapter interface {
	ID() commonv1.GameSystem
	Version() string
	Apply(context.Context, event.Event) error
	Snapshot(context.Context, string) (any, error)
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
	defaults map[commonv1.GameSystem]string
	mu       sync.RWMutex
}

// systemKey identifies a specific system version.
type systemKey struct {
	ID      commonv1.GameSystem
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
)

// NewAdapterRegistry creates a new system adapter registry.
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		adapters: make(map[systemKey]Adapter),
		defaults: make(map[commonv1.GameSystem]string),
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

// Get returns the adapter for the system + version.
func (r *AdapterRegistry) Get(id commonv1.GameSystem, version string) Adapter {
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
