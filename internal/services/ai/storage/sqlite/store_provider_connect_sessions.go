package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providerconnect"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func (s *Store) PutProviderConnectSession(ctx context.Context, session providerconnect.Session) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if session.ID == "" {
		return fmt.Errorf("connect session id is required")
	}
	if session.OwnerUserID == "" {
		return fmt.Errorf("owner user id is required")
	}
	if session.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if session.Status == "" {
		return fmt.Errorf("status is required")
	}
	if session.StateHash == "" {
		return fmt.Errorf("state hash is required")
	}
	if session.CodeVerifierCiphertext == "" {
		return fmt.Errorf("code verifier ciphertext is required")
	}
	if session.ExpiresAt.IsZero() {
		return fmt.Errorf("expires at is required")
	}
	scopesJSON, err := encodeScopes(session.RequestedScopes)
	if err != nil {
		return err
	}
	var completedAt sql.NullInt64
	if session.CompletedAt != nil {
		completedAt = sql.NullInt64{Int64: sqliteutil.ToMillis(*session.CompletedAt), Valid: true}
	}

	_, err = s.sqlDB.ExecContext(ctx, `
INSERT INTO ai_provider_connect_sessions (
	id, owner_user_id, provider, status, requested_scopes, state_hash, code_verifier_ciphertext, created_at, updated_at, expires_at, completed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	owner_user_id = excluded.owner_user_id,
	provider = excluded.provider,
	status = excluded.status,
	requested_scopes = excluded.requested_scopes,
	state_hash = excluded.state_hash,
	code_verifier_ciphertext = excluded.code_verifier_ciphertext,
	updated_at = excluded.updated_at,
	expires_at = excluded.expires_at,
	completed_at = excluded.completed_at
`,
		session.ID,
		session.OwnerUserID,
		session.Provider,
		session.Status,
		scopesJSON,
		session.StateHash,
		session.CodeVerifierCiphertext,
		sqliteutil.ToMillis(session.CreatedAt),
		sqliteutil.ToMillis(session.UpdatedAt),
		sqliteutil.ToMillis(session.ExpiresAt),
		completedAt,
	)
	if err != nil {
		return fmt.Errorf("put provider connect session: %w", err)
	}
	return nil
}

// GetProviderConnectSession fetches one provider connect session by ID.
func (s *Store) GetProviderConnectSession(ctx context.Context, connectSessionID string) (providerconnect.Session, error) {
	if err := ctx.Err(); err != nil {
		return providerconnect.Session{}, err
	}
	if s == nil || s.sqlDB == nil {
		return providerconnect.Session{}, fmt.Errorf("storage is not configured")
	}
	if connectSessionID == "" {
		return providerconnect.Session{}, fmt.Errorf("connect session id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT id, owner_user_id, provider, status, requested_scopes, state_hash, code_verifier_ciphertext, created_at, updated_at, expires_at, completed_at
FROM ai_provider_connect_sessions
WHERE id = ?
`, connectSessionID)

	rec, err := scanProviderConnectSession(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return providerconnect.Session{}, storage.ErrNotFound
		}
		return providerconnect.Session{}, fmt.Errorf("get provider connect session: %w", err)
	}
	return rec, nil
}

// CompleteProviderConnectSession marks one connect session as completed.
func (s *Store) CompleteProviderConnectSession(ctx context.Context, session providerconnect.Session) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if err := completeProviderConnectSession(ctx, s.sqlDB, session); err != nil {
		return err
	}
	return nil
}

func completeProviderConnectSession(ctx context.Context, execer sqlExecer, session providerconnect.Session) error {
	if session.OwnerUserID == "" {
		return fmt.Errorf("owner user id is required")
	}
	if session.ID == "" {
		return fmt.Errorf("connect session id is required")
	}
	if session.Status != providerconnect.StatusCompleted {
		return fmt.Errorf("connect session must be completed")
	}
	if session.CompletedAt == nil {
		return fmt.Errorf("completed at is required")
	}
	res, err := execer.ExecContext(ctx, `
UPDATE ai_provider_connect_sessions
SET status = 'completed', updated_at = ?, completed_at = ?
WHERE owner_user_id = ? AND id = ? AND status = 'pending'
`, sqliteutil.ToMillis(session.UpdatedAt), sqliteutil.ToMillis(session.CompletedAt.UTC()), session.OwnerUserID, session.ID)
	if err != nil {
		return fmt.Errorf("complete provider connect session: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("complete provider connect session rows affected: %w", err)
	}
	if affected == 0 {
		return storage.ErrNotFound
	}
	return nil
}
