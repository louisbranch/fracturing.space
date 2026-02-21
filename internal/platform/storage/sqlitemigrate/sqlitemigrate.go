package sqlitemigrate

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const migrationTable = "schema_migrations"

// ApplyMigrations executes embedded migrations from migrationRoot at most once per file.
func ApplyMigrations(sqlDB *sql.DB, migrationFS fs.FS, migrationRoot string) error {
	if sqlDB == nil {
		return fmt.Errorf("sql db is required")
	}

	root := strings.TrimSpace(migrationRoot)
	if root == "" {
		root = "."
	}
	readRoot := root
	migrationKeyRoot := root
	if migrationKeyRoot == "." {
		migrationKeyRoot = ""
	}

	entries, err := fs.ReadDir(migrationFS, readRoot)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sqlFiles = append(sqlFiles, entry.Name())
		}
	}
	sort.Strings(sqlFiles)

	createSQL := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
    name TEXT PRIMARY KEY,
    applied_at INTEGER NOT NULL
);
`, migrationTable)
	if _, err := sqlDB.Exec(createSQL); err != nil {
		return fmt.Errorf("ensure migration table: %w", err)
	}

	for _, file := range sqlFiles {
		filePath := file
		if migrationKeyRoot != "" {
			filePath = filepath.ToSlash(filepath.Join(migrationKeyRoot, file))
		}

		content, err := fs.ReadFile(migrationFS, filepath.Join(readRoot, file))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}

		applied, err := isApplied(sqlDB, filePath)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", file, err)
		}
		if applied {
			continue
		}

		upSQL := ExtractUpMigration(string(content))
		if strings.TrimSpace(upSQL) == "" {
			continue
		}

		tx, err := sqlDB.BeginTx(context.Background(), nil)
		if err != nil {
			return fmt.Errorf("begin migration transaction %s: %w", file, err)
		}

		if _, err := tx.Exec(upSQL); err != nil {
			if !IsAlreadyExistsError(err) {
				_ = tx.Rollback()
				return fmt.Errorf("exec migration %s: %w", file, err)
			}
		}

		if _, err := tx.Exec(
			fmt.Sprintf("INSERT OR IGNORE INTO %s (name, applied_at) VALUES (?, ?)", migrationTable),
			filePath,
			time.Now().UTC().UnixMilli(),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", file, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", file, err)
		}
	}

	return nil
}

// ExtractUpMigration returns the SQL in the -- +migrate Up section.
func ExtractUpMigration(content string) string {
	upIdx := strings.Index(content, "-- +migrate Up")
	if upIdx == -1 {
		return content
	}
	downIdx := strings.Index(content, "-- +migrate Down")
	if downIdx == -1 {
		return content[upIdx+len("-- +migrate Up"):]
	}
	return content[upIdx+len("-- +migrate Up") : downIdx]
}

// IsAlreadyExistsError reports whether this error indicates idempotent DDL success.
func IsAlreadyExistsError(err error) bool {
	value := strings.ToLower(err.Error())
	return strings.Contains(value, "already exists") || strings.Contains(value, "duplicate column name")
}

func isApplied(sqlDB *sql.DB, name string) (bool, error) {
	var found int
	row := sqlDB.QueryRow("SELECT 1 FROM "+migrationTable+" WHERE name = ?", name)
	err := row.Scan(&found)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
