// Package sqlite provides SQLite-backed override persistence for the status service.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	sqlitemigrate "github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	"github.com/louisbranch/fracturing.space/internal/services/status/domain"
	"github.com/louisbranch/fracturing.space/internal/services/status/storage/sqlite/migrations"
	_ "modernc.org/sqlite"
)

// Store persists capability overrides in SQLite.
type Store struct {
	sqlDB *sql.DB
}

// Open opens a SQLite override store and applies embedded migrations.
func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("storage path is required")
	}
	cleanPath := filepath.Clean(path)
	dsn := cleanPath + "?_journal_mode=WAL&_foreign_keys=ON&_busy_timeout=5000&_synchronous=NORMAL"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping sqlite db: %w", err)
	}
	if err := sqlitemigrate.ApplyMigrations(sqlDB, migrations.FS, ""); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	return &Store{sqlDB: sqlDB}, nil
}

// Close closes the SQLite handle.
func (s *Store) Close() error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}

// PutOverride upserts one capability override.
func (s *Store) PutOverride(ctx context.Context, ov domain.Override) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := s.sqlDB.ExecContext(ctx,
		`INSERT INTO capability_overrides (service, capability, status, reason, detail, set_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(service, capability) DO UPDATE SET
		   status = excluded.status,
		   reason = excluded.reason,
		   detail = excluded.detail,
		   set_at = excluded.set_at`,
		ov.Service, ov.Capability, int(ov.Status), int(ov.Reason), ov.Detail,
		ov.SetAt.UTC().UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("put override: %w", err)
	}
	return nil
}

// DeleteOverride removes one capability override.
func (s *Store) DeleteOverride(ctx context.Context, service, capability string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := s.sqlDB.ExecContext(ctx,
		`DELETE FROM capability_overrides WHERE service = ? AND capability = ?`,
		service, capability,
	)
	if err != nil {
		return fmt.Errorf("delete override: %w", err)
	}
	return nil
}

// ListOverrides returns all persisted overrides.
func (s *Store) ListOverrides(ctx context.Context) ([]domain.Override, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := s.sqlDB.QueryContext(ctx,
		`SELECT service, capability, status, reason, detail, set_at
		 FROM capability_overrides
		 ORDER BY service, capability`)
	if err != nil {
		return nil, fmt.Errorf("list overrides: %w", err)
	}
	defer rows.Close()

	var overrides []domain.Override
	for rows.Next() {
		var ov domain.Override
		var status, reason int
		var setAt int64
		if err := rows.Scan(&ov.Service, &ov.Capability, &status, &reason, &ov.Detail, &setAt); err != nil {
			return nil, fmt.Errorf("scan override: %w", err)
		}
		ov.Status = domain.CapabilityStatus(status)
		ov.Reason = domain.OverrideReason(reason)
		ov.SetAt = time.UnixMilli(setAt).UTC()
		overrides = append(overrides, ov)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list overrides: %w", err)
	}
	return overrides, nil
}
