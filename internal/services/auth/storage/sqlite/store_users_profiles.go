package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite/db"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
)

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
