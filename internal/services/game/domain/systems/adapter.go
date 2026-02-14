package systems

import (
	"context"
	"fmt"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
)

// Adapter applies system-specific events and exposes system snapshots.
type Adapter interface {
	ID() commonv1.GameSystem
	Version() string
	ApplyEvent(ctx context.Context, evt event.Event) error
	Snapshot(ctx context.Context, campaignID string) (any, error)
}

// AdapterRegistry routes system adapters by system ID + version.
type AdapterRegistry struct {
	adapters map[SystemKey]Adapter
	defaults map[commonv1.GameSystem]string
}

// NewAdapterRegistry creates a new adapter registry.
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		adapters: make(map[SystemKey]Adapter),
		defaults: make(map[commonv1.GameSystem]string),
	}
}

// Register registers an adapter for a system + version.
func (r *AdapterRegistry) Register(adapter Adapter) {
	if r == nil {
		panic("adapter registry is nil")
	}
	version := strings.TrimSpace(adapter.Version())
	if version == "" {
		panic(fmt.Sprintf("system %s must define a version", adapter.ID()))
	}
	key := SystemKey{ID: adapter.ID(), Version: version}
	if _, exists := r.adapters[key]; exists {
		panic(fmt.Sprintf("system %s version %s already registered", adapter.ID(), version))
	}
	if _, exists := r.defaults[adapter.ID()]; !exists {
		r.defaults[adapter.ID()] = version
	}
	r.adapters[key] = adapter
}

// Get returns the adapter for the system + version.
func (r *AdapterRegistry) Get(id commonv1.GameSystem, version string) Adapter {
	if r == nil {
		return nil
	}
	resolved := strings.TrimSpace(version)
	if resolved == "" {
		resolved = r.defaults[id]
	}
	if resolved == "" {
		return nil
	}
	return r.adapters[SystemKey{ID: id, Version: resolved}]
}
