package coreprojection

import (
	"context"
	"database/sql"
)

// projectionQueryable abstracts the database handle used by projection
// methods. There are two tx-routing paths in the Store:
//
//   - s.q (the sqlc *db.Queries handle) is replaced wholesale by txStore(),
//     which clones the Store and calls s.q.WithTx(tx). Most existing methods
//     use s.q directly.
//   - projectionQueryable() checks s.tx directly and returns either the active
//     transaction or the bare *sql.DB. This is used by methods that build
//     dynamic SQL outside of sqlc (e.g. ListEventsPage-style queries).
//
// New methods should prefer projectionQueryable() for consistency.
type projectionQueryable interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (s *Store) projectionQueryable() projectionQueryable {
	if s == nil {
		return nil
	}
	if s.tx != nil {
		return s.tx
	}
	return s.sqlDB
}
