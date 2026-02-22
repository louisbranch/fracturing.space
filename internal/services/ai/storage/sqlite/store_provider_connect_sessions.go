package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func (s *Store) PutProviderConnectSession(ctx context.Context, record storage.ProviderConnectSessionRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(record.ID) == "" {
		return fmt.Errorf("connect session id is required")
	}
	if strings.TrimSpace(record.OwnerUserID) == "" {
		return fmt.Errorf("owner user id is required")
	}
	if strings.TrimSpace(record.Provider) == "" {
		return fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(record.Status) == "" {
		return fmt.Errorf("status is required")
	}
	if strings.TrimSpace(record.StateHash) == "" {
		return fmt.Errorf("state hash is required")
	}
	if strings.TrimSpace(record.CodeVerifierCiphertext) == "" {
		return fmt.Errorf("code verifier ciphertext is required")
	}
	if record.ExpiresAt.IsZero() {
		return fmt.Errorf("expires at is required")
	}
	scopesJSON, err := encodeScopes(record.RequestedScopes)
	if err != nil {
		return err
	}
	var completedAt sql.NullInt64
	if record.CompletedAt != nil {
		completedAt = sql.NullInt64{Int64: toMillis(*record.CompletedAt), Valid: true}
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
		record.ID,
		record.OwnerUserID,
		record.Provider,
		record.Status,
		scopesJSON,
		record.StateHash,
		record.CodeVerifierCiphertext,
		toMillis(record.CreatedAt),
		toMillis(record.UpdatedAt),
		toMillis(record.ExpiresAt),
		completedAt,
	)
	if err != nil {
		return fmt.Errorf("put provider connect session: %w", err)
	}
	return nil
}

// GetProviderConnectSession fetches one provider connect session by ID.
func (s *Store) GetProviderConnectSession(ctx context.Context, connectSessionID string) (storage.ProviderConnectSessionRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.ProviderConnectSessionRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.ProviderConnectSessionRecord{}, fmt.Errorf("storage is not configured")
	}
	connectSessionID = strings.TrimSpace(connectSessionID)
	if connectSessionID == "" {
		return storage.ProviderConnectSessionRecord{}, fmt.Errorf("connect session id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT id, owner_user_id, provider, status, requested_scopes, state_hash, code_verifier_ciphertext, created_at, updated_at, expires_at, completed_at
FROM ai_provider_connect_sessions
WHERE id = ?
`, connectSessionID)

	rec, err := scanProviderConnectSessionRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ProviderConnectSessionRecord{}, storage.ErrNotFound
		}
		return storage.ProviderConnectSessionRecord{}, fmt.Errorf("get provider connect session: %w", err)
	}
	return rec, nil
}

// CompleteProviderConnectSession marks one connect session as completed.
func (s *Store) CompleteProviderConnectSession(ctx context.Context, ownerUserID string, connectSessionID string, completedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return fmt.Errorf("owner user id is required")
	}
	connectSessionID = strings.TrimSpace(connectSessionID)
	if connectSessionID == "" {
		return fmt.Errorf("connect session id is required")
	}

	updatedAt := completedAt.UTC()
	res, err := s.sqlDB.ExecContext(ctx, `
UPDATE ai_provider_connect_sessions
SET status = 'completed', updated_at = ?, completed_at = ?
WHERE owner_user_id = ? AND id = ? AND status = 'pending'
`, toMillis(updatedAt), toMillis(completedAt.UTC()), ownerUserID, connectSessionID)
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
