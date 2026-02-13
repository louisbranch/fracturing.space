package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite/db"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite/migrations"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	_ "modernc.org/sqlite"
)

const timeFormat = time.RFC3339Nano

// Store provides a SQLite-backed store implementing auth storage interfaces.
type Store struct {
	sqlDB *sql.DB
	q     *db.Queries
}

// DB returns the underlying sql.DB instance.
func (s *Store) DB() *sql.DB {
	if s == nil {
		return nil
	}
	return s.sqlDB
}

// Open opens a SQLite store at the provided path.
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

	store := &Store{
		sqlDB: sqlDB,
		q:     db.New(sqlDB),
	}

	if err := store.runMigrations(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return store, nil
}

// Close closes the underlying SQLite database.
func (s *Store) Close() error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}

// runMigrations runs embedded SQL migrations.
func (s *Store) runMigrations() error {
	entries, err := fs.ReadDir(migrations.FS, ".")
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

	for _, file := range sqlFiles {
		content, err := fs.ReadFile(migrations.FS, file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}

		upSQL := extractUpMigration(string(content))
		if upSQL == "" {
			continue
		}

		if _, err := s.sqlDB.Exec(upSQL); err != nil {
			if !isAlreadyExistsError(err) {
				return fmt.Errorf("exec migration %s: %w", file, err)
			}
		}
	}

	return nil
}

// extractUpMigration extracts the Up migration portion from a migration file.
func extractUpMigration(content string) string {
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

// isAlreadyExistsError checks if the error is a table/index already exists error.
func isAlreadyExistsError(err error) bool {
	return strings.Contains(err.Error(), "already exists")
}

// PutUser persists a user record.
func (s *Store) PutUser(ctx context.Context, u user.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(u.ID) == "" {
		return fmt.Errorf("user id is required")
	}

	return s.q.PutUser(ctx, db.PutUserParams{
		ID:          u.ID,
		DisplayName: u.DisplayName,
		CreatedAt:   u.CreatedAt.Format(timeFormat),
		UpdatedAt:   u.UpdatedAt.Format(timeFormat),
	})
}

// GetUser fetches a user record by ID.
func (s *Store) GetUser(ctx context.Context, userID string) (user.User, error) {
	if err := ctx.Err(); err != nil {
		return user.User{}, err
	}
	if s == nil || s.sqlDB == nil {
		return user.User{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(userID) == "" {
		return user.User{}, fmt.Errorf("user id is required")
	}

	row, err := s.q.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user.User{}, storage.ErrNotFound
		}
		return user.User{}, fmt.Errorf("get user: %w", err)
	}

	return dbUserToDomain(row)
}

// ListUsers returns a page of user records.
func (s *Store) ListUsers(ctx context.Context, pageSize int, pageToken string) (storage.UserPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.UserPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.UserPage{}, fmt.Errorf("storage is not configured")
	}
	if pageSize <= 0 {
		return storage.UserPage{}, fmt.Errorf("page size must be greater than zero")
	}

	var rows []db.User
	var err error

	if pageToken == "" {
		rows, err = s.q.ListUsersPagedFirst(ctx, int64(pageSize+1))
	} else {
		rows, err = s.q.ListUsersPaged(ctx, db.ListUsersPagedParams{
			ID:    pageToken,
			Limit: int64(pageSize + 1),
		})
	}
	if err != nil {
		return storage.UserPage{}, fmt.Errorf("list users: %w", err)
	}

	page := storage.UserPage{Users: make([]user.User, 0, pageSize)}

	for i, row := range rows {
		if i >= pageSize {
			page.NextPageToken = rows[pageSize-1].ID
			break
		}
		u, err := dbUserToDomain(row)
		if err != nil {
			return storage.UserPage{}, err
		}
		page.Users = append(page.Users, u)
	}

	return page, nil
}

func dbUserToDomain(row db.User) (user.User, error) {
	createdAt, err := time.Parse(timeFormat, row.CreatedAt)
	if err != nil {
		return user.User{}, fmt.Errorf("parse created_at: %w", err)
	}
	updatedAt, err := time.Parse(timeFormat, row.UpdatedAt)
	if err != nil {
		return user.User{}, fmt.Errorf("parse updated_at: %w", err)
	}

	return user.User{
		ID:          row.ID,
		DisplayName: row.DisplayName,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

var _ storage.UserStore = (*Store)(nil)
