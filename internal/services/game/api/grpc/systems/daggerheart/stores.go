package daggerheart

import (
	"context"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	daggerheartsystem "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
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
	DaggerheartContent storage.DaggerheartContentStore
	Event              storage.EventStore
	Domain             Domain
}

// Validate checks that Daggerheart gameplay service dependencies are configured.
func (s Stores) Validate() error {
	var missing []string
	if s.Campaign == nil {
		missing = append(missing, "Campaign")
	}
	if s.Character == nil {
		missing = append(missing, "Character")
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
	if s.Daggerheart == nil {
		missing = append(missing, "Daggerheart")
	}
	if s.Event == nil {
		missing = append(missing, "Event")
	}
	if s.Domain == nil {
		missing = append(missing, "Domain")
	}
	if len(missing) > 0 {
		return fmt.Errorf("stores not configured: %s", strings.Join(missing, ", "))
	}
	return nil
}

// ValidateContent checks that Daggerheart content service dependencies are configured.
func (s Stores) ValidateContent() error {
	if s.DaggerheartContent == nil {
		return fmt.Errorf("stores not configured: DaggerheartContent")
	}
	return nil
}

// Applier returns a projection Applier wired to the stores in this bundle.
// Only the stores available in the Daggerheart service are mapped; fields not
// present (e.g., Invite, CampaignFork) remain nil and are unused by dispatch.
func (s Stores) Applier() projection.Applier {
	adapters := systems.NewAdapterRegistry()
	if s.Daggerheart != nil {
		if err := adapters.Register(daggerheartsystem.NewAdapter(s.Daggerheart)); err != nil {
			panic("register daggerheart adapter: " + err.Error())
		}
	}
	return projection.Applier{
		Campaign:         s.Campaign,
		Character:        s.Character,
		Session:          s.Session,
		SessionGate:      s.SessionGate,
		SessionSpotlight: s.SessionSpotlight,
		Adapters:         adapters,
	}
}
