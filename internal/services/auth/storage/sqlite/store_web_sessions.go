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
		return fmt.Errorf("context is required")
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(session.ID) == "" {
		return fmt.Errorf("session id is required")
	}
	if strings.TrimSpace(session.UserID) == "" {
		return fmt.Errorf("user id is required")
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now().UTC()
	}
	if session.ExpiresAt.IsZero() {
		return fmt.Errorf("expires at is required")
	}
	var revokedAt sql.NullInt64
	if session.RevokedAt != nil {
		revokedAt = sql.NullInt64{Int64: toMillis(session.RevokedAt.UTC()), Valid: true}
	}
	_, err := s.sqlDB.ExecContext(ctx, `
INSERT INTO web_sessions (id, user_id, created_at, expires_at, revoked_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  user_id = excluded.user_id,
  created_at = excluded.created_at,
  expires_at = excluded.expires_at,
  revoked_at = excluded.revoked_at
`, session.ID, session.UserID, toMillis(session.CreatedAt.UTC()), toMillis(session.ExpiresAt.UTC()), revokedAt)
	if err != nil {
		return fmt.Errorf("put web session: %w", err)
	}
	return nil
}

// GetWebSession returns a durable authenticated web session by id.
func (s *Store) GetWebSession(ctx context.Context, id string) (storage.WebSession, error) {
	if ctx == nil {
		return storage.WebSession{}, fmt.Errorf("context is required")
	}
	if s == nil || s.sqlDB == nil {
		return storage.WebSession{}, fmt.Errorf("storage is not configured")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return storage.WebSession{}, fmt.Errorf("session id is required")
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
		return storage.WebSession{}, fmt.Errorf("get web session: %w", err)
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
		return fmt.Errorf("context is required")
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("session id is required")
	}
	if revokedAt.IsZero() {
		revokedAt = time.Now().UTC()
	}
	res, err := s.sqlDB.ExecContext(ctx, `UPDATE web_sessions SET revoked_at = ? WHERE id = ?`, toMillis(revokedAt.UTC()), id)
	if err != nil {
		return fmt.Errorf("revoke web session: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke web session rows affected: %w", err)
	}
	if affected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// DeleteExpiredWebSessions removes expired durable web sessions.
func (s *Store) DeleteExpiredWebSessions(ctx context.Context, now time.Time) error {
	if ctx == nil {
		return fmt.Errorf("context is required")
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	_, err := s.sqlDB.ExecContext(ctx, `DELETE FROM web_sessions WHERE expires_at <= ?`, toMillis(now.UTC()))
	if err != nil {
		return fmt.Errorf("delete expired web sessions: %w", err)
	}
	return nil
}
