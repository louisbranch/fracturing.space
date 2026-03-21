package daggerheartprojection

import (
	"database/sql"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

// Store provides the SQLite-backed Daggerheart projection backend.
//
// This backend binds to the root projections store's shared `sql.DB` and
// `db.Queries` handles so system adapters can rebind inside exact-once
// projection transactions without opening a second database.
type Store struct {
	sqlDB *sql.DB
	q     *db.Queries
}

var _ projectionstore.Store = (*Store)(nil)

// Bind creates a Daggerheart projection backend from an existing projections DB
// handle and query bundle.
func Bind(sqlDB *sql.DB, q *db.Queries) *Store {
	if sqlDB == nil || q == nil {
		return nil
	}
	return &Store{
		sqlDB: sqlDB,
		q:     q,
	}
}
