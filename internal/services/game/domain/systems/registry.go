package systems

import (
	"context"
	"fmt"
	"strings"
	"sync"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

// GameSystem defines the interface that all game systems must implement.
// This allows the API layer to dispatch to the correct system based on
// the campaign's system_id.
type GameSystem interface {
	// ID returns the system identifier (matches GameSystem proto enum).
	ID() commonv1.GameSystem

	// Version returns the system ruleset version.
	Version() string

	// Name returns the human-readable system name.
	Name() string

	// StateFactory returns the factory for creating system-specific state.
	// May return nil if the system doesn't manage state.
	StateFactory() StateFactory

	// OutcomeApplier returns the handler for applying roll outcomes.
	// May return nil if the system doesn't support outcome application.
	OutcomeApplier() OutcomeApplier
}

// CharacterKind represents the type of character (PC or NPC).
type CharacterKind int

const (
	CharacterKindUnspecified CharacterKind = iota
	CharacterKindPC
	CharacterKindNPC
)

// Healable represents entities that can be healed.
type Healable interface {
	Heal(amount int) (before, after int)
	MaxHP() int
}

// Damageable represents entities that can take damage.
type Damageable interface {
	TakeDamage(amount int) (before, after int)
	CurrentHP() int
}

// ResourceHolder represents entities with named resources.
// Resources are system-specific (e.g., Hope/Stress for Daggerheart, Blood Pool for VtM).
type ResourceHolder interface {
	// GainResource increases a named resource by amount.
	// Returns the before and after values, or error if the resource is unknown.
	GainResource(name string, amount int) (before, after int, err error)

	// SpendResource decreases a named resource by amount.
	// Returns the before and after values, or error if insufficient or unknown.
	SpendResource(name string, amount int) (before, after int, err error)

	// ResourceValue returns the current value of a named resource.
	ResourceValue(name string) int

	// ResourceCap returns the maximum value of a named resource.
	ResourceCap(name string) int

	// ResourceNames returns the names of all resources this holder manages.
	ResourceNames() []string
}

// CharacterStateHandler combines all character state behaviors.
// Systems implement this to provide character-level state management.
type CharacterStateHandler interface {
	Healable
	Damageable
	ResourceHolder

	// CampaignID returns the campaign this state belongs to.
	CampaignID() string

	// CharacterID returns the character this state belongs to.
	CharacterID() string
}

// SnapshotStateHandler manages campaign-level state for a system.
// Systems implement this for resources like GM Fear (Daggerheart).
type SnapshotStateHandler interface {
	ResourceHolder

	// CampaignID returns the campaign this state belongs to.
	CampaignID() string
}

// StateFactory creates system-specific state instances.
type StateFactory interface {
	// NewCharacterState creates initial character state for the given character.
	NewCharacterState(campaignID, characterID string, kind CharacterKind) (CharacterStateHandler, error)

	// NewSnapshotState creates an initial snapshot projection for the given campaign.
	NewSnapshotState(campaignID string) (SnapshotStateHandler, error)
}

// StateChange represents a change to game state from an outcome.
type StateChange struct {
	// CharacterID is set for character-level changes, empty for campaign-level.
	CharacterID string
	// Field is the name of the changed field (e.g., "hope", "gm_fear").
	Field string
	// Before is the value before the change.
	Before int
	// After is the value after the change.
	After int
}

// OutcomeContext provides context for applying a roll outcome.
type OutcomeContext struct {
	CampaignID  string
	SessionID   string
	CharacterID string
	RollSeq     int64
	Outcome     interface{} // System-specific outcome type
	Targets     []string    // Target character IDs
}

// OutcomeApplier applies roll outcomes to game state.
type OutcomeApplier interface {
	// ApplyOutcome applies the outcome to game state and returns the changes.
	ApplyOutcome(ctx context.Context, outcome OutcomeContext) ([]StateChange, error)
}

// Registry manages registered game systems.
type Registry struct {
	mu       sync.RWMutex
	systems  map[SystemKey]GameSystem
	defaults map[commonv1.GameSystem]string
}

// SystemKey identifies a specific version of a system.
type SystemKey struct {
	ID      commonv1.GameSystem
	Version string
}

// NewRegistry creates a new game system registry.
func NewRegistry() *Registry {
	return &Registry{
		systems:  make(map[SystemKey]GameSystem),
		defaults: make(map[commonv1.GameSystem]string),
	}
}

// Register adds a game system to the registry.
// Panics if a system with the same ID is already registered.
func (r *Registry) Register(system GameSystem) {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := system.ID()
	version := strings.TrimSpace(system.Version())
	if version == "" {
		panic(fmt.Sprintf("game system %s must define a version", id))
	}
	key := SystemKey{ID: id, Version: version}
	if _, exists := r.systems[key]; exists {
		panic(fmt.Sprintf("game system %s version %s already registered", id, version))
	}
	if _, exists := r.defaults[id]; !exists {
		r.defaults[id] = version
	}
	r.systems[key] = system
}

// Get returns the game system for the given ID, or nil if not found.
func (r *Registry) Get(id commonv1.GameSystem) GameSystem {
	return r.GetVersion(id, "")
}

// GetVersion returns the game system for the given ID and version.
// If version is empty, the default registered version is returned.
func (r *Registry) GetVersion(id commonv1.GameSystem, version string) GameSystem {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resolved := strings.TrimSpace(version)
	if resolved == "" {
		resolved = r.defaults[id]
	}
	if resolved == "" {
		return nil
	}
	return r.systems[SystemKey{ID: id, Version: resolved}]
}

// MustGet returns the game system for the given ID, or panics if not found.
func (r *Registry) MustGet(id commonv1.GameSystem) GameSystem {
	system := r.Get(id)
	if system == nil {
		panic(fmt.Sprintf("game system %s not registered", id))
	}
	return system
}

// List returns all registered game systems.
func (r *Registry) List() []GameSystem {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]GameSystem, 0, len(r.systems))
	for _, system := range r.systems {
		result = append(result, system)
	}
	return result
}

// DefaultRegistry is the global game system registry.
// Systems should register themselves via init() functions.
var DefaultRegistry = NewRegistry()

// RollRequest represents a generic roll request that systems can implement.
type RollRequest struct {
	Ctx  context.Context
	Seed int64
}

// RollResult represents a generic roll result.
type RollResult struct {
	Total int
}
