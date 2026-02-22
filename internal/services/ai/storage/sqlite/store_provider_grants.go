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

func (s *Store) PutProviderGrant(ctx context.Context, record storage.ProviderGrantRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(record.ID) == "" {
		return fmt.Errorf("provider grant id is required")
	}
	if strings.TrimSpace(record.OwnerUserID) == "" {
		return fmt.Errorf("owner user id is required")
	}
	if strings.TrimSpace(record.Provider) == "" {
		return fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(record.TokenCiphertext) == "" {
		return fmt.Errorf("token ciphertext is required")
	}
	// TokenCiphertext must be pre-sealed by the service layer.
	if strings.TrimSpace(record.Status) == "" {
		return fmt.Errorf("status is required")
	}
	scopesJSON, err := encodeScopes(record.GrantedScopes)
	if err != nil {
		return err
	}

	var revokedAt sql.NullInt64
	if record.RevokedAt != nil {
		revokedAt = sql.NullInt64{Int64: toMillis(*record.RevokedAt), Valid: true}
	}
	var expiresAt sql.NullInt64
	if record.ExpiresAt != nil {
		expiresAt = sql.NullInt64{Int64: toMillis(*record.ExpiresAt), Valid: true}
	}
	var lastRefreshedAt sql.NullInt64
	if record.LastRefreshedAt != nil {
		lastRefreshedAt = sql.NullInt64{Int64: toMillis(*record.LastRefreshedAt), Valid: true}
	}

	_, err = s.sqlDB.ExecContext(ctx, `
INSERT INTO ai_provider_grants (
	id, owner_user_id, provider, granted_scopes, token_ciphertext, refresh_supported, status, last_refresh_error, created_at, updated_at, revoked_at, expires_at, last_refreshed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	owner_user_id = excluded.owner_user_id,
	provider = excluded.provider,
	granted_scopes = excluded.granted_scopes,
	token_ciphertext = excluded.token_ciphertext,
	refresh_supported = excluded.refresh_supported,
	status = excluded.status,
	last_refresh_error = excluded.last_refresh_error,
	updated_at = excluded.updated_at,
	revoked_at = excluded.revoked_at,
	expires_at = excluded.expires_at,
	last_refreshed_at = excluded.last_refreshed_at
`,
		record.ID,
		record.OwnerUserID,
		record.Provider,
		scopesJSON,
		record.TokenCiphertext,
		record.RefreshSupported,
		record.Status,
		record.LastRefreshError,
		toMillis(record.CreatedAt),
		toMillis(record.UpdatedAt),
		revokedAt,
		expiresAt,
		lastRefreshedAt,
	)
	if err != nil {
		return fmt.Errorf("put provider grant: %w", err)
	}
	return nil
}

// GetProviderGrant fetches a provider grant record by ID.
func (s *Store) GetProviderGrant(ctx context.Context, providerGrantID string) (storage.ProviderGrantRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.ProviderGrantRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("storage is not configured")
	}
	providerGrantID = strings.TrimSpace(providerGrantID)
	if providerGrantID == "" {
		return storage.ProviderGrantRecord{}, fmt.Errorf("provider grant id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT id, owner_user_id, provider, granted_scopes, token_ciphertext, refresh_supported, status, last_refresh_error, created_at, updated_at, revoked_at, expires_at, last_refreshed_at
FROM ai_provider_grants
WHERE id = ?
`, providerGrantID)

	rec, err := scanProviderGrantRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ProviderGrantRecord{}, storage.ErrNotFound
		}
		return storage.ProviderGrantRecord{}, fmt.Errorf("get provider grant: %w", err)
	}
	return rec, nil
}

// ListProviderGrantsByOwner returns a page of provider grants for one owner.
func (s *Store) ListProviderGrantsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string, filter storage.ProviderGrantFilter) (storage.ProviderGrantPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.ProviderGrantPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.ProviderGrantPage{}, fmt.Errorf("storage is not configured")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return storage.ProviderGrantPage{}, fmt.Errorf("owner user id is required")
	}
	if pageSize <= 0 {
		return storage.ProviderGrantPage{}, fmt.Errorf("page size must be greater than zero")
	}
	provider := strings.ToLower(strings.TrimSpace(filter.Provider))
	status := strings.ToLower(strings.TrimSpace(filter.Status))

	limit := pageSize + 1
	pageToken = strings.TrimSpace(pageToken)
	whereParts := []string{"owner_user_id = ?"}
	args := []any{ownerUserID}
	if provider != "" {
		whereParts = append(whereParts, "provider = ?")
		args = append(args, provider)
	}
	if status != "" {
		whereParts = append(whereParts, "status = ?")
		args = append(args, status)
	}
	if pageToken != "" {
		whereParts = append(whereParts, "id > ?")
		args = append(args, pageToken)
	}
	args = append(args, limit)

	// Owner scope is always anchored in WHERE before optional filters so caller
	// input can only narrow visibility for that owner.
	query := fmt.Sprintf(`
SELECT id, owner_user_id, provider, granted_scopes, token_ciphertext, refresh_supported, status, last_refresh_error, created_at, updated_at, revoked_at, expires_at, last_refreshed_at
FROM ai_provider_grants
WHERE %s
ORDER BY id
LIMIT ?
`, strings.Join(whereParts, " AND "))
	rows, err := s.sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		return storage.ProviderGrantPage{}, fmt.Errorf("list provider grants: %w", err)
	}
	defer rows.Close()

	page := storage.ProviderGrantPage{ProviderGrants: make([]storage.ProviderGrantRecord, 0, pageSize)}
	for rows.Next() {
		rec, err := scanProviderGrantRows(rows)
		if err != nil {
			return storage.ProviderGrantPage{}, fmt.Errorf("scan provider grant row: %w", err)
		}
		page.ProviderGrants = append(page.ProviderGrants, rec)
	}
	if err := rows.Err(); err != nil {
		return storage.ProviderGrantPage{}, fmt.Errorf("iterate provider grant rows: %w", err)
	}

	if len(page.ProviderGrants) > pageSize {
		page.NextPageToken = page.ProviderGrants[pageSize-1].ID
		page.ProviderGrants = page.ProviderGrants[:pageSize]
	}
	return page, nil
}

// RevokeProviderGrant marks a provider grant as revoked.
func (s *Store) RevokeProviderGrant(ctx context.Context, ownerUserID string, providerGrantID string, revokedAt time.Time) error {
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
	providerGrantID = strings.TrimSpace(providerGrantID)
	if providerGrantID == "" {
		return fmt.Errorf("provider grant id is required")
	}

	updatedAt := revokedAt.UTC()
	res, err := s.sqlDB.ExecContext(ctx, `
UPDATE ai_provider_grants
SET status = 'revoked', updated_at = ?, revoked_at = ?
WHERE owner_user_id = ? AND id = ?
`, toMillis(updatedAt), toMillis(revokedAt.UTC()), ownerUserID, providerGrantID)
	if err != nil {
		return fmt.Errorf("revoke provider grant: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke provider grant rows affected: %w", err)
	}
	if affected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// UpdateProviderGrantToken updates token ciphertext and refresh metadata.
func (s *Store) UpdateProviderGrantToken(ctx context.Context, ownerUserID string, providerGrantID string, tokenCiphertext string, refreshedAt time.Time, expiresAt *time.Time, status string, lastRefreshError string) error {
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
	providerGrantID = strings.TrimSpace(providerGrantID)
	if providerGrantID == "" {
		return fmt.Errorf("provider grant id is required")
	}
	tokenCiphertext = strings.TrimSpace(tokenCiphertext)
	if tokenCiphertext == "" {
		return fmt.Errorf("token ciphertext is required")
	}
	status = strings.TrimSpace(status)
	if status == "" {
		return fmt.Errorf("status is required")
	}

	var expiresAtValue sql.NullInt64
	if expiresAt != nil {
		expiresAtValue = sql.NullInt64{Int64: toMillis(expiresAt.UTC()), Valid: true}
	}
	res, err := s.sqlDB.ExecContext(ctx, `
UPDATE ai_provider_grants
SET token_ciphertext = ?, status = ?, last_refresh_error = ?, updated_at = ?, expires_at = ?, last_refreshed_at = ?
WHERE owner_user_id = ? AND id = ?
`, tokenCiphertext, status, strings.TrimSpace(lastRefreshError), toMillis(refreshedAt.UTC()), expiresAtValue, toMillis(refreshedAt.UTC()), ownerUserID, providerGrantID)
	if err != nil {
		return fmt.Errorf("update provider grant token: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update provider grant token rows affected: %w", err)
	}
	if affected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// PutProviderConnectSession persists a provider connect session record.
