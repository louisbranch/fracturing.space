package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	sqlitemigrate "github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite/db"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite/migrations"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	_ "modernc.org/sqlite"
)

const authStatisticsQuery = `
SELECT COUNT(*)
FROM users
WHERE (?1 IS NULL OR created_at >= ?1);
`

// toMillis normalizes timestamps into millisecond precision for storage.
func toMillis(value time.Time) int64 {
	return value.UTC().UnixMilli()
}

// fromMillis restores millisecond precision and keeps UTC normalization.
func fromMillis(value int64) time.Time {
	return time.UnixMilli(value).UTC()
}

// Store implements auth persistence over SQLite.
//
// A single SQLite file backs identity state so every auth subflow can share the
// same transaction and visibility boundaries.
type Store struct {
	sqlDB *sql.DB
	q     *db.Queries
}

type execContexter interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// DB returns the raw database handle for OAuth and legacy callers.
func (s *Store) DB() *sql.DB {
	if s == nil {
		return nil
	}
	return s.sqlDB
}

// Open opens an auth SQLite store and applies bundled migrations.
//
// This keeps startup and schema evolution in one place, instead of requiring
// callers to coordinate migrations independently.
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

// Close releases the underlying SQLite database.
func (s *Store) Close() error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}

// runMigrations applies embedded DDL snapshots for known schema versions.
func (s *Store) runMigrations() error {
	return sqlitemigrate.ApplyMigrations(s.sqlDB, migrations.FS, "")
}

// extractUpMigration extracts only the upgrade section from a migration file.
func extractUpMigration(content string) string {
	return sqlitemigrate.ExtractUpMigration(content)
}

// isAlreadyExistsError detects SQLite "already exists" conditions during idempotent runs.
func isAlreadyExistsError(err error) bool {
	return sqlitemigrate.IsAlreadyExistsError(err)
}

// PutUser persists a user record and its primary email atomically.
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
	if strings.TrimSpace(u.Email) == "" {
		return fmt.Errorf("email is required")
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	qtx := s.q.WithTx(tx)

	if err := qtx.PutUser(ctx, db.PutUserParams{
		ID:        u.ID,
		Locale:    platformi18n.LocaleString(platformi18n.NormalizeLocale(u.Locale)),
		CreatedAt: toMillis(u.CreatedAt),
		UpdatedAt: toMillis(u.UpdatedAt),
	}); err != nil {
		return fmt.Errorf("put user: %w", err)
	}

	if err := qtx.PutUserPrimaryEmail(ctx, db.PutUserPrimaryEmailParams{
		ID:        u.ID,
		UserID:    u.ID,
		Email:     u.Email,
		CreatedAt: toMillis(u.CreatedAt),
		UpdatedAt: toMillis(u.UpdatedAt),
	}); err != nil {
		return fmt.Errorf("put user primary email: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit user: %w", err)
	}
	return nil
}

// PutUserWithIntegrationOutboxEvent persists user identity and one outbox event atomically.
func (s *Store) PutUserWithIntegrationOutboxEvent(ctx context.Context, u user.User, event storage.IntegrationOutboxEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(u.ID) == "" {
		return fmt.Errorf("user id is required")
	}
	if strings.TrimSpace(u.Email) == "" {
		return fmt.Errorf("email is required")
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	qtx := s.q.WithTx(tx)
	if err := qtx.PutUser(ctx, db.PutUserParams{
		ID:        u.ID,
		Locale:    platformi18n.LocaleString(platformi18n.NormalizeLocale(u.Locale)),
		CreatedAt: toMillis(u.CreatedAt),
		UpdatedAt: toMillis(u.UpdatedAt),
	}); err != nil {
		return fmt.Errorf("put user: %w", err)
	}
	if err := qtx.PutUserPrimaryEmail(ctx, db.PutUserPrimaryEmailParams{
		ID:        u.ID,
		UserID:    u.ID,
		Email:     u.Email,
		CreatedAt: toMillis(u.CreatedAt),
		UpdatedAt: toMillis(u.UpdatedAt),
	}); err != nil {
		return fmt.Errorf("put user primary email: %w", err)
	}
	if err := enqueueIntegrationOutboxEvent(ctx, tx, event); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit user + outbox: %w", err)
	}
	return nil
}

// PutAccountProfile stores account profile metadata for a user.
func (s *Store) PutAccountProfile(ctx context.Context, profile storage.AccountProfile) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(profile.UserID) == "" {
		return fmt.Errorf("user id is required")
	}

	return s.q.PutAccountProfile(ctx, db.PutAccountProfileParams{
		UserID:        profile.UserID,
		Name:          profile.Name,
		Locale:        platformi18n.LocaleString(profile.Locale),
		AvatarSetID:   profile.AvatarSetID,
		AvatarAssetID: profile.AvatarAssetID,
		CreatedAt:     toMillis(profile.CreatedAt),
		UpdatedAt:     toMillis(profile.UpdatedAt),
	})
}

// GetAccountProfile fetches profile metadata for a user.
func (s *Store) GetAccountProfile(ctx context.Context, userID string) (storage.AccountProfile, error) {
	if err := ctx.Err(); err != nil {
		return storage.AccountProfile{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.AccountProfile{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(userID) == "" {
		return storage.AccountProfile{}, fmt.Errorf("user id is required")
	}

	row, err := s.q.GetAccountProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.AccountProfile{}, storage.ErrNotFound
		}
		return storage.AccountProfile{}, fmt.Errorf("get account profile: %w", err)
	}

	profile, err := dbAccountProfileToDomain(row)
	if err != nil {
		return storage.AccountProfile{}, fmt.Errorf("parse account profile: %w", err)
	}

	return profile, nil
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

	return dbUserToDomain(row.ID, row.Email, row.Locale, row.CreatedAt, row.UpdatedAt), nil
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

	page := storage.UserPage{Users: make([]user.User, 0, pageSize)}

	switch {
	case pageToken == "":
		rows, err := s.q.ListUsersPagedFirst(ctx, int64(pageSize+1))
		if err != nil {
			return storage.UserPage{}, fmt.Errorf("list users: %w", err)
		}
		for i, row := range rows {
			if i >= pageSize {
				page.NextPageToken = rows[pageSize-1].ID
				break
			}
			page.Users = append(page.Users, dbUserToDomain(row.ID, row.Email, row.Locale, row.CreatedAt, row.UpdatedAt))
		}
	default:
		rows, err := s.q.ListUsersPaged(ctx, db.ListUsersPagedParams{
			ID:    pageToken,
			Limit: int64(pageSize + 1),
		})
		if err != nil {
			return storage.UserPage{}, fmt.Errorf("list users: %w", err)
		}
		for i, row := range rows {
			if i >= pageSize {
				page.NextPageToken = rows[pageSize-1].ID
				break
			}
			page.Users = append(page.Users, dbUserToDomain(row.ID, row.Email, row.Locale, row.CreatedAt, row.UpdatedAt))
		}
	}

	return page, nil
}

// PutContact stores one owner-scoped quick-lookup relationship.
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

	return s.q.PutContact(ctx, db.PutContactParams{
		OwnerUserID:   ownerUserID,
		ContactUserID: contactUserID,
		CreatedAt:     toMillis(contact.CreatedAt),
		UpdatedAt:     toMillis(contact.UpdatedAt),
	})
}

// GetContact fetches one owner-scoped contact.
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

	row, err := s.q.GetContact(ctx, db.GetContactParams{
		OwnerUserID:   ownerUserID,
		ContactUserID: contactUserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.Contact{}, storage.ErrNotFound
		}
		return storage.Contact{}, fmt.Errorf("get contact: %w", err)
	}

	return dbContactToDomain(row), nil
}

// DeleteContact removes one owner-scoped contact.
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

	return s.q.DeleteContact(ctx, db.DeleteContactParams{
		OwnerUserID:   ownerUserID,
		ContactUserID: contactUserID,
	})
}

// ListContacts returns one cursor page of owner-scoped contacts.
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

	page := storage.ContactPage{Contacts: make([]storage.Contact, 0, pageSize)}
	switch {
	case pageToken == "":
		rows, err := s.q.ListContactsPagedFirst(ctx, db.ListContactsPagedFirstParams{
			OwnerUserID: ownerUserID,
			Limit:       int64(pageSize + 1),
		})
		if err != nil {
			return storage.ContactPage{}, fmt.Errorf("list contacts: %w", err)
		}
		for i, row := range rows {
			if i >= pageSize {
				page.NextPageToken = rows[pageSize-1].ContactUserID
				break
			}
			page.Contacts = append(page.Contacts, dbContactToDomain(row))
		}
	default:
		rows, err := s.q.ListContactsPaged(ctx, db.ListContactsPagedParams{
			OwnerUserID:   ownerUserID,
			ContactUserID: pageToken,
			Limit:         int64(pageSize + 1),
		})
		if err != nil {
			return storage.ContactPage{}, fmt.Errorf("list contacts: %w", err)
		}
		for i, row := range rows {
			if i >= pageSize {
				page.NextPageToken = rows[pageSize-1].ContactUserID
				break
			}
			page.Contacts = append(page.Contacts, dbContactToDomain(row))
		}
	}

	return page, nil
}

// GetAuthStatistics returns aggregate counts across auth data.
func (s *Store) GetAuthStatistics(ctx context.Context, since *time.Time) (storage.AuthStatistics, error) {
	if err := ctx.Err(); err != nil {
		return storage.AuthStatistics{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.AuthStatistics{}, fmt.Errorf("storage is not configured")
	}

	var sinceValue any
	if since != nil {
		sinceValue = toMillis(*since)
	}

	var count int64
	row := s.sqlDB.QueryRowContext(ctx, authStatisticsQuery, sinceValue)
	if err := row.Scan(&count); err != nil {
		return storage.AuthStatistics{}, fmt.Errorf("get auth statistics: %w", err)
	}

	return storage.AuthStatistics{UserCount: count}, nil
}

func dbUserToDomain(id string, email string, locale string, createdAt int64, updatedAt int64) user.User {
	parsedLocale := platformi18n.DefaultLocale()
	if parsed, ok := platformi18n.ParseLocale(locale); ok {
		parsedLocale = parsed
	}

	return user.User{
		ID:        id,
		Email:     email,
		Locale:    parsedLocale,
		CreatedAt: fromMillis(createdAt),
		UpdatedAt: fromMillis(updatedAt),
	}
}

func dbContactToDomain(row db.UserContact) storage.Contact {
	return storage.Contact{
		OwnerUserID:   row.OwnerUserID,
		ContactUserID: row.ContactUserID,
		CreatedAt:     fromMillis(row.CreatedAt),
		UpdatedAt:     fromMillis(row.UpdatedAt),
	}
}

func dbAccountProfileToDomain(row db.GetAccountProfileRow) (storage.AccountProfile, error) {
	parsedLocale := platformi18n.DefaultLocale()
	if locale, ok := platformi18n.ParseLocale(row.Locale); ok {
		parsedLocale = locale
	}

	return storage.AccountProfile{
		UserID:        row.UserID,
		Name:          row.Name,
		Locale:        parsedLocale,
		AvatarSetID:   row.AvatarSetID,
		AvatarAssetID: row.AvatarAssetID,
		CreatedAt:     fromMillis(row.CreatedAt),
		UpdatedAt:     fromMillis(row.UpdatedAt),
	}, nil
}

// PutPasskeyCredential stores a WebAuthn credential.
func (s *Store) PutPasskeyCredential(ctx context.Context, credential storage.PasskeyCredential) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(credential.CredentialID) == "" {
		return fmt.Errorf("credential id is required")
	}
	if strings.TrimSpace(credential.UserID) == "" {
		return fmt.Errorf("user id is required")
	}
	if strings.TrimSpace(credential.CredentialJSON) == "" {
		return fmt.Errorf("credential json is required")
	}

	lastUsed := sql.NullInt64{}
	if credential.LastUsedAt != nil {
		lastUsed = sql.NullInt64{Int64: toMillis(*credential.LastUsedAt), Valid: true}
	}

	return s.q.PutPasskey(ctx, db.PutPasskeyParams{
		CredentialID:   credential.CredentialID,
		UserID:         credential.UserID,
		CredentialJson: credential.CredentialJSON,
		CreatedAt:      toMillis(credential.CreatedAt),
		UpdatedAt:      toMillis(credential.UpdatedAt),
		LastUsedAt:     lastUsed,
	})
}

// GetPasskeyCredential fetches a stored WebAuthn credential.
func (s *Store) GetPasskeyCredential(ctx context.Context, credentialID string) (storage.PasskeyCredential, error) {
	if err := ctx.Err(); err != nil {
		return storage.PasskeyCredential{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.PasskeyCredential{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(credentialID) == "" {
		return storage.PasskeyCredential{}, fmt.Errorf("credential id is required")
	}

	row, err := s.q.GetPasskey(ctx, credentialID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.PasskeyCredential{}, storage.ErrNotFound
		}
		return storage.PasskeyCredential{}, fmt.Errorf("get passkey: %w", err)
	}

	return dbPasskeyToDomain(row), nil
}

// ListPasskeyCredentials returns passkeys for a user.
func (s *Store) ListPasskeyCredentials(ctx context.Context, userID string) ([]storage.PasskeyCredential, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("user id is required")
	}

	rows, err := s.q.ListPasskeysByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list passkeys: %w", err)
	}

	credentials := make([]storage.PasskeyCredential, 0, len(rows))
	for _, row := range rows {
		credentials = append(credentials, dbPasskeyToDomain(row))
	}
	return credentials, nil
}

// DeletePasskeyCredential removes a passkey credential.
func (s *Store) DeletePasskeyCredential(ctx context.Context, credentialID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(credentialID) == "" {
		return fmt.Errorf("credential id is required")
	}
	return s.q.DeletePasskey(ctx, credentialID)
}

// PutPasskeySession stores a WebAuthn session.
func (s *Store) PutPasskeySession(ctx context.Context, session storage.PasskeySession) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(session.ID) == "" {
		return fmt.Errorf("session id is required")
	}
	if strings.TrimSpace(session.Kind) == "" {
		return fmt.Errorf("session kind is required")
	}
	if strings.TrimSpace(session.SessionJSON) == "" {
		return fmt.Errorf("session json is required")
	}

	userID := sql.NullString{}
	if strings.TrimSpace(session.UserID) != "" {
		userID = sql.NullString{String: session.UserID, Valid: true}
	}

	return s.q.PutPasskeySession(ctx, db.PutPasskeySessionParams{
		ID:          session.ID,
		Kind:        session.Kind,
		UserID:      userID,
		SessionJson: session.SessionJSON,
		ExpiresAt:   toMillis(session.ExpiresAt),
	})
}

// GetPasskeySession fetches a stored WebAuthn session.
func (s *Store) GetPasskeySession(ctx context.Context, id string) (storage.PasskeySession, error) {
	if err := ctx.Err(); err != nil {
		return storage.PasskeySession{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.PasskeySession{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.PasskeySession{}, fmt.Errorf("session id is required")
	}

	row, err := s.q.GetPasskeySession(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.PasskeySession{}, storage.ErrNotFound
		}
		return storage.PasskeySession{}, fmt.Errorf("get passkey session: %w", err)
	}

	return dbPasskeySessionToDomain(row), nil
}

// DeletePasskeySession removes a WebAuthn session.
func (s *Store) DeletePasskeySession(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("session id is required")
	}
	return s.q.DeletePasskeySession(ctx, id)
}

// DeleteExpiredPasskeySessions removes expired WebAuthn sessions.
func (s *Store) DeleteExpiredPasskeySessions(ctx context.Context, now time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	return s.q.DeleteExpiredPasskeySessions(ctx, toMillis(now))
}

func dbPasskeyToDomain(row db.Passkey) storage.PasskeyCredential {
	var lastUsed *time.Time
	if row.LastUsedAt.Valid {
		value := fromMillis(row.LastUsedAt.Int64)
		lastUsed = &value
	}
	return storage.PasskeyCredential{
		CredentialID:   row.CredentialID,
		UserID:         row.UserID,
		CredentialJSON: row.CredentialJson,
		CreatedAt:      fromMillis(row.CreatedAt),
		UpdatedAt:      fromMillis(row.UpdatedAt),
		LastUsedAt:     lastUsed,
	}
}

func dbPasskeySessionToDomain(row db.PasskeySession) storage.PasskeySession {
	userID := ""
	if row.UserID.Valid {
		userID = row.UserID.String
	}
	return storage.PasskeySession{
		ID:          row.ID,
		Kind:        row.Kind,
		UserID:      userID,
		SessionJSON: row.SessionJson,
		ExpiresAt:   fromMillis(row.ExpiresAt),
	}
}

// PutUserEmail stores a user email record.
func (s *Store) PutUserEmail(ctx context.Context, email storage.UserEmail) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(email.ID) == "" {
		return fmt.Errorf("email id is required")
	}
	if strings.TrimSpace(email.UserID) == "" {
		return fmt.Errorf("user id is required")
	}
	if strings.TrimSpace(email.Email) == "" {
		return fmt.Errorf("email is required")
	}

	verified := sql.NullInt64{}
	if email.VerifiedAt != nil {
		verified = sql.NullInt64{Int64: toMillis(*email.VerifiedAt), Valid: true}
	}

	return s.q.PutUserEmail(ctx, db.PutUserEmailParams{
		ID:         email.ID,
		UserID:     email.UserID,
		Email:      email.Email,
		IsPrimary:  0,
		VerifiedAt: verified,
		CreatedAt:  toMillis(email.CreatedAt),
		UpdatedAt:  toMillis(email.UpdatedAt),
	})
}

// GetUserEmailByEmail fetches a user email by email address.
func (s *Store) GetUserEmailByEmail(ctx context.Context, email string) (storage.UserEmail, error) {
	if err := ctx.Err(); err != nil {
		return storage.UserEmail{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.UserEmail{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(email) == "" {
		return storage.UserEmail{}, fmt.Errorf("email is required")
	}

	row, err := s.q.GetUserEmailByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.UserEmail{}, storage.ErrNotFound
		}
		return storage.UserEmail{}, fmt.Errorf("get user email: %w", err)
	}
	return dbUserEmailToDomain(row), nil
}

// ListUserEmailsByUser lists emails for a user.
func (s *Store) ListUserEmailsByUser(ctx context.Context, userID string) ([]storage.UserEmail, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("user id is required")
	}

	rows, err := s.q.ListUserEmailsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list user emails: %w", err)
	}
	emails := make([]storage.UserEmail, 0, len(rows))
	for _, row := range rows {
		emails = append(emails, dbUserEmailToDomain(row))
	}
	return emails, nil
}

// VerifyUserEmail marks an email as verified.
func (s *Store) VerifyUserEmail(ctx context.Context, userID string, email string, verifiedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("user id is required")
	}
	if strings.TrimSpace(email) == "" {
		return fmt.Errorf("email is required")
	}
	return s.q.UpdateUserEmailVerified(ctx, db.UpdateUserEmailVerifiedParams{
		VerifiedAt: sql.NullInt64{Int64: toMillis(verifiedAt), Valid: true},
		UpdatedAt:  toMillis(verifiedAt),
		Email:      email,
		UserID:     userID,
	})
}

// PutMagicLink stores a magic link token.
func (s *Store) PutMagicLink(ctx context.Context, link storage.MagicLink) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(link.Token) == "" {
		return fmt.Errorf("token is required")
	}
	if strings.TrimSpace(link.UserID) == "" {
		return fmt.Errorf("user id is required")
	}
	if strings.TrimSpace(link.Email) == "" {
		return fmt.Errorf("email is required")
	}

	pending := sql.NullString{}
	if strings.TrimSpace(link.PendingID) != "" {
		pending = sql.NullString{String: link.PendingID, Valid: true}
	}
	used := sql.NullInt64{}
	if link.UsedAt != nil {
		used = sql.NullInt64{Int64: toMillis(*link.UsedAt), Valid: true}
	}

	return s.q.PutMagicLink(ctx, db.PutMagicLinkParams{
		Token:     link.Token,
		UserID:    link.UserID,
		Email:     link.Email,
		PendingID: pending,
		CreatedAt: toMillis(link.CreatedAt),
		ExpiresAt: toMillis(link.ExpiresAt),
		UsedAt:    used,
	})
}

// GetMagicLink fetches a magic link token.
func (s *Store) GetMagicLink(ctx context.Context, token string) (storage.MagicLink, error) {
	if err := ctx.Err(); err != nil {
		return storage.MagicLink{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.MagicLink{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(token) == "" {
		return storage.MagicLink{}, fmt.Errorf("token is required")
	}

	row, err := s.q.GetMagicLink(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.MagicLink{}, storage.ErrNotFound
		}
		return storage.MagicLink{}, fmt.Errorf("get magic link: %w", err)
	}
	return dbMagicLinkToDomain(row), nil
}

// MarkMagicLinkUsed marks a magic link as used.
func (s *Store) MarkMagicLinkUsed(ctx context.Context, token string, usedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("token is required")
	}
	return s.q.MarkMagicLinkUsed(ctx, db.MarkMagicLinkUsedParams{
		UsedAt: sql.NullInt64{Int64: toMillis(usedAt), Valid: true},
		Token:  token,
	})
}

// EnqueueIntegrationOutboxEvent stores one integration outbox event.
//
// If a non-empty dedupe key already exists, the enqueue is treated as a no-op.
func (s *Store) EnqueueIntegrationOutboxEvent(ctx context.Context, event storage.IntegrationOutboxEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	return enqueueIntegrationOutboxEvent(ctx, s.sqlDB, event)
}

// GetIntegrationOutboxEvent returns one integration outbox event by ID.
func (s *Store) GetIntegrationOutboxEvent(ctx context.Context, id string) (storage.IntegrationOutboxEvent, error) {
	if err := ctx.Err(); err != nil {
		return storage.IntegrationOutboxEvent{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.IntegrationOutboxEvent{}, fmt.Errorf("storage is not configured")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return storage.IntegrationOutboxEvent{}, fmt.Errorf("event id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT
	id,
	event_type,
	payload_json,
	dedupe_key,
	status,
	attempt_count,
	next_attempt_at,
	lease_owner,
	lease_expires_at,
	last_error,
	processed_at,
	created_at,
	updated_at
FROM auth_integration_outbox
WHERE id = ?
`, id)
	event, err := scanIntegrationOutboxEvent(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.IntegrationOutboxEvent{}, storage.ErrNotFound
		}
		return storage.IntegrationOutboxEvent{}, fmt.Errorf("get integration outbox event: %w", err)
	}
	return event, nil
}

// LeaseIntegrationOutboxEvents leases due integration outbox events for one worker.
func (s *Store) LeaseIntegrationOutboxEvents(ctx context.Context, consumer string, limit int, now time.Time, leaseTTL time.Duration) ([]storage.IntegrationOutboxEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	consumer = strings.TrimSpace(consumer)
	if consumer == "" {
		return nil, fmt.Errorf("consumer is required")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}
	if leaseTTL <= 0 {
		return nil, fmt.Errorf("lease ttl must be greater than zero")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	leaseExpiresAt := now.Add(leaseTTL)

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("start lease transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	rows, err := tx.QueryContext(ctx, `
SELECT id
FROM auth_integration_outbox
WHERE (
	(status = ? AND next_attempt_at <= ?)
	OR
	(status = ? AND lease_expires_at IS NOT NULL AND lease_expires_at <= ?)
)
ORDER BY next_attempt_at ASC, created_at ASC, id ASC
LIMIT ?
`,
		storage.IntegrationOutboxStatusPending,
		toMillis(now),
		storage.IntegrationOutboxStatusLeased,
		toMillis(now),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("select lease candidates: %w", err)
	}
	candidateIDs := make([]string, 0, limit)
	for rows.Next() {
		var id string
		if scanErr := rows.Scan(&id); scanErr != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan lease candidate: %w", scanErr)
		}
		candidateIDs = append(candidateIDs, id)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate lease candidates: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close lease candidates: %w", err)
	}
	if len(candidateIDs) == 0 {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit empty lease transaction: %w", err)
		}
		return []storage.IntegrationOutboxEvent{}, nil
	}

	leased := make([]storage.IntegrationOutboxEvent, 0, len(candidateIDs))
	for _, id := range candidateIDs {
		result, updateErr := tx.ExecContext(ctx, `
UPDATE auth_integration_outbox
SET
	status = ?,
	lease_owner = ?,
	lease_expires_at = ?,
	updated_at = ?
WHERE id = ?
AND (
	(status = ? AND next_attempt_at <= ?)
	OR
	(status = ? AND lease_expires_at IS NOT NULL AND lease_expires_at <= ?)
)
`,
			storage.IntegrationOutboxStatusLeased,
			consumer,
			toMillis(leaseExpiresAt),
			toMillis(now),
			id,
			storage.IntegrationOutboxStatusPending,
			toMillis(now),
			storage.IntegrationOutboxStatusLeased,
			toMillis(now),
		)
		if updateErr != nil {
			return nil, fmt.Errorf("lease integration outbox event %s: %w", id, updateErr)
		}
		rowsAffected, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			return nil, fmt.Errorf("lease rows affected for %s: %w", id, rowsErr)
		}
		if rowsAffected == 0 {
			continue
		}

		row := tx.QueryRowContext(ctx, `
SELECT
	id,
	event_type,
	payload_json,
	dedupe_key,
	status,
	attempt_count,
	next_attempt_at,
	lease_owner,
	lease_expires_at,
	last_error,
	processed_at,
	created_at,
	updated_at
FROM auth_integration_outbox
WHERE id = ?
`, id)
		event, scanErr := scanIntegrationOutboxEvent(row.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("scan leased integration outbox event %s: %w", id, scanErr)
		}
		leased = append(leased, event)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit lease transaction: %w", err)
	}
	return leased, nil
}

// MarkIntegrationOutboxSucceeded marks one leased integration outbox event as succeeded.
func (s *Store) MarkIntegrationOutboxSucceeded(ctx context.Context, id string, consumer string, processedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}

	id = strings.TrimSpace(id)
	consumer = strings.TrimSpace(consumer)
	if id == "" {
		return fmt.Errorf("event id is required")
	}
	if consumer == "" {
		return fmt.Errorf("consumer is required")
	}
	if processedAt.IsZero() {
		processedAt = time.Now().UTC()
	}
	processedAt = processedAt.UTC()

	result, err := s.sqlDB.ExecContext(ctx, `
UPDATE auth_integration_outbox
SET
	status = ?,
	lease_owner = '',
	lease_expires_at = NULL,
	last_error = '',
	processed_at = ?,
	updated_at = ?
WHERE id = ?
AND status = ?
AND lease_owner = ?
`,
		storage.IntegrationOutboxStatusSucceeded,
		toMillis(processedAt),
		toMillis(processedAt),
		id,
		storage.IntegrationOutboxStatusLeased,
		consumer,
	)
	if err != nil {
		return fmt.Errorf("mark integration outbox succeeded: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark integration outbox succeeded rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// MarkIntegrationOutboxRetry marks one leased integration outbox event for retry.
func (s *Store) MarkIntegrationOutboxRetry(ctx context.Context, id string, consumer string, nextAttemptAt time.Time, lastError string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}

	id = strings.TrimSpace(id)
	consumer = strings.TrimSpace(consumer)
	lastError = strings.TrimSpace(lastError)
	if id == "" {
		return fmt.Errorf("event id is required")
	}
	if consumer == "" {
		return fmt.Errorf("consumer is required")
	}
	if nextAttemptAt.IsZero() {
		return fmt.Errorf("next attempt at is required")
	}
	now := time.Now().UTC()
	nextAttemptAt = nextAttemptAt.UTC()

	result, err := s.sqlDB.ExecContext(ctx, `
UPDATE auth_integration_outbox
SET
	status = ?,
	attempt_count = attempt_count + 1,
	next_attempt_at = ?,
	lease_owner = '',
	lease_expires_at = NULL,
	last_error = ?,
	processed_at = NULL,
	updated_at = ?
WHERE id = ?
AND status = ?
AND lease_owner = ?
`,
		storage.IntegrationOutboxStatusPending,
		toMillis(nextAttemptAt),
		lastError,
		toMillis(now),
		id,
		storage.IntegrationOutboxStatusLeased,
		consumer,
	)
	if err != nil {
		return fmt.Errorf("mark integration outbox retry: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark integration outbox retry rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// MarkIntegrationOutboxDead marks one leased integration outbox event as dead.
func (s *Store) MarkIntegrationOutboxDead(ctx context.Context, id string, consumer string, lastError string, processedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}

	id = strings.TrimSpace(id)
	consumer = strings.TrimSpace(consumer)
	lastError = strings.TrimSpace(lastError)
	if id == "" {
		return fmt.Errorf("event id is required")
	}
	if consumer == "" {
		return fmt.Errorf("consumer is required")
	}
	if processedAt.IsZero() {
		processedAt = time.Now().UTC()
	}
	processedAt = processedAt.UTC()

	result, err := s.sqlDB.ExecContext(ctx, `
UPDATE auth_integration_outbox
SET
	status = ?,
	attempt_count = attempt_count + 1,
	lease_owner = '',
	lease_expires_at = NULL,
	last_error = ?,
	processed_at = ?,
	updated_at = ?
WHERE id = ?
AND status = ?
AND lease_owner = ?
`,
		storage.IntegrationOutboxStatusDead,
		lastError,
		toMillis(processedAt),
		toMillis(processedAt),
		id,
		storage.IntegrationOutboxStatusLeased,
		consumer,
	)
	if err != nil {
		return fmt.Errorf("mark integration outbox dead: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark integration outbox dead rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

type integrationOutboxScanner func(dest ...any) error

func scanIntegrationOutboxEvent(scan integrationOutboxScanner) (storage.IntegrationOutboxEvent, error) {
	var event storage.IntegrationOutboxEvent
	var nextAttemptAt int64
	var createdAt int64
	var updatedAt int64
	var leaseExpiresAt sql.NullInt64
	var processedAt sql.NullInt64
	if err := scan(
		&event.ID,
		&event.EventType,
		&event.PayloadJSON,
		&event.DedupeKey,
		&event.Status,
		&event.AttemptCount,
		&nextAttemptAt,
		&event.LeaseOwner,
		&leaseExpiresAt,
		&event.LastError,
		&processedAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return storage.IntegrationOutboxEvent{}, err
	}
	event.NextAttemptAt = fromMillis(nextAttemptAt)
	event.CreatedAt = fromMillis(createdAt)
	event.UpdatedAt = fromMillis(updatedAt)
	if leaseExpiresAt.Valid {
		value := fromMillis(leaseExpiresAt.Int64)
		event.LeaseExpiresAt = &value
	}
	if processedAt.Valid {
		value := fromMillis(processedAt.Int64)
		event.ProcessedAt = &value
	}
	return event, nil
}

func normalizeIntegrationOutboxEvent(event storage.IntegrationOutboxEvent) (storage.IntegrationOutboxEvent, error) {
	event.ID = strings.TrimSpace(event.ID)
	event.EventType = strings.TrimSpace(event.EventType)
	event.PayloadJSON = strings.TrimSpace(event.PayloadJSON)
	event.DedupeKey = strings.TrimSpace(event.DedupeKey)
	event.Status = strings.TrimSpace(event.Status)
	event.LeaseOwner = strings.TrimSpace(event.LeaseOwner)
	event.LastError = strings.TrimSpace(event.LastError)
	if event.ID == "" {
		return storage.IntegrationOutboxEvent{}, fmt.Errorf("event id is required")
	}
	if event.EventType == "" {
		return storage.IntegrationOutboxEvent{}, fmt.Errorf("event type is required")
	}
	if event.PayloadJSON == "" {
		event.PayloadJSON = "{}"
	}
	if event.Status == "" {
		event.Status = storage.IntegrationOutboxStatusPending
	}
	if event.AttemptCount < 0 {
		return storage.IntegrationOutboxEvent{}, fmt.Errorf("attempt count must be greater than or equal to zero")
	}
	now := time.Now().UTC()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	if event.UpdatedAt.IsZero() {
		event.UpdatedAt = event.CreatedAt
	}
	if event.NextAttemptAt.IsZero() {
		event.NextAttemptAt = event.CreatedAt
	}
	return event, nil
}

func enqueueIntegrationOutboxEvent(ctx context.Context, target execContexter, event storage.IntegrationOutboxEvent) error {
	normalized, err := normalizeIntegrationOutboxEvent(event)
	if err != nil {
		return err
	}

	var leaseExpiresAt sql.NullInt64
	if normalized.LeaseExpiresAt != nil {
		leaseExpiresAt = sql.NullInt64{Int64: toMillis(normalized.LeaseExpiresAt.UTC()), Valid: true}
	}
	var processedAt sql.NullInt64
	if normalized.ProcessedAt != nil {
		processedAt = sql.NullInt64{Int64: toMillis(normalized.ProcessedAt.UTC()), Valid: true}
	}

	_, err = target.ExecContext(ctx, `
INSERT INTO auth_integration_outbox (
	id,
	event_type,
	payload_json,
	dedupe_key,
	status,
	attempt_count,
	next_attempt_at,
	lease_owner,
	lease_expires_at,
	last_error,
	processed_at,
	created_at,
	updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(dedupe_key) WHERE dedupe_key <> '' DO NOTHING
`,
		normalized.ID,
		normalized.EventType,
		normalized.PayloadJSON,
		normalized.DedupeKey,
		normalized.Status,
		normalized.AttemptCount,
		toMillis(normalized.NextAttemptAt),
		normalized.LeaseOwner,
		leaseExpiresAt,
		normalized.LastError,
		processedAt,
		toMillis(normalized.CreatedAt),
		toMillis(normalized.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("enqueue integration outbox event: %w", err)
	}
	return nil
}

func dbUserEmailToDomain(row db.UserEmail) storage.UserEmail {
	var verified *time.Time
	if row.VerifiedAt.Valid {
		value := fromMillis(row.VerifiedAt.Int64)
		verified = &value
	}
	return storage.UserEmail{
		ID:         row.ID,
		UserID:     row.UserID,
		Email:      row.Email,
		VerifiedAt: verified,
		CreatedAt:  fromMillis(row.CreatedAt),
		UpdatedAt:  fromMillis(row.UpdatedAt),
	}
}

func dbMagicLinkToDomain(row db.MagicLink) storage.MagicLink {
	pendingID := ""
	if row.PendingID.Valid {
		pendingID = row.PendingID.String
	}
	var used *time.Time
	if row.UsedAt.Valid {
		value := fromMillis(row.UsedAt.Int64)
		used = &value
	}
	return storage.MagicLink{
		Token:     row.Token,
		UserID:    row.UserID,
		Email:     row.Email,
		PendingID: pendingID,
		CreatedAt: fromMillis(row.CreatedAt),
		ExpiresAt: fromMillis(row.ExpiresAt),
		UsedAt:    used,
	}
}

var _ storage.UserStore = (*Store)(nil)
var _ storage.AccountProfileStore = (*Store)(nil)
var _ storage.StatisticsStore = (*Store)(nil)
var _ storage.PasskeyStore = (*Store)(nil)
var _ storage.EmailStore = (*Store)(nil)
var _ storage.MagicLinkStore = (*Store)(nil)
var _ storage.ContactStore = (*Store)(nil)
var _ storage.IntegrationOutboxStore = (*Store)(nil)
var _ storage.UserOutboxTransactionalStore = (*Store)(nil)
