// Package sqliteutil provides shared helper functions for SQLite storage packages.
package sqliteutil

import (
	"database/sql"
	"strings"
	"time"
)

// ToMillis converts a time.Time to milliseconds since the Unix epoch (UTC).
func ToMillis(value time.Time) int64 {
	return value.UTC().UnixMilli()
}

// FromMillis converts milliseconds since the Unix epoch to a UTC time.Time.
func FromMillis(value int64) time.Time {
	return time.UnixMilli(value).UTC()
}

// ToNullMillis maps optional domain times to sql.NullInt64 for nullable DB columns.
func ToNullMillis(value *time.Time) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: ToMillis(*value), Valid: true}
}

// FromNullMillis maps nullable SQL timestamps back into optional domain time values.
func FromNullMillis(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}
	t := FromMillis(value.Int64)
	return &t
}

// ToNullString converts a string to sql.NullString, treating whitespace-only strings as null.
func ToNullString(value string) sql.NullString {
	if strings.TrimSpace(value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

// MapPageRows applies a row-to-domain mapper to at most pageSize rows and returns
// the next-page token when additional rows are available.
//
// Callers are expected to query using pageSize+1 rows and validate pageSize before
// calling this helper.
func MapPageRows[Row any, Item any](
	rows []Row,
	pageSize int,
	rowID func(Row) string,
	mapRow func(Row) (Item, error),
) ([]Item, string, error) {
	capHint := pageSize
	if capHint > len(rows) {
		capHint = len(rows)
	}
	items := make([]Item, 0, capHint)

	for i, row := range rows {
		if i >= pageSize {
			return items, rowID(rows[pageSize-1]), nil
		}
		item, err := mapRow(row)
		if err != nil {
			return nil, "", err
		}
		items = append(items, item)
	}

	return items, "", nil
}
