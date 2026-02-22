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

// PutCredential persists a credential record.
func (s *Store) PutCredential(ctx context.Context, record storage.CredentialRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(record.ID) == "" {
		return fmt.Errorf("credential id is required")
	}
	if strings.TrimSpace(record.OwnerUserID) == "" {
		return fmt.Errorf("owner user id is required")
	}
	if strings.TrimSpace(record.Provider) == "" {
		return fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(record.Label) == "" {
		return fmt.Errorf("label is required")
	}
	if strings.TrimSpace(record.SecretCiphertext) == "" {
		return fmt.Errorf("secret ciphertext is required")
	}
	// SecretCiphertext is expected to already be sealed by the service layer.
	if strings.TrimSpace(record.Status) == "" {
		return fmt.Errorf("status is required")
	}

	var revokedAt sql.NullInt64
	if record.RevokedAt != nil {
		revokedAt = sql.NullInt64{Int64: toMillis(*record.RevokedAt), Valid: true}
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
		record.ID,
		record.OwnerUserID,
		record.Provider,
		record.Label,
		record.SecretCiphertext,
		record.Status,
		toMillis(record.CreatedAt),
		toMillis(record.UpdatedAt),
		revokedAt,
	)
	if err != nil {
		return fmt.Errorf("put credential: %w", err)
	}
	return nil
}

// GetCredential fetches a credential record by ID.
func (s *Store) GetCredential(ctx context.Context, credentialID string) (storage.CredentialRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.CredentialRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.CredentialRecord{}, fmt.Errorf("storage is not configured")
	}
	credentialID = strings.TrimSpace(credentialID)
	if credentialID == "" {
		return storage.CredentialRecord{}, fmt.Errorf("credential id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT id, owner_user_id, provider, label, secret_ciphertext, status, created_at, updated_at, revoked_at
FROM ai_credentials
WHERE id = ?
`, credentialID)

	rec, err := scanCredentialRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.CredentialRecord{}, storage.ErrNotFound
		}
		return storage.CredentialRecord{}, fmt.Errorf("get credential: %w", err)
	}
	return rec, nil
}

// ListCredentialsByOwner returns a page of credential records for one owner.
func (s *Store) ListCredentialsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string) (storage.CredentialPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.CredentialPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.CredentialPage{}, fmt.Errorf("storage is not configured")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return storage.CredentialPage{}, fmt.Errorf("owner user id is required")
	}
	if pageSize <= 0 {
		return storage.CredentialPage{}, fmt.Errorf("page size must be greater than zero")
	}

	limit := pageSize + 1
	var (
		rows *sql.Rows
		err  error
	)
	if strings.TrimSpace(pageToken) == "" {
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
`, ownerUserID, strings.TrimSpace(pageToken), limit)
	}
	if err != nil {
		return storage.CredentialPage{}, fmt.Errorf("list credentials: %w", err)
	}
	defer rows.Close()

	page := storage.CredentialPage{Credentials: make([]storage.CredentialRecord, 0, pageSize)}
	for rows.Next() {
		rec, err := scanCredentialRecord(rows)
		if err != nil {
			return storage.CredentialPage{}, fmt.Errorf("scan credential row: %w", err)
		}
		page.Credentials = append(page.Credentials, rec)
	}
	if err := rows.Err(); err != nil {
		return storage.CredentialPage{}, fmt.Errorf("iterate credential rows: %w", err)
	}

	if len(page.Credentials) > pageSize {
		page.NextPageToken = page.Credentials[pageSize-1].ID
		page.Credentials = page.Credentials[:pageSize]
	}
	return page, nil
}

// RevokeCredential marks a credential as revoked.
func (s *Store) RevokeCredential(ctx context.Context, ownerUserID string, credentialID string, revokedAt time.Time) error {
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
	credentialID = strings.TrimSpace(credentialID)
	if credentialID == "" {
		return fmt.Errorf("credential id is required")
	}

	// Revocation is a lifecycle state change; ciphertext is retained for audit
	// history and is no longer considered usable by service-level checks.
	updatedAt := revokedAt.UTC()
	res, err := s.sqlDB.ExecContext(ctx, `
UPDATE ai_credentials
SET status = 'revoked', updated_at = ?, revoked_at = ?
WHERE owner_user_id = ? AND id = ?
`, toMillis(updatedAt), toMillis(revokedAt.UTC()), ownerUserID, credentialID)
	if err != nil {
		return fmt.Errorf("revoke credential: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke credential rows affected: %w", err)
	}
	if affected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func scanCredentialRecord(scanner interface{ Scan(...any) error }) (storage.CredentialRecord, error) {
	var rec storage.CredentialRecord
	var createdAt int64
	var updatedAt int64
	var revokedAt sql.NullInt64
	if err := scanner.Scan(
		&rec.ID,
		&rec.OwnerUserID,
		&rec.Provider,
		&rec.Label,
		&rec.SecretCiphertext,
		&rec.Status,
		&createdAt,
		&updatedAt,
		&revokedAt,
	); err != nil {
		return storage.CredentialRecord{}, err
	}

	rec.CreatedAt = fromMillis(createdAt)
	rec.UpdatedAt = fromMillis(updatedAt)
	if revokedAt.Valid {
		value := fromMillis(revokedAt.Int64)
		rec.RevokedAt = &value
	}
	return rec, nil
}
