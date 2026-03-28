package sqlite

import (
	"database/sql"
	"fmt"
)

func requireStoreDB(s *Store) (*sql.DB, error) {
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	return s.sqlDB, nil
}

func keysetPageLimit(pageSize int) (int, error) {
	if pageSize <= 0 {
		return 0, fmt.Errorf("page size must be greater than zero")
	}
	return pageSize + 1, nil
}

func scanIDKeysetPage[T any](rows *sql.Rows, pageSize int, scanRow func(scanner) (T, error), rowName string, idOf func(T) string) ([]T, string, error) {
	items := make([]T, 0, pageSize)
	for rows.Next() {
		item, err := scanRow(rows)
		if err != nil {
			return nil, "", fmt.Errorf("scan %s row: %w", rowName, err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate %s rows: %w", rowName, err)
	}
	if len(items) <= pageSize {
		return items, "", nil
	}
	return items[:pageSize], idOf(items[pageSize-1]), nil
}
