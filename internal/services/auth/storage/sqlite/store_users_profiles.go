package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite/db"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	authusername "github.com/louisbranch/fracturing.space/internal/services/auth/username"
)

func (s *Store) PutUser(ctx context.Context, u user.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("Storage is not configured.")
	}
	params, err := normalizeUserPutParams(u)
	if err != nil {
		return err
	}

	return s.q.PutUser(ctx, params)
}

// PutUserWithIntegrationOutboxEvent persists user identity and one outbox event atomically.
func (s *Store) PutUserWithIntegrationOutboxEvent(ctx context.Context, u user.User, event storage.IntegrationOutboxEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("Storage is not configured.")
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("Start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	qtx := s.q.WithTx(tx)
	params, err := normalizeUserPutParams(u)
	if err != nil {
		return err
	}
	if err := qtx.PutUser(ctx, params); err != nil {
		return fmt.Errorf("Put user: %w", err)
	}
	if err := enqueueIntegrationOutboxEvent(ctx, tx, event); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("Commit user + outbox: %w", err)
	}
	return nil
}

// PutUserPasskeyWithIntegrationOutboxEvent persists signup identity, the first
// passkey, the initial web session, and one integration outbox event atomically.
func (s *Store) PutUserPasskeyWithIntegrationOutboxEvent(ctx context.Context, u user.User, credential storage.PasskeyCredential, session storage.WebSession, event storage.IntegrationOutboxEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("Storage is not configured.")
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("Start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	qtx := s.q.WithTx(tx)
	userParams, err := normalizeUserPutParams(u)
	if err != nil {
		return err
	}
	if err := qtx.PutUser(ctx, userParams); err != nil {
		return fmt.Errorf("Put user: %w", err)
	}

	passkeyParams, err := normalizePasskeyPutParams(credential)
	if err != nil {
		return err
	}
	if err := qtx.PutPasskey(ctx, passkeyParams); err != nil {
		return fmt.Errorf("Put passkey: %w", err)
	}

	if err := putWebSessionWithExecutor(ctx, tx, session); err != nil {
		return err
	}

	if err := enqueueIntegrationOutboxEvent(ctx, tx, event); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("Commit signup persistence: %w", err)
	}
	return nil
}

// GetUser fetches a user record by ID.
func (s *Store) GetUser(ctx context.Context, userID string) (user.User, error) {
	if err := ctx.Err(); err != nil {
		return user.User{}, err
	}
	if s == nil || s.sqlDB == nil {
		return user.User{}, fmt.Errorf("Storage is not configured.")
	}
	if strings.TrimSpace(userID) == "" {
		return user.User{}, fmt.Errorf("User ID is required.")
	}

	row, err := s.q.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user.User{}, storage.ErrNotFound
		}
		return user.User{}, fmt.Errorf("Get user: %w", err)
	}

	return dbUserToDomain(row), nil
}

// GetUserByUsername fetches a user record by username.
func (s *Store) GetUserByUsername(ctx context.Context, username string) (user.User, error) {
	if err := ctx.Err(); err != nil {
		return user.User{}, err
	}
	if s == nil || s.sqlDB == nil {
		return user.User{}, fmt.Errorf("Storage is not configured.")
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return user.User{}, fmt.Errorf("Username is required.")
	}

	row, err := s.q.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user.User{}, storage.ErrNotFound
		}
		return user.User{}, fmt.Errorf("Get user by username: %w", err)
	}
	return dbUserToDomain(row), nil
}

// ListUsers returns a page of user records.
func (s *Store) ListUsers(ctx context.Context, pageSize int, pageToken string) (storage.UserPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.UserPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.UserPage{}, fmt.Errorf("Storage is not configured.")
	}
	if pageSize <= 0 {
		return storage.UserPage{}, fmt.Errorf("Page size must be greater than zero.")
	}

	page := storage.UserPage{Users: make([]user.User, 0, pageSize)}

	switch {
	case pageToken == "":
		rows, err := s.q.ListUsersPagedFirst(ctx, int64(pageSize+1))
		if err != nil {
			return storage.UserPage{}, fmt.Errorf("List users: %w", err)
		}
		for i, row := range rows {
			if i >= pageSize {
				page.NextPageToken = rows[pageSize-1].ID
				break
			}
			page.Users = append(page.Users, dbUserToDomain(row))
		}
	default:
		rows, err := s.q.ListUsersPaged(ctx, db.ListUsersPagedParams{
			ID:    pageToken,
			Limit: int64(pageSize + 1),
		})
		if err != nil {
			return storage.UserPage{}, fmt.Errorf("List users: %w", err)
		}
		for i, row := range rows {
			if i >= pageSize {
				page.NextPageToken = rows[pageSize-1].ID
				break
			}
			page.Users = append(page.Users, dbUserToDomain(row))
		}
	}

	return page, nil
}

func normalizeUserPutParams(u user.User) (db.PutUserParams, error) {
	userID := strings.TrimSpace(u.ID)
	if userID == "" {
		return db.PutUserParams{}, fmt.Errorf("User ID is required.")
	}
	if strings.TrimSpace(u.Username) == "" {
		return db.PutUserParams{}, fmt.Errorf("Username is required.")
	}
	username, err := authusername.Canonicalize(u.Username)
	if err != nil {
		return db.PutUserParams{}, fmt.Errorf("Username must match the required format.")
	}
	if u.CreatedAt.IsZero() {
		return db.PutUserParams{}, fmt.Errorf("Created at is required.")
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = u.CreatedAt
	}
	if u.RecoveryCodeUpdatedAt.IsZero() {
		u.RecoveryCodeUpdatedAt = u.UpdatedAt
	}

	var recoveryReservedUntil sql.NullInt64
	if u.RecoveryReservedUntil != nil {
		recoveryReservedUntil = sql.NullInt64{Int64: sqliteutil.ToMillis(u.RecoveryReservedUntil.UTC()), Valid: true}
	}

	return db.PutUserParams{
		ID:                        userID,
		Username:                  username,
		Locale:                    platformi18n.LocaleString(platformi18n.NormalizeLocale(u.Locale)),
		RecoveryCodeHash:          strings.TrimSpace(u.RecoveryCodeHash),
		RecoveryReservedSessionID: strings.TrimSpace(u.RecoveryReservedSessionID),
		RecoveryReservedUntil:     recoveryReservedUntil,
		RecoveryCodeUpdatedAt:     sqliteutil.ToMillis(u.RecoveryCodeUpdatedAt.UTC()),
		CreatedAt:                 sqliteutil.ToMillis(u.CreatedAt.UTC()),
		UpdatedAt:                 sqliteutil.ToMillis(u.UpdatedAt.UTC()),
	}, nil
}

func (s *Store) GetAuthStatistics(ctx context.Context, since *time.Time) (storage.AuthStatistics, error) {
	if err := ctx.Err(); err != nil {
		return storage.AuthStatistics{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.AuthStatistics{}, fmt.Errorf("Storage is not configured.")
	}

	var sinceValue any
	if since != nil {
		sinceValue = sqliteutil.ToMillis(*since)
	}

	var count int64
	row := s.sqlDB.QueryRowContext(ctx, authStatisticsQuery, sinceValue)
	if err := row.Scan(&count); err != nil {
		return storage.AuthStatistics{}, fmt.Errorf("Get auth statistics: %w", err)
	}

	return storage.AuthStatistics{UserCount: count}, nil
}
