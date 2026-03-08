package daggerheart

import (
	"context"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Domain executes domain commands and returns the result.
type Domain interface {
	Execute(ctx context.Context, cmd command.Command) (engine.Result, error)
}

// Stores groups storage interfaces used by the Daggerheart service.
type Stores struct {
	Campaign           storage.CampaignStore
	Character          storage.CharacterStore
	Session            storage.SessionStore
	SessionGate        storage.SessionGateStore
	SessionSpotlight   storage.SessionSpotlightStore
	Daggerheart        storage.DaggerheartStore
	DaggerheartContent storage.DaggerheartContentReadStore
	Event              storage.EventStore
	Watermarks         storage.ProjectionWatermarkStore
	Events             *event.Registry

	// Write groups the domain executor, runtime controls, and audit store
	// used by the write path. It satisfies domainwriteexec.Deps so handlers
	// can pass it directly to executeAndApplyDomainCommand.
	Write domainwriteexec.WritePath

	// adapters is built eagerly during Validate and cached for Applier.
	adapters *bridge.AdapterRegistry
}

// ProjectionStoreBundle is the projection dependency contract for the
// Daggerheart gameplay service.
type ProjectionStoreBundle interface {
	storage.CampaignStore
	storage.CharacterStore
	storage.SessionStore
	storage.SessionGateStore
	storage.SessionSpotlightStore
	storage.DaggerheartStore
	storage.ProjectionWatermarkStore
}

// StoresFromProjectionConfig configures NewStoresFromProjection.
type StoresFromProjectionConfig struct {
	ProjectionStore ProjectionStoreBundle
	EventStore      storage.EventStore
	ContentStore    storage.DaggerheartContentReadStore
	Domain          Domain
	WriteRuntime    *domainwrite.Runtime
	Events          *event.Registry
}

// NewStoresFromProjection constructs Stores from a projection-oriented bundle
// plus runtime dependencies. This keeps startup wiring concise and explicit.
func NewStoresFromProjection(config StoresFromProjectionConfig) Stores {
	return Stores{
		Campaign:           config.ProjectionStore,
		Character:          config.ProjectionStore,
		Session:            config.ProjectionStore,
		SessionGate:        config.ProjectionStore,
		SessionSpotlight:   config.ProjectionStore,
		Daggerheart:        config.ProjectionStore,
		DaggerheartContent: config.ContentStore,
		Event:              config.EventStore,
		Watermarks:         config.ProjectionStore,
		Events:             config.Events,
		Write: domainwriteexec.WritePath{
			Executor: config.Domain,
			Runtime:  config.WriteRuntime,
		},
	}
}

// Validate checks that Daggerheart gameplay service dependencies are configured
// and eagerly builds the adapter registry so registration errors surface at
// startup instead of at runtime.
func (s *Stores) Validate() error {
	var missing []string
	missing = appendMissingRequirements(missing, s.projectionRequirements()...)
	missing = appendMissingRequirements(missing, s.infrastructureRequirements()...)
	missing = appendMissingRequirements(missing, s.runtimeRequirements()...)
	if len(missing) > 0 {
		return fmt.Errorf("stores not configured: %s", strings.Join(missing, ", "))
	}

	adapters, err := systemmanifest.AdapterRegistry(systemmanifest.ProjectionStores{
		Daggerheart: s.Daggerheart,
	})
	if err != nil {
		return fmt.Errorf("build adapter registry: %w", err)
	}
	s.adapters = adapters
	return nil
}

// ValidateContent checks that Daggerheart content service dependencies are configured.
func (s Stores) ValidateContent() error {
	if s.DaggerheartContent == nil {
		return fmt.Errorf("stores not configured: DaggerheartContent")
	}
	return nil
}

type dependencyRequirement struct {
	name       string
	configured bool
}

func appendMissingRequirements(missing []string, requirements ...dependencyRequirement) []string {
	for _, requirement := range requirements {
		if !requirement.configured {
			missing = append(missing, requirement.name)
		}
	}
	return missing
}

func (s Stores) projectionRequirements() []dependencyRequirement {
	return []dependencyRequirement{
		{name: "Campaign", configured: s.Campaign != nil},
		{name: "Character", configured: s.Character != nil},
		{name: "Session", configured: s.Session != nil},
		{name: "SessionGate", configured: s.SessionGate != nil},
		{name: "SessionSpotlight", configured: s.SessionSpotlight != nil},
		{name: "Daggerheart", configured: s.Daggerheart != nil},
	}
}

func (s Stores) infrastructureRequirements() []dependencyRequirement {
	return []dependencyRequirement{
		{name: "Event", configured: s.Event != nil},
	}
}

func (s Stores) runtimeRequirements() []dependencyRequirement {
	return []dependencyRequirement{
		{name: "Write.Executor", configured: s.Write.Executor != nil},
		{name: "Write.Runtime", configured: s.Write.Runtime != nil},
		{name: "Events", configured: s.Events != nil},
	}
}

// Applier returns a projection Applier wired to the stores in this bundle.
// Only the stores available in the Daggerheart service are mapped; fields not
// present (e.g., Invite, CampaignFork) remain nil and are unused by dispatch.
func (s Stores) Applier() projection.Applier {
	applier, err := s.TryApplier()
	if err != nil {
		return projection.Applier{BuildErr: err}
	}
	return applier
}

// TryApplier returns a projection Applier wired to the stores in this bundle.
// Only the stores available in the Daggerheart service are mapped; fields not
// present (e.g., Invite, CampaignFork) remain nil and are unused by dispatch.
//
// If Validate was called first the cached adapter registry is used; otherwise
// a fresh one is built on-the-fly.
func (s Stores) TryApplier() (projection.Applier, error) {
	adapters := s.adapters
	if adapters == nil {
		var err error
		adapters, err = systemmanifest.AdapterRegistry(systemmanifest.ProjectionStores{
			Daggerheart: s.Daggerheart,
		})
		if err != nil {
			return projection.Applier{}, fmt.Errorf("build adapter registry: %w", err)
		}
	}
	return projection.Applier{
		Events:           s.Events,
		Campaign:         s.Campaign,
		Character:        s.Character,
		Session:          s.Session,
		SessionGate:      s.SessionGate,
		SessionSpotlight: s.SessionSpotlight,
		Watermarks:       s.Watermarks,
		Adapters:         adapters,
	}, nil
}
