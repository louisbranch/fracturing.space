package game

import (
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/observability/audit"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
)

type adapterRegistry = *bridge.AdapterRegistry

// Applier returns a projection Applier wired to the stores in this bundle.
// The returned Applier can apply any event type; unused stores are simply not
// invoked by the dispatch.
func (s Stores) Applier() projection.Applier {
	applier, err := s.TryApplier()
	if err != nil {
		return projection.Applier{BuildErr: err}
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
		adapters, err = TryAdapterRegistryForProjectionStores(s.SystemStores)
		if err != nil {
			return projection.Applier{}, fmt.Errorf("build adapter registry: %w", err)
		}
	}
	return projection.Applier{
		Events:           s.Events,
		Campaign:         s.Campaign,
		Character:        s.Character,
		CampaignFork:     s.CampaignFork,
		ClaimIndex:       s.ClaimIndex,
		Invite:           s.Invite,
		Participant:      s.Participant,
		Session:          s.Session,
		SessionGate:      s.SessionGate,
		SessionSpotlight: s.SessionSpotlight,
		Scene:            s.Scene,
		SceneCharacter:   s.SceneCharacter,
		SceneGate:        s.SceneGate,
		SceneSpotlight:   s.SceneSpotlight,
		Watermarks:       s.Watermarks,
		Adapters:         adapters,
		Auditor:          audit.NewEmitter(s.Audit),
	}, nil
}
