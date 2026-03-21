package coreprojection

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	sqlitedaggerheartprojection "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/daggerheartprojection"
)

// DaggerheartProjectionStore binds the system-owned Daggerheart projection
// backend to this root projections store's current query bundle. Transaction
// clones keep working because txStore swaps `q` before callers request the
// system backend.
func (s *Store) DaggerheartProjectionStore() projectionstore.Store {
	if s == nil {
		return nil
	}
	return sqlitedaggerheartprojection.Bind(s.sqlDB, s.q)
}
