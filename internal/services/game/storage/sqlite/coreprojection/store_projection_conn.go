package coreprojection

import (
	"context"
	"database/sql"
)

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
