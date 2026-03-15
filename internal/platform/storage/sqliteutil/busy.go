package sqliteutil

import (
	"errors"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// IsSQLiteBusyError reports whether the error is a SQLite BUSY or LOCKED error.
func IsSQLiteBusyError(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	code := sqliteErr.Code()
	return code == sqlite3.SQLITE_BUSY || code == sqlite3.SQLITE_LOCKED
}
