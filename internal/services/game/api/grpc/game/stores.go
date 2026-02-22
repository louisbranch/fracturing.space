package game

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Stores groups all campaign-related storage interfaces for service injection.
type Stores struct {
	// Core projection stores — used by the projection applier for core events.
	Campaign         storage.CampaignStore
	Participant      storage.ParticipantStore
	ClaimIndex       storage.ClaimIndexStore
	Invite           storage.InviteStore
	Character        storage.CharacterStore
	Session          storage.SessionStore
	SessionGate      storage.SessionGateStore
	SessionSpotlight storage.SessionSpotlightStore
	CampaignFork     storage.CampaignForkStore

	// System adapter stores — used for system event projection via AdapterRegistry.
	// Daggerheart is wired through manifest.AdapterRegistry, not the core
	// projection.Applier directly.
	Daggerheart storage.DaggerheartStore

	// Infrastructure stores — event journal, snapshots, telemetry.
	Event      storage.EventStore
	Telemetry  storage.TelemetryStore
	Statistics storage.StatisticsStore
	Snapshot   storage.SnapshotStore

	// System content stores — read-only content used by gRPC handlers.
	DaggerheartContent storage.DaggerheartContentStore

	Domain Domain

	// Events is the event registry used for intent filtering at request time.
	Events *event.Registry

	// adapters is built eagerly during Validate and cached for Applier.
	adapters *bridge.AdapterRegistry
}

// Applier returns a projection Applier wired to the stores in this bundle.
// The returned Applier can apply any event type; unused stores are simply not
// invoked by the dispatch.
func (s Stores) Applier() projection.Applier {
	applier, err := s.TryApplier()
	if err != nil {
		panic(err)
	}
	return applier
}

// TryApplier returns a projection Applier wired to the stores in this bundle.
// The returned Applier can apply any event type; unused stores are simply not
// invoked by the dispatch.
//
// If Validate was called first the cached adapter registry is used; otherwise
// a fresh one is built on-the-fly so partial-Stores test helpers keep working.
func (s Stores) TryApplier() (projection.Applier, error) {
	adapters := s.adapters
	if adapters == nil {
		var err error
		adapters, err = TryAdapterRegistryForStores(s)
		if err != nil {
			return projection.Applier{}, fmt.Errorf("build adapter registry: %w", err)
		}
	}
	return projection.Applier{
		Campaign:         s.Campaign,
		Character:        s.Character,
		CampaignFork:     s.CampaignFork,
		ClaimIndex:       s.ClaimIndex,
		Invite:           s.Invite,
		Participant:      s.Participant,
		Session:          s.Session,
		SessionGate:      s.SessionGate,
		SessionSpotlight: s.SessionSpotlight,
		Adapters:         adapters,
	}, nil
}

// Validate checks that every store field is non-nil and eagerly builds the
// adapter registry. Call this at service construction time so that handlers
// do not need per-method nil guards and adapter registration errors surface
// at startup instead of at runtime.
func (s *Stores) Validate() error {
	var missing []string
	if s.Campaign == nil {
		missing = append(missing, "Campaign")
	}
	if s.Participant == nil {
		missing = append(missing, "Participant")
	}
	if s.ClaimIndex == nil {
		missing = append(missing, "ClaimIndex")
	}
	if s.Invite == nil {
		missing = append(missing, "Invite")
	}
	if s.Character == nil {
		missing = append(missing, "Character")
	}
	if s.Daggerheart == nil {
		missing = append(missing, "Daggerheart")
	}
	if s.Session == nil {
		missing = append(missing, "Session")
	}
	if s.SessionGate == nil {
		missing = append(missing, "SessionGate")
	}
	if s.SessionSpotlight == nil {
		missing = append(missing, "SessionSpotlight")
	}
	if s.Event == nil {
		missing = append(missing, "Event")
	}
	if s.Telemetry == nil {
		missing = append(missing, "Telemetry")
	}
	if s.Statistics == nil {
		missing = append(missing, "Statistics")
	}
	if s.Snapshot == nil {
		missing = append(missing, "Snapshot")
	}
	if s.CampaignFork == nil {
		missing = append(missing, "CampaignFork")
	}
	if s.DaggerheartContent == nil {
		missing = append(missing, "DaggerheartContent")
	}
	if len(missing) > 0 {
		return fmt.Errorf("stores not configured: %s", strings.Join(missing, ", "))
	}

	adapters, err := TryAdapterRegistryForStores(*s)
	if err != nil {
		return fmt.Errorf("build adapter registry: %w", err)
	}
	s.adapters = adapters

	applier, err := s.TryApplier()
	if err != nil {
		return fmt.Errorf("build projection applier: %w", err)
	}
	if err := applier.ValidateStorePreconditions(); err != nil {
		return err
	}
	return nil
}
