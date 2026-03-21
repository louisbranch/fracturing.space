package daggerheart

import (
	daggerheartadapter "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/adapter"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

// NewAdapter creates the Daggerheart projection adapter with the module-owned
// level-up applier wired in at the composition root.
func NewAdapter(store projectionstore.Store) *daggerheartadapter.Adapter {
	return daggerheartadapter.NewAdapter(store, applyLevelUpToCharacterProfile)
}
