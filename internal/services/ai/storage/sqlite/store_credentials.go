package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// PutCredential persists a credential.
func (s *Store) PutCredential(ctx context.Context, c credential.Credential) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if c.ID == "" {
		return fmt.Errorf("credential id is required")
	}
	if c.OwnerUserID == "" {
		return fmt.Errorf("owner user id is required")
	}
	if c.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if c.Label == "" {
		return fmt.Errorf("label is required")
	}
	if c.SecretCiphertext == "" {
		return fmt.Errorf("secret ciphertext is required")
	}
	if c.Status == "" {
		return fmt.Errorf("status is required")
	}

	var revokedAt sql.NullInt64
	if c.RevokedAt != nil {
		revokedAt = sql.NullInt64{Int64: sqliteutil.ToMillis(*c.RevokedAt), Valid: true}
	}

	_, err := s.sqlDB.ExecContext(ctx, `
INSERT INTO ai_credentials (
	id, owner_user_id, provider, label, secret_ciphertext, status, created_at, updated_at, revoked_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	owner_user_id = excluded.owner_user_id,
	provider = excluded.provider,
	label = excluded.label,
	secret_ciphertext = excluded.secret_ciphertext,
	status = excluded.status,
	updated_at = excluded.updated_at,
	revoked_at = excluded.revoked_at
`,
		c.ID,
		c.OwnerUserID,
		string(c.Provider),
		c.Label,
		c.SecretCiphertext,
		string(c.Status),
		sqliteutil.ToMillis(c.CreatedAt),
		sqliteutil.ToMillis(c.UpdatedAt),
		revokedAt,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return storage.ErrConflict
		}
		return fmt.Errorf("put credential: %w", err)
	}
	return nil
}

// GetCredential fetches a credential by ID.
func (s *Store) GetCredential(ctx context.Context, credentialID string) (credential.Credential, error) {
	if err := ctx.Err(); err != nil {
		return credential.Credential{}, err
	}
	if s == nil || s.sqlDB == nil {
		return credential.Credential{}, fmt.Errorf("storage is not configured")
	}
	if credentialID == "" {
		return credential.Credential{}, fmt.Errorf("credential id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT id, owner_user_id, provider, label, secret_ciphertext, status, created_at, updated_at, revoked_at
FROM ai_credentials
WHERE id = ?
`, credentialID)

	c, err := scanCredential(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return credential.Credential{}, storage.ErrNotFound
		}
		return credential.Credential{}, fmt.Errorf("get credential: %w", err)
	}
	return c, nil
}

// ListCredentialsByOwner returns a page of credentials for one owner.
func (s *Store) ListCredentialsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string) (credential.Page, error) {
	if err := ctx.Err(); err != nil {
		return credential.Page{}, err
	}
	if s == nil || s.sqlDB == nil {
		return credential.Page{}, fmt.Errorf("storage is not configured")
	}
	if ownerUserID == "" {
		return credential.Page{}, fmt.Errorf("owner user id is required")
	}
	if pageSize <= 0 {
		return credential.Page{}, fmt.Errorf("page size must be greater than zero")
	}

	limit := pageSize + 1
	var (
		rows *sql.Rows
		err  error
	)
	if pageToken == "" {
		rows, err = s.sqlDB.QueryContext(ctx, `
SELECT id, owner_user_id, provider, label, secret_ciphertext, status, created_at, updated_at, revoked_at
FROM ai_credentials
WHERE owner_user_id = ?
ORDER BY id
LIMIT ?
`, ownerUserID, limit)
	} else {
		rows, err = s.sqlDB.QueryContext(ctx, `
SELECT id, owner_user_id, provider, label, secret_ciphertext, status, created_at, updated_at, revoked_at
FROM ai_credentials
WHERE owner_user_id = ? AND id > ?
ORDER BY id
LIMIT ?
`, ownerUserID, pageToken, limit)
	}
	if err != nil {
		return credential.Page{}, fmt.Errorf("list credentials: %w", err)
	}
	defer rows.Close()

	page := credential.Page{Credentials: make([]credential.Credential, 0, pageSize)}
	for rows.Next() {
		c, err := scanCredential(rows)
		if err != nil {
			return credential.Page{}, fmt.Errorf("scan credential row: %w", err)
		}
		page.Credentials = append(page.Credentials, c)
	}
	if err := rows.Err(); err != nil {
		return credential.Page{}, fmt.Errorf("iterate credential rows: %w", err)
	}

	if len(page.Credentials) > pageSize {
		page.NextPageToken = page.Credentials[pageSize-1].ID
		page.Credentials = page.Credentials[:pageSize]
	}
	return page, nil
}

// scanCredential reads one credential row into a domain credential.
func scanCredential(s scanner) (credential.Credential, error) {
	var (
		c           credential.Credential
		providerStr string
		statusStr   string
		createdAt   int64
		updatedAt   int64
		revokedAt   sql.NullInt64
	)
	if err := s.Scan(
		&c.ID,
		&c.OwnerUserID,
		&providerStr,
		&c.Label,
		&c.SecretCiphertext,
		&statusStr,
		&createdAt,
		&updatedAt,
		&revokedAt,
	); err != nil {
		return credential.Credential{}, err
	}

	c.Provider, _ = provider.Normalize(providerStr)
	c.Status = credential.ParseStatus(statusStr)
	c.CreatedAt = sqliteutil.FromMillis(createdAt)
	c.UpdatedAt = sqliteutil.FromMillis(updatedAt)
	if revokedAt.Valid {
		value := sqliteutil.FromMillis(revokedAt.Int64)
		c.RevokedAt = &value
	}
	return c, nil
}
