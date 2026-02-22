package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite/db"
)

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
