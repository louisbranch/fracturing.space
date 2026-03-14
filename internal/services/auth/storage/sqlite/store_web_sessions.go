package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
)

// PutWebSession stores a durable authenticated web session.
func (s *Store) PutWebSession(ctx context.Context, session storage.WebSession) error {
	if ctx == nil {
		return fmt.Errorf("Context is required.")
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("Storage is not configured.")
	}
	return putWebSessionWithExecutor(ctx, s.sqlDB, session)
}

func normalizeWebSession(session storage.WebSession) (storage.WebSession, error) {
	if strings.TrimSpace(session.ID) == "" {
		return storage.WebSession{}, fmt.Errorf("Session ID is required.")
	}
	if strings.TrimSpace(session.UserID) == "" {
		return storage.WebSession{}, fmt.Errorf("User ID is required.")
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now().UTC()
	}
	if session.ExpiresAt.IsZero() {
		return storage.WebSession{}, fmt.Errorf("Expires at is required.")
	}
	return session, nil
}

func putWebSessionWithExecutor(ctx context.Context, exec execContexter, session storage.WebSession) error {
	normalized, err := normalizeWebSession(session)
	if err != nil {
		return err
	}
	var revokedAt sql.NullInt64
	if normalized.RevokedAt != nil {
		revokedAt = sql.NullInt64{Int64: toMillis(normalized.RevokedAt.UTC()), Valid: true}
	}
	_, err = exec.ExecContext(ctx, `
INSERT INTO web_sessions (id, user_id, created_at, expires_at, revoked_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  user_id = excluded.user_id,
  created_at = excluded.created_at,
  expires_at = excluded.expires_at,
  revoked_at = excluded.revoked_at
`, normalized.ID, normalized.UserID, toMillis(normalized.CreatedAt.UTC()), toMillis(normalized.ExpiresAt.UTC()), revokedAt)
	if err != nil {
		return fmt.Errorf("Put web session: %w", err)
	}
	return nil
}

// GetWebSession returns a durable authenticated web session by id.
func (s *Store) GetWebSession(ctx context.Context, id string) (storage.WebSession, error) {
	if ctx == nil {
		return storage.WebSession{}, fmt.Errorf("Context is required.")
	}
	if s == nil || s.sqlDB == nil {
		return storage.WebSession{}, fmt.Errorf("Storage is not configured.")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return storage.WebSession{}, fmt.Errorf("Session ID is required.")
	}
	row := s.sqlDB.QueryRowContext(ctx, `
SELECT id, user_id, created_at, expires_at, revoked_at
FROM web_sessions
WHERE id = ?
`, id)
	var session storage.WebSession
	var createdAt int64
	var expiresAt int64
	var revokedAt sql.NullInt64
	if err := row.Scan(&session.ID, &session.UserID, &createdAt, &expiresAt, &revokedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.WebSession{}, storage.ErrNotFound
		}
		return storage.WebSession{}, fmt.Errorf("Get web session: %w", err)
	}
	session.CreatedAt = fromMillis(createdAt)
	session.ExpiresAt = fromMillis(expiresAt)
	if revokedAt.Valid {
		value := fromMillis(revokedAt.Int64)
		session.RevokedAt = &value
	}
	return session, nil
}

// RevokeWebSession marks a durable session as revoked.
func (s *Store) RevokeWebSession(ctx context.Context, id string, revokedAt time.Time) error {
	if ctx == nil {
		return fmt.Errorf("Context is required.")
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("Storage is not configured.")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("Session ID is required.")
	}
	if revokedAt.IsZero() {
		revokedAt = time.Now().UTC()
	}
	res, err := s.sqlDB.ExecContext(ctx, `UPDATE web_sessions SET revoked_at = ? WHERE id = ?`, toMillis(revokedAt.UTC()), id)
	if err != nil {
		return fmt.Errorf("Revoke web session: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("Revoke web session rows affected: %w", err)
	}
	if affected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// RevokeWebSessionsByUser marks all durable sessions for a user as revoked.
func (s *Store) RevokeWebSessionsByUser(ctx context.Context, userID string, revokedAt time.Time) error {
	if ctx == nil {
		return fmt.Errorf("Context is required.")
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("Storage is not configured.")
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return fmt.Errorf("User ID is required.")
	}
	if revokedAt.IsZero() {
		revokedAt = time.Now().UTC()
	}
	_, err := s.sqlDB.ExecContext(ctx, `UPDATE web_sessions SET revoked_at = ? WHERE user_id = ?`, toMillis(revokedAt.UTC()), userID)
	if err != nil {
		return fmt.Errorf("Revoke web sessions by user: %w", err)
	}
	return nil
}

// DeleteExpiredWebSessions removes expired durable web sessions.
func (s *Store) DeleteExpiredWebSessions(ctx context.Context, now time.Time) error {
	if ctx == nil {
		return fmt.Errorf("Context is required.")
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("Storage is not configured.")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	_, err := s.sqlDB.ExecContext(ctx, `DELETE FROM web_sessions WHERE expires_at <= ?`, toMillis(now.UTC()))
	if err != nil {
		return fmt.Errorf("Delete expired web sessions: %w", err)
	}
	return nil
}
