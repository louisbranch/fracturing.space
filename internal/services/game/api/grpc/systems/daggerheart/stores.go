package daggerheart

import "github.com/louisbranch/fracturing.space/internal/services/game/storage"

// Stores groups storage interfaces used by the Daggerheart service.
type Stores struct {
	Campaign    storage.CampaignStore
	Character   storage.CharacterStore
	Session     storage.SessionStore
	Daggerheart storage.DaggerheartStore
	Event       storage.EventStore
}
