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

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
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

func toMillis(value time.Time) int64 {
	return value.UTC().UnixMilli()
}

func fromMillis(value int64) time.Time {
	return time.UnixMilli(value).UTC()
}

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
		Locale:      platformi18n.LocaleString(u.Locale),
		CreatedAt:   toMillis(u.CreatedAt),
		UpdatedAt:   toMillis(u.UpdatedAt),
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

func dbUserToDomain(row db.User) (user.User, error) {
	locale := platformi18n.DefaultLocale()
	if parsed, ok := platformi18n.ParseLocale(row.Locale); ok {
		locale = parsed
	}
	return user.User{
		ID:          row.ID,
		DisplayName: row.DisplayName,
		Locale:      locale,
		CreatedAt:   fromMillis(row.CreatedAt),
		UpdatedAt:   fromMillis(row.UpdatedAt),
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
var _ storage.StatisticsStore = (*Store)(nil)
var _ storage.PasskeyStore = (*Store)(nil)
var _ storage.EmailStore = (*Store)(nil)
var _ storage.MagicLinkStore = (*Store)(nil)
