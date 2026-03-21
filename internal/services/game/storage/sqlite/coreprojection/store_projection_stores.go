package coreprojection

import (
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
	sqlitedaggerheartprojection "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/daggerheartprojection"
)

// ProjectionStores binds the built-in system-owned projection backends to this
// root projections store's current query bundle. Transaction clones keep
// working because txStore swaps `q` before callers request the system bundle.
func (s *Store) ProjectionStores() systemmanifest.ProjectionStores {
	if s == nil {
		return systemmanifest.ProjectionStores{}
	}
	return systemmanifest.ProjectionStores{
		Daggerheart: sqlitedaggerheartprojection.Bind(s.sqlDB, s.q),
	}
}
