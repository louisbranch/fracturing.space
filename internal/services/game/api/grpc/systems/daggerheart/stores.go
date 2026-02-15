package daggerheart

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
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

// Applier returns a projection Applier wired to the stores in this bundle.
// Only the stores available in the Daggerheart service are mapped; fields not
// present (e.g., Invite, CampaignFork) remain nil and are unused by dispatch.
func (s Stores) Applier() projection.Applier {
	return projection.Applier{
		Campaign:         s.Campaign,
		Character:        s.Character,
		Session:          s.Session,
		SessionGate:      s.SessionGate,
		SessionSpotlight: s.SessionSpotlight,
		Daggerheart:      s.Daggerheart,
	}
}
