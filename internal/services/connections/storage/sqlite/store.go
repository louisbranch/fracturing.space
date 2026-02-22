// Package sqlite provides a SQLite-backed connections storage implementation.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	sqlitemigrate "github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	"github.com/louisbranch/fracturing.space/internal/services/connections/storage"
	"github.com/louisbranch/fracturing.space/internal/services/connections/storage/sqlite/migrations"
	_ "modernc.org/sqlite"
)

// Store persists connections state in SQLite.
type Store struct {
	sqlDB *sql.DB
}

func toMillis(value time.Time) int64 {
	return value.UTC().UnixMilli()
}

func fromMillis(value int64) time.Time {
	return time.UnixMilli(value).UTC()
}

// Open opens a SQLite connections store and applies embedded migrations.
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

// PutContact upserts one directed owner-scoped contact relationship.
func (s *Store) PutContact(ctx context.Context, contact storage.Contact) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	ownerUserID := strings.TrimSpace(contact.OwnerUserID)
	contactUserID := strings.TrimSpace(contact.ContactUserID)
	if ownerUserID == "" {
		return fmt.Errorf("owner user id is required")
	}
	if contactUserID == "" {
		return fmt.Errorf("contact user id is required")
	}
	if ownerUserID == contactUserID {
		return fmt.Errorf("contact user id must differ from owner user id")
	}

	_, err := s.sqlDB.ExecContext(
		ctx,
		`INSERT INTO contacts (owner_user_id, contact_user_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(owner_user_id, contact_user_id) DO UPDATE SET
		   updated_at = excluded.updated_at`,
		ownerUserID,
		contactUserID,
		toMillis(contact.CreatedAt),
		toMillis(contact.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("put contact: %w", err)
	}
	return nil
}

// GetContact returns one directed owner-scoped contact relationship.
func (s *Store) GetContact(ctx context.Context, ownerUserID string, contactUserID string) (storage.Contact, error) {
	if err := ctx.Err(); err != nil {
		return storage.Contact{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.Contact{}, fmt.Errorf("storage is not configured")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	contactUserID = strings.TrimSpace(contactUserID)
	if ownerUserID == "" {
		return storage.Contact{}, fmt.Errorf("owner user id is required")
	}
	if contactUserID == "" {
		return storage.Contact{}, fmt.Errorf("contact user id is required")
	}

	row := s.sqlDB.QueryRowContext(
		ctx,
		`SELECT owner_user_id, contact_user_id, created_at, updated_at
		 FROM contacts
		 WHERE owner_user_id = ? AND contact_user_id = ?`,
		ownerUserID,
		contactUserID,
	)
	var contact storage.Contact
	var createdAt int64
	var updatedAt int64
	err := row.Scan(
		&contact.OwnerUserID,
		&contact.ContactUserID,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.Contact{}, storage.ErrNotFound
		}
		return storage.Contact{}, fmt.Errorf("get contact: %w", err)
	}
	contact.CreatedAt = fromMillis(createdAt)
	contact.UpdatedAt = fromMillis(updatedAt)
	return contact, nil
}

// DeleteContact removes one directed owner-scoped contact relationship.
func (s *Store) DeleteContact(ctx context.Context, ownerUserID string, contactUserID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	contactUserID = strings.TrimSpace(contactUserID)
	if ownerUserID == "" {
		return fmt.Errorf("owner user id is required")
	}
	if contactUserID == "" {
		return fmt.Errorf("contact user id is required")
	}

	_, err := s.sqlDB.ExecContext(
		ctx,
		`DELETE FROM contacts
		 WHERE owner_user_id = ? AND contact_user_id = ?`,
		ownerUserID,
		contactUserID,
	)
	if err != nil {
		return fmt.Errorf("delete contact: %w", err)
	}
	return nil
}

// ListContacts returns one page of owner-scoped directed contacts.
func (s *Store) ListContacts(ctx context.Context, ownerUserID string, pageSize int, pageToken string) (storage.ContactPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.ContactPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.ContactPage{}, fmt.Errorf("storage is not configured")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return storage.ContactPage{}, fmt.Errorf("owner user id is required")
	}
	if pageSize <= 0 {
		return storage.ContactPage{}, fmt.Errorf("page size must be greater than zero")
	}

	page := storage.ContactPage{
		Contacts: make([]storage.Contact, 0, pageSize),
	}
	pageToken = strings.TrimSpace(pageToken)

	var (
		rows *sql.Rows
		err  error
	)
	if pageToken == "" {
		rows, err = s.sqlDB.QueryContext(
			ctx,
			`SELECT owner_user_id, contact_user_id, created_at, updated_at
			 FROM contacts
			 WHERE owner_user_id = ?
			 ORDER BY contact_user_id ASC
			 LIMIT ?`,
			ownerUserID,
			pageSize+1,
		)
	} else {
		rows, err = s.sqlDB.QueryContext(
			ctx,
			`SELECT owner_user_id, contact_user_id, created_at, updated_at
			 FROM contacts
			 WHERE owner_user_id = ? AND contact_user_id > ?
			 ORDER BY contact_user_id ASC
			 LIMIT ?`,
			ownerUserID,
			pageToken,
			pageSize+1,
		)
	}
	if err != nil {
		return storage.ContactPage{}, fmt.Errorf("list contacts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			contact   storage.Contact
			createdAt int64
			updatedAt int64
		)
		if err := rows.Scan(
			&contact.OwnerUserID,
			&contact.ContactUserID,
			&createdAt,
			&updatedAt,
		); err != nil {
			return storage.ContactPage{}, fmt.Errorf("list contacts: %w", err)
		}
		contact.CreatedAt = fromMillis(createdAt)
		contact.UpdatedAt = fromMillis(updatedAt)
		page.Contacts = append(page.Contacts, contact)
	}
	if err := rows.Err(); err != nil {
		return storage.ContactPage{}, fmt.Errorf("list contacts: %w", err)
	}
	if len(page.Contacts) > pageSize {
		page.NextPageToken = page.Contacts[pageSize-1].ContactUserID
		page.Contacts = page.Contacts[:pageSize]
	}

	return page, nil
}

var _ storage.ContactStore = (*Store)(nil)
